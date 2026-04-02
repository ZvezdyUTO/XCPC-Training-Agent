import { Link, Outlet, useLocation } from "react-router-dom";
import { useAuth } from "../features/auth/AuthContext";
import { useAgentRuns } from "../features/agent/AgentRunContext";
import { useState } from "react";

const navItems = [
  { to: "/", label: "总览" },
  { to: "/students", label: "学生" },
  { to: "/query", label: "查询" },
  { to: "/alerts", label: "预警" },
  { to: "/agent", label: "Agent" },
];

/** AppShell 负责登录后的统一导航、页头与页面容器。 */
export function AppShell() {
  const { pathname } = useLocation();
  const { user, refreshAuth, logout } = useAuth();
  const { runningCount } = useAgentRuns();
  const [refreshingAuth, setRefreshingAuth] = useState(false);

  async function handleRefreshAuth() {
    setRefreshingAuth(true);
    try {
      await refreshAuth();
    } finally {
      setRefreshingAuth(false);
    }
  }

  return (
    <div className="shell">
      <aside className="sidebar">
        <div>
          <div className="brand brand-stacked">
            <span>XCPC</span>
            <span>TRAINING</span>
            <span>AGENT</span>
          </div>
          <div className="brand-subtitle">教练端</div>
        </div>

        <nav className="nav">
          {navItems.map((item) => (
            <Link
              key={item.to}
              className={pathname === item.to || (item.to !== "/" && pathname.startsWith(item.to)) ? "nav-link active" : "nav-link"}
              to={item.to}
            >
              {item.label}
            </Link>
          ))}
        </nav>
      </aside>

      <div className="main">
        <header className="topbar">
          <div>
            <div className="page-title">教练工作台</div>
            <div className="page-subtitle">学生管理、训练同步和查询入口</div>
          </div>

          <div className="topbar-actions">
            {runningCount > 0 ? <span className="badge badge-running">Agent 运行中 {runningCount}</span> : null}
            <span className="badge">{user?.name}</span>
            <button className="ghost-button" onClick={handleRefreshAuth} type="button">
              {refreshingAuth ? "刷新中..." : "刷新登录状态"}
            </button>
            <button className="ghost-button" onClick={logout} type="button">
              退出
            </button>
          </div>
        </header>

        <main className="content">
          <Outlet />
        </main>
      </div>
    </div>
  );
}
