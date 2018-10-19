package shared

type Filename string
type FileData []byte
type FileInfo struct {
	Creator         string
	NumberOfRecords uint32
	Data            FileData
}

