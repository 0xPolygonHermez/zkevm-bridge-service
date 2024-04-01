package apolloconfig

import (
	"github.com/0xPolygonHermez/zkevm-bridge-service/log"
	"github.com/apolloconfig/agollo/v4"
)

func getLogger() *log.Logger {
	return log.WithFields(loggerFieldKey, loggerFieldValue)
}

func SetLogger() {
	agollo.SetLogger(getLogger())
}
