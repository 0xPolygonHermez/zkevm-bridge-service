package server

import (
	"testing"

	"github.com/0xPolygonHermez/zkevm-bridge-service/bridgectrl/pb"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestGetRespErrorInfo(t *testing.T) {
	cases := []struct {
		name       string
		inputResp  any
		inputErr   error
		outputCode int64
		outputMsg  string
	}{
		{
			"err is not nil",
			pb.CommonCoinsResponse{Code: 2, Msg: "this will not be returned"},
			errors.New("this is an error"),
			int64(pb.ErrorCode_ERROR_DEFAULT),
			"this is an error",
		},
		{
			"no error and resp is nil",
			nil,
			nil,
			int64(pb.ErrorCode_ERROR_OK),
			"",
		},
		{
			"return msg from the resp body",
			pb.CommonCoinsResponse{Code: 2, Msg: "this is an error"},
			nil,
			2,
			"this is an error",
		},
		{
			"no code and msg field",
			struct{}{},
			nil,
			int64(pb.ErrorCode_ERROR_OK),
			"",
		},
	}

	for _, c := range cases {
		code, msg := getRespErrorInfo(c.inputResp, c.inputErr)
		assert.Equal(t, c.outputCode, code, c.name)
		assert.Equal(t, c.outputMsg, msg, c.name)
	}
}
