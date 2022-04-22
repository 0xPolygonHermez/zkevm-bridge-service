package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"math/big"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/hermeznetwork/hermez-bridge/bridgectrl/pb"
	"github.com/hermeznetwork/hermez-bridge/db/pgstorage"
	"github.com/hermeznetwork/hermez-bridge/etherman"
	"github.com/hermeznetwork/hermez-bridge/server"
	"github.com/hermeznetwork/hermez-bridge/test/operations"
	"github.com/hermeznetwork/hermez-core/log"
	"google.golang.org/protobuf/encoding/protojson"
)

const (
	restPort = "8080"
)

func main() {
	var (
		preDepositCount = 10000
		networkIds      = []uint{0, 1000}
		depositAmount   = big.NewInt(1000000000000) //nolint:gomnd
		addressList     = []common.Address{}
		addressCount    = 10
		requestCount    = 20
		wg              sync.WaitGroup
	)

	for i := 0; i < addressCount; i++ {
		addressList = append(addressList, common.BigToAddress(big.NewInt(int64(i))))
	}
	ctx := context.Background()
	bt, err := server.RunMockServer()
	if err != nil {
		log.Error("error run the mock rest server. Error: ", err)
		panic(err)
	}
	dbCfg := pgstorage.NewConfigFromEnv()
	store, err := pgstorage.NewPostgresStorage(dbCfg)
	if err != nil {
		panic(err)
	}

	// add deposits
	start := time.Now()
	for i := 0; i < preDepositCount; i++ {
		deposit := &etherman.Deposit{
			OriginalNetwork:    networkIds[i%2],
			TokenAddress:       addressList[i%addressCount],
			Amount:             depositAmount.Add(depositAmount, big.NewInt(int64(i+6))), //nolint:gomnd
			DestinationNetwork: networkIds[(i+1)%2],
			DestinationAddress: addressList[(i+1)%addressCount],
			BlockID:            1,
			BlockNumber:        0,
			DepositCount:       uint(i + 6), //nolint:gomnd
		}
		err := store.AddDeposit(ctx, deposit)
		if err != nil {
			panic(err)
		}
		err = bt.MockAddDeposit(deposit)
		if err != nil {
			panic(err)
		}
	}
	log.Infof("%d pre-deposits execution time %s\n", preDepositCount, time.Since(start))
	// run the goroutine to add deposits
	go func(count int) {
		startTime := time.Now()
		for i := 0; i < preDepositCount; i++ {
			deposit := &etherman.Deposit{
				OriginalNetwork:    networkIds[i%2],
				TokenAddress:       addressList[i%addressCount],
				Amount:             depositAmount.Add(depositAmount, big.NewInt(int64(i+preDepositCount+6))), //nolint:gomnd
				DestinationNetwork: networkIds[(i+1)%2],
				DestinationAddress: addressList[(i+1)%addressCount],
				BlockID:            1,
				BlockNumber:        0,
				DepositCount:       uint(i + preDepositCount + 6), //nolint:gomnd
			}
			err := store.AddDeposit(ctx, deposit)
			if err != nil {
				panic(err)
			}
			err = bt.MockAddDeposit(deposit)
			if err != nil {
				panic(err)
			}
		}
		log.Infof("Goroutine add %d deposits execution time %s\n", preDepositCount, time.Since(startTime))
	}(preDepositCount)
	// parallel requests
	address := "http://localhost:" + restPort
	err = operations.WaitRestHealthy(address)
	if err != nil {
		panic(err)
	}

	execTime := time.Duration(0)
	for i := 0; i < preDepositCount; i++ {
		start = time.Now()
		wg.Add(requestCount)
		for j := 0; j < requestCount; j++ {
			go func(depositCount uint) {
				res, err := http.Get(fmt.Sprintf("%s%s?orig_net=%d&deposit_cnt=%d", address, "/merkle-proofs", networkIds[depositCount%2], depositCount))
				if err != nil {
					panic(err)
				}
				bodyBytes, _ := ioutil.ReadAll(res.Body)
				var proofResp pb.GetProofResponse
				err = protojson.Unmarshal(bodyBytes, &proofResp)
				if err != nil || len(proofResp.Proof.MerkleProof) != 32 {
					log.Error(string(bodyBytes), depositCount)
					panic(err)
				}
			}(uint(i + j + 6)) //nolint:gomnd
			wg.Done()
		}
		wg.Wait()
		execTime += time.Since(start)
		time.Sleep(30 * time.Millisecond) //nolint:gomnd
	}

	log.Infof("Average of %d times in %d parallel request execution time: %s\n", preDepositCount, requestCount, execTime/time.Duration(preDepositCount))
	// Wait for an in interrupt.
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)
	<-ch
}
