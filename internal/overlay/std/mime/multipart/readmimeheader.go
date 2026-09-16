//go:build dart

package multipart

import "net/textproto"

func readMIMEHeader(r *textproto.Reader, maxMemory, maxHeaders int64) (textproto.MIMEHeader, error) {
	return textproto.ReadMIMEHeaderLimited(r, maxMemory, maxHeaders)
}
