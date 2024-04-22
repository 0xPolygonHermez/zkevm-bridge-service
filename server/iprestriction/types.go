package iprestriction

type CheckResult struct {
	Hidden bool `json:"hidden"`
	Limit  bool `json:"limit"`
}

type CheckCountryLimitResponseData struct {
	XLayerBridge *CheckResult `json:"xLayerBridge"`
}

type CheckCountryLimitResponse struct {
	Code         int                           `json:"code"`
	Msg          string                        `json:"msg"`
	ErrorCode    string                        `json:"error_code"`
	ErrorMessage string                        `json:"error_message"`
	DetailMsg    string                        `json:"detailMsg"`
	Data         CheckCountryLimitResponseData `json:"data"`
}
