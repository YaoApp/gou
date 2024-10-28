package types

import (
	"fmt"
	"hash/fnv"
	"net/textproto"
	"strings"

	"github.com/google/uuid"
)

// UploadFile upload file
type UploadFile struct {
	UID      string               `json:"uid,omitempty"`   // Content-Uid The unique identifier of the file ( for chunk upload )
	Range    string               `json:"range,omitempty"` // Content-Range bytes start-end/total (for chunk upload)
	Sync     bool                 `json:"sync,omitempty"`  // Content-Sync sync upload or not. the default is false
	Name     string               `json:"name"`
	TempFile string               `json:"tempFile"`
	Size     int64                `json:"size"`
	Header   textproto.MIMEHeader `json:"header"`
	Error    string               `json:"error"`
}

// UploadProgress upload progress
type UploadProgress struct {
	Total     int64 `json:"total"`
	Uploaded  int64 `json:"uploaded"`
	Completed bool  `json:"completed"`
}

// IsChunk is chunk upload or not
func (upload UploadFile) IsChunk() bool {
	return upload.Range != ""
}

// IsError is error or not
func (upload UploadFile) IsError() bool {
	return upload.Error != ""
}

// Hash hash
func (upload UploadFile) Hash() string {

	h := fnv.New64a()
	// UUID Hash
	if upload.UID != "" {
		h.Write([]byte(fmt.Sprintf("%v", upload.UID)))
		return fmt.Sprintf("%x", h.Sum64())
	}

	// TempFile Hash (for chunk upload)
	if upload.TempFile != "" {
		h.Write([]byte(fmt.Sprintf("%v", upload.TempFile)))
		return fmt.Sprintf("%x", h.Sum64())
	}

	// General a uuid and hash
	uuid := uuid.NewString()
	h.Write([]byte(fmt.Sprintf("%v", uuid)))
	return fmt.Sprintf("%x", h.Sum64())
}

// ChunkFileName get chunk file name
func (upload UploadFile) ChunkFileName() string {
	name := strings.ReplaceAll(upload.Range, "bytes ", "")
	name = strings.ReplaceAll(name, "/", "_")
	return fmt.Sprintf("%s.chunk", name)
}

// TotalSize total size
func (upload UploadFile) TotalSize() int64 {
	var total int64
	nameInfo := strings.Split(upload.Range, "/")
	fmt.Sscanf(nameInfo[len(nameInfo)-1], "%d", &total)
	return total
}
