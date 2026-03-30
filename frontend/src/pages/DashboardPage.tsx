import { useEffect, useState } from "react";
import { api } from "../shared/api";
import { useAuth } from "../features/auth/AuthContext";
import { useAgentRuns } from "../features/agent/AgentRunContext";
import type { SyncStateListPayload, UserListPayload } from "../shared/types";

/** DashboardPage 展示教练端首屏概览信息。 */
export function DashboardPage() {
  const { user } = useAuth();
  const { runs } = useAgentRuns();
  const [users, setUsers] = useState<UserListPayload | null>(null);
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
        <h2>常用入口</h2>
        <p>这里查看系统概览、最近 Agent 运行和训练同步状态。</p>
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
          <div className="stat-text">{latestRun?.task ?? "暂无"}</div>
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
          <div className="empty-state">暂无运行记录。</div>
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
          <div className="empty-state">暂无匹配的同步状态记录。</div>
        )}
      </section>
    </div>
  );
}
