package anomaly

import (
	"aATA/internal/model"
	"time"
)

type studentDifficultyLevel struct {
	CF     int
	AC     int
	HasCF  bool
	HasAC  bool
}

func buildStudentDifficultyLevel(records []*model.ContestRecord, cfg RuleConfig) studentDifficultyLevel {
	level := studentDifficultyLevel{}
	for _, record := range records {
		if record == nil {
			continue
		}
		rounded := roundDown(record.NewRating, cfg.DifficultyLevelRoundBase)
		switch record.Platform {
		case "CF":
			level.CF = rounded
			level.HasCF = true
		case "AC":
			level.AC = rounded
			level.HasAC = true
		}
	}
	return level
}

func roundDown(value, base int) int {
	if base <= 0 {
		return value
	}
	if value < 0 {
		return 0
	}
	return (value / base) * base
}

func sumTotalSolved(items []*model.DailyTrainingStats) int {
	total := 0
	for _, item := range items {
		if item == nil {
			continue
		}
		total += item.CFNewTotal + item.ACNewTotal
	}
	return total
}

func totalHighEasyByLevel(
	stats *model.DailyTrainingStats,
	level studentDifficultyLevel,
	cfg RuleConfig,
) (int, int, int) {
	if stats == nil {
		return 0, 0, 0
	}

	total := 0
	high := 0
	easy := 0

	cfBuckets := map[int]int{
		800:  stats.CFNew800,
		900:  stats.CFNew900,
		1000: stats.CFNew1000,
		1100: stats.CFNew1100,
		1200: stats.CFNew1200,
		1300: stats.CFNew1300,
		1400: stats.CFNew1400,
		1500: stats.CFNew1500,
		1600: stats.CFNew1600,
		1700: stats.CFNew1700,
		1800: stats.CFNew1800,
		1900: stats.CFNew1900,
		2000: stats.CFNew2000,
		2100: stats.CFNew2100,
		2200: stats.CFNew2200,
		2300: stats.CFNew2300,
		2400: stats.CFNew2400,
		2500: stats.CFNew2500,
		2600: stats.CFNew2600,
		2700: stats.CFNew2700,
		2800: stats.CFNew2800Plus,
	}
	acBuckets := map[int]int{
		200:  stats.ACNew0_399,
		600:  stats.ACNew400_799,
		1000: stats.ACNew800_1199,
		1400: stats.ACNew1200_1599,
		1800: stats.ACNew1600_1999,
		2200: stats.ACNew2000_2399,
		2600: stats.ACNew2400_2799,
		2800: stats.ACNew2800Plus,
	}

	if level.HasCF {
		for diff, cnt := range cfBuckets {
			if cnt <= 0 {
				continue
			}
			total += cnt
			if diff >= level.CF+cfg.DifficultyRelativeHighDelta {
				high += cnt
			}
			if diff <= level.CF-cfg.DifficultyRelativeEasyDelta {
				easy += cnt
			}
		}
	}

	if level.HasAC {
		for diff, cnt := range acBuckets {
			if cnt <= 0 {
				continue
			}
			total += cnt
			if diff >= level.AC+cfg.DifficultyRelativeHighDelta {
				high += cnt
			}
			if diff <= level.AC-cfg.DifficultyRelativeEasyDelta {
				easy += cnt
			}
		}
	}

	return total, high, easy
}

func safeDiv(numerator, denominator float64) float64 {
	if denominator == 0 {
		return 0
	}
	return numerator / denominator
}

func dateOnly(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

func round2(v float64) float64 {
	return float64(int(v*100+0.5)) / 100
}

func round4(v float64) float64 {
	return float64(int(v*10000+0.5)) / 10000
}
