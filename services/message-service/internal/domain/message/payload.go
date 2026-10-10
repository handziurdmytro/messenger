package message

import "strings"

type Payload interface {
	Validate() error
}

type TextPayload struct {
	Text string `json:"text"`
}

func (p TextPayload) Validate() error {
	if strings.TrimSpace(p.Text) == "" {
		return ErrMessageEmpty
	}
	return nil
}

type FilePayload struct {
	FileID   string `json:"file_id"`
	MIMEType string `json:"mime_type"`
	Name     string `json:"name,omitempty"`
	Caption  string `json:"caption,omitempty"`
}

func (p FilePayload) Validate() error {
	if strings.TrimSpace(p.FileID) == "" {
		return ErrFileIDEmpty
	}
	return nil
}

func validatePayload(t Type, p Payload) error {
	if !t.Accepts(p) {
		return ErrPayloadTypeMismatch
	}
	return p.Validate()
}
