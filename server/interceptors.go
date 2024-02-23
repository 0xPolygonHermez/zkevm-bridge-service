package server

import (
	"context"
	"time"

	"github.com/0xPolygonHermez/zkevm-bridge-service/utils"
	"github.com/0xPolygonHermez/zkevm-node/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

func NewRequestLogInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		startTime := time.Now()
		methodName := info.FullMethod

		// Set up the trace id for logging
		traceID := utils.GenerateTraceID()
		logger := utils.LoggerWithTraceID(log.LoggerFromCtx(ctx), traceID)
		ctx = log.CtxWithLogger(ctx, logger)

		logger.Debugf("method[%v] start", methodName)

		// Return the trace id to the client through the header
		header := metadata.New(map[string]string{
			"trace-id": traceID,
		})
		err := grpc.SendHeader(ctx, header)
		if err != nil {
			logger.Infof("method[%v] SendHeader error[%v]", methodName, err)
			return nil, err
		}

		// Actual process of the request
		resp, err := handler(ctx, req)

		duration := time.Since(startTime)
		reqJson, _ := protojson.Marshal(req.(proto.Message))
		respJson, _ := protojson.Marshal(resp.(proto.Message))
		logger.Infof("method[%v] req[%v] resp[%v] err[%v] processTime[%v]", methodName, string(reqJson), string(respJson), err, duration.String())
		return resp, err
	}
}
