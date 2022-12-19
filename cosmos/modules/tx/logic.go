package tx

import (
	"fmt"
	"strings"
	"unicode"
)

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

// Get the Nth value for the given key (starting at 1)
func GetNthValueForAttribute(key string, n int, evt *LogMessageEvent) string {
	if evt == nil || evt.Attributes == nil {
		return ""
	}
	var count int
	for i := 0; i < len(evt.Attributes); i++ {
		attr := evt.Attributes[i]
		if attr.Key == key {
			count++
			if count == n {
				return attr.Value
			}
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

func IsMessageActionEquals(msgType string, msg *LogMessage) bool {
	logEvent := GetEventWithType("message", msg)
	altMsgType := getAltMsgType(msgType)
	if logEvent == nil {
		return false
	}

	for _, attr := range logEvent.Attributes {
		if attr.Key == "action" {
			if attr.Value == msgType || attr.Value == altMsgType {
				return true
			}
		}
	}

	return false
}

var altMsgMap = map[string]string{
	"/cosmos.staking.v1beta1.MsgUndelegate": "begin_unbonding",
}

func getAltMsgType(msgType string) string {
	if altMsg, ok := altMsgMap[msgType]; ok {
		return altMsg
	}

	var output string
	msgParts := strings.Split(msgType, ".Msg")
	if len(msgParts) == 2 {
		msgSuffix := msgParts[1]
		for i, char := range msgSuffix {
			if unicode.IsUpper(char) {
				if i != 0 {
					output = fmt.Sprintf("%v_", output)
				}
			}
			output = fmt.Sprintf("%v%v", output, string(unicode.ToLower(char)))
		}
	}
	return output
}
