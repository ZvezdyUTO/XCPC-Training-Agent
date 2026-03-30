import { FormEvent, useMemo, useState } from "react";
import { type AgentTraceMode, useAgentRuns } from "../features/agent/AgentRunContext";
import { useAuth } from "../features/auth/AuthContext";
import { api } from "../shared/api";
import type { ContestRankingPayload } from "../shared/types";

function formatUnknown(value: unknown): string {
  if (value === null || value === undefined || value === "") {
    return "-";
  }
  if (typeof value === "number") {
    return Number.isFinite(value) ? value.toString() : "-";
  }
  return String(value);
}

function getMetricsEntries(value: unknown): Array<[string, unknown]> {
  if (!value || typeof value !== "object" || Array.isArray(value)) {
    return [];
  }
  return Object.entries(value as Record<string, unknown>);
}

function getTraceCollection(value: unknown): Array<Record<string, unknown>> {
  if (!Array.isArray(value)) {
    return [];
  }
  return value.filter((item): item is Record<string, unknown> => typeof item === "object" && item !== null);
}

function getTraceLabel(item: Record<string, unknown>, primaryKey: string, fallback: string) {
  const primary = item[primaryKey];
  if (typeof primary === "string" && primary.trim() !== "") {
    return primary;
  }
  const summary = item.payload && typeof item.payload === "object" ? (item.payload as Record<string, unknown>).summary : undefined;
  if (typeof summary === "string" && summary.trim() !== "") {
    return summary;
  }
  return fallback;
}

/** AgentPage 提供自然语言提问、结果展示与 trace 查看入口。 */
export function AgentPage() {
  const { user } = useAuth();
  const { runs, startRun } = useAgentRuns();
  const [task, setTask] = useState("分析学号为 230000001 的学生近期训练情况");
  const [traceMode, setTraceMode] = useState<AgentTraceMode>("summary");
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState("");
  const [rankingPlatform, setRankingPlatform] = useState("CF");
  const [rankingContestID, setRankingContestID] = useState("");
  const [rankingLoading, setRankingLoading] = useState(false);
  const [rankingError, setRankingError] = useState("");
  const [rankingResult, setRankingResult] = useState<ContestRankingPayload | null>(null);

  const latestRun = useMemo(() => runs[0], [runs]);
  const latestResult = latestRun?.result?.result;
  const report = formatUnknown(latestResult?.report);
  const decisionType = formatUnknown(latestResult?.decision_type);
  const confidence = formatUnknown(latestResult?.confidence);
  const focusStudents = Array.isArray(latestResult?.focus_students)
    ? (latestResult?.focus_students as Array<unknown>).map((item) => String(item))
    : [];
  const metricsEntries = getMetricsEntries(latestResult?.metrics);
  const traceEvents = getTraceCollection(latestRun?.result?.trace?.events);
  const traceSpans = getTraceCollection(latestRun?.result?.trace?.spans);

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError("");

    if (task.trim() === "") {
      setError("任务描述不能为空");
      return;
    }

    setSubmitting(true);
    try {
      await startRun(task.trim(), traceMode);
    } catch (submitError) {
      setError(submitError instanceof Error ? submitError.message : "运行失败");
    } finally {
      setSubmitting(false);
    }
  }

  async function handleRankingQuery(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!user) {
      setRankingError("未登录");
      return;
    }
    if (rankingContestID.trim() === "") {
      setRankingError("比赛 ID 不能为空");
      return;
    }

    setRankingLoading(true);
    setRankingError("");
    try {
      const payload = await api.getContestRanking(user.token, rankingPlatform, rankingContestID.trim());
      setRankingResult(payload);
    } catch (queryError) {
      setRankingResult(null);
      setRankingError(queryError instanceof Error ? queryError.message : "查询失败");
    } finally {
      setRankingLoading(false);
    }
  }

  return (
    <div className="agent-page">
      <section className="panel agent-input-panel">
        <div className="agent-panel-head">
          <div>
            <div className="panel-title">发起分析</div>
            <div className="panel-hint">这里输入自然语言问题，Agent 会按当前模式返回结构化分析结果。</div>
          </div>
          <span className="status-chip agent-mode-chip">Trace: {traceMode}</span>
        </div>

        <form className="agent-form" onSubmit={handleSubmit}>
          <label className="field agent-task-field">
            <span>自然语言任务</span>
            <textarea
              className="code-input agent-task-input"
              rows={6}
              value={task}
              onChange={(event) => setTask(event.target.value)}
            />
          </label>

          <div className="agent-form-actions">
            <label className="field agent-select-field">
              <span>Trace 模式</span>
              <select className="agent-select" value={traceMode} onChange={(event) => setTraceMode(event.target.value as AgentTraceMode)}>
                <option value="none">none</option>
                <option value="summary">summary</option>
                <option value="debug">debug</option>
              </select>
            </label>

            <button className="primary-button agent-submit-button" disabled={submitting} type="submit">
              {submitting ? "分析中..." : "开始分析"}
            </button>
          </div>

          {error ? <div className="notice notice-error">{error}</div> : null}
        </form>
      </section>

      <section className="agent-content-stack">
        <article className="panel">
          <div className="panel-title">比赛排名查询</div>
          <form className="agent-form" onSubmit={handleRankingQuery}>
            <div className="agent-form-row">
              <label className="field agent-select-field">
                <span>平台</span>
                <select className="agent-select" value={rankingPlatform} onChange={(event) => setRankingPlatform(event.target.value)}>
                  <option value="CF">CF</option>
                  <option value="AC">AC</option>
                </select>
              </label>

              <label className="field agent-task-field">
                <span>比赛 ID</span>
                <input
                  className="agent-inline-input"
                  placeholder="例如 2050"
                  value={rankingContestID}
                  onChange={(event) => setRankingContestID(event.target.value)}
                />
              </label>

              <button className="secondary-button agent-submit-button" disabled={rankingLoading} type="submit">
                {rankingLoading ? "查询中..." : "查询比赛排名"}
              </button>
            </div>

            {rankingError ? <div className="notice notice-error">{rankingError}</div> : null}
          </form>

          {rankingResult ? (
            <div className="agent-section">
              <div className="agent-trace-grid">
                <div className="agent-meta-item">
                  <span>contest_name</span>
                  <strong>{formatUnknown(rankingResult.contest_name)}</strong>
                </div>
                <div className="agent-meta-item">
                  <span>contest_date</span>
                  <strong>{formatUnknown(rankingResult.contest_date)}</strong>
                </div>
                <div className="agent-meta-item">
                  <span>platform</span>
                  <strong>{rankingResult.platform}</strong>
                </div>
                <div className="agent-meta-item">
                  <span>count</span>
                  <strong>{rankingResult.count}</strong>
                </div>
              </div>

              <div className="table-wrap">
                <table className="data-table">
                  <thead>
                    <tr>
                      <th>排名</th>
                      <th>学号</th>
                      <th>姓名</th>
                      <th>旧分</th>
                      <th>新分</th>
                      <th>变化</th>
                    </tr>
                  </thead>
                  <tbody>
                    {rankingResult.items.map((item) => (
                      <tr key={`${item.student_id}-${item.platform}-${item.contest_id}`}>
                        <td>{item.rank}</td>
                        <td>{item.student_id}</td>
                        <td>{item.student_name || "-"}</td>
                        <td>{item.old_rating}</td>
                        <td>{item.new_rating}</td>
                        <td>{item.rating_change}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </div>
          ) : (
            <div className="empty-state">输入平台和比赛 ID 后，可以直接查询这场比赛在数据库内的队内排名。</div>
          )}
        </article>

        <article className="panel">
          <div className="panel-title">分析结果</div>
          {latestRun?.result ? (
            <div className="agent-result">
              <div className="agent-result-header">
                <div>
                  <div className="eyebrow">Latest Run</div>
                  <h3 className="agent-result-title">{latestRun.result.task}</h3>
                </div>
                <span
                  className={
                    latestRun.status === "success"
                      ? "status-chip status-ok"
                      : latestRun.status === "running"
                        ? "status-chip badge-running"
                        : "status-chip status-error"
                  }
                >
                  {latestRun.status === "success" ? "已完成" : latestRun.status === "running" ? "运行中" : "失败"}
                </span>
              </div>

              <div className="agent-summary-card">
                <div className="agent-summary-label">结论摘要</div>
                <div className="agent-summary-text">{report}</div>
              </div>

              <div className="agent-meta-grid">
                <div className="agent-meta-item">
                  <span>decision_type</span>
                  <strong>{decisionType}</strong>
                </div>
                <div className="agent-meta-item">
                  <span>confidence</span>
                  <strong>{confidence}</strong>
                </div>
                <div className="agent-meta-item">
                  <span>model calls</span>
                  <strong>{latestRun.result.token_usage.model_call_count}</strong>
                </div>
                <div className="agent-meta-item">
                  <span>total tokens</span>
                  <strong>{latestRun.result.token_usage.total_tokens}</strong>
                </div>
              </div>

              <div className="agent-section">
                <div className="agent-section-title">重点学生</div>
                {focusStudents.length ? (
                  <div className="chip-row">
                    {focusStudents.map((studentID) => (
                      <span key={studentID} className="status-chip agent-student-chip">
                        {studentID}
                      </span>
                    ))}
                  </div>
                ) : (
                  <div className="empty-state">本次结果没有返回重点关注学生。</div>
                )}
              </div>

              <div className="agent-section">
                <div className="agent-section-title">关键指标</div>
                {metricsEntries.length ? (
                  <div className="agent-metrics-grid">
                    {metricsEntries.map(([key, value]) => (
                      <div key={key} className="agent-metric-card">
                        <span>{key}</span>
                        <strong>{formatUnknown(value)}</strong>
                      </div>
                    ))}
                  </div>
                ) : (
                  <div className="empty-state">本次结果没有返回 metrics。</div>
                )}
              </div>

              <details className="details-box">
                <summary>查看完整结果 JSON</summary>
                <pre>{JSON.stringify(latestRun.result.result, null, 2)}</pre>
              </details>
            </div>
          ) : latestRun?.status === "error" ? (
            <div className="notice notice-error">{latestRun.error}</div>
          ) : latestRun?.status === "running" ? (
            <div className="empty-state">Agent 正在运行。你现在切到其他页面也不会中断这次请求。</div>
          ) : (
            <div className="empty-state">执行一次分析后，这里会显示结构化结果。</div>
          )}
        </article>

        <article className="panel">
          <div className="panel-title">Trace</div>
          {latestRun?.result?.trace ? (
            <div className="stack stack-column">
              <div className="agent-trace-grid">
                <div className="agent-meta-item">
                  <span>run_id</span>
                  <strong>{latestRun.result.trace.run_id}</strong>
                </div>
                <div className="agent-meta-item">
                  <span>mode</span>
                  <strong>{latestRun.result.trace.mode}</strong>
                </div>
                <div className="agent-meta-item">
                  <span>events</span>
                  <strong>{latestRun.result.trace.events.length}</strong>
                </div>
                <div className="agent-meta-item">
                  <span>spans</span>
                  <strong>{latestRun.result.trace.spans.length}</strong>
                </div>
              </div>

              <div className="agent-section">
                <div className="agent-section-title">Trace 概览</div>
                <div className="agent-trace-summary">
                  <div>当前模式：{latestRun.result.trace.mode}</div>
                  <div>事件数：{traceEvents.length}</div>
                  <div>跨度数：{traceSpans.length}</div>
                  <div>总 Token：{latestRun.result.token_usage.total_tokens}</div>
                </div>
              </div>

              <div className="agent-section">
                <div className="agent-section-title">事件列表</div>
                {traceEvents.length ? (
                  <div className="trace-list">
                    {traceEvents.map((item, index) => {
                      const payload = item.payload && typeof item.payload === "object"
                        ? (item.payload as Record<string, unknown>)
                        : {};
                      return (
                        <details key={String(item.event_id ?? index)} className="trace-card">
                          <summary className="trace-card-summary">
                            <div>
                              <div className="trace-card-title">{getTraceLabel(item, "event_type", `事件 ${index + 1}`)}</div>
                              <div className="trace-card-meta">
                                step {formatUnknown(item.step)} · {formatUnknown(payload.status)}
                              </div>
                            </div>
                            <span className="trace-card-time">{formatUnknown(item.timestamp)}</span>
                          </summary>
                          <div className="trace-card-body">
                            <div className="trace-payload-grid">
                              <div className="agent-meta-item">
                                <span>event_id</span>
                                <strong>{formatUnknown(item.event_id)}</strong>
                              </div>
                              <div className="agent-meta-item">
                                <span>parent_id</span>
                                <strong>{formatUnknown(item.parent_id)}</strong>
                              </div>
                              <div className="agent-meta-item">
                                <span>entity_name</span>
                                <strong>{formatUnknown(payload.entity_name)}</strong>
                              </div>
                              <div className="agent-meta-item">
                                <span>summary</span>
                                <strong>{formatUnknown(payload.summary)}</strong>
                              </div>
                            </div>
                            <details className="details-box">
                              <summary>查看事件 payload</summary>
                              <pre>{JSON.stringify(payload, null, 2)}</pre>
                            </details>
                          </div>
                        </details>
                      );
                    })}
                  </div>
                ) : (
                  <div className="empty-state">当前 trace 没有事件列表。</div>
                )}
              </div>

              <div className="agent-section">
                <div className="agent-section-title">跨度列表</div>
                {traceSpans.length ? (
                  <div className="trace-list">
                    {traceSpans.map((item, index) => {
                      const payload = item.payload && typeof item.payload === "object"
                        ? (item.payload as Record<string, unknown>)
                        : {};
                      return (
                        <details key={String(item.span_id ?? index)} className="trace-card">
                          <summary className="trace-card-summary">
                            <div>
                              <div className="trace-card-title">{getTraceLabel(item, "span_type", `跨度 ${index + 1}`)}</div>
                              <div className="trace-card-meta">
                                {formatUnknown(item.status)} · {formatUnknown(item.latency_ms)} ms
                              </div>
                            </div>
                            <span className="trace-card-time">step {formatUnknown(item.step)}</span>
                          </summary>
                          <div className="trace-card-body">
                            <div className="trace-payload-grid">
                              <div className="agent-meta-item">
                                <span>span_id</span>
                                <strong>{formatUnknown(item.span_id)}</strong>
                              </div>
                              <div className="agent-meta-item">
                                <span>parent_span_id</span>
                                <strong>{formatUnknown(item.parent_span_id)}</strong>
                              </div>
                              <div className="agent-meta-item">
                                <span>entity_name</span>
                                <strong>{formatUnknown(payload.entity_name)}</strong>
                              </div>
                              <div className="agent-meta-item">
                                <span>summary</span>
                                <strong>{formatUnknown(payload.summary)}</strong>
                              </div>
                            </div>
                            <details className="details-box">
                              <summary>查看跨度 payload</summary>
                              <pre>{JSON.stringify(payload, null, 2)}</pre>
                            </details>
                          </div>
                        </details>
                      );
                    })}
                  </div>
                ) : (
                  <div className="empty-state">当前 trace 没有跨度列表。</div>
                )}
              </div>

              <details className="details-box">
                <summary>查看完整 Trace JSON</summary>
                <pre>{JSON.stringify(latestRun.result.trace, null, 2)}</pre>
              </details>
            </div>
          ) : (
            <div className="empty-state">当前运行没有返回 Trace。把 Trace 模式切到 summary 或 debug 后再次运行即可查看。</div>
          )}
        </article>

        <article className="panel">
          <div className="panel-title">运行时间线</div>
          <div className="agent-timeline">
            {runs.length ? (
              runs.map((item) => (
                <div key={item.id} className="agent-timeline-item">
                  <div className="agent-timeline-dot" />
                  <div className="agent-timeline-content">
                    <div className="run-card-header">
                      <strong>{item.status === "running" ? "运行中" : item.status === "success" ? "已完成" : "失败"}</strong>
                      <span>{new Date(item.startedAt).toLocaleString("zh-CN")}</span>
                    </div>
                    <div className="run-task">{item.task}</div>
                    {item.error ? <div className="timeline-error">{item.error}</div> : null}
                  </div>
                </div>
              ))
            ) : (
              <div className="empty-state">还没有运行记录。</div>
            )}
          </div>
        </article>
        
        <article className="panel">
          <div className="panel-title">运行记录</div>
          <div className="run-list">
            {runs.length ? (
              runs.map((item) => (
                <article key={item.id} className="run-card">
                  <div className="run-card-header">
                    <strong>{item.status === "running" ? "运行中" : item.status === "success" ? "已完成" : "失败"}</strong>
                    <span>{new Date(item.startedAt).toLocaleString("zh-CN")}</span>
                  </div>
                  <div className="run-task">{item.task}</div>
                </article>
              ))
            ) : (
              <div className="empty-state">还没有运行记录。</div>
            )}
          </div>
        </article>
      </section>
    </div>
  );
}
