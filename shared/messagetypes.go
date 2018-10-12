package shared

// Client request types
const (
	CREATE_FILE = iota
	LIST_FILES = iota
	TOTAL_RECS = iota
	READ_REC = iota
	APPEND_REC = iota
)

// Failure types
const (
	RECORD_DOES_NOT_EXIST = iota
	BAD_FILENAME = iota
	FILE_DOES_NOT_EXIST = iota
	FILE_EXISTS = iota
	MAX_LEN_REACHED = iota
)

type RFSClientRequest struct {
	RequestType int
	FileName string
	RecordNum uint16
	Record []byte
}

type RFSMinerResponse struct {
	// Set the ErrorType to -1 if no error occurred while processing the client request
	ErrorType int
	FileNames []string
	NumRecords uint16
	RecordNum uint16
}
