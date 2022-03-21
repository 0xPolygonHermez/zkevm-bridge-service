package server

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"runtime"
	"testing"
	"time"

	"github.com/hermeznetwork/hermez-bridge/bridgetree/pb"
	"github.com/stretchr/testify/require"
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

func TestBridgeService(t *testing.T) {
	err := RunMockServer()
	require.NoError(t, err)

	address := "http://localhost:8080"

	// Check the api version
	checkLimit := 15

	for checkLimit > 0 {
		time.Sleep(1 * time.Second)
		checkLimit--

		resp, err := http.Get(address + "/api")
		if err != nil {
			continue
		}

		bodyBytes, err := ioutil.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if err != nil {
			continue
		}

		var apiResp pb.CheckAPIResponse
		_ = json.Unmarshal(bodyBytes, &apiResp)
		require.NoError(t, err)

		require.Equal(t, "v1", apiResp.Api)
		break
	}

	require.GreaterOrEqual(t, checkLimit, 1)

	resp, err := http.Get(fmt.Sprintf("%s%s/%s", address, "/bridges", "0xeB17ce701E9D92724AA2ABAdA7E4B28830597Dd9"))
	require.NoError(t, err)

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	var bridgeResp pb.GetBridgesResponse
	_ = json.Unmarshal(bodyBytes, &bridgeResp)
	require.Greater(t, len(bridgeResp.Deposits), 0)

	offset := 3
	resp, err = http.Get(fmt.Sprintf("%s%s/%s?offset=%d", address, "/bridges", "0xeB17ce701E9D92724AA2ABAdA7E4B28830597Dd9", offset))
	require.NoError(t, err)

	bodyBytes, err = ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	_ = json.Unmarshal(bodyBytes, &bridgeResp)
	require.Equal(t, len(bridgeResp.Deposits), offset-1)

	offset = 1
	resp, err = http.Get(fmt.Sprintf("%s%s/%s?offset=%d", address, "/claims", "0xeB17ce701E9D92724AA2ABAdA7E4B28830597Dd9", offset))
	require.NoError(t, err)

	bodyBytes, err = ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	var claimResp pb.GetClaimsResponse
	_ = json.Unmarshal(bodyBytes, &claimResp)
	require.Equal(t, len(claimResp.Claims), 2)

	resp, err = http.Get(fmt.Sprintf("%s%s?orig_net=%d&deposit_cnt=%d", address, "/merkle-proofs", 0, 2))
	require.NoError(t, err)

	bodyBytes, err = ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	var proofResp pb.GetProofResponse
	_ = json.Unmarshal(bodyBytes, &proofResp)
	require.Equal(t, len(proofResp.Proof.MerkleProof), 32)

	resp, err = http.Get(fmt.Sprintf("%s%s?orig_net=%d&deposit_cnt=%d", address, "/claim-status", 0, 2))
	require.NoError(t, err)

	bodyBytes, err = ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	var claimStatus pb.GetClaimStatusResponse
	_ = json.Unmarshal(bodyBytes, &claimStatus)
	require.Equal(t, claimStatus.Ready, true)
}
