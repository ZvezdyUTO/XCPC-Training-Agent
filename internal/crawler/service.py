from datetime import datetime
from typing import List, Tuple

from ac_client import get_ac_contest_history_in_range, get_ac_submissions_in_range
from aggregator import (
    build_ac_contest_records,
    build_ac_daily_stats,
    build_cf_contest_records,
    build_cf_daily_stats,
    merge_daily_stats,
)
from cf_client import get_cf_rating_history, get_cf_submissions_in_range
from models import ContestRecord, DailyTrainingStats


def fetch_range(
        student_id: str,
        cf_handle: str | None,
        ac_handle: str | None,
        from_dt: datetime,
        to_dt: datetime,
) -> Tuple[List[ContestRecord], List[DailyTrainingStats]]:
    """
    返回：
    - contests: CF+AC 的比赛记录（按时间排序）
    - daily: CF+AC 合并后的每日训练统计（一天一条，按日期排序）
    """
    from_ts = int(from_dt.timestamp())
    to_ts = int(to_dt.timestamp())

    contests: List[ContestRecord] = []
    cf_daily: List[DailyTrainingStats] = []
    ac_daily: List[DailyTrainingStats] = []

    # CF contests + CF daily
    if cf_handle:
        rating_hist = get_cf_rating_history(cf_handle)
        contests += build_cf_contest_records(student_id, rating_hist, from_ts, to_ts)

        subs = get_cf_submissions_in_range(cf_handle, from_ts, to_ts)
        cf_daily = build_cf_daily_stats(student_id, subs)

    # AC contests + AC daily
    if ac_handle:
        # contests: parse history HTML and filter by [from_dt,to_dt]
        history_rows = get_ac_contest_history_in_range(ac_handle, from_dt, to_dt)
        contests += build_ac_contest_records(student_id, history_rows)

        subs = get_ac_submissions_in_range(ac_handle, from_ts, to_ts)
        ac_daily = build_ac_daily_stats(student_id, subs)

    contests.sort(key=lambda x: x.date)
    daily = merge_daily_stats(student_id, cf_daily, ac_daily)
    return contests, daily