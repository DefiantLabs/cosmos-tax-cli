package tx

import (
	"errors"
	"fmt"
)

var ErrUnknownMessage = errors.New("no message handler for message type")

type MessageLogFormatError struct {
	Log         string
	MessageType string
}

func (e *MessageLogFormatError) Error() string {
	return fmt.Sprintf("Type: %s could not handle message log %s", e.MessageType, e.Log)
}
