package tx

func GetMessageLogForIndex(logs []TxLogMessage, index int) *TxLogMessage {
	for _, log := range logs {
		if log.MessageIndex == index {
			return &log
		}
	}

	return nil
}

func GetEventWithType(event_type string, msg *TxLogMessage) *LogMessageEvent {
	for _, log_event := range msg.Events {
		if log_event.Type == event_type {
			return &log_event
		}
	}

	return nil
}

func GetValueForAttribute(key string, evt *LogMessageEvent) string {
	for _, attr := range evt.Attributes {
		if attr.Key == key {
			return attr.Value
		}
	}

	return ""
}

func IsMessageActionEquals(message_type string, msg *TxLogMessage) bool {
	log_event := GetEventWithType("message", msg)
	if log_event == nil {
		return false
	}

	for _, attr := range log_event.Attributes {
		if attr.Key == "action" {
			return attr.Value == message_type
		}
	}

	return false
}