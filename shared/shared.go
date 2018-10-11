package shared

const (
	CREATE_FILE = iota
	LIST_FILES = iota
	TOTAL_RECS = iota
	READ_REC = iota
	APPEND_REC = iota
)

type RFSClientRequest struct {
	RequestType int
	FileName string
	RecordNum uint16
	Record []byte
}

type RFSMinerResponse struct {
	Err error
	FileNames []string
	NumRecords uint16
	RecordNum uint16
}
