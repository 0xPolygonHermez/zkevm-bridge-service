package autoclaim

import (
	"context"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"time"

	"github.com/0xPolygonHermez/zkevm-bridge-service/autoclaimservice/blockchainmanager"
	"github.com/0xPolygonHermez/zkevm-bridge-service/bridgectrl/pb"
	"github.com/0xPolygonHermez/zkevm-bridge-service/etherman/smartcontracts/claimcompressor"
	"github.com/0xPolygonHermez/zkevm-bridge-service/log"
	"github.com/ethereum/go-ethereum/common"
	"golang.org/x/crypto/sha3"
	"google.golang.org/protobuf/encoding/protojson"
)

type autoclaim struct {
	ctx context.Context
	bm  *blockchainmanager.Client
	cfg *Config
}

func NewAutoClaim(ctx context.Context, c *Config, blockchainManager *blockchainmanager.Client) (autoclaim, error) {
	if c.MaxNumberOfClaimsPerGroup == 0 || blockchainManager.ClaimCompressor == nil { //ClaimCompressor disabled
		log.Info("ClaimCompressor disabled")
	} else {
		log.Info("ClaimCompressor enabled")
	}
	return autoclaim{
		bm:  blockchainManager,
		cfg: c,
		ctx: ctx,
	}, nil
}

func (ac *autoclaim) Start() {
	ticker := time.NewTicker(ac.cfg.AutoClaimInterval.Duration)
	for {
		select {
		case <-ac.ctx.Done():
			ticker.Stop()
			return
		case <-ticker.C:
			if ac.cfg.MaxNumberOfClaimsPerGroup == 0 || ac.bm.ClaimCompressor == nil { //ClaimCompressor disabled
				err := ac.claimIndividually()
				if err != nil {
					log.Error("error claiming individually. Error: ", err)
				}
			} else { //ClaimCompressor enabled
				err := ac.claimGrouped()
				if err != nil {
					log.Error("error claiming grouped. Error: ", err)
				}
			}
		}
	}
}

func (ac *autoclaim) claimIndividually() error {
	bridges, err := ac.getBridgesToClaim()
	if err != nil {
		log.Errorf("error getBridgesToClaim. Error: %v", err)
		return err
	}
	if len(bridges) == 0 {
		log.Info("No bridges to claim were found")
		return nil
	}
	var proofs []*pb.GetProofResponse
	for _, b := range bridges {
		log.Debugf("Getting proof for bridge (depositCount: %d, networkID: %d)", b.DepositCnt, b.NetworkId)
		proof, err := ac.getProof(b)
		if err != nil {
			log.Errorf("error getting proof for deposit counter: %d and networkID: %d", b.DepositCnt, b.NetworkId)
			return err
		}
		proofs = append(proofs, proof)
	}
	if len(bridges) != len(proofs) {
		log.Error("error: bridges length and proofs length don't match in claimIndividually()")
		return fmt.Errorf("error: bridges length and proofs length don't match in claimIndividually()")
	}
	for i := range bridges {
		b := bridges[i]
		p := proofs[i].Proof
		globalIndex, _ := big.NewInt(0).SetString(b.GlobalIndex, 0)
		amount, _ := new(big.Int).SetString(b.Amount, 0)
		var smtProof, smtRollupProof [blockchainmanager.MtHeight][blockchainmanager.KeyLen]byte
		for i := 0; i < len(p.MerkleProof); i++ {
			smtProof[i] = common.HexToHash(p.MerkleProof[i])
			smtRollupProof[i] = common.HexToHash(p.RollupMerkleProof[i])
		}
		log.Debugf("Sending claim for bridge depositCount: %d, networkID: %d", b.DepositCnt, b.NetworkId)
		tx, err := ac.bm.SendClaim(b.LeafType, b.OrigNet, common.HexToAddress(b.OrigAddr), amount, b.DestNet, common.HexToAddress(b.DestAddr), b.NetworkId, common.FromHex(b.Metadata), globalIndex, smtProof, smtRollupProof, common.HexToHash(p.MainExitRoot), common.HexToHash(p.RollupExitRoot))
		if err != nil {
			log.Errorf("error sending claim for deposit counter: %d and networkID: %d, Error: %v", b.DepositCnt, b.NetworkId, err)
			// Here the error is not returned to avoid block the service. An specific claim can fail but others can work properly.
		} else {
			log.Infof("Claim tx (%s) sent for deposit counter %d from network %d", tx.Hash(), b.DepositCnt, b.NetworkId)
		}
	}
	return nil
}

func (ac *autoclaim) claimGrouped() error {
	bridges, err := ac.getBridgesToClaim()
	if err != nil {
		log.Errorf("error getBridgesToClaim. Error: %v", err)
		return err
	}
	if len(bridges) == 0 {
		log.Info("No bridges to claim were found")
		return nil
	}
	var proofs []*pb.GetProofResponse
	var ger, mainnetExitRoot, rollupExitRoot common.Hash
	for i, b := range bridges {
		log.Debugf("Getting proof for bridge (depositCount: %d, networkID: %d)", b.DepositCnt, b.NetworkId)
		if i == 0 {
			proof, err := ac.getProof(b)
			if err != nil {
				log.Errorf("error getting proof for deposit counter: %d and networkID: %d", b.DepositCnt, b.NetworkId)
				return err
			}
			mainnetExitRoot = common.HexToHash(proof.Proof.MainExitRoot)
			rollupExitRoot = common.HexToHash(proof.Proof.RollupExitRoot)
			ger = hash(mainnetExitRoot, rollupExitRoot)
			proofs = append(proofs, proof)
		} else {
			proof, err := ac.getProofByGER(b, ger)
			if err != nil {
				log.Errorf("error getting proof for deposit counter: %d and networkID: %d", b.DepositCnt, b.NetworkId)
				return err
			}
			proofs = append(proofs, proof)
		}
	}
	if len(bridges) != len(proofs) {
		log.Error("error: bridges length and proofs length don't match in claimGrouped()")
		return fmt.Errorf("error: bridges length and proofs length don't match in claimGrouped()")
	}
	splittedBridges := arraySplitter(bridges, ac.cfg.MaxNumberOfClaimsPerGroup)
	splittedProofs := arraySplitter(proofs, ac.cfg.MaxNumberOfClaimsPerGroup)
	for j, sb := range splittedBridges {
		var allClaimData []claimcompressor.ClaimCompressorCompressClaimCallData
		for i := range sb {
			b := sb[i]
			p := splittedProofs[j][i].Proof
			globalIndex, _ := big.NewInt(0).SetString(b.GlobalIndex, 0)
			amount, _ := new(big.Int).SetString(b.Amount, 0)
			var smtProof, smtRollupProof [blockchainmanager.MtHeight][blockchainmanager.KeyLen]byte
			for i := 0; i < len(p.MerkleProof); i++ {
				smtProof[i] = common.HexToHash(p.MerkleProof[i])
				smtRollupProof[i] = common.HexToHash(p.RollupMerkleProof[i])
			}
			metadata := common.FromHex(b.Metadata)
			originalAddress := common.HexToAddress(b.OrigAddr)
			destinationAddress := common.HexToAddress(b.DestAddr)

			log.Debugf("Estimating gas for bridge (depositCount: %d, networkID: %d)", b.DepositCnt, b.NetworkId)
			// Estimate gas to see if it is claimable
			_, err := ac.bm.EstimateGasClaim(metadata, amount, originalAddress, destinationAddress, b.OrigNet, b.DestNet, mainnetExitRoot, rollupExitRoot, b.LeafType, globalIndex, smtProof, smtRollupProof)
			if err != nil {
				log.Infof("error estimating gas for deposit counter: %d from networkID: %d, Skipping... Error: %v", b.DepositCnt, b.NetworkId, err)
				continue
			}

			claimData := claimcompressor.ClaimCompressorCompressClaimCallData{
				SmtProofLocalExitRoot:  smtProof,
				SmtProofRollupExitRoot: smtRollupProof,
				GlobalIndex:            globalIndex,
				OriginNetwork:          b.OrigNet,
				OriginAddress:          originalAddress,
				DestinationAddress:     destinationAddress,
				Amount:                 amount,
				Metadata:               metadata,
				IsMessage:              b.LeafType == blockchainmanager.LeafTypeMessage,
			}
			allClaimData = append(allClaimData, claimData)
		}
		if len(allClaimData) == 0 {
			log.Info("Nothing to compress and send")
			return nil
		}
		log.Debug("Compressing data")
		compressedTxData, err := ac.bm.CompressClaimCall(mainnetExitRoot, rollupExitRoot, allClaimData)
		if err != nil {
			log.Errorf("error compressing claim data, Error: %v", err)
			return err
		}
		log.Debug("Sending compressed claim tx")
		tx, err := ac.bm.SendCompressedClaims(compressedTxData)
		if err != nil {
			log.Errorf("error sending compressed claims, Error: %v", err)
			return err
		}
		log.Info("compressed claim sent. TxHash: ", tx.Hash())
	}
	return nil
}

func arraySplitter[T *pb.Deposit | *pb.GetProofResponse](arr []T, size int) [][]T {
	var chunks [][]T
	for {
		if len(arr) == 0 {
			break
		}
		if len(arr) < size {
			size = len(arr)
		}

		chunks = append(chunks, arr[0:size])
		arr = arr[size:]
	}

	return chunks
}

func hash(data ...[32]byte) [32]byte {
	var res [32]byte
	hash := sha3.NewLegacyKeccak256()
	for _, d := range data {
		hash.Write(d[:]) //nolint:errcheck,gosec
	}
	copy(res[:], hash.Sum(nil))
	return res
}

func (ac *autoclaim) getProof(bridge *pb.Deposit) (*pb.GetProofResponse, error) {
	requestURL := fmt.Sprintf("%s/merkle-proof?net_id=%d&deposit_cnt=%d", ac.cfg.BridgeURL, bridge.NetworkId, bridge.DepositCnt)
	req, err := http.NewRequest(http.MethodGet, requestURL, nil)
	if err != nil {
		log.Errorf("error creating newRequest. merkle-proof endpoint. Error: %v", err)
		return nil, err
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Errorf("error doing the request. merkle-proof endpoint. Error: %v", err)
		return nil, err
	}
	defer func() {
		err := res.Body.Close()
		if err != nil {
			log.Error("error closing response body in merkle-proof endpoint call")
		}
	}()

	var bodyBytes []byte
	if res.StatusCode == http.StatusOK {
		bodyBytes, err = io.ReadAll(res.Body)
		if err != nil {
			log.Errorf("error reading response body of merkle-proof endpoint. Error: %v", err)
			return nil, err
		}
	} else {
		return nil, fmt.Errorf("error calling /merkle-proof endpoint. Status Code: %d", res.StatusCode)
	}
	var proofResp pb.GetProofResponse
	err = protojson.Unmarshal(bodyBytes, &proofResp)
	if err != nil {
		return nil, err
	}
	return &proofResp, nil
}

func (ac *autoclaim) getProofByGER(bridge *pb.Deposit, ger common.Hash) (*pb.GetProofResponse, error) {
	requestURL := fmt.Sprintf("%s/merkle-proof-by-ger?net_id=%d&deposit_cnt=%d&ger=%s", ac.cfg.BridgeURL, bridge.NetworkId, bridge.DepositCnt, ger)
	req, err := http.NewRequest(http.MethodGet, requestURL, nil)
	if err != nil {
		log.Errorf("error creating newRequest. merkle-proof-by-ger endpoint. Error: %v", err)
		return nil, err
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Errorf("error doing the request. merkle-proof-by-ger endpoint. Error: %v", err)
		return nil, err
	}
	defer func() {
		err := res.Body.Close()
		if err != nil {
			log.Error("error closing response body in merkle-proof-by-ger endpoint call")
		}
	}()

	var bodyBytes []byte
	if res.StatusCode == http.StatusOK {
		bodyBytes, err = io.ReadAll(res.Body)
		if err != nil {
			log.Errorf("error reading response body of merkle-proof-by-ger endpoint. Error: %v", err)
			return nil, err
		}
	} else {
		return nil, fmt.Errorf("error calling merkle-proof-by-ger endpoint. Status Code: %d", res.StatusCode)
	}
	var proofResp pb.GetProofResponse
	err = protojson.Unmarshal(bodyBytes, &proofResp)
	if err != nil {
		return nil, err
	}
	return &proofResp, nil
}

func (ac *autoclaim) getBridgesToClaim() ([]*pb.Deposit, error) {
	log.Debug("Getting Bridges to claim (assets)")
	bridges, _, err := ac.getPendingBridges(blockchainmanager.LeafTypeAsset, common.Address{})
	if err != nil {
		log.Errorf("error getPendingBridges. Error: %v", err)
		return nil, err
	}
	for _, addr := range ac.cfg.AuthorizedClaimMessageAddresses {
		log.Debug("Getting Bridges to claim (messages) from address: ", addr.String())
		authorizedBridges, _, err := ac.getPendingBridges(blockchainmanager.LeafTypeMessage, addr)
		if err != nil {
			log.Errorf("error getPendingBridges. Error: %v", err)
			return nil, err
		}
		bridges = append(bridges, authorizedBridges...)
	}
	return bridges, nil
}

func (ac *autoclaim) getPendingBridges(leafType uint32, destAddress common.Address) ([]*pb.Deposit, uint64, error) {
	destNetwork := ac.bm.NetworkID
	requestURL := fmt.Sprintf("%s/pending-bridges?dest_net=%d&leaf_type=%d&dest_addr=%s", ac.cfg.BridgeURL, destNetwork, leafType, destAddress)
	req, err := http.NewRequest(http.MethodGet, requestURL, nil)
	if err != nil {
		log.Errorf("error creating newRequest. pending-bridges endpoint. Error: %v", err)
		return nil, 0, err
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Errorf("error doing the request. pending-bridges endpoint. Error: %v", err)
		return nil, 0, err
	}
	defer func() {
		err := res.Body.Close()
		if err != nil {
			log.Error("error closing response body in pending-bridges endpoint call")
		}
	}()

	var bodyBytes []byte
	if res.StatusCode == http.StatusOK {
		bodyBytes, err = io.ReadAll(res.Body)
		if err != nil {
			log.Errorf("error reading response body of pending-bridges endpoint. Error: %v", err)
			return nil, 0, err
		}
	}
	var bridgeResp pb.GetBridgesResponse
	err = protojson.Unmarshal(bodyBytes, &bridgeResp)
	if err != nil {
		return nil, 0, err
	}
	return bridgeResp.Deposits, bridgeResp.TotalCnt, nil
}
