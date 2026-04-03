package anomaly

import "fmt"

// RuleConfig 表示异常检测规则参数。
// 当前通过代码文件统一管理；后续可替换为数据库或管理接口下发。
type RuleConfig struct {
	// CurrentWindowDays 表示“当前观测窗口”天数。
	// 例如 7 表示最近 7 天（包含 asOf 当天）。
	CurrentWindowDays int `json:"current_window_days"`
	// BaselineWindowDays 表示“历史基线窗口”天数。
	// 例如 30 表示在当前窗口之前，向前取 30 天作为参考基线。
	BaselineWindowDays int `json:"baseline_window_days"`
	// BaselineMinDaily 是触发检测前的基线日均下限。
	// 当历史基线日均低于该值时，认为样本不足，不触发“训练量突降”告警。
	BaselineMinDaily float64 `json:"baseline_min_daily"`
	// CurrentMinDailyForAlert 是当前窗口的绝对日均保护阈值。
	// 当前日均高于或等于该值时，即使相对历史下降，也不触发告警。
	CurrentMinDailyForAlert float64 `json:"current_min_daily_for_alert"`
	// VolumeRecoveryRatio1D 是“最近一天恢复抑制比例”。
	// 若最近 1 天题量 >= 基线日均 * 该比例，则抑制题量下降告警。
	VolumeRecoveryRatio1D float64 `json:"volume_recovery_ratio_1d"`

	// DropLowThreshold 是低等级告警的降幅阈值（0~1）。
	// 例如 0.35 表示较基线下降 35% 触发 low。
	DropLowThreshold float64 `json:"drop_low_threshold"`
	// DropMediumThreshold 是中等级告警的降幅阈值（0~1）。
	DropMediumThreshold float64 `json:"drop_medium_threshold"`
	// DropHighThreshold 是高等级告警的降幅阈值（0~1）。
	DropHighThreshold float64 `json:"drop_high_threshold"`

	// InactiveDaysThreshold 是“连续停训告警”的最小连续天数。
	InactiveDaysThreshold int `json:"inactive_days_threshold"`
	// InactiveDaysMediumThreshold 是“连续停训告警”触发 medium 的连续天数阈值。
	InactiveDaysMediumThreshold int `json:"inactive_days_medium_threshold"`
	// InactiveDaysHighThreshold 是“连续停训告警”触发 high 的连续天数阈值。
	InactiveDaysHighThreshold int `json:"inactive_days_high_threshold"`
	// InactiveBaselineMinDaily 是触发停训告警前要求的历史基线日均题量。
	// 用于避免对长期低活跃学生频繁误报。
	InactiveBaselineMinDaily float64 `json:"inactive_baseline_min_daily"`

	// DifficultyDropCurrentWindowDays 是“高难题连续未达标告警”的连续天数阈值。
	DifficultyDropCurrentWindowDays int `json:"difficulty_drop_current_window_days"`
	// DifficultyDropMediumDaysThreshold 是“高难题连续未达标告警”触发 medium 的连续天数阈值。
	DifficultyDropMediumDaysThreshold int `json:"difficulty_drop_medium_days_threshold"`
	// DifficultyDropHighDaysThreshold 是“高难题连续未达标告警”触发 high 的连续天数阈值。
	DifficultyDropHighDaysThreshold int `json:"difficulty_drop_high_days_threshold"`
	// DifficultyDropBaselineWindowDays 是兼容保留字段（当前规则未使用）。
	DifficultyDropBaselineWindowDays int `json:"difficulty_drop_baseline_window_days"`
	// DifficultyDropMinCurrentTotal 是“每日高难题达标阈值”。
	// 单日高难题数量小于该值，即记为该日“高难题未达标”。
	DifficultyDropMinCurrentTotal int `json:"difficulty_drop_min_current_total"`
	// DifficultyDropMinBaselineHighRatio 是兼容保留字段（当前规则未使用）。
	DifficultyDropMinBaselineHighRatio float64 `json:"difficulty_drop_min_baseline_high_ratio"`
	// DifficultyLevelRoundBase 是“个人水平分”的取整基数。
	// 例如 100 表示舍去个位和十位，只保留千百位（1743 -> 1700）。
	DifficultyLevelRoundBase int `json:"difficulty_level_round_base"`
	// DifficultyRelativeHighDelta 表示高难题阈值：题目难度 >= 个人水平 + 该值。
	DifficultyRelativeHighDelta int `json:"difficulty_relative_high_delta"`
	// DifficultyRelativeEasyDelta 表示简单题阈值：题目难度 <= 个人水平 - 该值。
	DifficultyRelativeEasyDelta int `json:"difficulty_relative_easy_delta"`

	// DifficultyDropLowThreshold 是兼容保留字段（当前规则未使用）。
	DifficultyDropLowThreshold float64 `json:"difficulty_drop_low_threshold"`
	// DifficultyDropMediumThreshold 是兼容保留字段（当前规则未使用）。
	DifficultyDropMediumThreshold float64 `json:"difficulty_drop_medium_threshold"`
	// DifficultyDropHighThreshold 是兼容保留字段（当前规则未使用）。
	DifficultyDropHighThreshold float64 `json:"difficulty_drop_high_threshold"`
}

// RuleConfigPatch 表示规则配置的部分更新请求。
// 字段为 nil 表示不修改；非 nil 表示覆盖对应值。
type RuleConfigPatch struct {
	CurrentWindowDays                  *int     `json:"current_window_days"`
	BaselineWindowDays                 *int     `json:"baseline_window_days"`
	BaselineMinDaily                   *float64 `json:"baseline_min_daily"`
	CurrentMinDailyForAlert            *float64 `json:"current_min_daily_for_alert"`
	VolumeRecoveryRatio1D              *float64 `json:"volume_recovery_ratio_1d"`
	DropLowThreshold                   *float64 `json:"drop_low_threshold"`
	DropMediumThreshold                *float64 `json:"drop_medium_threshold"`
	DropHighThreshold                  *float64 `json:"drop_high_threshold"`
	InactiveDaysThreshold              *int     `json:"inactive_days_threshold"`
	InactiveDaysMediumThreshold        *int     `json:"inactive_days_medium_threshold"`
	InactiveDaysHighThreshold          *int     `json:"inactive_days_high_threshold"`
	InactiveBaselineMinDaily           *float64 `json:"inactive_baseline_min_daily"`
	DifficultyDropCurrentWindowDays    *int     `json:"difficulty_drop_current_window_days"`
	DifficultyDropMediumDaysThreshold  *int     `json:"difficulty_drop_medium_days_threshold"`
	DifficultyDropHighDaysThreshold    *int     `json:"difficulty_drop_high_days_threshold"`
	DifficultyDropBaselineWindowDays   *int     `json:"difficulty_drop_baseline_window_days"`
	DifficultyDropMinCurrentTotal      *int     `json:"difficulty_drop_min_current_total"`
	DifficultyDropMinBaselineHighRatio *float64 `json:"difficulty_drop_min_baseline_high_ratio"`
	DifficultyLevelRoundBase           *int     `json:"difficulty_level_round_base"`
	DifficultyRelativeHighDelta        *int     `json:"difficulty_relative_high_delta"`
	DifficultyRelativeEasyDelta        *int     `json:"difficulty_relative_easy_delta"`
	DifficultyDropLowThreshold         *float64 `json:"difficulty_drop_low_threshold"`
	DifficultyDropMediumThreshold      *float64 `json:"difficulty_drop_medium_threshold"`
	DifficultyDropHighThreshold        *float64 `json:"difficulty_drop_high_threshold"`
}

const (
	// defaultCurrentWindowDays: 当前观测窗口天数，默认最近 7 天。
	defaultCurrentWindowDays = 7
	// defaultBaselineWindowDays: 历史基线窗口天数，默认前 30 天。
	defaultBaselineWindowDays = 30
	// defaultBaselineMinDaily: 历史基线触发检测的最小日均题量。
	defaultBaselineMinDaily = 1.0
	// defaultCurrentMinDailyForAlert: 当前窗口“绝对水平仍健康”保护阈值。
	defaultCurrentMinDailyForAlert = 2.0
	// defaultVolumeRecoveryRatio1D: 最近 1 天恢复到基线日均 80% 时，抑制题量下降告警。
	defaultVolumeRecoveryRatio1D = 0.8

	// defaultDropLowThreshold: 降幅达到 35% 记为 low。
	defaultDropLowThreshold = 0.35
	// defaultDropMediumThreshold: 降幅达到 50% 记为 medium。
	defaultDropMediumThreshold = 0.5
	// defaultDropHighThreshold: 降幅达到 70% 记为 high。
	defaultDropHighThreshold = 0.7

	// defaultInactiveDaysThreshold: 连续 3 天无训练触发停训告警。
	defaultInactiveDaysThreshold = 3
	// defaultInactiveDaysMediumThreshold: 连续 5 天无训练触发中等级停训告警。
	defaultInactiveDaysMediumThreshold = 5
	// defaultInactiveDaysHighThreshold: 连续 7 天无训练触发高等级停训告警。
	defaultInactiveDaysHighThreshold = 7
	// defaultInactiveBaselineMinDaily: 停训告警要求历史基线日均至少 1 题。
	defaultInactiveBaselineMinDaily = 1.0

	// defaultDifficultyDropCurrentWindowDays: 连续 3 天高难题未达标触发告警。
	defaultDifficultyDropCurrentWindowDays = 3
	// defaultDifficultyDropMediumDaysThreshold: 连续 5 天高难题未达标触发中等级告警。
	defaultDifficultyDropMediumDaysThreshold = 5
	// defaultDifficultyDropHighDaysThreshold: 连续 7 天高难题未达标触发高等级告警。
	defaultDifficultyDropHighDaysThreshold = 7
	// defaultDifficultyDropBaselineWindowDays: 高难占比历史基线默认前 30 天。
	defaultDifficultyDropBaselineWindowDays = 30
	// defaultDifficultyDropMinCurrentTotal: 单日高难题达标阈值默认 1 题。
	defaultDifficultyDropMinCurrentTotal = 1
	// defaultDifficultyDropMinBaselineHighRatio: 历史高难占比至少 15% 才参与该规则。
	defaultDifficultyDropMinBaselineHighRatio = 0.15
	// defaultDifficultyLevelRoundBase: 个人水平默认按百位取整。
	defaultDifficultyLevelRoundBase = 100
	// defaultDifficultyRelativeHighDelta: 高难题默认定义为高于个人水平 200 分及以上。
	defaultDifficultyRelativeHighDelta = 200
	// defaultDifficultyRelativeEasyDelta: 简单题默认定义为低于个人水平 200 分及以下。
	defaultDifficultyRelativeEasyDelta = 200

	// defaultDifficultyDropLowThreshold: 高难占比相对降幅达到 35% 记为 low。
	defaultDifficultyDropLowThreshold = 0.35
	// defaultDifficultyDropMediumThreshold: 高难占比相对降幅达到 50% 记为 medium。
	defaultDifficultyDropMediumThreshold = 0.5
	// defaultDifficultyDropHighThreshold: 高难占比相对降幅达到 70% 记为 high。
	defaultDifficultyDropHighThreshold = 0.7
)

// defaultRuleConfig 返回异常检测默认参数。
// 业务方可直接修改本文件常量完成阈值调参。
func defaultRuleConfig() RuleConfig {
	return RuleConfig{
		CurrentWindowDays:                  defaultCurrentWindowDays,
		BaselineWindowDays:                 defaultBaselineWindowDays,
		BaselineMinDaily:                   defaultBaselineMinDaily,
		CurrentMinDailyForAlert:            defaultCurrentMinDailyForAlert,
		VolumeRecoveryRatio1D:              defaultVolumeRecoveryRatio1D,
		DropLowThreshold:                   defaultDropLowThreshold,
		DropMediumThreshold:                defaultDropMediumThreshold,
		DropHighThreshold:                  defaultDropHighThreshold,
		InactiveDaysThreshold:              defaultInactiveDaysThreshold,
		InactiveDaysMediumThreshold:        defaultInactiveDaysMediumThreshold,
		InactiveDaysHighThreshold:          defaultInactiveDaysHighThreshold,
		InactiveBaselineMinDaily:           defaultInactiveBaselineMinDaily,
		DifficultyDropCurrentWindowDays:    defaultDifficultyDropCurrentWindowDays,
		DifficultyDropMediumDaysThreshold:  defaultDifficultyDropMediumDaysThreshold,
		DifficultyDropHighDaysThreshold:    defaultDifficultyDropHighDaysThreshold,
		DifficultyDropBaselineWindowDays:   defaultDifficultyDropBaselineWindowDays,
		DifficultyDropMinCurrentTotal:      defaultDifficultyDropMinCurrentTotal,
		DifficultyDropMinBaselineHighRatio: defaultDifficultyDropMinBaselineHighRatio,
		DifficultyLevelRoundBase:           defaultDifficultyLevelRoundBase,
		DifficultyRelativeHighDelta:        defaultDifficultyRelativeHighDelta,
		DifficultyRelativeEasyDelta:        defaultDifficultyRelativeEasyDelta,
		DifficultyDropLowThreshold:         defaultDifficultyDropLowThreshold,
		DifficultyDropMediumThreshold:      defaultDifficultyDropMediumThreshold,
		DifficultyDropHighThreshold:        defaultDifficultyDropHighThreshold,
	}
}

// Validate 校验规则参数是否合法。
// 返回错误时应拒绝执行检测，避免因为配置错误导致误报或漏报。
func (c RuleConfig) Validate() error {
	if c.CurrentWindowDays <= 0 {
		return fmt.Errorf("当前窗口天数必须大于 0")
	}
	if c.BaselineWindowDays <= 0 {
		return fmt.Errorf("基线窗口天数必须大于 0")
	}
	if c.BaselineMinDaily < 0 {
		return fmt.Errorf("基线最小日均题量必须大于或等于 0")
	}
	if c.CurrentMinDailyForAlert < 0 {
		return fmt.Errorf("当前窗口告警保护日均阈值必须大于或等于 0")
	}
	if c.VolumeRecoveryRatio1D < 0 || c.VolumeRecoveryRatio1D > 1 {
		return fmt.Errorf("最近一天恢复抑制比例必须在 [0,1] 区间内")
	}
	if c.DropLowThreshold < 0 || c.DropLowThreshold > 1 {
		return fmt.Errorf("低等级降幅阈值必须在 [0,1] 区间内")
	}
	if c.DropMediumThreshold < 0 || c.DropMediumThreshold > 1 {
		return fmt.Errorf("中等级降幅阈值必须在 [0,1] 区间内")
	}
	if c.DropHighThreshold < 0 || c.DropHighThreshold > 1 {
		return fmt.Errorf("高等级降幅阈值必须在 [0,1] 区间内")
	}
	if !(c.DropLowThreshold <= c.DropMediumThreshold && c.DropMediumThreshold <= c.DropHighThreshold) {
		return fmt.Errorf("降幅阈值必须满足 low <= medium <= high")
	}
	if c.InactiveDaysThreshold <= 0 {
		return fmt.Errorf("连续停训阈值天数必须大于 0")
	}
	if c.InactiveDaysMediumThreshold <= 0 {
		return fmt.Errorf("连续停训中等级阈值天数必须大于 0")
	}
	if c.InactiveDaysHighThreshold <= 0 {
		return fmt.Errorf("连续停训高等级阈值天数必须大于 0")
	}
	if !(c.InactiveDaysThreshold <= c.InactiveDaysMediumThreshold &&
		c.InactiveDaysMediumThreshold <= c.InactiveDaysHighThreshold) {
		return fmt.Errorf("连续停训天数阈值必须满足 low <= medium <= high")
	}
	if c.InactiveBaselineMinDaily < 0 {
		return fmt.Errorf("停训告警基线最小日均题量必须大于或等于 0")
	}
	if c.DifficultyDropCurrentWindowDays <= 0 {
		return fmt.Errorf("高难题连续未达标天数阈值必须大于 0")
	}
	if c.DifficultyDropMediumDaysThreshold <= 0 {
		return fmt.Errorf("高难题中等级连续未达标天数阈值必须大于 0")
	}
	if c.DifficultyDropHighDaysThreshold <= 0 {
		return fmt.Errorf("高难题高等级连续未达标天数阈值必须大于 0")
	}
	if !(c.DifficultyDropCurrentWindowDays <= c.DifficultyDropMediumDaysThreshold &&
		c.DifficultyDropMediumDaysThreshold <= c.DifficultyDropHighDaysThreshold) {
		return fmt.Errorf("高难题连续未达标天数阈值必须满足 low <= medium <= high")
	}
	if c.DifficultyDropBaselineWindowDays <= 0 {
		return fmt.Errorf("兼容字段 difficulty_drop_baseline_window_days 必须大于 0")
	}
	if c.DifficultyDropMinCurrentTotal < 0 {
		return fmt.Errorf("每日高难题达标阈值必须大于或等于 0")
	}
	if c.DifficultyDropMinBaselineHighRatio < 0 || c.DifficultyDropMinBaselineHighRatio > 1 {
		return fmt.Errorf("难度占比检测基线最小高难占比必须在 [0,1] 区间内")
	}
	if c.DifficultyLevelRoundBase <= 0 {
		return fmt.Errorf("个人水平取整基数必须大于 0")
	}
	if c.DifficultyRelativeHighDelta < 0 {
		return fmt.Errorf("高难题相对个人水平阈值必须大于或等于 0")
	}
	if c.DifficultyRelativeEasyDelta < 0 {
		return fmt.Errorf("简单题相对个人水平阈值必须大于或等于 0")
	}
	if c.DifficultyDropLowThreshold < 0 || c.DifficultyDropLowThreshold > 1 {
		return fmt.Errorf("难度占比低等级降幅阈值必须在 [0,1] 区间内")
	}
	if c.DifficultyDropMediumThreshold < 0 || c.DifficultyDropMediumThreshold > 1 {
		return fmt.Errorf("难度占比中等级降幅阈值必须在 [0,1] 区间内")
	}
	if c.DifficultyDropHighThreshold < 0 || c.DifficultyDropHighThreshold > 1 {
		return fmt.Errorf("难度占比高等级降幅阈值必须在 [0,1] 区间内")
	}
	if !(c.DifficultyDropLowThreshold <= c.DifficultyDropMediumThreshold &&
		c.DifficultyDropMediumThreshold <= c.DifficultyDropHighThreshold) {
		return fmt.Errorf("难度占比降幅阈值必须满足 low <= medium <= high")
	}
	return nil
}
