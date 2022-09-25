package accointing

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

func DateFromString(dateString string) (time.Time, error) {
	//See the explanation of the time.Parse function for why we use a layout string
	//like this
	layout := "01/02/2006 15:04:05"
	return time.Parse(layout, dateString)
}
