package sentinel

import (
	"encoding/json"
	"strings"
	"sync"

	"github.com/0xPolygonHermez/zkevm-bridge-service/config/apolloconfig"
	"github.com/0xPolygonHermez/zkevm-node/log"
	"github.com/alibaba/sentinel-golang/api"
	"github.com/alibaba/sentinel-golang/core/config"
	"github.com/alibaba/sentinel-golang/core/flow"
	"github.com/alibaba/sentinel-golang/ext/datasource"
	"github.com/alibaba/sentinel-golang/ext/datasource/file"
	"github.com/alibaba/sentinel-golang/pkg/datasource/apollo"
	agolloConfig "github.com/apolloconfig/agollo/v4/env/config"
	"github.com/pkg/errors"
)

const (
	apolloConfigKey  = "sentinel.rules"
	loggerFieldKey   = "component"
	loggerFieldValue = "sentinel"
)

var (
	initOnce sync.Once
	logger   *log.Logger
)

func initSentinel() {
	logger = log.WithFields(loggerFieldKey, loggerFieldValue)
	initOnce.Do(func() {
		conf := config.NewDefaultConfig()
		err := api.InitWithConfig(conf)
		if err != nil {
			logger.Errorf("initSentinel error: %v, ignored", err)
		}
	})
}

func InitFileDataSource(filePath string) error {
	if filePath == "" {
		logger.Info("No sentinel config file name, ignored")
		return nil
	}

	initSentinel()

	// Initialize the file data source
	source := file.NewFileDataSource(filePath, getPropertyHandler())
	err := source.Initialize()
	if err != nil {
		return errors.Wrap(err, "init sentinel data source err")
	}
	return nil
}

func InitApolloDataSource(c apolloconfig.Config) error {
	initSentinel()

	cfg := &agolloConfig.AppConfig{
		AppID:          c.AppID,
		Cluster:        c.Cluster,
		IP:             c.MetaAddress,
		NamespaceName:  strings.Join(c.Namespaces, ","),
		Secret:         c.Secret,
		IsBackupConfig: c.IsBackupConfig,
	}

	// Init Apollo datasource
	source, err := apollo.NewDatasource(cfg, apolloConfigKey, apollo.WithPropertyHandlers(getPropertyHandler()), apollo.WithLogger(logger))
	if err != nil {
		return errors.Wrap(err, "init sentinel apollo data source err")
	}
	err = source.Initialize()
	if err != nil {
		return errors.Wrap(err, "init sentinel apollo data source err")
	}
	return nil
}

// Handler to handle the config update
func getPropertyHandler() datasource.PropertyHandler {
	return datasource.NewDefaultPropertyHandler(
		func(src []byte) (interface{}, error) {
			logger.Debugf("sentinel raw config received: %v", string(src))
			cfg := &Config{FlowRules: make([]*flow.Rule, 0)}
			err := json.Unmarshal(src, cfg)
			if err != nil {
				logger.Errorf("fail to unmarshal sentinel config[%v] err[%v]", src, err)
				return nil, err
			}

			return cfg, nil
		},
		func(data interface{}) error {
			cfg, ok := data.(*Config)
			if !ok {
				logger.Errorf("invalid sentinel config data [%v]", data)
				return errors.New("invalid config")
			}
			// Load flow rules
			_, err := flow.LoadRules(cfg.FlowRules)
			if err != nil {
				logger.Errorf("sentinel load flow rules[%v] err[%v]", cfg.FlowRules, err)
				return err
			}
			return nil
		})
}
