import { useEffect, useState } from "react";
import { api } from "../shared/api";
import { useAuth } from "../features/auth/AuthContext";
import { useAgentRuns } from "../features/agent/AgentRunContext";
import type { SyncAllTrainingPayload, SyncStateListPayload, UserListPayload } from "../shared/types";

/** DashboardPage 展示教练端首屏概览信息。 */
export function DashboardPage() {
  const { user } = useAuth();
  const { runs } = useAgentRuns();
  const [users, setUsers] = useState<UserListPayload | null>(null);
  const [syncing, setSyncing] = useState(false);
  const [syncMessage, setSyncMessage] = useState("");
  const [syncResult, setSyncResult] = useState<SyncAllTrainingPayload | null>(null);
  const [syncStates, setSyncStates] = useState<SyncStateListPayload | null>(null);
  const [syncStateKeyword, setSyncStateKeyword] = useState("");
  const [syncStateLoading, setSyncStateLoading] = useState(false);

  async function loadSyncStates() {
    if (!user) {
      return;
    }
    setSyncStateLoading(true);
    try {
      const payload = await api.listSyncStates(user.token);
      setSyncStates(payload);
    } catch {
      setSyncStates(null);
    } finally {
      setSyncStateLoading(false);
    }
  }

  useEffect(() => {
    if (!user) {
      return;
    }
    api.listUsers(user.token, "")
      .then(setUsers)
      .catch(() => {
        setUsers(null);
      });

    void loadSyncStates();
  }, [user]);

  async function handleSyncAll() {
    if (!user) {
      return;
    }

    setSyncing(true);
    setSyncMessage("");
    setSyncResult(null);
    try {
      const payload = await api.syncAllTraining(user.token);
      setSyncResult(payload);
      setSyncMessage(`同步完成：成功 ${payload.success_cnt}${payload.failed_cnt ? `，失败 ${payload.failed_cnt}` : ""}`);
      await loadSyncStates();
    } catch (error) {
      setSyncMessage(error instanceof Error ? error.message : "同步触发失败");
    } finally {
      setSyncing(false);
    }
  }

  const latestRun = runs[0];
  const filteredSyncStates = syncStates?.list.filter((item) => {
    const keyword = syncStateKeyword.trim();
    if (keyword === "") {
      return true;
    }
    return item.student_id.includes(keyword);
  }) ?? [];

  return (
    <div className="page-grid">
      <section className="panel hero-panel">
        <span className="eyebrow">总览</span>
        <h2>训练管理保持轻量，但关键入口集中。</h2>
        <p>学生导入、训练同步与 Agent 分析都集中在一个控制台内，适合教练快速处理日常工作。</p>
        <div className="toolbar">
          <button className="primary-button" disabled={syncing} onClick={handleSyncAll} type="button">
            {syncing ? "同步中..." : "触发训练同步"}
          </button>
          {syncMessage ? <span className="helper-text">{syncMessage}</span> : null}
        </div>
      </section>

      <section className="stats-grid">
        <article className="panel stat-card">
          <div className="stat-label">学生总数</div>
          <div className="stat-value">{users?.count ?? "-"}</div>
        </article>

        <article className="panel stat-card">
          <div className="stat-label">最近 Agent 运行</div>
          <div className="stat-value">{latestRun ? latestRun.status : "暂无"}</div>
        </article>

        <article className="panel stat-card">
          <div className="stat-label">最近分析任务</div>
          <div className="stat-text">{latestRun?.task ?? "还没有执行过 Agent 分析"}</div>
        </article>
      </section>

      <section className="panel">
        <div className="panel-title">最近 Agent 运行</div>
        {latestRun ? (
          <div className="stack">
            <div><strong>任务：</strong>{latestRun.task}</div>
            <div><strong>状态：</strong>{latestRun.status}</div>
            <div><strong>开始时间：</strong>{new Date(latestRun.startedAt).toLocaleString("zh-CN")}</div>
            {latestRun.finishedAt ? (
              <div><strong>结束时间：</strong>{new Date(latestRun.finishedAt).toLocaleString("zh-CN")}</div>
            ) : null}
          </div>
        ) : (
          <div className="empty-state">还没有 Agent 运行记录，可以前往 Agent 页面发起分析。</div>
        )}
      </section>

      <section className="panel">
        <div className="panel-title">最近训练同步结果</div>
        {syncResult ? (
          <div className="stack stack-column">
            <div className="agent-meta-grid">
              <div className="agent-meta-item">
                <span>状态</span>
                <strong>{syncResult.msg}</strong>
              </div>
              <div className="agent-meta-item">
                <span>成功数量</span>
                <strong>{syncResult.success_cnt}</strong>
              </div>
              <div className="agent-meta-item">
                <span>失败数量</span>
                <strong>{syncResult.failed_cnt ?? 0}</strong>
              </div>
            </div>

            <div>
              <div className="subsection-title">成功学生</div>
              {syncResult.success.length ? (
                <div className="chip-row">
                  {syncResult.success.map((item) => (
                    <span key={`${item.student_id}-${item.mode}`} className="status-chip agent-student-chip">
                      {item.student_id} · {item.mode}
                    </span>
                  ))}
                </div>
              ) : (
                <div className="empty-state">本次没有成功同步的学生。</div>
              )}
            </div>

            <div>
              <div className="subsection-title">失败学生</div>
              {syncResult.failed?.length ? (
                <div className="run-list">
                  {syncResult.failed.map((item) => (
                    <article key={item.student_id} className="run-card">
                      <div className="run-card-header">
                        <strong>{item.student_id}</strong>
                        <span className="status-chip status-error">失败</span>
                      </div>
                      <div className="run-task">{item.error}</div>
                    </article>
                  ))}
                </div>
              ) : (
                <div className="empty-state">本次没有失败项。</div>
              )}
            </div>
          </div>
        ) : (
          <div className="empty-state">点击“触发训练同步”后，这里会展示每个学生的同步模式和失败原因。</div>
        )}
      </section>

      <section className="panel">
        <div className="panel-title">同步状态表</div>
        <div className="toolbar">
          <input
            className="search-input"
            placeholder="按学号筛选同步状态"
            value={syncStateKeyword}
            onChange={(event) => setSyncStateKeyword(event.target.value)}
          />
          <button className="secondary-button" disabled={syncStateLoading} onClick={() => void loadSyncStates()} type="button">
            {syncStateLoading ? "查询中..." : "查询同步状态"}
          </button>
        </div>

        {syncStateLoading ? (
          <div className="empty-state">同步状态加载中...</div>
        ) : filteredSyncStates.length ? (
          <div className="table-wrap">
            <table className="data-table">
              <thead>
                <tr>
                  <th>学号</th>
                  <th>是否已初始化</th>
                  <th>最近成功日期</th>
                  <th>更新时间</th>
                </tr>
              </thead>
              <tbody>
                {filteredSyncStates.map((item) => (
                  <tr key={item.student_id}>
                    <td>{item.student_id}</td>
                    <td>
                      <span className={item.is_fully_initialized === 1 ? "status-chip status-ok" : "status-chip status-error"}>
                        {item.is_fully_initialized === 1 ? "已初始化" : "未初始化"}
                      </span>
                    </td>
                    <td>{item.latest_successful_date ? new Date(item.latest_successful_date).toLocaleDateString("zh-CN") : "-"}</td>
                    <td>{new Date(item.updated_at).toLocaleString("zh-CN")}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        ) : (
          <div className="empty-state">没有匹配的同步状态记录。你可以直接点“查询同步状态”刷新当前状态表。</div>
        )}
      </section>
    </div>
  );
}
