package estimatetime

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/0xPolygonHermez/zkevm-node/log"
	"github.com/pkg/errors"
)

const (
	// Number of deposits to get from DB to predict the estimate time
	sampleLimit           = 10
	refreshInterval       = 5 * time.Minute
	defaultL1EstimateTime = 15
	defaultL2EstimateTime = 60
)

type calculatorImpl struct {
	storage      DBStorage
	estimateTime []uint32 // In minutes
}

func NewCalculator(storage interface{}) (Calculator, error) {
	if storage == nil {
		return nil, errors.New("EstimateTime calculator: storage is nil")
	}
	c := &calculatorImpl{
		storage:      storage.(DBStorage),
		estimateTime: []uint32{defaultL1EstimateTime, defaultL2EstimateTime},
	}
	c.init()
	return c, nil
}

func (c *calculatorImpl) init() {
	c.refreshAll()

	go func() {
		ticker := time.NewTicker(refreshInterval)
		for range ticker.C {
			c.refreshAll()
		}
	}()
}

func (c *calculatorImpl) refreshAll() {
	ctx := context.Background()
	for i := 0; i < 2; i++ {
		err := c.refresh(ctx, uint(i))
		if err != nil {
			log.Errorf("Refresh estimate time for networkId %v error: %v", i, err)
		}
	}
	log.Infof("Refresh deposit estimate time, new estimations: %v", c.estimateTime)
}

func (c *calculatorImpl) refresh(ctx context.Context, networkID uint) error {
	if networkID > 1 {
		return fmt.Errorf("invalid networkID %v", networkID)
	}
	deposits, err := c.storage.GetLatestReadyDeposits(ctx, networkID, sampleLimit, nil)
	if err != nil {
		log.Errorf("GetLatestReadyDeposits err:%v", err)
		return err
	}

	fMinutes := make([]float64, 0)
	for _, deposit := range deposits {
		// Filter out the edge cases where the times are not valid
		if deposit.Time.IsZero() || deposit.ReadyTime.IsZero() || deposit.Time.After(deposit.ReadyTime) {
			continue
		}

		// Collect the valid time ranges
		fMinutes = append(fMinutes, deposit.ReadyTime.Sub(deposit.Time).Minutes())
	}

	if len(fMinutes) == 0 {
		return nil
	}

	// Calculate the average minutes
	sum := float64(0)
	for _, m := range fMinutes {
		sum += m
	}
	newTime := uint32(math.Ceil(sum / float64(len(fMinutes))))
	log.Debugf("Re-calculate estimate time, networkID[%v], fMinutes[%v], newTime[%v]", networkID, fMinutes, newTime)
	defaultTime := uint32(defaultL1EstimateTime)
	if networkID != 0 {
		defaultTime = defaultL2EstimateTime
	}
	if newTime > defaultTime {
		newTime = defaultTime
	}
	c.estimateTime[networkID] = newTime
	return nil
}

// Get returns the estimated deposit time for the network by networkID
func (c *calculatorImpl) Get(networkID uint) uint32 {
	return c.estimateTime[networkID]
}
