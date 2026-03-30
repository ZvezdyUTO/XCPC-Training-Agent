import { createContext, type ReactNode, useContext, useMemo, useState } from "react";
import { api } from "../../shared/api";
import type { AgentRunPayload } from "../../shared/types";
import { useAuth } from "../auth/AuthContext";

export type AgentTraceMode = "none" | "summary" | "debug";

export interface AgentRunItem {
  id: string;
  task: string;
  traceMode: AgentTraceMode;
  status: "running" | "success" | "error";
  startedAt: string;
  finishedAt?: string;
  result?: AgentRunPayload;
  error?: string;
}

type AgentRunContextValue = {
  runs: AgentRunItem[];
  runningCount: number;
  startRun: (task: string, traceMode: AgentTraceMode) => Promise<void>;
};

const AgentRunContext = createContext<AgentRunContextValue | null>(null);

function createRunId() {
  return `run-${Date.now()}-${Math.random().toString(16).slice(2)}`;
}

/** AgentRunProvider 在应用根部维护运行中的 Agent 任务，保证切页时请求不被中断。 */
export function AgentRunProvider({ children }: { children: ReactNode }) {
  const { user } = useAuth();
  const [runs, setRuns] = useState<AgentRunItem[]>([]);

  async function startRun(task: string, traceMode: AgentTraceMode) {
    if (!user) {
      throw new Error("未登录");
    }

    const runId = createRunId();
    const startedAt = new Date().toISOString();
    setRuns((current) => [
      {
        id: runId,
        task,
        traceMode,
        status: "running",
        startedAt,
      },
      ...current,
    ]);

    try {
      const payload = await api.runAgent(user.token, task, traceMode);
      setRuns((current) =>
        current.map((item) =>
          item.id === runId
            ? {
                ...item,
                status: "success",
                finishedAt: new Date().toISOString(),
                result: payload,
              }
            : item,
        ),
      );
    } catch (error) {
      setRuns((current) =>
        current.map((item) =>
          item.id === runId
            ? {
                ...item,
                status: "error",
                finishedAt: new Date().toISOString(),
                error: error instanceof Error ? error.message : "运行失败",
              }
            : item,
        ),
      );
      throw error;
    }
  }

  const value = useMemo<AgentRunContextValue>(
    () => ({
      runs,
      runningCount: runs.filter((item) => item.status === "running").length,
      startRun,
    }),
    [runs],
  );

  return <AgentRunContext.Provider value={value}>{children}</AgentRunContext.Provider>;
}

/** useAgentRuns 暴露全局 Agent 运行状态。 */
export function useAgentRuns() {
  const context = useContext(AgentRunContext);
  if (!context) {
    throw new Error("useAgentRuns must be used within AgentRunProvider");
  }
  return context;
}
