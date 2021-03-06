package shared

// Client request types
type RequestType int

const (
	CREATE_FILE RequestType = iota
	LIST_FILES
	TOTAL_RECS
	READ_REC
	APPEND_REC
	DELETE_FILE
)

// Failure types
type FailureType int

const (
	BAD_FILENAME = iota
	DISCONNECTED
	FILE_DOES_NOT_EXIST
	FILE_EXISTS
	MAX_LEN_REACHED
	NOT_ENOUGH_MONEY
	APPEND_DUPLICATE
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
