package server

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"runtime"
	"testing"

	"github.com/hermeznetwork/hermez-bridge/bridgectrl/pb"
	"github.com/hermeznetwork/hermez-bridge/test/operations"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
)

const (
	grpcPort = "9090"
	restPort = "8080"
)

func init() {
	// Change dir to project root
	// This is important because we have relative paths to files containing test vectors
	_, filename, _, _ := runtime.Caller(0)
	dir := path.Join(path.Dir(filename), "../")
	err := os.Chdir(dir)
	if err != nil {
		panic(err)
	}
}

func TestBridgeMock(t *testing.T) {
	_, err := RunMockServer()
	require.NoError(t, err)

	err = operations.WaitGRPCHealthy("0.0.0.0:" + grpcPort)
	require.NoError(t, err)

	address := "http://localhost:" + restPort
	err = operations.WaitRestHealthy(address)
	require.NoError(t, err)

	resp, err := http.Get(address + "/api")
	require.NoError(t, err)

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	_ = resp.Body.Close()

	var apiResp pb.CheckAPIResponse
	err = protojson.Unmarshal(bodyBytes, &apiResp)
	require.NoError(t, err)

	require.Equal(t, "v1", apiResp.Api)

	resp, err = http.Get(fmt.Sprintf("%s%s/%s", address, "/bridges", "0xeB17ce701E9D92724AA2ABAdA7E4B28830597Dd9"))
	require.NoError(t, err)

	bodyBytes, err = ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	var bridgeResp pb.GetBridgesResponse
	err = protojson.Unmarshal(bodyBytes, &bridgeResp)
	require.NoError(t, err)
	require.Greater(t, len(bridgeResp.Deposits), 0)

	offset := 0
	resp, err = http.Get(fmt.Sprintf("%s%s/%s?offset=%d", address, "/bridges", "0xeB17ce701E9D92724AA2ABAdA7E4B28830597Dd9", offset))
	require.NoError(t, err)

	bodyBytes, err = ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	err = protojson.Unmarshal(bodyBytes, &bridgeResp)
	require.NoError(t, err)
	require.Equal(t, len(bridgeResp.Deposits), 1)

	offset = 0
	resp, err = http.Get(fmt.Sprintf("%s%s/%s?offset=%d", address, "/claims", "0xabCcEd19d7f290B84608feC510bEe872CC8F5112", offset))
	require.NoError(t, err)

	bodyBytes, err = ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	var claimResp pb.GetClaimsResponse
	err = protojson.Unmarshal(bodyBytes, &claimResp)
	require.NoError(t, err)
	require.Equal(t, len(claimResp.Claims), 1)

	resp, err = http.Get(fmt.Sprintf("%s%s?net_id=%d&deposit_cnt=%d", address, "/merkle-proofs", 0, 2))
	require.NoError(t, err)

	bodyBytes, err = ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	var proofResp pb.GetProofResponse
	err = protojson.Unmarshal(bodyBytes, &proofResp)
	require.NoError(t, err)
	require.Equal(t, len(proofResp.Proof.MerkleProof), 32)

	resp, err = http.Get(fmt.Sprintf("%s%s?net_id=%d&deposit_cnt=%d", address, "/claim-status", 0, 2))
	require.NoError(t, err)

	bodyBytes, err = ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	var claimStatus pb.GetClaimStatusResponse
	err = protojson.Unmarshal(bodyBytes, &claimStatus)
	require.NoError(t, err)
	require.Equal(t, claimStatus.Ready, true)
}
