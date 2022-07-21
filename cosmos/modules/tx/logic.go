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
	if msg == nil || msg.Events == nil {
		return nil
	}

	for _, log_event := range msg.Events {
		if log_event.Type == event_type {
			return &log_event
		}
	}

	return nil
}

//If order is reversed, the last attribute containing the given key will be returned
//otherwise the first attribute will be returned
func GetValueForAttribute(key string, evt *LogMessageEvent) string {
	if evt == nil || evt.Attributes == nil {
		return ""
	}

	for _, attr := range evt.Attributes {
		if attr.Key == key {
			return attr.Value
		}
	}

	return ""
}

func GetLastValueForAttribute(key string, evt *LogMessageEvent) string {
	if evt == nil || evt.Attributes == nil {
		return ""
	}

	for i := len(evt.Attributes) - 1; i >= 0; i-- {
		attr := evt.Attributes[i]
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
