package sentinel

import (
	"github.com/alibaba/sentinel-golang/core/flow"
)

type Config struct {
	FlowRules []*flow.Rule `json:"flowRules"`
}
