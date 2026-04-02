/** ApiEnvelope 约束后端统一响应结构。 */
export interface ApiEnvelope<T> {
  code: number;
  data: T;
  msg: string;
  err_code?: string;
}

/** LoginUser 描述登录接口中 user 字段的结构。 */
export interface LoginUser {
  id: string;
  name: string;
  status: number;
  is_system: number;
}

/** LoginPayload 表示登录接口返回的 user 与 token 包装。 */
export interface LoginPayload {
  token: string;
  user: LoginUser;
}

/** AuthUser 是前端缓存的登录态用户信息。 */
export interface AuthUser {
  token: string;
  id: string;
  name: string;
  isAdmin: boolean;
}

/** UserItem 对应用户列表与导入接口的用户字段。 */
export interface UserItem {
  id: string;
  name: string;
  password: string;
  status: number;
  is_system: number;
  cf_handle: string;
  ac_handle: string;
}

/** UserListPayload 表示用户列表响应。 */
export interface UserListPayload {
  count: number;
  list: UserItem[];
}

/** BatchCreatePayload 表示批量导入的执行结果。 */
export interface BatchCreatePayload {
  total: number;
  success: number;
  failed: Array<{
    student_id: string;
    error: string;
  }>;
}

/** SyncAllTrainingPayload 表示训练同步接口返回的逐学生执行结果。 */
export interface SyncAllTrainingPayload {
  msg: string;
  success_cnt: number;
  success: Array<{
    student_id: string;
    mode: string;
  }>;
  alert_cnt?: number;
  failed_cnt?: number;
  failed?: Array<{
    student_id: string;
    error: string;
  }>;
}

/** SyncOneTrainingPayload 表示单个学生训练同步接口返回体。 */
export interface SyncOneTrainingPayload {
  msg: string;
  student_id: string;
  mode: string;
  alert_cnt?: number;
}

/** DetectRunPayload 表示手动触发异常检测接口返回。 */
export interface DetectRunPayload {
  msg: string;
  alert_cnt: number;
  detected_at: string;
}

/** AnomalyRuleConfig 表示异常检测规则参数。 */
export interface AnomalyRuleConfig {
  current_window_days: number;
  baseline_window_days: number;
  baseline_min_daily: number;
  current_min_daily_for_alert: number;
  drop_low_threshold: number;
  drop_medium_threshold: number;
  drop_high_threshold: number;
  inactive_days_threshold: number;
  inactive_days_medium_threshold: number;
  inactive_days_high_threshold: number;
  inactive_baseline_min_daily: number;
  difficulty_drop_current_window_days: number;
  difficulty_drop_medium_days_threshold: number;
  difficulty_drop_high_days_threshold: number;
  difficulty_drop_baseline_window_days: number;
  difficulty_drop_min_current_total: number;
  difficulty_drop_min_baseline_high_ratio: number;
  difficulty_level_round_base: number;
  difficulty_relative_high_delta: number;
  difficulty_relative_easy_delta: number;
  difficulty_drop_low_threshold: number;
  difficulty_drop_medium_threshold: number;
  difficulty_drop_high_threshold: number;
}

/** AnomalyRuleConfigPatch 表示规则参数的部分更新请求。 */
export type AnomalyRuleConfigPatch = Partial<AnomalyRuleConfig>;

/** AnomalyRuleConfigUpdatePayload 表示规则更新接口返回。 */
export interface AnomalyRuleConfigUpdatePayload {
  msg: string;
  config: AnomalyRuleConfig;
}

/** TrainingAlertItem 表示单条训练异常预警。 */
export interface TrainingAlertItem {
  id: number;
  student_id: string;
  alert_date: string;
  alert_type: string;
  severity: "low" | "medium" | "high";
  status: "new" | "ack" | "resolved";
  title: string;
  evidence: Record<string, unknown>;
  actions: string[];
  created_at: string;
  updated_at: string;
}

/** TrainingAlertsListPayload 表示预警列表响应。 */
export interface TrainingAlertsListPayload {
  count: number;
  list: TrainingAlertItem[];
}

/** SyncStateItem 表示同步状态表中的单条记录。 */
export interface SyncStateItem {
  student_id: string;
  is_fully_initialized: number;
  latest_successful_date?: string | null;
  created_at: string;
  updated_at: string;
}

/** SyncStateListPayload 表示同步状态列表接口返回体。 */
export interface SyncStateListPayload {
  count: number;
  list: SyncStateItem[];
}

/** AgentTokenUsage 描述一次 Agent 运行的 token 使用概况。 */
export interface AgentTokenUsage {
  model_call_count: number;
  input_tokens: number;
  output_tokens: number;
  total_tokens: number;
}

/** AgentTrace 对应后端可选返回的 trace。 */
export interface AgentTrace {
  mode: string;
  started_at?: string;
  finished_at?: string;
  token_usage?: AgentTokenUsage;
  spans: Array<Record<string, unknown>>;
  events: Array<Record<string, unknown>>;
}

/** AgentRunPayload 是 Agent HTTP 接口返回给前端的运行主体。 */
export interface AgentRunPayload {
  task: string;
  result?: Record<string, unknown>;
  token_usage: AgentTokenUsage;
  trace?: AgentTrace;
}

/** ContestRankingItem 表示某场比赛中的单个学生排名记录。 */
export interface ContestRankingItem {
  student_id: string;
  student_name: string;
  platform: string;
  contest_id: string;
  name: string;
  date: string;
  rank: number;
  old_rating: number;
  new_rating: number;
  rating_change: number;
}

/** ContestRankingPayload 表示某场比赛的队内排名查询结果。 */
export interface ContestRankingPayload {
  platform: string;
  contest_id: string;
  contest_name: string;
  contest_date: string;
  count: number;
  items: ContestRankingItem[];
}

/** TrainingSummaryPayload 表示某学生在指定时间段内的训练累计结果。 */
export interface TrainingSummaryPayload {
  student_id: string;
  from: string;
  to: string;
  cf_total: number;
  cf_distribution: Record<string, number>;
  ac_total?: number;
  ac_distribution?: Record<string, number>;
  training_value?: {
    scoring_version: string;
    solved_total: number;
    daily_average: number;
    score: number;
    volume_score: number;
    difficulty_score: number;
    challenge_score: number;
    contest_score: number;
    undefined_total: number;
    undefined_ratio: number;
    cf_rating: TrainingLeaderboardRatingProfile;
    ac_rating: TrainingLeaderboardRatingProfile;
  };
}

/** TrainingLeaderboardRatingProfile 表示某个平台的当前分、峰值和能力参考线。 */
export interface TrainingLeaderboardRatingProfile {
  current: number | null;
  peak: number | null;
  ability_anchor: number | null;
}

/** TrainingLeaderboardPlatformScore 表示单个平台上的训练贡献拆解。 */
export interface TrainingLeaderboardPlatformScore {
  solved_total: number;
  known_total: number;
  undefined_total: number;
  score: number;
  volume_score: number;
  difficulty_score: number;
  challenge_score: number;
}

/** TrainingLeaderboardItem 表示排行榜中的单个学生结果。 */
export interface TrainingLeaderboardItem {
  rank: number;
  student_id: string;
  student_name: string;
  solved_total: number;
  daily_average: number;
  score: number;
  volume_score: number;
  difficulty_score: number;
  challenge_score: number;
  contest_score: number;
  undefined_total: number;
  undefined_ratio: number;
  cf_rating: TrainingLeaderboardRatingProfile;
  ac_rating: TrainingLeaderboardRatingProfile;
  cf: TrainingLeaderboardPlatformScore;
  ac: TrainingLeaderboardPlatformScore;
}

/** TrainingLeaderboardPayload 表示训练价值排行榜查询结果。 */
export interface TrainingLeaderboardPayload {
  scoring_version: string;
  from: string;
  to: string;
  top_n: number;
  count: number;
  items: TrainingLeaderboardItem[];
}

/** ImportPreviewRow 表示学生导入预览中的单行解析结果。 */
export interface ImportPreviewRow {
  lineNo: number;
  raw: string;
  id: string;
  name: string;
  password: string;
  cfHandle: string;
  acHandle: string;
  valid: boolean;
  error: string;
}
