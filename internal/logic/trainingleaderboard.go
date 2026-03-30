package logic

import (
	"aATA/internal/domain"
	"aATA/internal/model"
	"context"
	"fmt"
	"math"
	"sort"
	"time"
)

const (
	trainingLeaderboardVersion          = "training_value_v2"
	trainingLeaderboardDefaultTopN      = 20
	trainingLeaderboardMaxTopN          = 100
	trainingLeaderboardBaseScore        = 1.0
	trainingLeaderboardPeakCap          = 300
	trainingLeaderboardPeakFactor       = 0.2
	trainingLeaderboardUndefinedK       = 0.85
	trainingLeaderboardTargetDaily      = 3.0
	trainingLeaderboardLowDaily         = 2.0
	trainingLeaderboardLowDailyFactor   = 0.78
	trainingLeaderboardMidDailyFactor   = 0.9
	trainingLeaderboardContestLookback  = 60
	trainingLeaderboardContestRecentMax = 4
	trainingLeaderboardContestScale     = 3.0
	trainingLeaderboardContestDivisor   = 80.0
	trainingLeaderboardContestRecordCap = 1.4
	trainingLeaderboardTargetDelta      = 250.0
	trainingLeaderboardChallengePeak    = 0.45
	trainingLeaderboardChallengeTail    = 0.12
	trainingLeaderboardDifficultyPeak   = 0.4
	trainingLeaderboardDifficultyTail   = 0.08
	trainingLeaderboardDifficultyFloor  = 0.04
	trainingLeaderboardAbsoluteDiffBase = 900
	trainingLeaderboardAbsoluteDiffStep = 0.18
	trainingLeaderboardAbsoluteDiffCap  = 4.0
)

var (
	cfFallbackDifficulty = 1200
	acFallbackDifficulty = 1000
)

// TrainingLeaderboard 负责按时间区间计算训练价值排行榜。
// 该逻辑只读取已落库的训练统计和比赛 rating，不负责触发抓取或补数据。
type TrainingLeaderboard interface {
	Query(ctx context.Context, req *domain.TrainingLeaderboardReq) (*domain.TrainingLeaderboardResp, error)
}

type trainingLeaderboard struct {
	users   model.UsersModel
	daily   model.DailyTrainingStatsModel
	contest model.ContestRecordModel
}

type ratingProfile struct {
	current *int
	peak    *int
	ability *float64
}

type platformScore struct {
	solvedTotal     int
	knownTotal      int
	undefinedTotal  int
	score           float64
	volumeScore     float64
	difficultyScore float64
	challengeScore  float64
}

// NewTrainingLeaderboard 创建训练价值排行榜逻辑。
// 排行分数由题量、难度和相对本人能力线的挑战价值共同决定。
func NewTrainingLeaderboard(
	users model.UsersModel,
	daily model.DailyTrainingStatsModel,
	contest model.ContestRecordModel,
) TrainingLeaderboard {
	return &trainingLeaderboard{
		users:   users,
		daily:   daily,
		contest: contest,
	}
}

// Query 计算指定时间区间内的训练价值排行榜。
// 这里不会对时间区间做隐式扩展，输入什么范围就计算什么范围。
func (l *trainingLeaderboard) Query(
	ctx context.Context,
	req *domain.TrainingLeaderboardReq,
) (*domain.TrainingLeaderboardResp, error) {
	fromTime, err := time.Parse("2006-01-02", req.From)
	if err != nil {
		return nil, err
	}
	toTime, err := time.Parse("2006-01-02", req.To)
	if err != nil {
		return nil, err
	}
	if toTime.Before(fromTime) {
		return nil, fmt.Errorf("to must be greater than or equal to from")
	}

	topN := req.TopN
	if topN <= 0 {
		topN = trainingLeaderboardDefaultTopN
	}
	if topN > trainingLeaderboardMaxTopN {
		topN = trainingLeaderboardMaxTopN
	}

	users, _, err := l.users.List(ctx, &domain.UserListReq{})
	if err != nil {
		return nil, err
	}

	items := make([]domain.TrainingLeaderboardItem, 0, len(users))
	for _, user := range users {
		if user.IsSystem == model.IsSystemUser {
			continue
		}

		stats, err := l.daily.SumRange(ctx, user.Id, fromTime, toTime)
		if err != nil {
			return nil, fmt.Errorf("sum training stats for %s failed: %w", user.Id, err)
		}

		records, err := l.contest.FindByStudent(ctx, user.Id)
		if err != nil {
			return nil, fmt.Errorf("load contest records for %s failed: %w", user.Id, err)
		}

		summary := BuildTrainingValueSummary(stats, records, fromTime, toTime)
		if summary.SolvedTotal == 0 {
			continue
		}

		items = append(items, domain.TrainingLeaderboardItem{
			StudentID:       user.Id,
			StudentName:     user.Name,
			SolvedTotal:     summary.SolvedTotal,
			DailyAverage:    summary.DailyAverage,
			Score:           summary.Score,
			VolumeScore:     summary.VolumeScore,
			DifficultyScore: summary.DifficultyScore,
			ChallengeScore:  summary.ChallengeScore,
			ContestScore:    summary.ContestScore,
			UndefinedTotal:  summary.UndefinedTotal,
			UndefinedRatio:  summary.UndefinedRatio,
			CFRating:        summary.CFRating,
			ACRating:        summary.ACRating,
			CF:              summary.CF,
			AC:              summary.AC,
		})
	}

	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Score != items[j].Score {
			return items[i].Score > items[j].Score
		}
		if items[i].ChallengeScore != items[j].ChallengeScore {
			return items[i].ChallengeScore > items[j].ChallengeScore
		}
		if items[i].DifficultyScore != items[j].DifficultyScore {
			return items[i].DifficultyScore > items[j].DifficultyScore
		}
		if items[i].SolvedTotal != items[j].SolvedTotal {
			return items[i].SolvedTotal > items[j].SolvedTotal
		}
		return items[i].StudentID < items[j].StudentID
	})

	if len(items) > topN {
		items = items[:topN]
	}
	for i := range items {
		items[i].Rank = i + 1
	}

	return &domain.TrainingLeaderboardResp{
		ScoringVersion: trainingLeaderboardVersion,
		From:           req.From,
		To:             req.To,
		TopN:           topN,
		Count:          len(items),
		Items:          items,
	}, nil
}

// buildRatingProfiles 从全部比赛记录中提取 CF/AC 的当前分、最高分和能力参考线。
// 能力参考线以当前分为主，只让历史峰值做有限上调，避免榜单退化成“本来就强的人天然领先”。
func buildRatingProfiles(records []*model.ContestRecord) (ratingProfile, ratingProfile) {
	profiles := map[string]ratingProfile{
		"CF": {},
		"AC": {},
	}

	for _, record := range records {
		profile := profiles[record.Platform]
		if profile.current == nil {
			current := record.NewRating
			peak := record.NewRating
			profile.current = &current
			profile.peak = &peak
		} else {
			current := record.NewRating
			profile.current = &current
			if record.NewRating > *profile.peak {
				peak := record.NewRating
				profile.peak = &peak
			}
		}
		profiles[record.Platform] = profile
	}

	cf := finalizeRatingProfile(profiles["CF"])
	ac := finalizeRatingProfile(profiles["AC"])
	return cf, ac
}

func finalizeRatingProfile(profile ratingProfile) ratingProfile {
	if profile.current == nil || profile.peak == nil {
		return profile
	}

	peakGain := *profile.peak - *profile.current
	if peakGain < 0 {
		peakGain = 0
	}
	if peakGain > trainingLeaderboardPeakCap {
		peakGain = trainingLeaderboardPeakCap
	}

	ability := float64(*profile.current) + float64(peakGain)*trainingLeaderboardPeakFactor
	profile.ability = float64Ptr(round2(ability))
	return profile
}

func buildCFBuckets(stats *model.DailyTrainingStats) map[int]int {
	if stats == nil {
		return map[int]int{}
	}
	return map[int]int{
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
}

func buildACBuckets(stats *model.DailyTrainingStats) map[int]int {
	if stats == nil {
		return map[int]int{}
	}
	return map[int]int{
		200:  stats.ACNew0_399,
		600:  stats.ACNew400_799,
		1000: stats.ACNew800_1199,
		1400: stats.ACNew1200_1599,
		1800: stats.ACNew1600_1999,
		2200: stats.ACNew2000_2399,
		2600: stats.ACNew2400_2799,
		2800: stats.ACNew2800Plus,
	}
}

// scorePlatform 计算单个平台的训练分。
// 分数拆成题量、难度、挑战三部分；undefined 题做谨慎折算，不再按明显低质量处理。
func scorePlatform(
	knownBuckets map[int]int,
	totalCount int,
	undefinedCount int,
	rating ratingProfile,
	fallbackDifficulty int,
) platformScore {
	knownTotal := 0
	difficultyScore := 0.0
	challengeScore := 0.0

	for diff, count := range knownBuckets {
		if count <= 0 {
			continue
		}
		knownTotal += count
		difficultyScore += float64(count) * difficultyBonus(diff, rating)
		challengeScore += float64(count) * challengeBonus(diff, rating)
	}

	resolvedTotal := totalCount
	if resolvedTotal < knownTotal+undefinedCount {
		resolvedTotal = knownTotal + undefinedCount
	}

	if undefinedCount > 0 {
		avgDiff, avgChallenge := estimateUndefinedBonuses(
			knownTotal,
			difficultyScore,
			challengeScore,
			fallbackDifficulty,
			rating,
		)
		difficultyScore += float64(undefinedCount) * avgDiff * trainingLeaderboardUndefinedK
		challengeScore += float64(undefinedCount) * avgChallenge * trainingLeaderboardUndefinedK
	}

	volumeScore := float64(resolvedTotal) * trainingLeaderboardBaseScore
	totalScore := volumeScore + difficultyScore + challengeScore

	return platformScore{
		solvedTotal:     resolvedTotal,
		knownTotal:      knownTotal,
		undefinedTotal:  undefinedCount,
		score:           totalScore,
		volumeScore:     volumeScore,
		difficultyScore: difficultyScore,
		challengeScore:  challengeScore,
	}
}

func estimateUndefinedBonuses(
	knownTotal int,
	difficultyScore float64,
	challengeScore float64,
	fallbackDifficulty int,
	rating ratingProfile,
) (float64, float64) {
	if knownTotal > 0 {
		return difficultyScore / float64(knownTotal), challengeScore / float64(knownTotal)
	}
	return difficultyBonus(fallbackDifficulty, rating), challengeBonus(fallbackDifficulty, rating)
}

// calcRangeDays 返回包含起止日期的自然日数量。
// 训练量按天评估时采用闭区间，避免同一天查询被误判成零天。
func calcRangeDays(from, to time.Time) int {
	if to.Before(from) {
		return 0
	}
	return int(to.Sub(from).Hours()/24) + 1
}

// calcVolumeFactor 根据区间日均题量给题量部分加一个温和系数。
// 这里把 3 题/天视为稳定训练量，2 题/天以下只做轻度折减，因为系统未覆盖全部 OJ。
func calcVolumeFactor(totalSolved, days int) (float64, float64) {
	if days <= 0 {
		return 1, 0
	}

	dailyAverage := safeDiv(float64(totalSolved), float64(days))
	switch {
	case dailyAverage >= trainingLeaderboardTargetDaily:
		return 1, dailyAverage
	case dailyAverage >= trainingLeaderboardLowDaily:
		progress := (dailyAverage - trainingLeaderboardLowDaily) /
			(trainingLeaderboardTargetDaily - trainingLeaderboardLowDaily)
		return lerp(trainingLeaderboardMidDailyFactor, 1, progress), dailyAverage
	default:
		progress := safeDiv(dailyAverage, trainingLeaderboardLowDaily)
		return lerp(trainingLeaderboardLowDailyFactor, trainingLeaderboardMidDailyFactor, progress), dailyAverage
	}
}

// buildContestScore 根据最近真实比赛的 rating 变化给一次温和校准。
// 比赛结果是训练判断的重要锚点，但这里只做校准，不让比赛分完全压过训练分。
func buildContestScore(records []*model.ContestRecord, to time.Time) float64 {
	if len(records) == 0 {
		return 0
	}

	cutoff := to.AddDate(0, 0, -trainingLeaderboardContestLookback)
	recent := make([]*model.ContestRecord, 0, trainingLeaderboardContestRecentMax)
	for i := len(records) - 1; i >= 0; i-- {
		record := records[i]
		if record == nil {
			continue
		}
		if record.ContestDate.After(to) || record.ContestDate.Before(cutoff) {
			continue
		}
		recent = append(recent, record)
		if len(recent) == trainingLeaderboardContestRecentMax {
			break
		}
	}
	if len(recent) == 0 {
		return 0
	}

	weights := []float64{1.0, 0.8, 0.65, 0.5}
	score := 0.0
	for i, record := range recent {
		changeUnits := clampFloat(
			float64(record.RatingChange)/trainingLeaderboardContestDivisor,
			-trainingLeaderboardContestRecordCap,
			trainingLeaderboardContestRecordCap,
		)
		score += weights[i] * changeUnits
	}

	return score * trainingLeaderboardContestScale
}

// difficultyBonus 评估题目难度是否落在相对本人能力线的合理训练区间。
// 当前分附近到高出约 200~300 分的题给更高价值；过低偏保温，过高则降低效率收益。
func difficultyBonus(difficulty int, rating ratingProfile) float64 {
	if rating.ability == nil {
		return absoluteDifficultyBonus(difficulty)
	}

	delta := float64(difficulty) - *rating.ability
	switch {
	case delta <= -300:
		return trainingLeaderboardDifficultyFloor
	case delta < 0:
		return lerp(trainingLeaderboardDifficultyFloor, 0.18, (delta+300)/300)
	case delta <= trainingLeaderboardTargetDelta:
		return lerp(0.18, trainingLeaderboardDifficultyPeak, delta/trainingLeaderboardTargetDelta)
	case delta <= 700:
		return lerp(trainingLeaderboardDifficultyPeak, 0.16, (delta-trainingLeaderboardTargetDelta)/(700-trainingLeaderboardTargetDelta))
	default:
		return trainingLeaderboardDifficultyTail
	}
}

// absoluteDifficultyBonus 在缺少 rating 能力线时，仅按绝对难度给一个保守奖励。
func absoluteDifficultyBonus(difficulty int) float64 {
	units := float64(difficulty-trainingLeaderboardAbsoluteDiffBase) / 500.0
	if units < 0 {
		units = 0
	}
	if units > trainingLeaderboardAbsoluteDiffCap {
		units = trainingLeaderboardAbsoluteDiffCap
	}
	return units * trainingLeaderboardAbsoluteDiffStep
}

// challengeBonus 评估题目是否对当前能力线形成有效挑战。
// 最优区间在高出本人能力线约 200~300 分，过高仍有价值，但不继续线性抬升。
func challengeBonus(difficulty int, rating ratingProfile) float64 {
	if rating.ability == nil {
		return 0
	}

	gap := float64(difficulty) - *rating.ability
	switch {
	case gap <= 0:
		return 0
	case gap <= trainingLeaderboardTargetDelta:
		return lerp(0, trainingLeaderboardChallengePeak, gap/trainingLeaderboardTargetDelta)
	case gap <= 700:
		return lerp(trainingLeaderboardChallengePeak, 0.18, (gap-trainingLeaderboardTargetDelta)/(700-trainingLeaderboardTargetDelta))
	default:
		return trainingLeaderboardChallengeTail
	}
}

func toDomainRatingProfile(profile ratingProfile) domain.TrainingLeaderboardRatingProfile {
	return domain.TrainingLeaderboardRatingProfile{
		Current:       profile.current,
		Peak:          profile.peak,
		AbilityAnchor: profile.ability,
	}
}

func toDomainPlatformScore(score platformScore) domain.TrainingLeaderboardPlatformScore {
	return domain.TrainingLeaderboardPlatformScore{
		SolvedTotal:     score.solvedTotal,
		KnownTotal:      score.knownTotal,
		UndefinedTotal:  score.undefinedTotal,
		Score:           round2(score.score),
		VolumeScore:     round2(score.volumeScore),
		DifficultyScore: round2(score.difficultyScore),
		ChallengeScore:  round2(score.challengeScore),
	}
}

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}

func round4(v float64) float64 {
	return math.Round(v*10000) / 10000
}

func safeDiv(a, b float64) float64 {
	if b == 0 {
		return 0
	}
	return a / b
}

func lerp(from, to, progress float64) float64 {
	return from + (to-from)*clampFloat(progress, 0, 1)
}

func clampFloat(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

func float64Ptr(v float64) *float64 {
	return &v
}
