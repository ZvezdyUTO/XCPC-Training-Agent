# Agent 工具说明

## 概述

当前 Agent 只注册 4 个工具，目标是把训练分析所需的核心查询能力收敛到最小闭环：

- 单人训练记录查询
- 单人比赛记录查询
- 训练价值排行榜查询
- 某场比赛队内排名查询

对应注册位置在 [service.go](/home/zvezdyuto/GolandProjects/XCPC-Training-Agent/internal/logic/agent/service/service.go)。

## 工具列表

### `training_summary_range`

职责：

- 查询某个学生在指定时间范围内的训练累计数据
- 返回 CF / AC 题量分布
- 返回训练价值评分拆解

输入参数：

- `student_id`：学生学号
- `from`：开始日期，格式 `2006-01-02`
- `to`：结束日期，格式 `2006-01-02`

输出重点：

- `cf_total`
- `cf_distribution`
- `ac_total`
- `ac_distribution`
- `training_value`

说明：

- `training_value` 与排行榜复用同一套公式
- 该工具只读训练统计表和比赛记录表，不触发抓取

### `student_contest_records`

职责：

- 查询单个学生的比赛记录
- 支持按平台过滤
- 支持限制返回条数

输入参数：

- `student_id`：学生学号
- `platform`：可选，`CF` 或 `AC`
- `limit`：可选，默认 `20`，最大 `50`

输出重点：

- `count`
- `items[].contest_name`
- `items[].contest_date`
- `items[].rank`
- `items[].old_rating`
- `items[].new_rating`
- `items[].rating_change`
- `items[].performance`

说明：

- 该工具按“查记录”语义返回，不额外生成高层分析结论

### `training_value_leaderboard`

职责：

- 查询指定时间范围内的训练价值排行榜

输入参数：

- `from`：开始日期，格式 `2006-01-02`
- `to`：结束日期，格式 `2006-01-02`
- `top_n`：可选，默认 `20`

输出重点：

- `scoring_version`
- `count`
- `items[].rank`
- `items[].student_id`
- `items[].student_name`
- `items[].score`
- `items[].volume_score`
- `items[].difficulty_score`
- `items[].challenge_score`
- `items[].undefined_total`
- `items[].cf_rating`
- `items[].ac_rating`

评分说明：

- 题量是底盘
- 难度体现质量
- 相对本人能力线的挑战度体现训练价值
- `undefined` 题会谨慎按折扣估计参与，不直接当作高难题

### `contest_ranking`

职责：

- 查询某一场比赛在数据库中的队内排名

输入参数：

- `platform`：`CF` 或 `AC`
- `contest_id`：比赛 ID

输出重点：

- `contest_name`
- `contest_date`
- `count`
- `items[].student_id`
- `items[].student_name`
- `items[].rank`
- `items[].old_rating`
- `items[].new_rating`
- `items[].rating_change`

说明：

- 该工具面向“同一场比赛队内对比”
- 不返回额外的分析性总结

## 评分口径

`training_summary_range` 和 `training_value_leaderboard` 复用同一套训练价值公式，公共实现位于：

- [trainingvalue.go](/home/zvezdyuto/GolandProjects/XCPC-Training-Agent/internal/logic/trainingvalue.go)
- [trainingleaderboard.go](/home/zvezdyuto/GolandProjects/XCPC-Training-Agent/internal/logic/trainingleaderboard.go)

这保证了：

- 前端手动查询和 Agent 查询口径一致
- 单人分析和排行榜排序口径一致
- 后续如果调整评分，只需要改一处核心逻辑
