package nacos

import (
	"fmt"
	"strconv"
	"time"

	"github.com/0xPolygonHermez/zkevm-bridge-service/log"
	"github.com/nacos-group/nacos-sdk-go/v2/clients"
	"github.com/nacos-group/nacos-sdk-go/v2/clients/naming_client"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v2/model"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
)

var (
	client naming_client.INamingClient
)

// InitNacosClient start nacos client and register rest service in nacos
func InitNacosClient(urls string, namespace string, name string, externalAddr string) {
	ip, port, err := resolveIPAndPort(externalAddr)
	if err != nil {
		log.Error(fmt.Sprintf("failed to resolve %s error: %s", externalAddr, err.Error()))
		return
	}

	serverConfigs, err := getServerConfigs(urls)
	if err != nil {
		log.Error(fmt.Sprintf("failed to resolve nacos server url %s: %s", urls, err.Error()))
		return
	}
	const timeoutMs = 5000
	client, err = clients.CreateNamingClient(map[string]interface{}{
		"serverConfigs": serverConfigs,
		"clientConfig": constant.ClientConfig{
			TimeoutMs:           timeoutMs,
			NotLoadCacheAtStart: true,
			NamespaceId:         namespace,
			LogDir:              "/dev/null",
			LogLevel:            "error",
		},
	})
	if err != nil {
		log.Error(fmt.Sprintf("failed to create nacos client. error: %s", err.Error()))
		return
	}

	const weight = 10
	_, err = client.RegisterInstance(vo.RegisterInstanceParam{
		Ip:          ip,
		Port:        uint64(port),
		ServiceName: name,
		Weight:      weight,
		ClusterName: "DEFAULT",
		Enable:      true,
		Healthy:     true,
		Ephemeral:   true,
		Metadata: map[string]string{
			"preserved.register.source": "GO",
			"app_registry_tag":          strconv.FormatInt(time.Now().Unix(), 10),
		},
	})
	if err != nil {
		log.Error(fmt.Sprintf("failed to register instance in nacos server. error: %s", err.Error()))
		return
	}
	log.Info("register application instance in nacos successfully")
}

// GetOneInstance returns the info of one healthy instance of the service
func GetOneInstance(serviceName string) (*model.Instance, error) {
	params := vo.SelectOneHealthInstanceParam{ServiceName: serviceName}
	return client.SelectOneHealthyInstance(params)
}

// GetOneURL returns the URL address of one healthy instance of the service
func GetOneURL(serviceName string) (string, error) {
	instance, err := GetOneInstance(serviceName)
	if err != nil {
		return "", err
	}
	url := fmt.Sprintf("%v:%v", instance.Ip, instance.Port)
	return url, nil
}
