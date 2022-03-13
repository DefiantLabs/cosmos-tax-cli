package csv

import "time"

func FormatDatetime(dt time.Time) string {
	formatter := "mm/dd/yyyy hh:MM:s"
	return dt.Format(formatter)
}
