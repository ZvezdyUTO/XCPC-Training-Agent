import { useEffect, useMemo, useState } from "react";
import { useAuth } from "../features/auth/AuthContext";
import { DEFAULT_STUDENT_PASSWORD, parseImportText, previewToUsers } from "../features/students/importParser";
import { api } from "../shared/api";
import type { ImportPreviewRow, SyncAllTrainingPayload, SyncStateListPayload, UserListPayload } from "../shared/types";

function formatShortDate(value?: string | null): string {
  if (!value) {
    return "-";
  }
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return "-";
  }
  const year = String(date.getFullYear()).slice(-2);
  const month = String(date.getMonth() + 1).padStart(2, "0");
  const day = String(date.getDate()).padStart(2, "0");
  return `${year}.${month}.${day}`;
}

/** StudentsPage 负责学生批量导入预览与列表查看。 */
export function StudentsPage() {
  const { user } = useAuth();
  const [source, setSource] = useState("");
  const [defaultPassword, setDefaultPassword] = useState(DEFAULT_STUDENT_PASSWORD);
  const [keyword, setKeyword] = useState("");
  const [users, setUsers] = useState<UserListPayload | null>(null);
  const [syncStates, setSyncStates] = useState<SyncStateListPayload | null>(null);
  const [listLoading, setListLoading] = useState(false);
  const [importing, setImporting] = useState(false);
  const [message, setMessage] = useState("");
  const [syncing, setSyncing] = useState(false);
  const [syncMessage, setSyncMessage] = useState("");
  const [syncResult, setSyncResult] = useState<SyncAllTrainingPayload | null>(null);
  const [syncingStudentID, setSyncingStudentID] = useState("");
  const [deletingStudentID, setDeletingStudentID] = useState("");
  const [rowSyncHint, setRowSyncHint] = useState<Record<string, string>>({});

  const previewRows = useMemo<ImportPreviewRow[]>(
    () => parseImportText(source, defaultPassword),
    [source, defaultPassword],
  );
  const validRows = previewRows.filter((item) => item.valid);

  async function loadUsers(nextKeyword: string) {
    if (!user) {
      return;
    }
    setListLoading(true);
    try {
      const [userPayload, syncStatePayload] = await Promise.all([
        api.listUsers(user.token, nextKeyword),
        api.listSyncStates(user.token),
      ]);
      setUsers({
        ...userPayload,
        count: userPayload.list.filter((item) => item.is_system !== 1).length,
        list: userPayload.list.filter((item) => item.is_system !== 1),
      });
      setSyncStates(syncStatePayload);
    } finally {
      setListLoading(false);
    }
  }

  useEffect(() => {
    void loadUsers("");
  }, [user]);

  async function handleImport() {
    if (!user) {
      return;
    }
    setMessage("");

    if (validRows.length === 0) {
      setMessage("没有可导入的有效行");
      return;
    }

    if (!window.confirm(`确认导入 ${validRows.length} 个学生吗？`)) {
      return;
    }

    setImporting(true);
    try {
      const payload = await api.createUsers(user.token, previewToUsers(validRows));
      setMessage(`导入完成：成功 ${payload.success} / 总计 ${payload.total}`);
      setSource("");
      await loadUsers(keyword);
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "导入失败");
    } finally {
      setImporting(false);
    }
  }

  async function handleSyncAll() {
    if (!user) {
      return;
    }

    if (!window.confirm("确认触发全体学生训练同步吗？")) {
      return;
    }

    setSyncing(true);
    setSyncMessage("");
    setSyncResult(null);
    try {
      const payload = await api.syncAllTraining(user.token);
      setSyncResult(payload);
      setSyncMessage(`同步完成：成功 ${payload.success_cnt}${payload.failed_cnt ? `，失败 ${payload.failed_cnt}` : ""}`);
      await loadUsers(keyword);
    } catch (error) {
      setSyncMessage(error instanceof Error ? error.message : "同步触发失败");
    } finally {
      setSyncing(false);
    }
  }

  async function handleSyncOne(studentID: string) {
    if (!user) {
      return;
    }

    setSyncingStudentID(studentID);
    setRowSyncHint((prev) => ({
      ...prev,
      [studentID]: "",
    }));
    try {
      const payload = await api.syncOneTraining(user.token, studentID);
      setRowSyncHint((prev) => ({
        ...prev,
        [studentID]: payload.mode,
      }));
      await loadUsers(keyword);
    } catch (error) {
      setRowSyncHint((prev) => ({
        ...prev,
        [studentID]: error instanceof Error ? error.message : "同步失败",
      }));
    } finally {
      setSyncingStudentID("");
    }
  }

  async function handleDeleteUser(studentID: string, studentName: string) {
    if (!user) {
      return;
    }

    const confirmed = window.confirm(
      `确认删除 ${studentID} ${studentName} 吗？\n这会同时删除该学生的训练记录、比赛记录和同步状态。`,
    );
    if (!confirmed) {
      return;
    }

    setDeletingStudentID(studentID);
    setRowSyncHint((prev) => ({
      ...prev,
      [studentID]: "",
    }));
    try {
      await api.deleteUser(user.token, studentID);
      setRowSyncHint((prev) => ({
        ...prev,
        [studentID]: "已删除",
      }));
      await loadUsers(keyword);
    } catch (error) {
      setRowSyncHint((prev) => ({
        ...prev,
        [studentID]: error instanceof Error ? error.message : "删除失败",
      }));
    } finally {
      setDeletingStudentID("");
    }
  }

  const syncStateMap = useMemo(() => {
    const map = new Map<string, SyncStateListPayload["list"][number]>();
    syncStates?.list.forEach((item) => {
      map.set(item.student_id, item);
    });
    return map;
  }, [syncStates]);

  return (
    <div className="students-layout">
      <div className="students-main-column">
        <section className="panel">
          <div className="panel-title">训练同步</div>
          <p className="muted">这里负责触发全体学生训练同步，并查看本次同步结果。</p>

          <div className="toolbar">
            <button className="primary-button" disabled={syncing} onClick={handleSyncAll} type="button">
              {syncing ? "同步中..." : "触发训练同步"}
            </button>
            {syncMessage ? <span className="helper-text">{syncMessage}</span> : null}
          </div>

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
            <div className="empty-state">暂无同步结果。</div>
          )}
        </section>

        <section className="panel">
          <div className="panel-title">批量导入学生</div>
          <p className="muted">
            每行格式：学号, 姓名, 初始密码, cfid, acid。学号和姓名必填，密码为空时使用默认密码，CF/AC
            留空会按空字符串提交。
          </p>

          <label className="field">
            <span>默认密码</span>
            <input value={defaultPassword} onChange={(event) => setDefaultPassword(event.target.value)} />
          </label>

          <label className="field">
            <span>导入文本</span>
            <textarea
              className="code-input"
              placeholder={"230000001,张三,123456,demo_cf,demo_ac\n230000002,李四,,,atcoder_id"}
              rows={12}
              value={source}
              onChange={(event) => setSource(event.target.value)}
            />
          </label>

          <div className="toolbar">
            <button className="primary-button" disabled={importing} onClick={handleImport} type="button">
              {importing ? "导入中..." : "确认导入"}
            </button>
            {message ? <span className="helper-text">{message}</span> : null}
          </div>

          <div className="subsection-title">导入预览</div>
          {previewRows.length === 0 ? (
            <div className="empty-state">粘贴内容后会在这里显示预览与校验结果。</div>
          ) : (
            <div className="table-wrap">
              <table className="data-table">
                <thead>
                  <tr>
                    <th>行号</th>
                    <th>学号</th>
                    <th>姓名</th>
                    <th>密码</th>
                    <th>CF</th>
                    <th>AC</th>
                    <th>结果</th>
                  </tr>
                </thead>
                <tbody>
                  {previewRows.map((item) => (
                    <tr key={`${item.lineNo}-${item.raw}`}>
                      <td>{item.lineNo}</td>
                      <td>{item.id || "-"}</td>
                      <td>{item.name || "-"}</td>
                      <td>{item.password || "-"}</td>
                      <td>{item.cfHandle || "-"}</td>
                      <td>{item.acHandle || "-"}</td>
                      <td>
                        <span className={item.valid ? "status-chip status-ok" : "status-chip status-error"}>
                          {item.valid ? "可导入" : item.error}
                        </span>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </section>
      </div>

      <section className="panel students-side-column">
        <div className="panel-title">学生列表</div>
        <div className="toolbar">
          <input
            className="search-input"
            placeholder="按姓名筛选"
            value={keyword}
            onChange={(event) => setKeyword(event.target.value)}
          />
          <button className="secondary-button" onClick={() => void loadUsers(keyword)} type="button">
            查询
          </button>
        </div>

        {listLoading ? (
          <div className="empty-state">加载中...</div>
        ) : (
          <div className="table-wrap">
            <table className="data-table">
              <thead>
                <tr>
                  <th>学号</th>
                  <th>姓名</th>
                  <th>最新日期</th>
                  <th>CF</th>
                  <th>AC</th>
                  <th>操作</th>
                </tr>
              </thead>
              <tbody>
                {users?.list?.length ? (
                  users.list.map((item) => {
                    const syncState = syncStateMap.get(item.id);
                    const isSynced = syncState?.is_fully_initialized === 1;
                    const isRowSyncing = syncingStudentID === item.id;
                    const isRowDeleting = deletingStudentID === item.id;
                    const canSync = item.cf_handle !== "" || item.ac_handle !== "";
                    return (
                      <tr key={item.id} className={isSynced ? undefined : "table-row-danger"}>
                        <td>{item.id}</td>
                        <td>{item.name}</td>
                        <td>{formatShortDate(syncState?.latest_successful_date)}</td>
                        <td>{item.cf_handle || "-"}</td>
                        <td>{item.ac_handle || "-"}</td>
                        <td>
                          <div className="row-action">
                            <button
                              className="secondary-button row-action-button"
                              disabled={isRowSyncing || isRowDeleting || !canSync}
                              onClick={() => void handleSyncOne(item.id)}
                              type="button"
                            >
                              {isRowSyncing ? "同步中..." : canSync ? "同步" : "无账号"}
                            </button>
                            <button
                              className="danger-button row-action-button"
                              disabled={isRowSyncing || isRowDeleting}
                              onClick={() => void handleDeleteUser(item.id, item.name)}
                              type="button"
                            >
                              {isRowDeleting ? "删除中..." : "删除"}
                            </button>
                            {rowSyncHint[item.id] ? (
                              <span className="row-action-hint">{rowSyncHint[item.id]}</span>
                            ) : null}
                          </div>
                        </td>
                      </tr>
                    );
                  })
                ) : (
                  <tr>
                    <td colSpan={6}>
                      <div className="empty-state">没有查到学生数据。</div>
                    </td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>
        )}
      </section>
    </div>
  );
}
