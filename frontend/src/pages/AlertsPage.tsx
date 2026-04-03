import { useEffect, useMemo, useState } from "react";
import { useAuth } from "../features/auth/AuthContext";
import { api } from "../shared/api";
import type { AnomalyRuleConfig, TrainingAlertItem, TrainingAlertsListPayload } from "../shared/types";

type RuleFieldType = "int" | "float";

type RuleFieldMeta = {
  key: keyof AnomalyRuleConfig;
  label: string;
  desc: string;
  type: RuleFieldType;
  step?: string;
};

const ruleFieldMetas: RuleFieldMeta[] = [
  { key: "current_window_days", label: "题量下降预警-取样窗口天数", desc: "题量下降预警：例如近 7 天", type: "int", step: "1" },
  { key: "baseline_window_days", label: "题量下降预警-基线窗口天数", desc: "题量下降预警：例如前 30 天", type: "int", step: "1" },
  { key: "current_min_daily_for_alert", label: "题量下降预警-当前保护日均阈值", desc: "题量下降预警：每天写的题数高于等于该值即使下降也不报警", type: "float", step: "0.01" },
  { key: "volume_recovery_ratio_1d", label: "题量下降预警-最近1天恢复抑制比例", desc: "题量下降预警：最近1天题量达到基线日均×该比例时，不触发题量下降预警", type: "float", step: "0.01" },
  { key: "drop_low_threshold", label: "题量下降预警-low阈值", desc: "题量下降预警：相对降幅达到该比例触发 low", type: "float", step: "0.01" },
  { key: "drop_medium_threshold", label: "题量下降预警-medium阈值", desc: "题量下降预警：相对降幅达到该比例触发 medium", type: "float", step: "0.01" },
  { key: "drop_high_threshold", label: "题量下降预警-high阈值", desc: "题量下降预警：相对降幅达到该比例触发 high", type: "float", step: "0.01" },
  { key: "inactive_days_threshold", label: "连续停训预警-low天数阈值", desc: "连续停训预警：连续无训练达到该天数触发 low", type: "int", step: "1" },
  { key: "inactive_days_medium_threshold", label: "连续停训预警-medium天数阈值", desc: "连续停训预警：连续无训练达到该天数触发 medium", type: "int", step: "1" },
  { key: "inactive_days_high_threshold", label: "连续停训预警-high天数阈值", desc: "连续停训预警：连续无训练达到该天数触发 high", type: "int", step: "1" },
  { key: "inactive_baseline_min_daily", label: "连续停训预警-统一基线最小日均题量", desc: "每日做题少于等于该值视为当天停训", type: "float", step: "0.01" },
  { key: "difficulty_drop_current_window_days", label: "高难题预警-low天数阈值", desc: "高难题预警：连续未达标达到该天数触发 low", type: "int", step: "1" },
  { key: "difficulty_drop_medium_days_threshold", label: "高难题预警-medium天数阈值", desc: "高难题预警：连续未达标达到该天数触发 medium", type: "int", step: "1" },
  { key: "difficulty_drop_high_days_threshold", label: "高难题预警-high天数阈值", desc: "高难题预警：连续未达标达到该天数触发 high", type: "int", step: "1" },
  { key: "difficulty_drop_min_current_total", label: "高难题预警-每日高难题达标阈值", desc: "高难题预警：单日高难题数量低于该值记为未达标", type: "int", step: "1" },
  { key: "difficulty_level_round_base", label: "高难题预警-个人水平取整基数", desc: "高难题预警：例如 100 表示取整到百位", type: "int", step: "1" },
  { key: "difficulty_relative_high_delta", label: "高难题预警-高难题相对阈值", desc: "高难题预警：高于个人水平该分值及以上算高难题", type: "int", step: "1" },
];

type AlertFilters = {
  student_id: string;
  status: string;
  severity: string;
  from: string;
  to: string;
  count: number;
};

function asNumber(value: unknown): number | null {
  if (typeof value === "number" && Number.isFinite(value)) {
    return value;
  }
  if (typeof value === "string" && value.trim() !== "") {
    const parsed = Number(value);
    if (Number.isFinite(parsed)) {
      return parsed;
    }
  }
  return null;
}

function asString(value: unknown): string {
  if (typeof value === "string") {
    return value;
  }
  if (value === null || value === undefined) {
    return "-";
  }
  return String(value);
}

function formatPercent(value: number | null): string {
  if (value === null) return "-";
  return `${(value * 100).toFixed(1)}%`;
}

function formatFloat(value: number | null): string {
  if (value === null) return "-";
  return Number.isInteger(value) ? String(value) : value.toFixed(2);
}

function getWindowText(value: unknown): string {
  if (!value || typeof value !== "object") return "-";
  const from = asString((value as Record<string, unknown>).from);
  const to = asString((value as Record<string, unknown>).to);
  if (from === "-" || to === "-") return "-";
  return `${from} ~ ${to}`;
}

function buildEvidenceLines(item: TrainingAlertItem): string[] {
  const evidence = item.evidence ?? {};

  if (item.alert_type === "volume_drop_7d") {
    const dropRatio = asNumber(evidence.drop_ratio);
    const currentAvg = asNumber(evidence.current_avg);
    const baselineAvg = asNumber(evidence.baseline_avg);
    return [
      `当前窗口：${getWindowText(evidence.current_window)}，日均题量 ${formatFloat(currentAvg)}。`,
      `基线窗口：${getWindowText(evidence.baseline_window)}，日均题量 ${formatFloat(baselineAvg)}。`,
      `题量相对下降：${formatPercent(dropRatio)}，触发等级：${asString(evidence.threshold_level)}。`,
    ];
  }

  if (item.alert_type === "inactive_days") {
    const inactiveDays = asNumber(evidence.inactive_days);
    const baselineAvg = asNumber(evidence.baseline_avg_daily);
    return [
      `已连续 ${formatFloat(inactiveDays)} 天无训练记录。`,
      `连续停训区间：${getWindowText(evidence.current_window)}。`,
      `历史基线日均题量：${formatFloat(baselineAvg)}（基线区间：${getWindowText(evidence.baseline_window)}）。`,
    ];
  }

  if (item.alert_type === "difficulty_drop_7d") {
    const inactiveDays = asNumber(evidence.inactive_days);
    const highMin = asNumber(evidence.high_min_per_day);
    const highTotal = asNumber(evidence.window_high_total);
    const total = asNumber(evidence.window_total);
    const ratio = asNumber(evidence.window_high_ratio);
    return [
      `已连续 ${formatFloat(inactiveDays)} 天高难题未达标（每日阈值：${formatFloat(highMin)} 题）。`,
      `统计区间：${getWindowText(evidence.current_window)}。`,
      `区间内高难题 ${formatFloat(highTotal)} / 总题量 ${formatFloat(total)}，高难占比 ${formatPercent(ratio)}。`,
    ];
  }

  return Object.entries(evidence).map(([key, value]) => `${key}：${typeof value === "object" ? JSON.stringify(value) : asString(value)}`);
}

function formatDateTime(value: string): string {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }
  return date.toLocaleString("zh-CN");
}

function getSeverityText(value: TrainingAlertItem["severity"]): string {
  if (value === "high") return "高";
  if (value === "medium") return "中";
  return "低";
}

function getStatusText(value: TrainingAlertItem["status"]): string {
  if (value === "resolved") return "已处理";
  if (value === "ack") return "已确认";
  return "待处理";
}

function buildConfigPatch(base: AnomalyRuleConfig, next: AnomalyRuleConfig): Partial<AnomalyRuleConfig> {
  const patch: Partial<AnomalyRuleConfig> = {};
  for (const meta of ruleFieldMetas) {
    const key = meta.key;
    const oldValue = base[key];
    const newValue = next[key];
    const changed = meta.type === "float"
      ? Math.abs(oldValue - newValue) > 1e-9
      : oldValue !== newValue;
    if (changed) {
      patch[key] = newValue;
    }
  }
  // 题量下降与连续停训共用同一“基线最小日均题量”参数：前端始终保持两字段同值。
  if (Math.abs(base.baseline_min_daily - next.inactive_baseline_min_daily) > 1e-9) {
    patch.baseline_min_daily = next.inactive_baseline_min_daily;
  }
  if (Math.abs(base.inactive_baseline_min_daily - next.inactive_baseline_min_daily) > 1e-9) {
    patch.inactive_baseline_min_daily = next.inactive_baseline_min_daily;
  }
  return patch;
}

/** AlertsPage 承接训练异常预警中心：检测、规则配置、预警列表与状态流转。 */
export function AlertsPage() {
  const { user } = useAuth();

  const [configLoading, setConfigLoading] = useState(false);
  const [configSaving, setConfigSaving] = useState(false);
  const [configMessage, setConfigMessage] = useState("");
  const [configError, setConfigError] = useState("");
  const [config, setConfig] = useState<AnomalyRuleConfig | null>(null);
  const [configDraft, setConfigDraft] = useState<AnomalyRuleConfig | null>(null);

  const [detecting, setDetecting] = useState(false);
  const [detectMessage, setDetectMessage] = useState("");
  const [detectError, setDetectError] = useState("");

  const [alertsLoading, setAlertsLoading] = useState(false);
  const [alertsError, setAlertsError] = useState("");
  const [alertsMessage, setAlertsMessage] = useState("");
  const [alertsPayload, setAlertsPayload] = useState<TrainingAlertsListPayload | null>(null);
  const [studentNameMap, setStudentNameMap] = useState<Record<string, string>>({});
  const [alertActionID, setAlertActionID] = useState<number | null>(null);
  const [resolvingAll, setResolvingAll] = useState(false);
  const [page, setPage] = useState(1);
  const [filters, setFilters] = useState<AlertFilters>({
    student_id: "",
    status: "",
    severity: "",
    from: "",
    to: "",
    count: 20,
  });

  async function loadConfig() {
    if (!user) return;
    setConfigLoading(true);
    setConfigError("");
    try {
      const payload = await api.getAnomalyConfig(user.token);
      const normalized: AnomalyRuleConfig = {
        ...payload,
        baseline_min_daily: payload.inactive_baseline_min_daily,
      };
      setConfig(normalized);
      setConfigDraft(normalized);
    } catch (error) {
      setConfigError(error instanceof Error ? error.message : "加载规则配置失败");
    } finally {
      setConfigLoading(false);
    }
  }

  async function loadAlerts(nextPage = page, nextFilters = filters) {
    if (!user) return;
    setAlertsLoading(true);
    setAlertsError("");
    try {
      const payload = await api.listTrainingAlerts(user.token, {
        page: nextPage,
        count: nextFilters.count,
        student_id: nextFilters.student_id.trim() || undefined,
        status: nextFilters.status || undefined,
        severity: nextFilters.severity || undefined,
        from: nextFilters.from || undefined,
        to: nextFilters.to || undefined,
      });
      setAlertsPayload(payload);
      setPage(nextPage);
    } catch (error) {
      setAlertsPayload(null);
      setAlertsError(error instanceof Error ? error.message : "加载预警列表失败");
    } finally {
      setAlertsLoading(false);
    }
  }

  useEffect(() => {
    if (!user) return;
    void loadConfig();
    void loadAlerts(1);
    api.listUsers(user.token, "")
      .then((payload) => {
        const map: Record<string, string> = {};
        payload.list.forEach((item) => {
          map[item.id] = item.name;
        });
        setStudentNameMap(map);
      })
      .catch(() => {
        setStudentNameMap({});
      });
  }, [user]);

  const totalPages = useMemo(() => {
    const total = alertsPayload?.count ?? 0;
    return Math.max(1, Math.ceil(total / filters.count));
  }, [alertsPayload, filters.count]);

  async function handleRunDetect() {
    if (!user) return;
    setDetecting(true);
    setDetectError("");
    setDetectMessage("");
    try {
      const payload = await api.runTrainingDetect(user.token);
      setDetectMessage(`检测完成：新增或更新 ${payload.alert_cnt} 条，时间 ${payload.detected_at}`);
      await loadAlerts(1);
    } catch (error) {
      setDetectError(error instanceof Error ? error.message : "触发检测失败");
    } finally {
      setDetecting(false);
    }
  }

  function handleConfigInput(meta: RuleFieldMeta, rawValue: string) {
    if (!configDraft) return;
    const parsed = rawValue.trim() === ""
      ? 0
      : (meta.type === "int" ? Number.parseInt(rawValue, 10) : Number.parseFloat(rawValue));
    const next = Number.isFinite(parsed) ? parsed : 0;
    const nextDraft: AnomalyRuleConfig = {
      ...configDraft,
      [meta.key]: next,
    };
    if (meta.key === "inactive_baseline_min_daily") {
      nextDraft.baseline_min_daily = next;
    }
    setConfigDraft(nextDraft);
  }

  async function handleSaveConfig() {
    if (!user || !config || !configDraft) return;
    setConfigSaving(true);
    setConfigError("");
    setConfigMessage("");
    try {
      const patch = buildConfigPatch(config, configDraft);
      if (!Object.keys(patch).length) {
        setConfigMessage("没有检测到变更字段，无需保存");
        return;
      }
      const payload = await api.patchAnomalyConfig(user.token, patch);
      setConfig(payload.config);
      setConfigDraft(payload.config);
      setConfigMessage(`保存成功：已更新 ${Object.keys(patch).length} 个字段`);
    } catch (error) {
      setConfigError(error instanceof Error ? error.message : "保存规则配置失败");
    } finally {
      setConfigSaving(false);
    }
  }

  function handleResetConfigDraft() {
    if (!config) return;
    setConfigDraft(config);
    setConfigMessage("已恢复到当前已保存配置");
    setConfigError("");
  }

  async function handleChangeAlertStatus(alertID: number, nextStatus: "ack" | "resolved") {
    if (!user) return;
    setAlertsMessage("");
    setAlertActionID(alertID);
    try {
      if (nextStatus === "ack") {
        await api.ackTrainingAlert(user.token, alertID);
      } else {
        await api.resolveTrainingAlert(user.token, alertID);
      }
      await loadAlerts(page);
    } catch (error) {
      setAlertsError(error instanceof Error ? error.message : "更新预警状态失败");
    } finally {
      setAlertActionID(null);
    }
  }

  async function handleResolveAllAlerts() {
    if (!user) return;
    if (!window.confirm("确认一键处理所有异常吗？这会把所有待处理或已确认预警标记为已处理。")) {
      return;
    }
    setResolvingAll(true);
    setAlertsError("");
    setAlertsMessage("");
    try {
      const payload = await api.resolveAllTrainingAlerts(user.token);
      setAlertsMessage(`一键处理完成：本次处理 ${payload.resolved_cnt} 条预警`);
      await loadAlerts(1);
    } catch (error) {
      setAlertsError(error instanceof Error ? error.message : "一键处理失败");
    } finally {
      setResolvingAll(false);
    }
  }

  return (
    <div className="agent-page">
      <section className="panel">
        <div className="panel-title">手动检测</div>
        <p className="muted">不依赖同步，直接触发一次全量训练异常检测。</p>
        <div className="toolbar">
          <button className="primary-button" disabled={detecting} onClick={handleRunDetect} type="button">
            {detecting ? "检测中..." : "立即检测"}
          </button>
          {detectMessage ? <span className="helper-text">{detectMessage}</span> : null}
        </div>
        {detectError ? <div className="notice notice-error">{detectError}</div> : null}
      </section>

      <section className="panel">
        <div className="panel-title">预警规则配置</div>
        <p className="muted">只会提交修改过的字段，未修改字段保持原值。保存后会持久化到数据库。</p>
        <div className="stack stack-column">
          <div className="muted">
            题量下降预警：当学生最近一段时间的日均做题量，相比历史基线明显下降时触发，用于发现训练量下滑。
          </div>
          <div className="muted">
            连续停训预警：当学生连续多天没有训练记录时触发，用于尽早识别训练中断风险。
          </div>
          <div className="muted">
            高难题预警：按“每日高难题是否达标”记录，连续多天未达标即触发，逻辑与连续停训预警一致。
          </div>
        </div>

        {configLoading || !configDraft ? (
          <div className="empty-state">配置加载中...</div>
        ) : (
          <div className="alert-config-grid">
            {ruleFieldMetas.map((meta) => (
              <label key={String(meta.key)} className="field">
                <span>{meta.label}</span>
                <input
                  type="number"
                  step={meta.step ?? "1"}
                  value={configDraft[meta.key]}
                  onChange={(event) => handleConfigInput(meta, event.target.value)}
                />
                <small className="muted">{meta.desc}</small>
              </label>
            ))}
          </div>
        )}

        <div className="toolbar">
          <button className="primary-button" disabled={configSaving || !configDraft} onClick={handleSaveConfig} type="button">
            {configSaving ? "保存中..." : "保存配置"}
          </button>
          <button className="ghost-button" disabled={configSaving || !configDraft} onClick={handleResetConfigDraft} type="button">
            重置编辑内容
          </button>
          {configMessage ? <span className="helper-text">{configMessage}</span> : null}
        </div>
        {configError ? <div className="notice notice-error">{configError}</div> : null}
      </section>

      <section className="panel">
        <div className="panel-title">预警列表</div>
        <div className="alert-filter-grid">
          <label className="field">
            <span>学号</span>
            <input
              value={filters.student_id}
              onChange={(event) => setFilters((prev) => ({ ...prev, student_id: event.target.value }))}
              placeholder="可选"
            />
          </label>

          <label className="field">
            <span>状态</span>
            <select value={filters.status} onChange={(event) => setFilters((prev) => ({ ...prev, status: event.target.value }))}>
              <option value="">全部</option>
              <option value="new">待处理</option>
              <option value="ack">已确认</option>
              <option value="resolved">已处理</option>
            </select>
          </label>

          <label className="field">
            <span>等级</span>
            <select value={filters.severity} onChange={(event) => setFilters((prev) => ({ ...prev, severity: event.target.value }))}>
              <option value="">全部</option>
              <option value="low">低</option>
              <option value="medium">中</option>
              <option value="high">高</option>
            </select>
          </label>

          <label className="field">
            <span>开始日期</span>
            <input type="date" value={filters.from} onChange={(event) => setFilters((prev) => ({ ...prev, from: event.target.value }))} />
          </label>

          <label className="field">
            <span>结束日期</span>
            <input type="date" value={filters.to} onChange={(event) => setFilters((prev) => ({ ...prev, to: event.target.value }))} />
          </label>

          <label className="field">
            <span>每页数量</span>
            <select
              value={String(filters.count)}
              onChange={(event) => setFilters((prev) => ({ ...prev, count: Number.parseInt(event.target.value, 10) || 20 }))}
            >
              <option value="10">10</option>
              <option value="20">20</option>
              <option value="50">50</option>
              <option value="100">100</option>
            </select>
          </label>
        </div>

        <div className="toolbar">
          <button className="secondary-button" onClick={() => void loadAlerts(1)} type="button" disabled={alertsLoading}>
            {alertsLoading ? "查询中..." : "查询预警"}
          </button>
          <button className="primary-button" onClick={() => void handleResolveAllAlerts()} type="button" disabled={alertsLoading || resolvingAll}>
            {resolvingAll ? "处理中..." : "一键处理全部异常"}
          </button>
          <button className="ghost-button" onClick={() => {
            const next: AlertFilters = { student_id: "", status: "", severity: "", from: "", to: "", count: 20 };
            setFilters(next);
            void loadAlerts(1, next);
          }} type="button" disabled={alertsLoading}>
            清空筛选
          </button>
          {alertsMessage ? <span className="helper-text">{alertsMessage}</span> : null}
          <span className="helper-text">总数 {alertsPayload?.count ?? 0}</span>
        </div>

        {alertsError ? <div className="notice notice-error">{alertsError}</div> : null}

        {alertsLoading ? (
          <div className="empty-state">加载中...</div>
        ) : (
          <div className="table-wrap">
            <table className="data-table">
              <thead>
                <tr>
                  <th>姓名</th>
                  <th>学号</th>
                  <th>日期</th>
                  <th>类型</th>
                  <th>等级</th>
                  <th>状态</th>
                  <th>标题</th>
                  <th>证据</th>
                  <th>操作</th>
                </tr>
              </thead>
              <tbody>
                {(alertsPayload?.list ?? []).map((item) => (
                  <tr key={item.id}>
                    <td>{studentNameMap[item.student_id] ?? "-"}</td>
                    <td>{item.student_id}</td>
                    <td>{item.alert_date}</td>
                    <td>{item.alert_type}</td>
                    <td>{getSeverityText(item.severity)}</td>
                    <td>{getStatusText(item.status)}</td>
                    <td>
                      <div>{item.title}</div>
                      <div className="muted">{formatDateTime(item.updated_at)}</div>
                    </td>
                    <td>
                      <div className="stack stack-column">
                        {buildEvidenceLines(item).map((line) => (
                          <div key={`${item.id}-${line}`} className="muted">{line}</div>
                        ))}
                        <details className="details-box">
                          <summary>查看原始证据</summary>
                          <pre>{JSON.stringify(item.evidence, null, 2)}</pre>
                        </details>
                      </div>
                    </td>
                    <td>
                      <div className="row-action">
                        <button
                          className="secondary-button row-action-button"
                          type="button"
                          disabled={item.status !== "new" || alertActionID === item.id}
                          onClick={() => void handleChangeAlertStatus(item.id, "ack")}
                        >
                          确认
                        </button>
                        <button
                          className="primary-button row-action-button"
                          type="button"
                          disabled={item.status === "resolved" || alertActionID === item.id}
                          onClick={() => void handleChangeAlertStatus(item.id, "resolved")}
                        >
                          处理完成
                        </button>
                      </div>
                    </td>
                  </tr>
                ))}
                {!alertsPayload?.list.length ? (
                  <tr>
                    <td colSpan={8}>
                      <div className="empty-state">暂无预警数据。</div>
                    </td>
                  </tr>
                ) : null}
              </tbody>
            </table>
          </div>
        )}

        <div className="toolbar">
          <button
            className="ghost-button"
            type="button"
            disabled={alertsLoading || page <= 1}
            onClick={() => void loadAlerts(page - 1)}
          >
            上一页
          </button>
          <span className="helper-text">第 {page} / {totalPages} 页</span>
          <button
            className="ghost-button"
            type="button"
            disabled={alertsLoading || page >= totalPages}
            onClick={() => void loadAlerts(page + 1)}
          >
            下一页
          </button>
        </div>
      </section>
    </div>
  );
}
