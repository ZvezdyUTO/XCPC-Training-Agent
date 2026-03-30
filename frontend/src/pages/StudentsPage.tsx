import { useEffect, useMemo, useState } from "react";
import { useAuth } from "../features/auth/AuthContext";
import { DEFAULT_STUDENT_PASSWORD, parseImportText, previewToUsers } from "../features/students/importParser";
import { api } from "../shared/api";
import type { ImportPreviewRow, UserListPayload } from "../shared/types";

/** StudentsPage 负责学生批量导入预览与列表查看。 */
export function StudentsPage() {
  const { user } = useAuth();
  const [source, setSource] = useState("");
  const [defaultPassword, setDefaultPassword] = useState(DEFAULT_STUDENT_PASSWORD);
  const [keyword, setKeyword] = useState("");
  const [users, setUsers] = useState<UserListPayload | null>(null);
  const [listLoading, setListLoading] = useState(false);
  const [importing, setImporting] = useState(false);
  const [message, setMessage] = useState("");

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
      const payload = await api.listUsers(user.token, nextKeyword);
      setUsers(payload);
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

  return (
    <div className="page-grid two-columns">
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

      <section className="panel">
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
                  <th>CF</th>
                  <th>AC</th>
                  <th>身份</th>
                </tr>
              </thead>
              <tbody>
                {users?.list?.length ? (
                  users.list.map((item) => (
                    <tr key={item.id}>
                      <td>{item.id}</td>
                      <td>{item.name}</td>
                      <td>{item.cf_handle || "-"}</td>
                      <td>{item.ac_handle || "-"}</td>
                      <td>{item.is_system === 1 ? "管理员" : "学生"}</td>
                    </tr>
                  ))
                ) : (
                  <tr>
                    <td colSpan={5}>
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
