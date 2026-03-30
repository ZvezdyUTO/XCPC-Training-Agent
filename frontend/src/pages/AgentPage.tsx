import { type FormEvent, type ReactNode, useMemo, useState } from "react";
import { type AgentTraceMode, useAgentRuns } from "../features/agent/AgentRunContext";

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

function getStringList(value: unknown): string[] {
  if (!Array.isArray(value)) {
    return [];
  }
  return value
    .map((item) => (typeof item === "string" ? item.trim() : String(item)))
    .filter((item) => item !== "");
}

function getTraceCollection(value: unknown): Array<Record<string, unknown>> {
  if (!Array.isArray(value)) {
    return [];
  }
  return value.filter((item): item is Record<string, unknown> => typeof item === "object" && item !== null);
}

function getObject(value: unknown): Record<string, unknown> | null {
  if (!value || typeof value !== "object" || Array.isArray(value)) {
    return null;
  }
  return value as Record<string, unknown>;
}

function isPrimitiveValue(value: unknown) {
  return value === null || value === undefined || typeof value === "string" || typeof value === "number" || typeof value === "boolean";
}

function renderStructuredValue(value: unknown) {
  if (isPrimitiveValue(value)) {
    return <strong>{formatUnknown(value)}</strong>;
  }

  if (Array.isArray(value)) {
    if (!value.length) {
      return <strong>[]</strong>;
    }
    return (
      <details className="details-box">
        <summary>查看数组内容 ({value.length})</summary>
        <pre>{JSON.stringify(value, null, 2)}</pre>
      </details>
    );
  }

  if (value && typeof value === "object") {
    return (
      <details className="details-box">
        <summary>查看对象内容</summary>
        <pre>{JSON.stringify(value, null, 2)}</pre>
      </details>
    );
  }

  return <strong>{formatUnknown(value)}</strong>;
}

function renderPayloadFields(payload: Record<string, unknown>, omitKeys: string[] = []) {
  const entries = Object.entries(payload).filter(([key]) => !omitKeys.includes(key));
  if (!entries.length) {
    return <div className="empty-state">这一部分没有更多字段。</div>;
  }

  const compactEntries = entries.filter(([, value]) => {
    if (!isPrimitiveValue(value)) {
      return false;
    }
    return formatUnknown(value).length <= 48;
  });
  const extendedEntries = entries.filter(([key]) => !compactEntries.some(([compactKey]) => compactKey === key));

  return (
    <div className="trace-payload-stack">
      {compactEntries.length ? (
        <div className="trace-inline-grid">
          {compactEntries.map(([key, value]) => (
            <div key={key} className="trace-inline-item">
              <span>{key}</span>
              <strong>{formatUnknown(value)}</strong>
            </div>
          ))}
        </div>
      ) : null}

      {extendedEntries.length ? (
        <div className="trace-payload-grid">
          {extendedEntries.map(([key, value]) => (
            <div key={key} className="agent-meta-item trace-payload-item">
              <span>{key}</span>
              {renderStructuredValue(value)}
            </div>
          ))}
        </div>
      ) : null}
    </div>
  );
}

function getTracePlanState(trace: unknown): Record<string, unknown> | null {
  const traceObject = getObject(trace);
  const events = getTraceCollection(traceObject?.events);
  for (let index = events.length - 1; index >= 0; index -= 1) {
    const payload = getObject(events[index].payload);
    const planState = getObject(payload?.plan_state);
    if (planState) {
      return planState;
    }
  }
  return null;
}

function renderTracePlanSteps(planState: Record<string, unknown> | null) {
  const steps = Array.isArray(planState?.steps) ? planState.steps : [];
  if (!steps.length) {
    return <div className="empty-state">当前 trace 没有计划步骤。</div>;
  }

  return (
    <div className="trace-plan-list">
      {steps.map((item, index) => {
        const step = getObject(item);
        if (!step) {
          return null;
        }
        const status = formatUnknown(step.status);
        return (
          <div key={String(step.index ?? index)} className={`trace-plan-step trace-plan-step-${status}`}>
            <div className="trace-plan-step-index">#{formatUnknown(step.index)}</div>
            <div className="trace-plan-step-main">
              <div className="trace-plan-step-title">{formatUnknown(step.title)}</div>
              <div className="trace-plan-step-status">{status}</div>
            </div>
          </div>
        );
      })}
    </div>
  );
}

function renderToolCalls(toolCalls: unknown) {
  const calls = getTraceCollection(toolCalls);
  if (!calls.length) {
    return <div className="empty-state">这一轮没有工具调用。</div>;
  }

  return (
    <div className="trace-list">
      {calls.map((item, index) => {
        const functionInfo = getObject(item.function);
        return (
          <details key={`${formatUnknown(functionInfo?.name)}-${index}`} className="trace-card trace-inner-card">
            <summary className="trace-card-summary">
              <div>
                <div className="trace-card-title">{formatUnknown(functionInfo?.name) || `工具调用 ${index + 1}`}</div>
                <div className="trace-card-meta">step 内工具调用</div>
              </div>
            </summary>
            <div className="trace-card-body">
              {renderPayloadFields(item)}
              <details className="details-box">
                <summary>查看参数</summary>
                <pre>{JSON.stringify(functionInfo?.arguments ?? item, null, 2)}</pre>
              </details>
            </div>
          </details>
        );
      })}
    </div>
  );
}

function getNumericStep(item: Record<string, unknown>) {
  const raw = item.step;
  return typeof raw === "number" && Number.isFinite(raw) ? raw : -1;
}

function getCompactBlockTitle(eventType: string) {
  if (eventType === "model_called") {
    return "模型调用";
  }
  if (eventType === "model_returned") {
    return "模型返回";
  }
  if (eventType === "tool_called") {
    return "工具调度";
  }
  if (eventType === "tool_returned") {
    return "工具返回";
  }
  if (eventType === "run_failed") {
    return "失败";
  }
  if (eventType === "run_finished") {
    return "完成";
  }
  return formatUnknown(eventType);
}

function getStepGroups(events: Array<Record<string, unknown>>, spans: Array<Record<string, unknown>>) {
  const grouped = new Map<number, { events: Array<Record<string, unknown>>; spans: Array<Record<string, unknown>> }>();

  for (const item of events) {
    const step = getNumericStep(item);
    if (!grouped.has(step)) {
      grouped.set(step, { events: [], spans: [] });
    }
    grouped.get(step)?.events.push(item);
  }

  for (const item of spans) {
    const step = getNumericStep(item);
    if (!grouped.has(step)) {
      grouped.set(step, { events: [], spans: [] });
    }
    grouped.get(step)?.spans.push(item);
  }

  return Array.from(grouped.entries())
    .sort((left, right) => left[0] - right[0])
    .map(([step, value]) => ({
      step,
      events: value.events,
      spans: value.spans,
    }));
}

function renderCompactEventCard(item: Record<string, unknown>, index: number) {
  const payload = getObject(item.payload) ?? {};
  return (
    <div key={`${formatUnknown(item.event_type)}-${index}`} className="trace-compact-card">
      <div className="trace-compact-head">
        <div className="trace-compact-title">{getCompactBlockTitle(formatUnknown(item.event_type))}</div>
        <div className="trace-compact-time">{formatUnknown(item.timestamp)}</div>
      </div>
      {renderPayloadFields(
        {
          step: item.step,
          event_type: item.event_type,
          timestamp: item.timestamp,
          ...payload,
        },
        ["plan_state", "tool_calls", "raw_response", "content", "debug"],
      )}
      {typeof payload.content === "string" && payload.content.trim() !== "" ? (
        <details className="details-box">
          <summary>查看模型正文</summary>
          <pre>{payload.content}</pre>
        </details>
      ) : null}
      {"tool_calls" in payload ? (
        <div className="trace-compact-subsection">
          <div className="trace-compact-subtitle">工具调用</div>
          {renderToolCalls(payload.tool_calls)}
        </div>
      ) : null}
      {payload.debug ? (
        <details className="details-box">
          <summary>查看 debug 原文</summary>
          <pre>{JSON.stringify(payload.debug, null, 2)}</pre>
        </details>
      ) : null}
      {payload.raw_response ? (
        <details className="details-box">
          <summary>查看原始模型响应</summary>
          <pre>{formatUnknown(payload.raw_response)}</pre>
        </details>
      ) : null}
      <details className="details-box">
        <summary>查看事件 payload</summary>
        <pre>{JSON.stringify(payload, null, 2)}</pre>
      </details>
    </div>
  );
}

function renderCompactSpanCard(item: Record<string, unknown>, index: number) {
  const payload = getObject(item.payload) ?? {};
  return (
    <div key={`${formatUnknown(item.span_type)}-${index}`} className="trace-compact-card">
      <div className="trace-compact-head">
        <div className="trace-compact-title">{formatUnknown(item.span_type)}</div>
        <div className="trace-compact-time">{formatUnknown(item.latency_ms)} ms</div>
      </div>
      {renderPayloadFields(
        {
          step: item.step,
          span_type: item.span_type,
          started_at: item.started_at,
          finished_at: item.finished_at,
          status: item.status,
          latency_ms: item.latency_ms,
          ...payload,
        },
        ["debug"],
      )}
      {payload.debug ? (
        <details className="details-box">
          <summary>查看 debug 原文</summary>
          <pre>{JSON.stringify(payload.debug, null, 2)}</pre>
        </details>
      ) : null}
      <details className="details-box">
        <summary>查看跨度 payload</summary>
        <pre>{JSON.stringify(payload, null, 2)}</pre>
      </details>
    </div>
  );
}

function renderTraceStepSection(title: string, count: number, content: ReactNode, defaultOpen = false) {
  return (
    <details className="trace-step-section" open={defaultOpen}>
      <summary className="trace-step-section-summary">
        <div className="trace-step-section-title">{title}</div>
        <div className="trace-step-section-count">{count}</div>
      </summary>
      <div className="trace-step-section-body">{content}</div>
    </details>
  );
}

/** AgentPage 提供自然语言提问、结果展示与 trace 查看入口。 */
export function AgentPage() {
  const { runs, startRun } = useAgentRuns();
  const [task, setTask] = useState("分析学号为 230000001 的学生近期训练情况");
  const [traceMode, setTraceMode] = useState<AgentTraceMode>("summary");
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState("");

  const latestRun = useMemo(() => runs[0], [runs]);
  const latestResult = latestRun?.result?.result;
  const report = formatUnknown(latestResult?.report);
  const overallSummary = formatUnknown(latestResult?.overall_summary);
  const decisionType = formatUnknown(latestResult?.decision_type);
  const confidence = formatUnknown(latestResult?.confidence);
  const focusStudents = getStringList(latestResult?.focus_students);
  const keyFindings = getStringList(latestResult?.key_findings);
  const metricsEntries = getMetricsEntries(latestResult?.metrics);
  const traceEvents = getTraceCollection(latestRun?.result?.trace?.events);
  const traceSpans = getTraceCollection(latestRun?.result?.trace?.spans);
  const tracePlanState = getTracePlanState(latestRun?.result?.trace);
  const traceStepGroups = useMemo(() => getStepGroups(traceEvents, traceSpans), [traceEvents, traceSpans]);

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
          <div className="panel-title">分析结果</div>
          {latestRun?.result ? (
            <div className="agent-result">
              {latestRun.status === "error" && latestRun.error ? <div className="notice notice-error">{latestRun.error}</div> : null}

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
                <div className="agent-summary-label">整体概览</div>
                <div className="agent-summary-text">{latestResult ? overallSummary : "本次运行未产出最终分析结果。可以结合下方 Trace 查看失败阶段和当前计划状态。"}</div>
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
                <div className="agent-section-title">完整分析</div>
                <div className="agent-summary-text">{latestResult ? report : "-"}</div>
              </div>

              <div className="agent-section">
                <div className="agent-section-title">关键发现</div>
                {keyFindings.length ? (
                  <div className="agent-list-block">
                    {keyFindings.map((item, index) => (
                      <div key={`${item}-${index}`} className="agent-list-item">
                        {item}
                      </div>
                    ))}
                  </div>
                ) : (
                  <div className="empty-state">本次结果没有返回关键发现。</div>
                )}
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

              {latestResult ? (
                <details className="details-box">
                  <summary>查看完整结果 JSON</summary>
                  <pre>{JSON.stringify(latestRun.result.result, null, 2)}</pre>
                </details>
              ) : null}
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
                <div className="agent-section-title">计划状态</div>
                {tracePlanState ? (
                  <div className="stack stack-column">
                    <div className="agent-meta-grid">
                      <div className="agent-meta-item">
                        <span>current_step</span>
                        <strong>{formatUnknown(tracePlanState.current_step)}</strong>
                      </div>
                      <div className="agent-meta-item">
                        <span>version</span>
                        <strong>{formatUnknown(tracePlanState.version)}</strong>
                      </div>
                    </div>
                    {renderTracePlanSteps(tracePlanState)}
                  </div>
                ) : (
                  <div className="empty-state">当前 trace 没有记录计划状态。</div>
                )}
              </div>

              <div className="agent-section">
                <div className="agent-section-title">步骤日志</div>
                {traceStepGroups.length ? (
                  <div className="trace-step-list">
                    {traceStepGroups.map((group) => {
                      const modelCalledEvents = group.events.filter((item) => item.event_type === "model_called");
                      const modelReturnedEvents = group.events.filter((item) => item.event_type === "model_returned");
                      const toolCalledEvents = group.events.filter((item) => item.event_type === "tool_called");
                      const toolReturnedEvents = group.events.filter((item) => item.event_type === "tool_returned");
                      const otherEvents = group.events.filter(
                        (item) =>
                          item.event_type !== "model_called" &&
                          item.event_type !== "model_returned" &&
                          item.event_type !== "tool_called" &&
                          item.event_type !== "tool_returned",
                      );
                      const toolBlockCount = Math.max(toolCalledEvents.length, toolReturnedEvents.length);
                      return (
                        <div key={`step-group-${group.step}`} className="trace-step-card">
                          <div className="trace-step-card-head">
                            <div>
                              <div className="trace-step-card-title">Step {formatUnknown(group.step)}</div>
                              <div className="trace-step-card-meta">
                                事件 {group.events.length} · 跨度 {group.spans.length}
                              </div>
                            </div>
                          </div>

                          <div className="trace-step-card-body">
                            {modelCalledEvents.length || modelReturnedEvents.length ? (
                              renderTraceStepSection(
                                "模型",
                                modelCalledEvents.length + modelReturnedEvents.length,
                                <div className="trace-step-block-grid">
                                  {modelCalledEvents.map((item, index) => renderCompactEventCard(item, index))}
                                  {modelReturnedEvents.map((item, index) => renderCompactEventCard(item, index + modelCalledEvents.length))}
                                </div>,
                                true,
                              )
                            ) : null}

                            {toolBlockCount ? (
                              renderTraceStepSection(
                                "工具",
                                toolBlockCount,
                                <div className="trace-tool-stack">
                                  {Array.from({ length: toolBlockCount }).map((_, index) => (
                                    <div key={`tool-stack-${group.step}-${index}`} className="trace-tool-row">
                                      {toolCalledEvents[index] ? renderCompactEventCard(toolCalledEvents[index], index) : null}
                                      {toolReturnedEvents[index] ? renderCompactEventCard(toolReturnedEvents[index], index + toolCalledEvents.length) : null}
                                    </div>
                                  ))}
                                </div>,
                              )
                            ) : null}

                            {otherEvents.length ? (
                              renderTraceStepSection(
                                "其它事件",
                                otherEvents.length,
                                <div className="trace-step-block-grid">
                                  {otherEvents.map((item, index) => renderCompactEventCard(item, index))}
                                </div>,
                              )
                            ) : null}

                            {group.spans.length ? (
                              renderTraceStepSection(
                                "跨度",
                                group.spans.length,
                                <div className="trace-step-block-grid">
                                  {group.spans.map((item, index) => renderCompactSpanCard(item, index))}
                                </div>,
                              )
                            ) : null}
                          </div>
                        </div>
                      );
                    })}
                  </div>
                ) : (
                  <div className="empty-state">当前 trace 没有步骤日志。</div>
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
