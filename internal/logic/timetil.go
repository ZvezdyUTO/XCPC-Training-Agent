package logic

import "time"

// dayRange returns [day 00:00:00, day 23:59:59.999999999] in the given location.
func dayRange(day time.Time, loc *time.Location) (time.Time, time.Time) {
	d := day.In(loc)
	from := time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, loc)
	to := from.Add(24*time.Hour - time.Nanosecond)
	return from, to
}

func allHistoryRange(now time.Time, loc *time.Location) (time.Time, time.Time) {
	from := time.Date(1970, 1, 1, 0, 0, 0, 0, loc)
	to := time.Date(now.In(loc).Year(), now.In(loc).Month(), now.In(loc).Day(), 23, 59, 59, int(time.Second-time.Nanosecond), loc)
	return from, to
}
