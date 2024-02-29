package server

import (
	"testing"

	"github.com/0xPolygonHermez/zkevm-bridge-service/bridgectrl/pb"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestCheckRespIsSuccess(t *testing.T) {
	cases := []struct {
		name      string
		inputResp any
		inputErr  error
		output    bool
	}{
		{
			"false if error is not nil",
			nil,
			errors.New("any error"),
			false,
		},
		{
			"error is nil and code is 0, should return true",
			&pb.CommonCoinsResponse{Code: 0},
			nil,
			true,
		},
		{
			"code is non-zero, should return false",
			&pb.CommonCoinsResponse{Code: 1},
			nil,
			false,
		},
		{
			"error is nil and no code field, return true",
			struct{}{},
			nil,
			true,
		},
	}

	for _, c := range cases {
		assert.Equal(t, c.output, checkRespIsSuccess(c.inputResp, c.inputErr), c.name)
	}
}
