package sentinel

import (
	"encoding/json"

	"github.com/0xPolygonHermez/zkevm-node/log"
	"github.com/alibaba/sentinel-golang/core/flow"
	"github.com/alibaba/sentinel-golang/ext/datasource"
	"github.com/alibaba/sentinel-golang/ext/datasource/file"
	"github.com/pkg/errors"
)

func InitFileDataSource(filePath string) error {
	if filePath == "" {
		log.Info("No sentinel config file name, ignored")
		return nil
	}

	// Handler to handle the config update
	propertyHandler := datasource.NewDefaultPropertyHandler(
		func(src []byte) (interface{}, error) {
			log.Debugf("sentinel raw config received: %v", string(src))
			cfg := &Config{FlowRules: make([]*flow.Rule, 0)}
			err := json.Unmarshal(src, cfg)
			if err != nil {
				log.Errorf("fail to unmarshal sentinel config[%v] err[%v]", src, err)
				return nil, err
			}

			return cfg, nil
		},
		func(data interface{}) error {
			cfg, ok := data.(*Config)
			if !ok {
				log.Errorf("invalid sentinel config data [%v]", data)
				return errors.New("invalid config")
			}
			// Load flow rules
			_, err := flow.LoadRules(cfg.FlowRules)
			if err != nil {
				log.Errorf("sentinel load flow rules[%v] err[%v]", cfg.FlowRules, err)
				return err
			}
			return nil
		})

	// Initialize the file data source
	source := file.NewFileDataSource(filePath, propertyHandler)
	err := source.Initialize()
	if err != nil {
		return errors.Wrap(err, "init sentinel data source err")
	}
	return nil
}
