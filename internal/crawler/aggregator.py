from collections import defaultdict
from datetime import datetime
from typing import Dict, List, Tuple

from ac_difficulty import load_ac_difficulty
from models import ContestRecord, DailyTrainingStats

ATCODER_BUCKETS = [
    (0, 399, "0-399"),
    (400, 799, "400-799"),
    (800, 1199, "800-1199"),
    (1200, 1599, "1200-1599"),
    (1600, 1999, "1600-1999"),
    (2000, 2399, "2000-2399"),
    (2400, 2799, "2400-2799"),
    (2800, float("inf"), "2800+"),
]


def normalize_date(dt: datetime) -> datetime:
    return datetime(dt.year, dt.month, dt.day)


# -------- CF contests --------


def build_cf_contest_records(
        student_id: str, rating_history: list, from_ts: int, to_ts: int
) -> List[ContestRecord]:
    records: List[ContestRecord] = []
    for item in rating_history:
        ts = item.get("ratingUpdateTimeSeconds", 0)
        if not (from_ts <= ts <= to_ts):
            continue

        old_rating = item["oldRating"]
        new_rating = item["newRating"]

        records.append(
            ContestRecord(
                student_id=student_id,
                platform="CF",
                contest_id=str(item["contestId"]),
                name=item["contestName"],
                date=datetime.fromtimestamp(ts),
                rank=item["rank"],
                old_rating=old_rating,
                new_rating=new_rating,
                rating_change=new_rating - old_rating,
                performance=0,
            )
        )

    records.sort(key=lambda x: x.date)
    return records


# -------- CF daily --------


def build_cf_daily_stats(
        student_id: str, submissions: list
) -> List[DailyTrainingStats]:
    """
    CF：同一题多次 AC 只算一次；按 creationTimeSeconds 的日期聚合；按 rating 分桶（rating 整数）。
    """
    solved = set()
    daily: Dict[datetime, Dict[int, int]] = defaultdict(lambda: defaultdict(int))

    for sub in submissions:
        if sub.get("verdict") != "OK":
            continue

        problem = sub.get("problem", {})
        contest_id = problem.get("contestId")
        index = problem.get("index")

        if contest_id is None:
            continue

        key = (contest_id, index)
        if key in solved:
            continue
        solved.add(key)

        rating = problem.get("rating")
        if rating is None:
            # 没 rating 的题直接不入桶（你若要统计可加 "undefined"）
            continue

        ts = datetime.fromtimestamp(sub["creationTimeSeconds"])
        d = normalize_date(ts)
        daily[d][int(rating)] += 1

    out: List[DailyTrainingStats] = []
    for d, bucket in sorted(daily.items()):
        total = sum(bucket.values())
        out.append(
            DailyTrainingStats(
                student_id=student_id,
                date=d,
                cf_new_total=total,
                cf_new=dict(bucket),
                ac_new_total=0,
                ac_new_range={},
            )
        )
    return out


# -------- AC contests --------


def build_ac_contest_records(
        student_id: str, ac_history_rows: list
) -> List[ContestRecord]:
    """
    ac_history_rows 来自 ac_client.get_ac_contest_history_in_range
    """
    out: List[ContestRecord] = []

    for row in ac_history_rows:
        new_rating = row["new_rating"]
        diff = row["rating_change"]
        old_rating = new_rating - diff

        out.append(
            ContestRecord(
                student_id=student_id,
                platform="AC",
                contest_id=row["contest_id"],
                name=row["name"],
                date=row["date"],
                rank=row["rank"],
                old_rating=old_rating,
                new_rating=new_rating,
                rating_change=diff,
                performance=row.get("performance", 0),
            )
        )

    out.sort(key=lambda x: x.date)
    return out


# -------- AC daily --------


def build_ac_daily_stats(
        student_id: str, submissions: list
) -> List[DailyTrainingStats]:
    """
    AC：同一 problem 多次 AC 只算一次；按 epoch_second 的日期聚合；difficulty -> bucket。
    """
    diff_map = load_ac_difficulty()

    solved = set()
    daily: Dict[datetime, Dict[str, int]] = defaultdict(lambda: defaultdict(int))

    for sub in submissions:
        if sub.get("result") != "AC":
            continue

        pid = sub.get("problem_id")
        if not pid:
            continue
        if pid in solved:
            continue
        solved.add(pid)

        raw = diff_map.get(pid, {}).get("difficulty")
        try:
            diff = float(raw)
        except Exception:
            diff = None

        ts = datetime.fromtimestamp(sub["epoch_second"])
        d = normalize_date(ts)

        label = None
        if diff is not None:
            for lo, hi, lbl in ATCODER_BUCKETS:
                if lo <= diff <= hi:
                    label = lbl
                    break

        if label is None:
            # 建议保留 undefined，别强行塞 0-399（否则会污染低段）
            label = "undefined"

        daily[d][label] += 1

    out: List[DailyTrainingStats] = []
    for d, bucket in sorted(daily.items()):
        total = sum(bucket.values())
        out.append(
            DailyTrainingStats(
                student_id=student_id,
                date=d,
                cf_new_total=0,
                cf_new={},
                ac_new_total=total,
                ac_new_range=dict(bucket),
            )
        )
    return out


# -------- merge CF+AC by date --------


def merge_daily_stats(
        student_id: str,
        cf_list: List[DailyTrainingStats],
        ac_list: List[DailyTrainingStats],
) -> List[DailyTrainingStats]:
    by_date: Dict[datetime, DailyTrainingStats] = {}

    def ensure(d: datetime) -> DailyTrainingStats:
        if d not in by_date:
            by_date[d] = DailyTrainingStats(
                student_id=student_id,
                date=d,
                cf_new_total=0,
                cf_new={},
                ac_new_total=0,
                ac_new_range={},
            )
        return by_date[d]

    for s in cf_list:
        x = ensure(s.date)
        x.cf_new_total += s.cf_new_total
        for k, v in s.cf_new.items():
            x.cf_new[k] = x.cf_new.get(k, 0) + v

    for s in ac_list:
        x = ensure(s.date)
        x.ac_new_total += s.ac_new_total
        for k, v in s.ac_new_range.items():
            x.ac_new_range[k] = x.ac_new_range.get(k, 0) + v

    return [by_date[d] for d in sorted(by_date.keys())]