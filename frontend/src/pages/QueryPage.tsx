import { FormEvent, useEffect, useMemo, useState } from "react";
import { useAuth } from "../features/auth/AuthContext";
import { api } from "../shared/api";
import type { ContestRankingPayload, TrainingLeaderboardPayload, TrainingSummaryPayload, UserItem } from "../shared/types";

function formatUnknown(value: unknown): string {
  if (value === null || value === undefined || value === "") {
    return "-";
  }
  if (typeof value === "number") {
    return Number.isFinite(value) ? value.toString() : "-";
  }
  return String(value);
}

function formatScore(value: number): string {
  return Number.isFinite(value) ? value.toFixed(2) : "-";
}

function formatPercent(value: number): string {
  return Number.isFinite(value) ? `${(value * 100).toFixed(1)}%` : "-";
}

/** QueryPage 承接不依赖模型的管理员直查能力。 */
export function QueryPage() {
  const { user } = useAuth();
  const [studentOptions, setStudentOptions] = useState<UserItem[]>([]);
  const [rankingPlatform, setRankingPlatform] = useState("CF");
  const [rankingContestID, setRankingContestID] = useState("");
  const [rankingLoading, setRankingLoading] = useState(false);
  const [rankingError, setRankingError] = useState("");
  const [rankingResult, setRankingResult] = useState<ContestRankingPayload | null>(null);

  const [summaryStudentID, setSummaryStudentID] = useState("");
  const [summaryStudentQuery, setSummaryStudentQuery] = useState("");
  const [summaryPickedStudent, setSummaryPickedStudent] = useState<UserItem | null>(null);
  const [summaryFrom, setSummaryFrom] = useState("");
  const [summaryTo, setSummaryTo] = useState("");
  const [summaryLoading, setSummaryLoading] = useState(false);
  const [summaryError, setSummaryError] = useState("");
  const [summaryResult, setSummaryResult] = useState<TrainingSummaryPayload | null>(null);
  const [leaderboardFrom, setLeaderboardFrom] = useState("");
  const [leaderboardTo, setLeaderboardTo] = useState("");
  const [leaderboardTopN, setLeaderboardTopN] = useState("20");
  const [leaderboardLoading, setLeaderboardLoading] = useState(false);
  const [leaderboardError, setLeaderboardError] = useState("");
  const [leaderboardResult, setLeaderboardResult] = useState<TrainingLeaderboardPayload | null>(null);

  useEffect(() => {
    if (!user) {
      return;
    }
    api.listUsers(user.token, "")
      .then((payload) => {
        setStudentOptions(payload.list.filter((item) => item.is_system !== 1));
      })
      .catch(() => {
        setStudentOptions([]);
      });
  }, [user]);

  const studentCandidates = useMemo(() => {
    const keyword = summaryStudentQuery.trim().toLowerCase();
    if (keyword === "") {
      return [];
    }
    return studentOptions
      .filter((item) => item.id.toLowerCase().startsWith(keyword) || item.name.toLowerCase().startsWith(keyword))
      .slice(0, 8);
  }, [studentOptions, summaryStudentQuery]);

  function handlePickStudent(student: UserItem) {
    setSummaryPickedStudent(student);
    setSummaryStudentID(student.id);
    setSummaryStudentQuery(`${student.name} · ${student.id}`);
    setSummaryError("");
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

  async function handleSummaryQuery(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!user) {
      setSummaryError("未登录");
      return;
    }
    const normalizedQuery = summaryStudentQuery.trim();
    const matchedStudent = summaryPickedStudent
      ?? studentOptions.find((item) => item.id === normalizedQuery || item.name === normalizedQuery);
    const resolvedStudentID = summaryStudentID.trim() || matchedStudent?.id || normalizedQuery;
    if (resolvedStudentID === "" || summaryFrom.trim() === "" || summaryTo.trim() === "") {
      setSummaryError("学生和时间范围不能为空");
      return;
    }
    if (!/^\d+$/.test(resolvedStudentID) && matchedStudent == null) {
      setSummaryError("请输入学号，或从姓名补全候选中选择学生");
      return;
    }

    setSummaryLoading(true);
    setSummaryError("");
    try {
      if (matchedStudent) {
        setSummaryPickedStudent(matchedStudent);
        setSummaryStudentID(matchedStudent.id);
        setSummaryStudentQuery(`${matchedStudent.name} · ${matchedStudent.id}`);
      }
      const payload = await api.getTrainingSummary(user.token, resolvedStudentID, summaryFrom.trim(), summaryTo.trim());
      setSummaryResult(payload);
    } catch (queryError) {
      setSummaryResult(null);
      setSummaryError(queryError instanceof Error ? queryError.message : "查询失败");
    } finally {
      setSummaryLoading(false);
    }
  }

  async function handleLeaderboardQuery(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!user) {
      setLeaderboardError("未登录");
      return;
    }
    if (leaderboardFrom.trim() === "" || leaderboardTo.trim() === "") {
      setLeaderboardError("时间范围不能为空");
      return;
    }

    const parsedTopN = Number.parseInt(leaderboardTopN.trim(), 10);
    const topN = Number.isFinite(parsedTopN) && parsedTopN > 0 ? parsedTopN : 20;

    setLeaderboardLoading(true);
    setLeaderboardError("");
    try {
      const payload = await api.getTrainingLeaderboard(user.token, leaderboardFrom.trim(), leaderboardTo.trim(), topN);
      setLeaderboardResult(payload);
    } catch (queryError) {
      setLeaderboardResult(null);
      setLeaderboardError(queryError instanceof Error ? queryError.message : "查询失败");
    } finally {
      setLeaderboardLoading(false);
    }
  }

  return (
    <div className="agent-page">
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
          <div className="empty-state">输入平台和比赛 ID 后，可以直接查询数据库内的队内排名。</div>
        )}
      </article>

      <article className="panel">
        <div className="panel-title">时间段题量查询</div>
        <form className="agent-form" onSubmit={handleSummaryQuery}>
          <div className="query-form-grid">
            <label className="field">
              <span>学生</span>
              <div className="autocomplete-wrap">
                <input
                  className="agent-inline-input"
                  placeholder="输入姓名或学号"
                  value={summaryStudentQuery}
                  onChange={(event) => {
                    setSummaryStudentQuery(event.target.value);
                    setSummaryStudentID("");
                    setSummaryPickedStudent(null);
                  }}
                />
                {studentCandidates.length > 0 && summaryPickedStudent == null ? (
                  <div className="autocomplete-list">
                    {studentCandidates.map((item) => (
                      <button
                        key={item.id}
                        className="autocomplete-item"
                        type="button"
                        onClick={() => handlePickStudent(item)}
                      >
                        <span>{item.name}</span>
                        <span>{item.id}</span>
                      </button>
                    ))}
                  </div>
                ) : null}
              </div>
            </label>

            <label className="field">
              <span>开始日期</span>
              <input className="agent-inline-input" type="date" value={summaryFrom} onChange={(event) => setSummaryFrom(event.target.value)} />
            </label>

            <label className="field">
              <span>结束日期</span>
              <input className="agent-inline-input" type="date" value={summaryTo} onChange={(event) => setSummaryTo(event.target.value)} />
            </label>

            <button className="secondary-button agent-submit-button" disabled={summaryLoading} type="submit">
              {summaryLoading ? "查询中..." : "查询题量"}
            </button>
          </div>

          {summaryError ? <div className="notice notice-error">{summaryError}</div> : null}
        </form>

        {summaryResult ? (
          <div className="agent-section">
            <div className="agent-trace-grid">
              <div className="agent-meta-item">
                <span>student_name</span>
                <strong>{summaryPickedStudent?.name ?? "-"}</strong>
              </div>
              <div className="agent-meta-item">
                <span>student_id</span>
                <strong>{summaryResult.student_id}</strong>
              </div>
              <div className="agent-meta-item">
                <span>时间范围</span>
                <strong>{summaryResult.from} ~ {summaryResult.to}</strong>
              </div>
              <div className="agent-meta-item">
                <span>CF 总题量</span>
                <strong>{summaryResult.cf_total}</strong>
              </div>
              <div className="agent-meta-item">
                <span>AC 总题量</span>
                <strong>{summaryResult.ac_total ?? 0}</strong>
              </div>
              {summaryResult.training_value ? (
                <>
                  <div className="agent-meta-item">
                    <span>综合分</span>
                    <strong>{formatScore(summaryResult.training_value.score)}</strong>
                  </div>
                  <div className="agent-meta-item">
                    <span>评分版本</span>
                    <strong>{summaryResult.training_value.scoring_version}</strong>
                  </div>
                </>
              ) : null}
            </div>

            {summaryResult.training_value ? (
              <div className="agent-metrics-grid">
                <div className="agent-metric-card">
                  <span>总题量</span>
                  <strong>{summaryResult.training_value.solved_total}</strong>
                </div>
                <div className="agent-metric-card">
                  <span>题量分</span>
                  <strong>{formatScore(summaryResult.training_value.volume_score)}</strong>
                </div>
                <div className="agent-metric-card">
                  <span>难度分</span>
                  <strong>{formatScore(summaryResult.training_value.difficulty_score)}</strong>
                </div>
                <div className="agent-metric-card">
                  <span>挑战分</span>
                  <strong>{formatScore(summaryResult.training_value.challenge_score)}</strong>
                </div>
                <div className="agent-metric-card">
                  <span>Undefined</span>
                  <strong>
                    {summaryResult.training_value.undefined_total} / {formatPercent(summaryResult.training_value.undefined_ratio)}
                  </strong>
                </div>
                <div className="agent-metric-card">
                  <span>CF 当前/峰值</span>
                  <strong>
                    {formatUnknown(summaryResult.training_value.cf_rating.current)} / {formatUnknown(summaryResult.training_value.cf_rating.peak)}
                  </strong>
                </div>
                <div className="agent-metric-card">
                  <span>AC 当前/峰值</span>
                  <strong>
                    {formatUnknown(summaryResult.training_value.ac_rating.current)} / {formatUnknown(summaryResult.training_value.ac_rating.peak)}
                  </strong>
                </div>
              </div>
            ) : null}

            <div className="agent-metrics-grid">
              {Object.entries(summaryResult.cf_distribution).map(([key, value]) => (
                <div key={key} className="agent-metric-card">
                  <span>{key}</span>
                  <strong>{value}</strong>
                </div>
              ))}
              {summaryResult.ac_distribution
                ? Object.entries(summaryResult.ac_distribution).map(([key, value]) => (
                    <div key={`ac-${key}`} className="agent-metric-card">
                      <span>{`AC ${key}`}</span>
                      <strong>{value}</strong>
                    </div>
                  ))
                : null}
            </div>
          </div>
        ) : (
          <div className="empty-state">输入学号和时间范围后，可以直接查询指定时间段内的题量累计。</div>
        )}
      </article>

      <article className="panel">
        <div className="panel-title">训练价值排行榜</div>
        <form className="agent-form" onSubmit={handleLeaderboardQuery}>
          <div className="query-form-grid">
            <label className="field">
              <span>开始日期</span>
              <input className="agent-inline-input" type="date" value={leaderboardFrom} onChange={(event) => setLeaderboardFrom(event.target.value)} />
            </label>

            <label className="field">
              <span>结束日期</span>
              <input className="agent-inline-input" type="date" value={leaderboardTo} onChange={(event) => setLeaderboardTo(event.target.value)} />
            </label>

            <label className="field">
              <span>前 N 名</span>
              <input
                className="agent-inline-input"
                type="number"
                min="1"
                max="100"
                value={leaderboardTopN}
                onChange={(event) => setLeaderboardTopN(event.target.value)}
              />
            </label>

            <button className="secondary-button agent-submit-button" disabled={leaderboardLoading} type="submit">
              {leaderboardLoading ? "查询中..." : "查询排行榜"}
            </button>
          </div>

          {leaderboardError ? <div className="notice notice-error">{leaderboardError}</div> : null}
        </form>

        {leaderboardResult ? (
          <div className="agent-section">
            <div className="agent-trace-grid">
              <div className="agent-meta-item">
                <span>scoring_version</span>
                <strong>{leaderboardResult.scoring_version}</strong>
              </div>
              <div className="agent-meta-item">
                <span>时间范围</span>
                <strong>{leaderboardResult.from} ~ {leaderboardResult.to}</strong>
              </div>
              <div className="agent-meta-item">
                <span>top_n</span>
                <strong>{leaderboardResult.top_n}</strong>
              </div>
              <div className="agent-meta-item">
                <span>count</span>
                <strong>{leaderboardResult.count}</strong>
              </div>
            </div>

            <div className="table-wrap">
              <table className="data-table">
                <thead>
                  <tr>
                    <th>排名</th>
                    <th>学号</th>
                    <th>姓名</th>
                    <th>综合分</th>
                    <th>总题量</th>
                    <th>题量分</th>
                    <th>难度分</th>
                    <th>挑战分</th>
                    <th>Undefined</th>
                    <th>CF 当前/峰值</th>
                    <th>AC 当前/峰值</th>
                  </tr>
                </thead>
                <tbody>
                  {leaderboardResult.items.map((item) => (
                    <tr key={item.student_id}>
                      <td>{item.rank}</td>
                      <td>{item.student_id}</td>
                      <td>{item.student_name || "-"}</td>
                      <td>{formatScore(item.score)}</td>
                      <td>{item.solved_total}</td>
                      <td>{formatScore(item.volume_score)}</td>
                      <td>{formatScore(item.difficulty_score)}</td>
                      <td>{formatScore(item.challenge_score)}</td>
                      <td>{item.undefined_total} / {formatPercent(item.undefined_ratio)}</td>
                      <td>{formatUnknown(item.cf_rating.current)} / {formatUnknown(item.cf_rating.peak)}</td>
                      <td>{formatUnknown(item.ac_rating.current)} / {formatUnknown(item.ac_rating.peak)}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>
        ) : (
          <div className="empty-state">输入时间范围和前 N 名后，可以直接查询训练价值排行榜。</div>
        )}
      </article>
    </div>
  );
}
