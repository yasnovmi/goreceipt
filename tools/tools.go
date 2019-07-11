package tools

import (
	"time"
)

func DateFromString(st string) (time.Time, error) {
	var err error
	var date time.Time
	if len(st) == 13 {
		date, err = time.Parse("20060102T1504", st)

	} else if len(st) == 15 {
		date, err = time.Parse("20060102T150405", st)
	}
	return date, err

}
