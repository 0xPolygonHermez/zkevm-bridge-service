package server

import (
	"context"
	"reflect"
	"strings"
	"time"

	"github.com/0xPolygonHermez/zkevm-bridge-service/bridgectrl/pb"
	"github.com/0xPolygonHermez/zkevm-bridge-service/log"
	"github.com/0xPolygonHermez/zkevm-bridge-service/metrics"
	"github.com/0xPolygonHermez/zkevm-bridge-service/server/iprestriction"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

const (
	ipRestrictionErrorMsg = "XLayer product isn't available in your region"
)

func NewRequestLogInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		startTime := time.Now()
		methodName := info.FullMethod

		// Actual process of the request
		resp, err := handler(ctx, req)

		duration := time.Since(startTime)
		var reqJson, respJson []byte
		if req != nil {
			reqJson, _ = protojson.Marshal(req.(proto.Message))
		} else {
			reqJson = []byte("<nil>")
		}
		if resp != nil {
			respJson, _ = protojson.Marshal(resp.(proto.Message))
		} else {
			respJson = []byte("<nil>")
		}

		log.Infof("method[%v] req[%v] resp[%v] err[%v] processTime[%v]", methodName, string(reqJson), string(respJson), err, duration.String())
		return resp, err
	}
}

// NewRequestMetricsInterceptor returns a GRPC interceptor to record the request metrics to prometheus
func NewRequestMetricsInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		startTime := time.Now()
		methodName := info.FullMethod

		// Actual process of the request
		resp, err := handler(ctx, req)

		duration := time.Since(startTime)
		code, msg := getRespErrorInfo(resp, err)

		metrics.RecordRequest(methodName, code, msg)
		metrics.RecordRequestLatency(methodName, duration, code)

		return resp, err
	}
}

// getRespErrorInfo returns the error code and msg from the resp
func getRespErrorInfo(resp any, err error) (code int64, msg string) {
	if err != nil {
		return defaultErrorCode, err.Error()
	}

	if resp == nil {
		// This should not happen
		return defaultSuccessCode, ""
	}

	// Check `Msg" field in the resp body
	v := reflect.Indirect(reflect.ValueOf(resp))
	codeField := v.FieldByName("Code")
	msgField := v.FieldByName("Msg")

	if codeField.CanInt() {
		code = codeField.Int()
	}
	if codeField.CanUint() {
		code = int64(codeField.Uint())
	}
	if msgField.Kind() == reflect.String {
		msg = msgField.String()
	}

	return
}

func NewIPCheckInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		headers, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			log.Warnf("cannot get headers from incoming context, skipped checking IP")
			return handler(ctx, req)
		}
		ip := getIPAddrFromHeaders(headers)
		log.Debugf("method[%v] client IP: %v", info.FullMethod, ip)
		if ip != "" && iprestriction.GetClient().CheckIPRestricted(ip) {
			// IP is restricted, need to block the request
			return nil, status.Error(codes.Code(pb.ErrorCode_ERROR_IP_RESTRICTED), ipRestrictionErrorMsg)
		}

		// Not restricted, continue the flow
		return handler(ctx, req)
	}
}

func getIPAddrFromHeaders(headers metadata.MD) string {
	ipHeaders := []string{"x-real-ip", "x-forwarded-for", "Proxy-Client-IP", "WL-Proxy-Client-IP"}

	// Check each header in order
	for _, h := range ipHeaders {
		// Find the first valid IP address from the headers
		vals := headers.Get(h)
		if len(vals) == 0 {
			continue
		}
		ipArray := strings.Split(vals[0], ",")
		if len(ipArray) == 0 {
			continue
		}
		ip := strings.TrimSpace(ipArray[0])
		// Return the first valid IP address found
		if ip != "" && strings.ToLower(ip) != "unknown" {
			return ip
		}
	}

	return ""
}
