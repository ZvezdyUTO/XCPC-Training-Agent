import re
import time
from datetime import datetime
from typing import List

import requests
from bs4 import BeautifulSoup


def get_ac_submissions_in_range(
        handle: str,
        from_ts: int,
        to_ts: int,
        sleep_sec: float = 0.2,
) -> List[dict]:
    """
    Kenkoooo API: up to 500 submissions after from_second.
    时间是升序返回（通常如此），我们可以在超过 to_ts 后停止。
    """
    subs: List[dict] = []
    cur = from_ts

    while True:
        r = requests.get(
            "https://kenkoooo.com/atcoder/atcoder-api/v3/user/submissions",
            params={"user": handle, "from_second": cur},
            timeout=20,
        )
        r.raise_for_status()
        data = r.json()
        if not data:
            break

        # 收集 <= to_ts 的，遇到 > to_ts 的可提前停止（如果数据按时间升序）
        stop = False
        for s in data:
            ts = s.get("epoch_second", 0)
            if ts > to_ts:
                stop = True
                break
            subs.append(s)

        if stop:
            break

        if len(data) < 500:
            break

        cur = max(s.get("epoch_second", 0) for s in data) + 1
        time.sleep(sleep_sec)

    return subs


def get_ac_contest_history_in_range(
        handle: str, from_dt: datetime, to_dt: datetime
) -> List[dict]:
    url = f"https://atcoder.jp/users/{handle}/history"
    r = requests.get(
        url,
        timeout=20,
        headers={"User-Agent": "Mozilla/5.0 (compatible; aATA-crawler/1.0)"},
    )
    r.raise_for_status()

    soup = BeautifulSoup(r.text, "html.parser")

    # AtCoder 页面里一般表格 class="table table-bordered ..."，不要用 soup.find("table")（可能抓到别的表）
    tables = soup.select("table")
    if not tables:
        return []

    # 找到包含 "Rated Only" 下面那张：其表头应包含 Date/Contest/Rank/Performance/New Rating/Diff
    target = None
    for t in tables:
        head = t.find("tr")
        if not head:
            continue
        head_text = " ".join(
            th.get_text(" ", strip=True) for th in head.find_all(["th", "td"])
        )
        if all(
                k in head_text.lower()
                for k in ["date", "contest", "rank", "performance", "rating", "diff"]
        ):
            target = t
            break

    if target is None:
        return []

    # 读取表头，确定列索引
    header_row = target.find("tr")
    headers = [
        c.get_text(" ", strip=True).lower() for c in header_row.find_all(["th", "td"])
    ]

    def idx_of(keywords):
        for i, h in enumerate(headers):
            for kw in keywords:
                if kw in h:
                    return i
        return None

    i_date = idx_of(["date"])
    i_contest = idx_of(["contest"])
    i_rank = idx_of(["rank"])
    i_perf = idx_of(["performance"])
    i_new = idx_of(["new rating", "newrating", "rating"])  # 有时表头就写 "New Rating"
    i_diff = idx_of(["diff"])

    # 必须具备这些列，否则无法解析
    need = [i_date, i_contest, i_rank, i_new, i_diff]
    if any(x is None for x in need):
        return []

    # 从字符串里抽取 "YYYY-MM-DD" 和 "HH:MM"
    date_re = re.compile(r"(\d{4}-\d{2}-\d{2}).*?(\d{2}:\d{2})")
    # 从字符串里抽取一个整数（支持 + / -）
    int_re = re.compile(r"[+-]?\d+")

    def parse_dt(s: str):
        m = date_re.search(s)
        if not m:
            return None
        ymd, hhmm = m.group(1), m.group(2)
        try:
            return datetime.strptime(f"{ymd} {hhmm}", "%Y-%m-%d %H:%M")
        except Exception:
            return None

    def parse_int_optional(s: str):
        s = s.strip()
        if s in ("", "-"):
            return None
        m = int_re.findall(s)
        if not m:
            return None
        # 单元格里可能有多段文本（比如图标 + 数字），取最后一个数字通常最稳
        return int(m[-1])

    out: List[dict] = []

    for row in target.find_all("tr")[1:]:
        tds = row.find_all("td")
        if not tds:
            continue

        # Contest link
        a = tds[i_contest].find("a") if i_contest < len(tds) else None
        if not a or not a.get("href"):
            continue

        name = a.get_text(strip=True)
        contest_id = a["href"].rstrip("/").split("/")[-1]

        # Date
        date_str = tds[i_date].get_text(" ", strip=True) if i_date < len(tds) else ""
        dt = parse_dt(date_str)
        if dt is None:
            continue
        if not (from_dt <= dt <= to_dt):
            continue

        # Rank / Perf / New / Diff
        rank = (
            parse_int_optional(tds[i_rank].get_text(" ", strip=True))
            if i_rank < len(tds)
            else None
        )
        perf = (
            parse_int_optional(tds[i_perf].get_text(" ", strip=True))
            if (i_perf is not None and i_perf < len(tds))
            else 0
        )
        new_rating = (
            parse_int_optional(tds[i_new].get_text(" ", strip=True))
            if i_new < len(tds)
            else None
        )
        diff = (
            parse_int_optional(tds[i_diff].get_text(" ", strip=True))
            if i_diff < len(tds)
            else None
        )

        # rated only：new_rating 和 diff 必须是数字
        if rank is None or new_rating is None or diff is None:
            continue

        out.append(
            {
                "contest_id": contest_id,
                "name": name,
                "date": dt,
                "rank": rank,
                "performance": perf or 0,
                "new_rating": new_rating,
                "rating_change": diff,
            }
        )

    out.sort(key=lambda x: x["date"])
    return out