import type {
  AgentRunPayload,
  ApiEnvelope,
  BatchCreatePayload,
  ContestRankingPayload,
  LoginPayload,
  SyncAllTrainingPayload,
  SyncOneTrainingPayload,
  SyncStateListPayload,
  TrainingLeaderboardPayload,
  TrainingSummaryPayload,
  UserItem,
  UserListPayload,
} from "./types";

type RequestOptions = {
  method?: string;
  body?: unknown;
  token?: string;
  signal?: AbortSignal;
};

/** ApiRequestError 保留后端错误响应中的 data，便于失败态继续展示 trace。 */
export class ApiRequestError<T = unknown> extends Error {
  data?: T;

  constructor(message: string, data?: T) {
    super(message);
    this.name = "ApiRequestError";
    this.data = data;
  }
}

async function request<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const response = await fetch(path, {
    method: options.method ?? "GET",
    headers: {
      "Content-Type": "application/json",
      ...(options.token ? { Authorization: `Bearer ${options.token}` } : {}),
    },
    body: options.body === undefined ? undefined : JSON.stringify(options.body),
    signal: options.signal,
  });

  const text = await response.text();
  let payload: ApiEnvelope<T>;
  try {
    payload = JSON.parse(text) as ApiEnvelope<T>;
  } catch {
    throw new Error(`接口 ${path} 返回了非 JSON 内容，通常表示代理未生效、接口地址错误或服务返回了 HTML 页面`);
  }
  if (!response.ok || payload.code !== 200) {
    throw new ApiRequestError<T>(payload.msg || "请求失败", payload.data);
  }
  return payload.data;
}

/** api 封装前端当前使用的最小后端能力。 */
export const api = {
  login(username: string, password: string) {
    return request<LoginPayload>("/v1/user/login", {
      method: "POST",
      body: { username, password },
    });
  },
  listUsers(token: string, keyword: string) {
    const params = new URLSearchParams();
    params.set("page", "1");
    params.set("count", "200");
    if (keyword.trim() !== "") {
      params.set("name", keyword.trim());
    }
    return request<UserListPayload | { count: number; List: UserItem[] }>(
      `/v1/admin/users/list?${params.toString()}`,
      {
        token,
      },
    ).then((payload) => ({
      count: payload.count,
      list: "list" in payload ? payload.list : payload.List,
    }));
  },
  createUsers(token: string, users: UserItem[]) {
    return request<BatchCreatePayload>("/v1/admin/users/create", {
      method: "POST",
      token,
      body: { users },
    });
  },
  deleteUser(token: string, studentID: string) {
    return request<unknown>(`/v1/admin/users/${studentID}`, {
      method: "DELETE",
      token,
    });
  },
  syncAllTraining(token: string) {
    return request<SyncAllTrainingPayload>("/v1/admin/op/training/syncall", {
      method: "POST",
      token,
      body: {},
    });
  },
  syncOneTraining(token: string, studentID: string) {
    return request<SyncOneTrainingPayload>("/v1/admin/op/training/syncone", {
      method: "POST",
      token,
      body: { student_id: studentID },
    });
  },
  listSyncStates(token: string) {
    return request<SyncStateListPayload>("/v1/admin/op/training/syncstate/list", {
      token,
    });
  },
  getContestRanking(token: string, platform: string, contestID: string) {
    const params = new URLSearchParams({
      platform,
      contest_id: contestID,
    });
    return request<ContestRankingPayload>(`/v1/admin/op/contest/ranking?${params.toString()}`, {
      token,
    });
  },
  getTrainingSummary(token: string, studentID: string, from: string, to: string) {
    const params = new URLSearchParams({
      student_id: studentID,
      from,
      to,
    });
    return request<TrainingSummaryPayload>(`/v1/admin/op/training/summary?${params.toString()}`, {
      token,
    });
  },
  getTrainingLeaderboard(token: string, from: string, to: string, topN: number) {
    const params = new URLSearchParams({
      from,
      to,
      top_n: String(topN),
    });
    return request<TrainingLeaderboardPayload>(`/v1/admin/op/training/leaderboard?${params.toString()}`, {
      token,
    });
  },
  me(token: string) {
    return request<UserItem>("/v1/user/me", {
      token,
    });
  },
  runAgent(token: string, task: string, traceMode: string) {
    return request<AgentRunPayload>("/v1/admin/agent/task/run", {
      method: "POST",
      token,
      body: {
        task,
        trace_mode: traceMode,
      },
    });
  },
};
