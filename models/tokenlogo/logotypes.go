package tokenlogo

type QueryLogoParam struct {
	ChainId              uint32 `json:"chainId"`
	TokenContractAddress string `json:"tokenContractAddress"`
}

type TokenLogoInfo struct {
	LogoUrl              string `json:"logoUrl"`
	LogoOssUrl           string `json:"logoOssUrl"`
	ChainId              uint32 `json:"chainId"`
	TokenContractAddress string `json:"tokenContractAddress"`
	TokenSymbol          string `json:"tokenSymbol"`
	Unit                 uint32 `json:"unit"`
	TokenName            string `json:"tokenName"`
}

type GetTokenLogosResponse struct {
	Code         int             `json:"code"`
	Msg          string          `json:"msg"`
	ErrorCode    string          `json:"error_code"`
	ErrorMessage string          `json:"error_message"`
	DetailMsg    string          `json:"detailMsg"`
	Data         []TokenLogoInfo `json:"data"`
}
