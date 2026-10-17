package message

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"

	"github.com/handziurdmytro/messenger/pkg/errkit"
)

var ErrInvalidPayload = errors.New("invalid message payload")

type TextPayload struct {
	Text string `json:"text"`
}

type FilePayload struct {
	FileID   string `json:"file_id"`
	MIMEType string `json:"mime_type"`
	Name     string `json:"name,omitempty"`
	Caption  string `json:"caption,omitempty"`
}

func ValidatePayload(typ Type, raw json.RawMessage) error {
	if len(raw) == 0 || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return ErrInvalidPayload
	}

	switch typ {
	case TypeText, TypeSystem:
		var p TextPayload
		if err := json.Unmarshal(raw, &p); err != nil {
			return errkit.Wrap(ErrInvalidPayload, "decode text payload", err)
		}
		if strings.TrimSpace(p.Text) == "" {
			return ErrInvalidPayload
		}

	case TypeImage, TypeMusic, TypeVideo, TypeFile, TypeAudioMessage, TypeVideoMessage:
		var p FilePayload
		if err := json.Unmarshal(raw, &p); err != nil {
			return errkit.Wrap(ErrInvalidPayload, "decode file payload", err)
		}
		if strings.TrimSpace(p.FileID) == "" {
			return ErrInvalidPayload
		}

	default:
		return ErrInvalidMessageType
	}

	return nil
}
