import type { ImportPreviewRow, UserItem } from "../../shared/types";

/** DEFAULT_STUDENT_PASSWORD 约定学生导入时的默认密码。 */
export const DEFAULT_STUDENT_PASSWORD = "000000";

function splitLine(raw: string): string[] {
  return raw.split(/,|，/).map((item) => item.trim());
}

/** parseImportText 负责把多行文本解析为导入预览结果。 */
export function parseImportText(source: string, defaultPassword: string): ImportPreviewRow[] {
  return source
    .split(/\r?\n/)
    .map((line, index) => ({ line, lineNo: index + 1 }))
    .filter(({ line }) => line.trim() !== "")
    .map(({ line, lineNo }) => {
      const fields = splitLine(line);
      const [id = "", name = "", password = "", cfHandle = "", acHandle = ""] = fields;

      if (fields.length > 5) {
        return {
          lineNo,
          raw: line,
          id,
          name,
          password: password || defaultPassword,
          cfHandle,
          acHandle,
          valid: false,
          error: "字段数量超过 5 列",
        };
      }

      if (id === "" || name === "") {
        return {
          lineNo,
          raw: line,
          id,
          name,
          password: password || defaultPassword,
          cfHandle,
          acHandle,
          valid: false,
          error: "学号和姓名不能为空",
        };
      }

      return {
        lineNo,
        raw: line,
        id,
        name,
        password: password || defaultPassword,
        cfHandle,
        acHandle,
        valid: true,
        error: "",
      };
    });
}

/** previewToUsers 把通过校验的预览行转换为后端导入格式。 */
export function previewToUsers(previewRows: ImportPreviewRow[]): UserItem[] {
  return previewRows
    .filter((item) => item.valid)
    .map((item) => ({
      id: item.id,
      name: item.name,
      password: item.password,
      status: 0,
      is_system: 0,
      cf_handle: item.cfHandle,
      ac_handle: item.acHandle,
    }));
}
