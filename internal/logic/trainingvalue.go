package logic

import (
	"aATA/internal/domain"
	"aATA/internal/model"
)

// BuildTrainingDistributions 把落库统计转成前端和 Agent 直接使用的区间分布。
// 这里只做字段整理，不追加任何推断或补全逻辑。
func BuildTrainingDistributions(stats *model.DailyTrainingStats) (map[string]int, map[string]int) {
	if stats == nil {
		return map[string]int{}, map[string]int{}
	}

	cfDist := map[string]int{
		"undefined": stats.CFNewUndefined,
		"800_1100":  stats.CFNew800 + stats.CFNew900 + stats.CFNew1000 + stats.CFNew1100,
		"1200_1300": stats.CFNew1200 + stats.CFNew1300,
		"1400_1500": stats.CFNew1400 + stats.CFNew1500,
		"1600_1800": stats.CFNew1600 + stats.CFNew1700 + stats.CFNew1800,
		"1900_2000": stats.CFNew1900 + stats.CFNew2000,
		"2100_2200": stats.CFNew2100 + stats.CFNew2200,
		"2300_plus": stats.CFNew2300 + stats.CFNew2400 +
			stats.CFNew2500 + stats.CFNew2600 +
			stats.CFNew2700 + stats.CFNew2800Plus,
	}
	acDist := map[string]int{
		"undefined": stats.ACNewUndefined,
		"0_399":     stats.ACNew0_399,
		"400_799":   stats.ACNew400_799,
		"800_1199":  stats.ACNew800_1199,
		"1200_1599": stats.ACNew1200_1599,
		"1600_1999": stats.ACNew1600_1999,
		"2000_2399": stats.ACNew2000_2399,
		"2400_2799": stats.ACNew2400_2799,
		"2800_plus": stats.ACNew2800Plus,
	}
	return cfDist, acDist
}

// BuildTrainingValueSummary 按排行榜同一套公式计算单个学生的训练价值拆解。
// 该函数只依赖已落库的训练统计和比赛记录，不负责抓取、补数或时间范围解析。
func BuildTrainingValueSummary(
	stats *model.DailyTrainingStats,
	records []*model.ContestRecord,
) domain.TrainingValueSummary {
	if stats == nil {
		stats = &model.DailyTrainingStats{}
	}

	cfRating, acRating := buildRatingProfiles(records)
	cfScore := scorePlatform(
		buildCFBuckets(stats),
		stats.CFNewTotal,
		stats.CFNewUndefined,
		cfRating,
		cfFallbackDifficulty,
	)
	acScore := scorePlatform(
		buildACBuckets(stats),
		stats.ACNewTotal,
		stats.ACNewUndefined,
		acRating,
		acFallbackDifficulty,
	)

	solvedTotal := cfScore.solvedTotal + acScore.solvedTotal
	undefinedTotal := cfScore.undefinedTotal + acScore.undefinedTotal

	return domain.TrainingValueSummary{
		ScoringVersion:  trainingLeaderboardVersion,
		SolvedTotal:     solvedTotal,
		Score:           round2(cfScore.score + acScore.score),
		VolumeScore:     round2(cfScore.volumeScore + acScore.volumeScore),
		DifficultyScore: round2(cfScore.difficultyScore + acScore.difficultyScore),
		ChallengeScore:  round2(cfScore.challengeScore + acScore.challengeScore),
		UndefinedTotal:  undefinedTotal,
		UndefinedRatio:  round4(safeDiv(float64(undefinedTotal), float64(solvedTotal))),
		CFRating:        toDomainRatingProfile(cfRating),
		ACRating:        toDomainRatingProfile(acRating),
		CF:              toDomainPlatformScore(cfScore),
		AC:              toDomainPlatformScore(acScore),
	}
}
