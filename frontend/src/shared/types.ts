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
  failed_cnt?: number;
  failed?: Array<{
    student_id: string;
    error: string;
  }>;
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
  run_id: string;
  mode: string;
  started_at?: string;
  finished_at?: string;
  token_usage?: AgentTokenUsage;
  spans: Array<Record<string, unknown>>;
  events: Array<Record<string, unknown>>;
}

/** AgentRunPayload 是 Agent HTTP 接口成功返回的主体。 */
export interface AgentRunPayload {
  task: string;
  result: Record<string, unknown>;
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
