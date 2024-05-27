package utils

import (
	"fmt"
	"time"
)

const (
	largeTxCacheHourLimit   = 19
	toL1                    = "toL1"
	toL2                    = "toL2"
	durationForDay          = 24 * time.Hour
	largeTxCacheMaxAliveDay = 2
	OpRead                  = 0
	OpWrite                 = 1
	OpDel                   = 2
)

func GetLargeTxRedisKeySuffix(toNetworkId uint, opType int) string {
	year, month, day := getCacheDate(opType)
	direction := toL2
	if toNetworkId == 0 {
		direction = toL1
	}
	return fmt.Sprintf("%s-%d-%02d-%02d", direction, year, month, day)
}

func getCacheDate(opType int) (year int, month time.Month, day int) {
	if opType == OpRead {
		return time.Now().Date()
	}
	var date time.Time
	if time.Now().Hour() < largeTxCacheHourLimit {
		date = time.Now()
	} else {
		date = time.Now().Add(durationForDay)
	}
	if opType == OpWrite {
		return date.Date()
	}
	return date.Add(GetLargeTxCacheExpireDuration()).Date()
}

func GetLargeTxCacheExpireDuration() time.Duration {
	return largeTxCacheMaxAliveDay * durationForDay
}
