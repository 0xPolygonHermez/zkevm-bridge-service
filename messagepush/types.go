package messagepush

const (
	BizCodeBridgeOrder = "x1_bridge_order"
)

type PushMessage struct {
	BizCode       string `json:"bizCode"`
	WalletAddress string `json:"walletAddress"`
	RequestID     string `json:"requestId"`
	PushContent   string `json:"pushContent"`
	Time          int64  `json:"time"`
}
