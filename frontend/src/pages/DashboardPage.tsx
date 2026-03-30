import { useEffect, useState } from "react";
import { api } from "../shared/api";
import { useAuth } from "../features/auth/AuthContext";
import { useAgentRuns } from "../features/agent/AgentRunContext";
import type { UserListPayload } from "../shared/types";

/** DashboardPage 展示教练端首屏概览信息。 */
export function DashboardPage() {
  const { user } = useAuth();
  const { runs } = useAgentRuns();
  const [users, setUsers] = useState<UserListPayload | null>(null);

  useEffect(() => {
    if (!user) {
      return;
    }
    api.listUsers(user.token, "")
      .then(setUsers)
      .catch(() => {
        setUsers(null);
      });
  }, [user]);

  const latestRun = runs[0];

  return (
    <div className="page-grid">
      <section className="panel hero-panel">
        <span className="eyebrow">总览</span>
        <h2>常用入口</h2>
        <p>这里查看系统概览和最近 Agent 运行。</p>
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
    </div>
  );
}
