package runtimeconfig

import "sync/atomic"

var monthRotateDay atomic.Int32

func MonthRotateDay() int {
	return int(monthRotateDay.Load())
}

func SetMonthRotateDay(day int) {
	monthRotateDay.Store(int32(day))
}
