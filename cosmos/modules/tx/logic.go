package tx

func GetMessageLogForIndex(logs []LogMessage, index int) *LogMessage {
	for _, log := range logs {
		if log.MessageIndex == index {
			return &log
		}
	}

	return nil
}

func GetEventWithType(eventType string, msg *LogMessage) *LogMessageEvent {
	if msg == nil || msg.Events == nil {
		return nil
	}

	for _, logEvent := range msg.Events {
		if logEvent.Type == eventType {
			return &logEvent
		}
	}

	return nil
}

// If order is reversed, the last attribute containing the given key will be returned
// otherwise the first attribute will be returned
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

func IsMessageActionEquals(messageType string, msg *LogMessage) bool {
	logEvent := GetEventWithType("message", msg)
	if logEvent == nil {
		return false
	}

	for _, attr := range logEvent.Attributes {
		if attr.Key == "action" {
			return attr.Value == messageType
		}
	}

	return false
}
