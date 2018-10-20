package shared

// Client request types
type RequestType int

const (
	CREATE_FILE RequestType = iota
	LIST_FILES
	TOTAL_RECS
	READ_REC
	APPEND_REC
)

// Failure types
type FailureType int

const (
	BAD_FILENAME = iota
	FILE_DOES_NOT_EXIST
	FILE_EXISTS
	MAX_LEN_REACHED
	NO_ERROR = -1
)

type RFSClientRequest struct {
	RequestType  RequestType
	FileName     string
	RecordNum    uint16
	AppendRecord [512]byte
}

type RFSMinerResponse struct {
	// Set the ErrorType to -1 if no error occurred while processing the client request
	ErrorType  FailureType
	FileNames  []string
	NumRecords uint16
	RecordNum  uint16
	ReadRecord [512]byte
}
