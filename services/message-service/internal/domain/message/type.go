package message

type (
	ID     string
	ChatID string
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
