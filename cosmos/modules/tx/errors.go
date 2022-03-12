package tx

import "fmt"

type UnknownMessageError struct {
	MessageType string
}

func (e *UnknownMessageError) Error() string {
	return fmt.Sprintf("No message handler for message type '%s'\n", e.MessageType)
}

func (e *UnknownMessageError) Type() string {
	return e.MessageType
}

type MessageLogFormatError struct {
	Log         string
	MessageType string
}

func (e *MessageLogFormatError) Error() string {
	return fmt.Sprintf("Type: %s could not handle message log %s\n", e.MessageType, e.Log)
}
