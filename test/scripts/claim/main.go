package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/hermeznetwork/hermez-bridge/test/operations"
	"github.com/hermeznetwork/hermez-core/encoding"
	"github.com/hermeznetwork/hermez-core/etherman/smartcontracts/bridge"
	"github.com/hermeznetwork/hermez-core/log"
)

const (
	l2BridgeAddr = "0x9d98deabc42dd696deb9e40b4f1cab7ddbf55988"

	l2AccHexAddress    = "0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266"
	l2AccHexPrivateKey = "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
	l2NetworkURL       = "http://localhost:8123"
	brdigeURL          = "http://localhost:8080"

	funds = 90000000000000000
)

func main() {
	ctx := context.Background()
	client, err := ethclient.Dial(l2NetworkURL)
	if err != nil {
		log.Error(err)
		return
	}
	br, err := bridge.NewBridge(common.HexToAddress(l2BridgeAddr), client)
	if err != nil {
		log.Error(err)
		return
	}
	privateKey, err := crypto.HexToECDSA(strings.TrimPrefix(l2AccHexPrivateKey, "0x"))
	if err != nil {
		log.Error(err)
		return
	}
	chainID, err := client.NetworkID(ctx)
	if err != nil {
		log.Error(err)
		return
	}
	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, chainID)
	if err != nil {
		log.Error(err)
		return
	}
	auth.GasPrice = big.NewInt(0)

	// Get Claim data
	bridgeData, proofData, err := getClaimData()
	if err != nil {
		log.Fatal("error getting claimData: ", err)
	} else if proofData == nil {
		log.Fatal("error. Merkeltree proof not found. Please wait for a new batch to do the claim")
	}
	log.Debug("bridge: ", bridgeData)
	log.Debug("mainnetExitRoot: ", proofData.MainExitRoot)
	log.Debug("rollupExitRoot: ", proofData.RollupExitRoot)
	log.Debug("L2ExitRootNum: ", proofData.L2ExitRootNum)
	log.Debug("ExitRootNum: ", proofData.ExitRootNum)

	amount := big.NewInt(funds)
	var destNetwork uint32 = 1
	var origNetwork uint32 = 0
	destAddr := common.HexToAddress(l2AccHexAddress)
	l2ExitRootNum, _ := new(big.Int).SetString(proofData.L2ExitRootNum, encoding.Base10)
	depositCount, _ := new(big.Int).SetString(bridgeData.DepositCnt, encoding.Base10)
	var smt [][32]byte
	for i := 0; i < len(proofData.MerkleProof); i++ {
		log.Debug("smt: ", common.HexToHash(proofData.MerkleProof[i]).String())
		smt = append(smt, common.HexToHash(proofData.MerkleProof[i]))
	}
	log.Info("Sending claim tx...")
	tx, err := br.Claim(auth, common.HexToAddress(bridgeData.TokenAddr), amount, origNetwork, destNetwork, destAddr, smt,
		uint32(depositCount.Uint64()), l2ExitRootNum, common.HexToHash(proofData.MainExitRoot),
		common.HexToHash(proofData.RollupExitRoot))
	if err != nil {
		log.Error(err)
		return
	}
	const txTimeout = 30 * time.Second
	err = operations.WaitTxToBeMined(ctx, client, tx.Hash(), txTimeout)
	if err != nil {
		log.Error(err)
		return
	}
	log.Info("Success! txHash: ", tx.Hash())
	balance, err := client.BalanceAt(ctx, common.HexToAddress(l2AccHexAddress), nil)
	if err != nil {
		log.Error("error getting balance: ", err)
	}
	log.Info("L2 balance: ", balance)
}

func getClaimData() (*deposit, *proof, error) {
	resp, err := http.Get(brdigeURL + "/bridges/" + l2AccHexAddress)
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, err
	}

	var bridges getBridgesResponse
	err = json.Unmarshal([]byte(body), &bridges)
	if err != nil {
		log.Error("error unmarshaling bridges: ", err)
	}

	resp, err = http.Get(fmt.Sprintf("%s/merkle-proofs?net_id=%d&deposit_cnt=%s", brdigeURL, bridges.Deposits[0].NetworkID, bridges.Deposits[0].DepositCnt))
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, err
	}
	var proof getProofResponse
	err = json.Unmarshal([]byte(body), &proof)
	if err != nil {
		log.Error("error unmarshaling proof: ", err)
	}
	return bridges.Deposits[0], proof.Proof, nil
}

type getProofResponse struct {
	Proof *proof `json:"proof,omitempty"`
}

type proof struct {
	MerkleProof    []string `json:"merkle_proof,omitempty"`
	ExitRootNum    string   `json:"exit_root_num,omitempty"`
	L2ExitRootNum  string   `json:"l2_exit_root_num,omitempty"`
	MainExitRoot   string   `json:"main_exit_root,omitempty"`
	RollupExitRoot string   `json:"rollup_exit_root,omitempty"`
}

type getBridgesResponse struct {
	Deposits []*deposit `json:"deposits,omitempty"`
}

type deposit struct {
	OrigNet    uint32 `json:"orig_net,omitempty"`
	TokenAddr  string `json:"token_addr,omitempty"`
	Amount     string `json:"amount,omitempty"`
	DestNet    uint32 `json:"dest_net,omitempty"`
	DestAddr   string `json:"dest_addr,omitempty"`
	BlockNum   string `json:"block_num,omitempty"`
	DepositCnt string `json:"deposit_cnt,omitempty"`
	NetworkID  uint32 `json:"network_id,omitempty"`
	TxHash     string `json:"tx_hash,omitempty"`
}
