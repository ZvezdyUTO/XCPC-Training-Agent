import { FormEvent, useState } from "react";
import { useAuth } from "../features/auth/AuthContext";

/** LoginPage 提供管理员登录入口。 */
export function LoginPage() {
  const { login, loading } = useAuth();
  const [username, setUsername] = useState("20001");
  const [password, setPassword] = useState("000000");
  const [error, setError] = useState("");

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError("");
    try {
      await login(username.trim(), password);
    } catch (submitError) {
      setError(submitError instanceof Error ? submitError.message : "登录失败");
    }
  }

  return (
    <div className="login-page">
      <div className="login-hero">
        <span className="eyebrow">XCPC Training Agent</span>
        <h1>教练端控制台</h1>
        <p>集中处理学生导入、训练同步与 Agent 分析，保持操作路径直接、页面信息清晰。</p>
      </div>

      <form className="panel login-panel" onSubmit={handleSubmit}>
        <div className="panel-title">登录</div>

        <label className="field">
          <span>账号</span>
          <input value={username} onChange={(event) => setUsername(event.target.value)} />
        </label>

        <label className="field">
          <span>密码</span>
          <input
            type="password"
            value={password}
            onChange={(event) => setPassword(event.target.value)}
          />
        </label>

        {error ? <div className="notice notice-error">{error}</div> : null}

        <button className="primary-button" disabled={loading} type="submit">
          {loading ? "登录中..." : "进入教练端"}
        </button>
      </form>
    </div>
  );
}
