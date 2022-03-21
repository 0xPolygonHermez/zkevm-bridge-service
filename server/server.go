package server

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/hermeznetwork/hermez-bridge/bridgetree"
	"github.com/hermeznetwork/hermez-bridge/bridgetree/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// RunServer runs gRPC server and HTTP gateway
func RunServer(storage bridgetree.BridgeServiceStorage, bridgeCtrl *bridgetree.BridgeTree, cfg Config) error {
	ctx := context.Background()

	if len(cfg.GRPCPort) == 0 {
		return fmt.Errorf("invalid TCP port for gRPC server: '%s'", cfg.GRPCPort)
	}

	if len(cfg.HTTPPort) == 0 {
		return fmt.Errorf("invalid TCP port for HTTP gateway: '%s'", cfg.HTTPPort)
	}

	bridgeService := bridgetree.NewBridgeService(storage, bridgeCtrl)

	go func() {
		_ = runRestServer(ctx, cfg.GRPCPort, cfg.HTTPPort)
	}()

	go func() {
		_ = runGRPCServer(ctx, bridgeService, cfg.GRPCPort)
	}()

	return nil
}

func runGRPCServer(ctx context.Context, bridgeServer pb.BridgeServiceServer, port string) error {
	listen, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return err
	}

	server := grpc.NewServer()
	pb.RegisterBridgeServiceServer(server, bridgeServer)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for range c {
			server.GracefulStop()
			<-ctx.Done()
		}
	}()

	return server.Serve(listen)
}

func runRestServer(ctx context.Context, grpcPort, httpPort string) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	mux := runtime.NewServeMux()
	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	if err := pb.RegisterBridgeServiceHandlerFromEndpoint(ctx, mux, "localhost:"+grpcPort, opts); err != nil {
		return err
	}

	srv := &http.Server{
		Addr:    ":" + httpPort,
		Handler: mux,
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

	return srv.ListenAndServe()
}
