package server

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/0xPolygonHermez/zkevm-bridge-service/bridgectrl/pb"
	"github.com/0xPolygonHermez/zkevm-bridge-service/log"
	"github.com/0xPolygonHermez/zkevm-bridge-service/nacos"
	"github.com/alibaba/sentinel-golang/core/base"
	sentinelGrpc "github.com/alibaba/sentinel-golang/pkg/adapters/grpc"
	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
)

const (
	bridgeEndpointPath = "/priapi/v1/ob/bridge"
)

func RegisterNacos(cfg nacos.Config) {
	log.Info(fmt.Sprintf("nacos config NacosUrls %s NamespaceId %s ApplicationName %s ExternalListenAddr %s", cfg.NacosUrls, cfg.NamespaceId, cfg.ApplicationName, cfg.ExternalListenAddr))
	if cfg.NacosUrls != "" {
		nacos.InitNacosClient(cfg.NacosUrls, cfg.NamespaceId, cfg.ApplicationName, cfg.ExternalListenAddr)
	}
}

// RunServer runs gRPC server and HTTP gateway
func RunServer(cfg Config, bridgeService pb.BridgeServiceServer) error {
	ctx := context.Background()

	if len(cfg.GRPCPort) == 0 {
		return fmt.Errorf("invalid TCP port for gRPC server: '%s'", cfg.GRPCPort)
	}

	if len(cfg.HTTPPort) == 0 {
		return fmt.Errorf("invalid TCP port for HTTP gateway: '%s'", cfg.HTTPPort)
	}

	go func() {
		_ = runRestServer(ctx, cfg.GRPCPort, cfg.HTTPPort)
	}()

	go func() {
		_ = runGRPCServer(ctx, bridgeService, cfg.GRPCPort)
	}()

	return nil
}

// HealthChecker will provide an implementation of the HealthCheck interface.
type healthChecker struct{}

// NewHealthChecker returns a health checker according to standard package
// grpc.health.v1.
func newHealthChecker() *healthChecker {
	return &healthChecker{}
}

// HealthCheck interface implementation.

// Check returns the current status of the server for unary gRPC health requests,
// for now if the server is up and able to respond we will always return SERVING.
func (s *healthChecker) Check(ctx context.Context, req *grpc_health_v1.HealthCheckRequest) (*grpc_health_v1.HealthCheckResponse, error) {
	return &grpc_health_v1.HealthCheckResponse{
		Status: grpc_health_v1.HealthCheckResponse_SERVING,
	}, nil
}

// Watch returns the current status of the server for stream gRPC health requests,
// for now if the server is up and able to respond we will always return SERVING.
func (s *healthChecker) Watch(req *grpc_health_v1.HealthCheckRequest, server grpc_health_v1.Health_WatchServer) error {
	return server.Send(&grpc_health_v1.HealthCheckResponse{
		Status: grpc_health_v1.HealthCheckResponse_SERVING,
	})
}

func runGRPCServer(ctx context.Context, bridgeServer pb.BridgeServiceServer, port string) error {
	listen, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return err
	}

	// Fallback function to be triggered when there's a block error from Sentinel
	blockErrFallbackFn := func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, blockErr *base.BlockError) (interface{}, error) {
		// ResourceExhausted status will be mapped to code 429 in the HTTP transcoder
		// In the future, return more different codes based on different type of BlockError?
		return nil, status.Error(codes.ResourceExhausted, blockErr.Error())
	}

	server := grpc.NewServer(grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(
		sentinelGrpc.NewUnaryServerInterceptor(sentinelGrpc.WithUnaryServerBlockFallback(blockErrFallbackFn)),
		NewRequestMetricsInterceptor(),
		NewRequestLogInterceptor(),
	)))
	pb.RegisterBridgeServiceServer(server, bridgeServer)

	healthService := newHealthChecker()
	grpc_health_v1.RegisterHealthServer(server, healthService)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for range c {
			server.GracefulStop()
			<-ctx.Done()
		}
	}()

	log.Info("gRPC Server is serving at ", port)
	return server.Serve(listen)
}

func preflightHandler(w http.ResponseWriter, r *http.Request) {
	//headers := []string{"Content-Type", "Accept", "X-Locale", "X-Utc", "X-Zkdex-Env", "App-Type", "Referer", "User-Agent", "Devid"}
	w.Header().Set("Access-Control-Allow-Headers", "*")
	//methods := []string{"GET", "HEAD", "POST", "PUT", "DELETE"}
	w.Header().Set("Access-Control-Allow-Methods", "*")
	w.Header().Set("Access-Control-Allow-Origin", "*")
}

// allowCORS allows Cross Origin Resource Sharing from any origin.
// Don't do this without consideration in production systems.
func allowCORS(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if origin := r.Header.Get("Origin"); origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			if r.Method == "OPTIONS" && r.Header.Get("Access-Control-Request-Method") != "" {
				preflightHandler(w, r)
				return
			}
		}
		h.ServeHTTP(w, r)
	})
}

func runRestServer(ctx context.Context, grpcPort, httpPort string) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	endpoint := "localhost:" + grpcPort
	conn, err := grpc.Dial(endpoint, opts...)
	if err != nil {
		return err
	}

	muxHealthOpt := runtime.WithHealthzEndpoint(grpc_health_v1.NewHealthClient(conn))
	muxJSONOpt := runtime.WithMarshalerOption(runtime.MIMEWildcard, &runtime.JSONPb{
		MarshalOptions: protojson.MarshalOptions{
			UseProtoNames:   true,
			EmitUnpopulated: true,
		},
		UnmarshalOptions: protojson.UnmarshalOptions{
			DiscardUnknown: true,
		},
	})
	mux := runtime.NewServeMux(muxJSONOpt, muxHealthOpt)

	httpMux := http.NewServeMux()
	httpMux.Handle(bridgeEndpointPath+"/", http.StripPrefix(bridgeEndpointPath, mux))
	httpMux.Handle("/", mux)

	if err := pb.RegisterBridgeServiceHandler(ctx, mux, conn); err != nil {
		return err
	}

	srv := &http.Server{
		ReadTimeout: 1 * time.Second, //nolint:gomnd
		Addr:        ":" + httpPort,
		Handler:     allowCORS(httpMux),
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for range c {
			_ = srv.Shutdown(ctx)
			<-ctx.Done()
		}

		_, cancel := context.WithTimeout(ctx, 5*time.Second) //nolint:gomnd
		defer cancel()

		_ = srv.Shutdown(ctx)
	}()

	log.Info("Restful Server is serving at ", httpPort)
	return srv.ListenAndServe()
}
