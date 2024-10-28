package types

import "net/textproto"

// UploadFile upload file
type UploadFile struct {
	UID      string               `json:"uid,omitempty"`   // Content-Uid The unique identifier of the file ( for chunk upload )
	Range    string               `json:"range,omitempty"` // Content-Range bytes start-end/total (for chunk upload)
	Chunk    string               `json:"chunk,omitempty"` // Content-Chunk current/total (for chunk upload) ** this is not standard **
	Name     string               `json:"name"`
	TempFile string               `json:"tempFile"`
	Size     int64                `json:"size"`
	Header   textproto.MIMEHeader `json:"header"`
	Error    string               `json:"error"`
}
