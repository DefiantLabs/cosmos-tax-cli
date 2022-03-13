package csv

import (
	"fmt"
	"time"
)

func FormatDatetime(t time.Time) string {
	//mm/dd/yyyy hh:MM:s
	result := fmt.Sprintf("%02d/%02d/%d %02d:%02d:%02d",
		t.Month(), t.Day(), t.Year(),
		t.Hour(), t.Minute(), t.Second())

	return result
}
