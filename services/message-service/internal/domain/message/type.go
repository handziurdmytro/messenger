package message

type (
	ID     string
	RoomID string
	Type   string
)

const (
	TypeText         Type = "text"
	TypeImage        Type = "image"
	TypeMusic        Type = "music"
	TypeVideo        Type = "video"
	TypeFile         Type = "file"
	TypeAudioMessage Type = "audio_message"
	TypeVideoMessage Type = "video_message"
	TypeSystem       Type = "system"
)

func (t Type) Valid() bool {
	switch t {
	case TypeText, TypeImage, TypeMusic, TypeVideo, TypeFile, TypeAudioMessage, TypeVideoMessage, TypeSystem:
		return true
	default:
		return false
	}
}

func (t Type) CanEdit() bool {
	switch t {
	case TypeText, TypeImage, TypeMusic, TypeVideo, TypeFile:
		return true
	default:
		return false
	}
}

func (t Type) CanDelete() bool {
	return t.Valid() && t != TypeSystem
}

func (t Type) Accepts(p Payload) bool {
	switch t {
	case TypeText, TypeSystem:
		_, ok := p.(TextPayload)
		return ok
	case TypeImage, TypeMusic, TypeVideo, TypeFile, TypeAudioMessage, TypeVideoMessage:
		_, ok := p.(FilePayload)
		return ok
	default:
		return false
	}
}
