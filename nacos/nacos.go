package nacos

import (
	"fmt"
	"strconv"
	"time"

	"github.com/0xPolygonHermez/zkevm-bridge-service/log"
	"github.com/nacos-group/nacos-sdk-go/clients"
	"github.com/nacos-group/nacos-sdk-go/clients/naming_client"
	"github.com/nacos-group/nacos-sdk-go/common/constant"
	"github.com/nacos-group/nacos-sdk-go/model"
	"github.com/nacos-group/nacos-sdk-go/vo"
	"github.com/pkg/errors"
)

var (
	client naming_client.INamingClient
)

// InitNacosClient start nacos client and register rest service in nacos
func InitNacosClient(urls string, namespace string, name string, externalAddr string) error {
	ip, port, err := resolveIPAndPort(externalAddr)
	if err != nil {
		log.Error(fmt.Sprintf("failed to resolve %s error: %s", externalAddr, err.Error()))
		return err
	}

	serverConfigs, err := getServerConfigs(urls)
	if err != nil {
		log.Error(fmt.Sprintf("failed to resolve nacos server url %s: %s", urls, err.Error()))
		return err
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
		return err
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
		return err
	}
	log.Info("register application instance in nacos successfully")
	return nil
}

// GetOneInstance returns the info of one healthy instance of the service
func GetOneInstance(serviceName string) (*model.Instance, error) {
	if client == nil {
		return nil, errors.New("nacos client is not initialized")
	}
	params := vo.SelectOneHealthInstanceParam{ServiceName: serviceName}
	return client.SelectOneHealthyInstance(params)
}

// GetOneURL returns the URL address of one healthy instance of the service
func GetOneURL(serviceName string) (string, error) {
	instance, err := GetOneInstance(serviceName)
	log.Debugf("Nacos GetOneInstance serviceName[%v] err[%v] instance[%v]", serviceName, instance, err)
	if err != nil {
		return "", err
	}

	// There's no scheme info (HTTP/HTTPS) in the instance info (?!)
	// Default it to HTTPS (sending requests will require having the scheme)
	// What if the service only supports HTTP?
	url := fmt.Sprintf("https://%v:%v", instance.Ip, instance.Port)
	return url, nil
}
