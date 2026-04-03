package student_data

import "time"

const repairLookbackDays = 5

// defaultUTC8Location 返回爬取系统统一使用的 UTC+8 时区。
// 这里优先复用 IANA 时区名，避免直接依赖宿主机的本地时区配置。
func defaultUTC8Location() *time.Location {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err == nil {
		return loc
	}
	return time.FixedZone("UTC+8", 8*60*60)
}

// nowInLoc 返回统一时区下的当前时间，避免业务逻辑直接混用 time.Now() 与不同 Location。
func nowInLoc(loc *time.Location) time.Time {
	return time.Now().In(normalizeLocation(loc))
}

// normalizeLocation 负责将外部传入的空时区收敛为系统统一的 UTC+8。
func normalizeLocation(loc *time.Location) *time.Location {
	if loc != nil {
		return loc
	}
	return defaultUTC8Location()
}

// repairSyncRange 基于最近一次成功日期计算修复爬取区间。
// 设计意图是每次回刷最近 5 天，覆盖平台延迟、补评测或补抓取导致的晚到数据。
func repairSyncRange(latest time.Time, now time.Time, loc *time.Location) (time.Time, time.Time) {
	baseLoc := normalizeLocation(loc)
	to := now.In(baseLoc)
	from := dateOnly(latest.In(baseLoc).AddDate(0, 0, -repairLookbackDays), baseLoc)
	return from, to
}

// allHistoryRange 返回全量初始化使用的固定历史区间。
// 上界按 UTC+8 的“今天 00:00”计算，表示补到当前自然日。
func allHistoryRange(now time.Time, loc *time.Location) (time.Time, time.Time) {
	loc = normalizeLocation(loc)
	now = now.In(loc)

	// 历史起点
	from := time.Date(2009, 1, 1, 0, 0, 0, 0, loc)

	// 今天 00:00
	//today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)

	return from, now
}
