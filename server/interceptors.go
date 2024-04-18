package server

import (
	"context"
	"reflect"
	"time"

	"github.com/0xPolygonHermez/zkevm-bridge-service/log"
	"github.com/0xPolygonHermez/zkevm-bridge-service/metrics"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

func NewRequestLogInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		startTime := time.Now()
		methodName := info.FullMethod

		// Actual process of the request
		resp, err := handler(ctx, req)

		duration := time.Since(startTime)
		reqJson, _ := protojson.Marshal(req.(proto.Message))
		respJson, _ := protojson.Marshal(resp.(proto.Message))
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
	msg = msgField.String()

	return
}
