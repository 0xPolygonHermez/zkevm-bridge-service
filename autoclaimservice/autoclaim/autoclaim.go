package autoclaim

import(

	"github.com/0xPolygonHermez/zkevm-bridge-service/log"
)
type autoclaim struct {

}

func NewAutoClaim(c *Config) (autoclaim, error) {
	log.Infof("autoclaim config: %+v", c)
	return autoclaim{}, nil
}

func (ac autoclaim) Start() {


}