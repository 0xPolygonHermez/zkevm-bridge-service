package server

import (
	"context"
	"reflect"
	"time"

	"github.com/0xPolygonHermez/zkevm-bridge-service/metrics"
	"github.com/0xPolygonHermez/zkevm-node/log"
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
		isSuccess := checkRespIsSuccess(resp, err)

		metrics.RecordRequest(methodName, isSuccess)
		metrics.RecordRequestLatency(methodName, duration, isSuccess)

		return resp, err
	}
}

func checkRespIsSuccess(resp any, err error) bool {
	// Check the returned error
	if err != nil {
		return false
	}

	if resp == nil {
		return true
	}
	// Check the `code` field in the response body
	v := reflect.Indirect(reflect.ValueOf(resp))
	codeField := v.FieldByName("Code")
	if codeField.CanInt() && codeField.Int() != 0 {
		return false
	}
	if codeField.CanUint() && codeField.Uint() != 0 {
		return false
	}

	// If code is 0 or cannot properly check code field, assume it's successful
	return true
}
