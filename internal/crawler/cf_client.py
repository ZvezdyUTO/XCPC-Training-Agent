import time
from typing import List, Optional

import requests

BASE = "https://codeforces.com/api"


def get_cf_rating_history(handle: str) -> list:
    url = f"{BASE}/user.rating"
    r = requests.get(url, params={"handle": handle}, timeout=15)
    r.raise_for_status()
    data = r.json()
    if data.get("status") != "OK":
        raise RuntimeError(f"CF user.rating failed: {data.get('comment')}")
    return data["result"]


def get_cf_submissions_in_range(
        handle: str,
        from_ts: int,
        to_ts: int,
        chunk_size: int = 200,
        sleep_sec: float = 0.2,
) -> list:
    """
    拉取 [from_ts, to_ts] 的 submissions。
    CF API: user.status?handle=...&from=...&count=...
    返回按时间倒序（最新在前）。
    我们分页拉，直到“这一页最早的 submission 时间 < from_ts”为止。
    """
    submissions: List[dict] = []
    start = 1

    while True:
        r = requests.get(
            f"{BASE}/user.status",
            params={"handle": handle, "from": start, "count": chunk_size},
            timeout=20,
        )
        r.raise_for_status()
        data = r.json()
        if data.get("status") != "OK":
            raise RuntimeError(f"CF user.status failed: {data.get('comment')}")

        page = data.get("result", [])
        if not page:
            break

        # 过滤区间内
        for s in page:
            ts = s.get("creationTimeSeconds", 0)
            if from_ts <= ts <= to_ts:
                submissions.append(s)

        # 判断是否可以停止分页：这一页最早的时间已经 < from_ts
        min_ts_in_page = min((s.get("creationTimeSeconds", 0) for s in page), default=0)

        # 下一页条件：本页满 chunk_size 且 min_ts_in_page >= from_ts
        if len(page) < chunk_size:
            break
        if min_ts_in_page < from_ts:
            break

        start += chunk_size
        time.sleep(sleep_sec)

    return submissions