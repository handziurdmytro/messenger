package message

type Type string

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

func (t Type) ClientSendable() bool {
	return t.Valid() && t != TypeSystem
}
