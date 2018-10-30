package shared

import (
	"github.com/DistributedClocks/GoVector/govec"
	"math"
	"time"
)

const (
	CLIENT_RETRY_COUNT = 10
	MAX_FILENAME_LENGTH = 64
	MAX_RECORD_COUNT uint16 = math.MaxUint16
	NUM_COINS_PER_FILE_APPEND = 1
	LISTENER_EXPIRATION = time.Minute * 30
	LOGFILE                   = "miner"
)

var GoVecOpts = govec.GoLogConfig{
	Buffered:      false,
	PrintOnScreen: false,
	AppendLog:     false,
	UseTimestamps: true,
	LogToFile:     true,
	Priority:      govec.DEBUG,
}
