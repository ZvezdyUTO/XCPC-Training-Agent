import { createContext, type ReactNode, useContext, useMemo, useState } from "react";
import { api } from "../../shared/api";
import type { AuthUser } from "../../shared/types";

type AuthContextValue = {
  user: AuthUser | null;
  loading: boolean;
  login: (username: string, password: string) => Promise<void>;
  refreshAuth: () => Promise<boolean>;
  logout: () => void;
};

const STORAGE_KEY = "xcpc-coach-auth";
const AuthContext = createContext<AuthContextValue | null>(null);

function loadStoredUser(): AuthUser | null {
  const raw = window.localStorage.getItem(STORAGE_KEY);
  if (!raw) {
    return null;
  }

  try {
    return JSON.parse(raw) as AuthUser;
  } catch {
    window.localStorage.removeItem(STORAGE_KEY);
    return null;
  }
}

/** AuthProvider 负责维护登录态与 token 持久化。 */
export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<AuthUser | null>(() => loadStoredUser());
  const [loading, setLoading] = useState(false);

  async function login(username: string, password: string) {
    setLoading(true);
    try {
      const payload = await api.login(username, password);
      const nextUser: AuthUser = {
        token: payload.token,
        id: payload.user.id,
        name: payload.user.name,
        isAdmin: payload.user.is_system === 1,
      };
      window.localStorage.setItem(STORAGE_KEY, JSON.stringify(nextUser));
      setUser(nextUser);
    } finally {
      setLoading(false);
    }
  }

  // refreshAuth 主动向后端校验当前 token 是否仍然有效。
  // 有效时同步最新用户名和管理员标记；无效时清理本地登录态。
  async function refreshAuth(): Promise<boolean> {
    if (!user) {
      return false;
    }

    setLoading(true);
    try {
      const profile = await api.me(user.token);
      const nextUser: AuthUser = {
        token: user.token,
        id: profile.id,
        name: profile.name,
        isAdmin: profile.is_system === 1,
      };
      window.localStorage.setItem(STORAGE_KEY, JSON.stringify(nextUser));
      setUser(nextUser);
      return true;
    } catch {
      window.localStorage.removeItem(STORAGE_KEY);
      setUser(null);
      return false;
    } finally {
      setLoading(false);
    }
  }

  function logout() {
    window.localStorage.removeItem(STORAGE_KEY);
    setUser(null);
  }

  const value = useMemo<AuthContextValue>(
    () => ({ user, loading, login, refreshAuth, logout }),
    [user, loading],
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

/** useAuth 暴露统一的登录态访问入口。 */
export function useAuth() {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error("useAuth must be used within AuthProvider");
  }
  return context;
}
