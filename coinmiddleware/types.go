package coinmiddleware

type MessageBody struct {
	Data  *MessageData `json:"data"`
	Topic string       `json:"topic"`
}

type MessageData struct {
	ID        string       `json:"id"`
	PriceList []*PriceInfo `json:"priceList"`
}

type PriceInfo struct {
	CoinID             int64   `json:"coinId"`
	ChainID            int64   `json:"chainId"`
	Symbol             string  `json:"symbol"`
	FullName           string  `json:"fullName"`
	Chain              string  `json:"chain"`
	TokenAddress       string  `json:"tokenAddress"`
	Price              float64 `json:"price"`
	PriceStatus        int     `json:"priceStatus"`
	MaxPrice24H        float64 `json:"maxPrice24H"`
	MinPrice24H        float64 `json:"minPrice24H"`
	MarketCap          float64 `json:"marketCap"`
	Timestamp          int64   `json:"timestamp"`
	Vol24H             float64 `json:"vol24h"`
	CirculatingSupply  float64 `json:"circulatingSupply"`
	MaxSupply          float64 `json:"maxSupply"`
	TotalSupply        float64 `json:"totalSupply"`
	PriceChange24H     float64 `json:"priceChange24H"`
	PriceChangeRate24H float64 `json:"priceChangeRate24H"`
	DexVol24H          float64 `json:"dexVol24H"`
	DexLiquidity       float64 `json:"dexLiquidity"`
	DexReserve         float64 `json:"dexReserve"`
	IsDexSource        bool    `json:"isDexSource"`
	QuoteLiquidity     float64 `json:"quoteLiquidity"`
}
