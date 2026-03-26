package student_data

import "time"

func allHistoryRange(now time.Time, loc *time.Location) (time.Time, time.Time) {
	now = now.In(loc)

	// 历史起点
	from := time.Date(2009, 1, 1, 0, 0, 0, 0, loc)

	// 今天 00:00
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)

	return from, today
}
