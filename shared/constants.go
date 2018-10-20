package shared

import "github.com/DistributedClocks/GoVector/govec"

const (
	NUM_COINS_PER_FILE_APPEND = 1
	LOGFILE = "miner"
)

var GoVecOpts = govec.GoLogConfig{
	Buffered:      false,
	PrintOnScreen: false,
	AppendLog:     false,
	UseTimestamps: true,
	LogToFile:     true,
	Priority:      govec.DEBUG,
}
