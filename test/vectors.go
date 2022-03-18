package test

// DepositVectorRaw represents the deposit vector
type DepositVectorRaw struct {
	OriginalNetwork    uint   `json:"origNetwork"`
	TokenAddress       string `json:"tokenAddress"`
	Amount             string `json:"amount"`
	DestinationNetwork uint   `json:"destNetwork"`
	DestinationAddress string `json:"destAddress"`
	BlockNumber        uint64 `json:"blockNumber"`
	DepositCount       uint   `json:"depositCount"`
	ExpectedHash       string `json:"expectedHash"`
	ExpectedRoot       string `json:"expectedRoot"`
}

// ClaimVectorRaw represents the claim vector
type ClaimVectorRaw struct {
	Index              uint   `json:"index"`
	OriginalNetwork    uint   `json:"origNetwork"`
	Token              string `json:"token"`
	Amount             string `json:"amount"`
	DestinationNetwork uint   `json:"destNetwork"`
	DestinationAddress string `json:"destAddress"`
	BlockNumber        uint64 `json:"blockNumber"`
}

// BlockVectorRaw represents the block vector
type BlockVectorRaw struct {
	BlockNumber uint64 `json:"blockNumber"`
	BlockHash   string `json:"blockHash"`
	ParentHash  string `json:"parentHash"`
	NetworkID   uint   `json:"networkID"`
}
