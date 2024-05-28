package tokenlogoinfo

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/0xPolygonHermez/zkevm-bridge-service/models/tokenlogo"
	"github.com/0xPolygonHermez/zkevm-bridge-service/nacos"
	"github.com/0xPolygonHermez/zkevm-node/log"
)

const (
	endpointGetLogoInfos = "/priapi/v1/oc/web-common/coin/getLogoInfoBatch"
	ChainNativeTokenAddr = "0x0000000000000000000000000000000000000000"
	EmptyStr             = ""
)

type Client struct {
	cfg        Config
	httpClient *http.Client
}

var (
	client *Client
)

func GetClient() *Client {
	return client
}

func InitClient(c Config) {
	if !c.Enabled {
		return
	}
	if c.LogoServiceNacosName == "" {
		log.Errorf("token logo service name is empty")
		return
	}
	client = &Client{
		cfg: c,
		httpClient: &http.Client{
			Timeout: c.Timeout.Duration,
		},
	}
}

func (c *Client) GetTokenLogoInfos(tokenAddArr []*tokenlogo.QueryLogoParam) (map[string]tokenlogo.LogoInfo, error) {
	if c == nil {
		log.Infof("init logo info client failed, so skip")
		return nil, nil
	}
	if !c.cfg.Enabled {
		log.Infof("get token logo enable is false, so skip")
		return nil, nil
	}
	host, err := nacos.GetOneURL(c.cfg.LogoServiceNacosName)
	if err != nil {
		log.Errorf("[getTokenLogoInfos] cannot get URL from nacos service, name[%v] err[%v]", c.cfg.LogoServiceNacosName, err)
		return nil, err
	}
	host = c.cfg.Scheme + "://" + host
	fullPath, err := url.JoinPath(host, endpointGetLogoInfos)
	if err != nil {
		log.Errorf("[getTokenLogoInfos] JoinPath err[%v] host[%v] endpoint[%v]", err, host, endpointGetLogoInfos)
		return nil, err
	}
	postBody, _ := json.Marshal(tokenAddArr)
	requestBody := bytes.NewBuffer(postBody)
	req, err := http.NewRequest("POST", fullPath, requestBody)
	if err != nil {
		log.Errorf("[getTokenLogoInfos] create request failed err[%v] full path[%v]", err, fullPath)
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		log.Errorf("[getTokenLogoInfos] call logo service failed err[%v] full path[%v]", err, fullPath)
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		log.Errorf("[getTokenLogoInfos] http status code [%v] url[%v]", resp.StatusCode, fullPath)
		return nil, nil
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Errorf("[getTokenLogoInfos] close response body failed, err [%v]", err)
		}
	}(resp.Body)
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Errorf("[getTokenLogoInfos] failed to read resp body err[%v]", err)
		return nil, err
	}
	respStruct := &tokenlogo.GetTokenLogosResponse{}
	err = json.Unmarshal(respBody, respStruct)
	if err != nil {
		log.Errorf("[getTokenLogoInfos] failed to convert resp to struct, resp [%v] err[%v]", string(respBody), err)
		return nil, err
	}
	logoMap := make(map[string]tokenlogo.LogoInfo, len(respStruct.Data))
	for _, v := range respStruct.Data {
		chainId, errConvert := strconv.ParseInt(v.ChainIdStr, 10, 64)
		if errConvert != nil {
			log.Errorf("token: %v convert chain id to uint failed, chain str: %v, err: %v", v.TokenContractAddress, v.ChainIdStr, errConvert)
			continue
		}
		logoMap[GetTokenLogoMapKey(v.TokenContractAddress, uint32(chainId))] = v
	}
	log.Debugf("request token info by rpc, final length: %v", len(logoMap))
	return logoMap, nil
}

func GetTokenLogoMapKey(tokenAddr string, chainId uint32) string {
	if tokenAddr == ChainNativeTokenAddr {
		tokenAddr = EmptyStr
	} else if tokenAddr == EmptyStr {
		tokenAddr = ChainNativeTokenAddr
	}
	return fmt.Sprintf("%s_%d", strings.ToLower(tokenAddr), chainId)
}
