package iprestriction

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"

	"github.com/0xPolygonHermez/zkevm-bridge-service/log"
	"github.com/0xPolygonHermez/zkevm-bridge-service/nacos"
)

type Client struct {
	cfg        Config
	httpClient *http.Client
}

var (
	client *Client
)

func InitClient(c Config) {
	if !c.Enabled {
		return
	}
	if c.Host == "" {
		log.Errorf("host is empty")
		return
	}
	client = &Client{
		cfg: c,
		httpClient: &http.Client{
			Timeout: c.Timeout.Duration,
		},
	}
}

func GetClient() *Client {
	return client
}

func (c *Client) CheckIPRestricted(ip string) bool {
	if c == nil || !c.cfg.Enabled {
		log.Debugf("IP restriction is disabled, skipped")
		return false
	}
	for _, blockedIP := range c.cfg.IPBlocklist {
		if ip == blockedIP {
			return true
		}
	}
	host := c.cfg.Host
	if c.cfg.UseNacos {
		// Resolve nacos service name to URL
		nacosUrl, err := nacos.GetOneURL(c.cfg.Host)
		if err != nil {
			log.Errorf("[CheckIPRestricted] cannot get URL from nacos service, name[%v] err[%v]", c.cfg.Host, err)
			return false
		}
		host = nacosUrl
	}

	fullPath, err := url.JoinPath(host, endpointCheckCountryLimit)
	if err != nil {
		log.Errorf("[CheckIPRestricted] JoinPath err[%v] host[%v] endpoint[%v]", err, host, endpointCheckCountryLimit)
		return false
	}
	req, err := http.NewRequest("GET", fullPath, nil)
	if err != nil {
		log.Errorf("[CheckIPRestricted] create new GET request err[%v] path[%v]", err, fullPath)
		return false
	}
	// Add request params
	q := req.URL.Query()
	q.Add("ip", ip)
	q.Add("type", queryTypeXLayerBridge)
	q.Add("siteCode", siteCodeOKXGLobal)
	req.URL.RawQuery = q.Encode()

	// Call the API
	resp, err := c.httpClient.Do(req)
	if err != nil {
		log.Errorf("[CheckIPRestricted] call GET API err[%v] url[%v] ip[%v]", err, fullPath, ip)
		return false
	}

	if resp.StatusCode != http.StatusOK {
		log.Errorf("[CheckIPRestrictied] GET API status code %v url[%v] ip[%v]", resp.StatusCode, fullPath, ip)
		return false
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Errorf("[CheckIPRestricted] failed to read resp body err[%v]")
		return false
	}
	log.Debugf("[CheckIPRestricted] CheckCountryLimit ip[%v] respBody[%v]", string(respBody))

	respStruct := new(CheckCountryLimitResponse)
	err = json.Unmarshal(respBody, respStruct)
	if err != nil {
		log.Errorf("[CheckIPRestricted] failed to unmarshal response, err[%v]", err)
		return false
	}
	log.Debugf("[CheckIPRestricted] URL[%v] IP[%v] Result[%+v]", fullPath, ip, respStruct.Data.XLayerBridge)
	if respStruct.Data.XLayerBridge != nil && (respStruct.Data.XLayerBridge.Hidden || respStruct.Data.XLayerBridge.Limit) {
		return true
	}

	return false
}
