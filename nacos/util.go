package nacos

import (
	"net"
	"strconv"
	"strings"

	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
)

func resolveIPAndPort(addr string) (string, int, error) {
	laddr := strings.Split(addr, ":")
	ip := laddr[0]
	if ip == "127.0.0.1" {
		const port = 26659
		return getLocalIP(), port, nil
	}
	port, err := strconv.Atoi(laddr[1])
	if err != nil {
		return "", 0, err
	}
	return ip, port, nil
}

// getLocalIP get local ip
func getLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}
	for _, address := range addrs {
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return ""
}

func getServerConfigs(urls string) ([]constant.ServerConfig, error) {
	// nolint
	var configs []constant.ServerConfig
	for _, url := range strings.Split(urls, ",") {
		laddr := strings.Split(url, ":")
		serverPort, err := strconv.Atoi(laddr[1])
		if err != nil {
			return nil, err
		}
		configs = append(configs, constant.ServerConfig{
			IpAddr: laddr[0],
			Port:   uint64(serverPort),
		})
	}
	return configs, nil
}
