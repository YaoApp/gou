package types

import "net/textproto"

// UploadFile upload file
type UploadFile struct {
	Name     string               `json:"name"`
	TempFile string               `json:"tempFile"`
	Size     int64                `json:"size"`
	Header   textproto.MIMEHeader `json:"mimeType"`
}
