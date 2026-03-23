import argparse
import os
import sys
import time
from datetime import datetime, timedelta
from typing import List, Tuple

import pymysql
import yaml

from cf_client import get_cf_rating_history, get_cf_submissions_in_range
from aggregator import build_cf_contest_records, build_cf_daily_stats


def parse_args():
    ap = argparse.ArgumentParser(description="Batch backfill Codeforces data for all users")
    ap.add_argument("--config", default="../../etc/local/api.yaml", help="path to api.yaml")
    ap.add_argument("--from", dest="from_date", required=True, help="YYYY-MM-DD")
    ap.add_argument("--to", dest="to_date", required=True, help="YYYY-MM-DD")
    ap.add_argument("--only", default="", help="only one student_id")
    ap.add_argument("--sleep", type=float, default=0.5, help="sleep seconds between users")
    ap.add_argument("--limit", type=int, default=0, help="max users to process, 0 means no limit")
    return ap.parse_args()


def load_mysql_dsn(config_path: str) -> str:
    with open(config_path, "r", encoding="utf-8") as f:
        cfg = yaml.safe_load(f)
    # 仓库里的配置键名有 MySql / DataSource
    dsn = cfg["MySql"]["DataSource"]
    return dsn


def parse_go_mysql_dsn(dsn: str):
    """
    解析类似:
    root:123456@tcp(127.0.0.1:3307)/aATAdb?charset=utf8mb4&parseTime=True&loc=Local
    """
    try:
        user_pass, rest = dsn.split("@tcp(", 1)
        host_port, rest = rest.split(")/", 1)
        db_name, _query = rest.split("?", 1)

        user, password = user_pass.split(":", 1)
        host, port = host_port.split(":", 1)

        return {
            "host": host,
            "port": int(port),
            "user": user,
            "password": password,
            "database": db_name,
            "charset": "utf8mb4",
            "autocommit": False,
            "cursorclass": pymysql.cursors.DictCursor,
        }
    except Exception as e:
        raise RuntimeError(f"无法解析 DataSource: {dsn}") from e


def get_conn(config_path: str):
    dsn = load_mysql_dsn(config_path)
    conn_kwargs = parse_go_mysql_dsn(dsn)
    return pymysql.connect(**conn_kwargs)


def get_users(conn, only_student_id: str = "", limit: int = 0):
    sql = """
        SELECT id, name, cf_handle
        FROM users
        WHERE delete_at IS NULL
          AND cf_handle IS NOT NULL
          AND TRIM(cf_handle) <> ''
    """
    params = []

    if only_student_id:
        sql += " AND id = %s"
        params.append(only_student_id)

    sql += " ORDER BY id"

    if limit > 0:
        sql += " LIMIT %s"
        params.append(limit)

    with conn.cursor() as cur:
        cur.execute(sql, params)
        return cur.fetchall()


def to_day_start(dt: datetime) -> datetime:
    return datetime(dt.year, dt.month, dt.day)


def build_daily_row(stat):
    """
    把 aggregator.build_cf_daily_stats 产出的 cf_new: Dict[int, int]
    映射到 daily_training_stats 表结构
    """
    cf_map = stat.cf_new or {}

    def g(x: int) -> int:
        return int(cf_map.get(x, 0))

    # 仓库表里是这些固定列
    return (
        stat.student_id,
        stat.date.strftime("%Y-%m-%d"),
        stat.cf_new_total,
        g(800),
        g(900),
        g(1000),
        g(1100),
        g(1200),
        g(1300),
        g(1400),
        g(1500),
        g(1600),
        g(1700),
        g(1800),
        g(1900),
        g(2000),
        g(2100),
        g(2200),
        g(2300),
        g(2400),
        g(2500),
        g(2600),
        g(2700),
        sum(v for k, v in cf_map.items() if isinstance(k, int) and k >= 2800),
    )


def upsert_contests(conn, contest_records):
    if not contest_records:
        return 0

    sql = """
        INSERT INTO contest_records (
            student_id,
            platform,
            contest_id,
            contest_name,
            contest_date,
            contest_rank,
            old_rating,
            new_rating,
            rating_change,
            performance
        ) VALUES (
            %s, %s, %s, %s, %s, %s, %s, %s, %s, %s
        )
        ON DUPLICATE KEY UPDATE
            contest_name = VALUES(contest_name),
            contest_date = VALUES(contest_date),
            contest_rank = VALUES(contest_rank),
            old_rating = VALUES(old_rating),
            new_rating = VALUES(new_rating),
            rating_change = VALUES(rating_change),
            performance = VALUES(performance),
            deleted_at = NULL
    """

    rows = []
    for c in contest_records:
        rows.append((
            c.student_id,
            c.platform,         # "CF"
            c.contest_id,
            c.name,
            c.date.strftime("%Y-%m-%d %H:%M:%S"),
            c.rank,
            c.old_rating,
            c.new_rating,
            c.rating_change,
            c.performance if c.performance is not None else 0,
        ))

    with conn.cursor() as cur:
        cur.executemany(sql, rows)
    return len(rows)


def upsert_daily_stats(conn, daily_stats):
    if not daily_stats:
        return 0

    sql = """
        INSERT INTO daily_training_stats (
            student_id,
            stat_date,
            cf_new_total,
            cf_new_800,
            cf_new_900,
            cf_new_1000,
            cf_new_1100,
            cf_new_1200,
            cf_new_1300,
            cf_new_1400,
            cf_new_1500,
            cf_new_1600,
            cf_new_1700,
            cf_new_1800,
            cf_new_1900,
            cf_new_2000,
            cf_new_2100,
            cf_new_2200,
            cf_new_2300,
            cf_new_2400,
            cf_new_2500,
            cf_new_2600,
            cf_new_2700,
            cf_new_2800_plus,
            ac_new_total,
            ac_new_0_399,
            ac_new_400_799,
            ac_new_800_1199,
            ac_new_1200_1599,
            ac_new_1600_1999,
            ac_new_2000_2399,
            ac_new_2400_2799,
            ac_new_2800_plus
        ) VALUES (
            %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s,
            %s, %s, %s, %s,
            0, 0, 0, 0, 0, 0, 0, 0, 0
        )
        ON DUPLICATE KEY UPDATE
            cf_new_total = VALUES(cf_new_total),
            cf_new_800 = VALUES(cf_new_800),
            cf_new_900 = VALUES(cf_new_900),
            cf_new_1000 = VALUES(cf_new_1000),
            cf_new_1100 = VALUES(cf_new_1100),
            cf_new_1200 = VALUES(cf_new_1200),
            cf_new_1300 = VALUES(cf_new_1300),
            cf_new_1400 = VALUES(cf_new_1400),
            cf_new_1500 = VALUES(cf_new_1500),
            cf_new_1600 = VALUES(cf_new_1600),
            cf_new_1700 = VALUES(cf_new_1700),
            cf_new_1800 = VALUES(cf_new_1800),
            cf_new_1900 = VALUES(cf_new_1900),
            cf_new_2000 = VALUES(cf_new_2000),
            cf_new_2100 = VALUES(cf_new_2100),
            cf_new_2200 = VALUES(cf_new_2200),
            cf_new_2300 = VALUES(cf_new_2300),
            cf_new_2400 = VALUES(cf_new_2400),
            cf_new_2500 = VALUES(cf_new_2500),
            cf_new_2600 = VALUES(cf_new_2600),
            cf_new_2700 = VALUES(cf_new_2700),
            cf_new_2800_plus = VALUES(cf_new_2800_plus),
            deleted_at = NULL
    """

    rows = [build_daily_row(s) for s in daily_stats]

    with conn.cursor() as cur:
        cur.executemany(sql, rows)
    return len(rows)


def fetch_one_user(student_id: str, cf_handle: str, from_dt: datetime, to_dt: datetime):
    from_ts = int(from_dt.timestamp())
    to_ts = int(to_dt.timestamp())

    rating_hist = get_cf_rating_history(cf_handle)
    contests = build_cf_contest_records(student_id, rating_hist, from_ts, to_ts)

    subs = get_cf_submissions_in_range(cf_handle, from_ts, to_ts)
    daily = build_cf_daily_stats(student_id, subs)

    return contests, daily


def main():
    args = parse_args()

    from_dt = datetime.strptime(args.from_date, "%Y-%m-%d")
    to_dt = datetime.strptime(args.to_date, "%Y-%m-%d").replace(hour=23, minute=59, second=59)

    conn = get_conn(args.config)
    users = get_users(conn, only_student_id=args.only, limit=args.limit)

    total_users = len(users)
    ok_users = 0
    failed_users = 0
    total_contests = 0
    total_daily = 0

    print(f"[INFO] users to process: {total_users}")

    for idx, u in enumerate(users, start=1):
        student_id = u["id"]
        name = u["name"]
        cf_handle = (u["cf_handle"] or "").strip()

        print(f"[INFO] ({idx}/{total_users}) student_id={student_id} name={name} cf={cf_handle}")

        try:
            contests, daily = fetch_one_user(student_id, cf_handle, from_dt, to_dt)

            upsert_c = upsert_contests(conn, contests)
            upsert_d = upsert_daily_stats(conn, daily)

            conn.commit()

            ok_users += 1
            total_contests += upsert_c
            total_daily += upsert_d

            print(
                f"[OK] student_id={student_id} contests={upsert_c} daily={upsert_d}"
            )
        except Exception as e:
            conn.rollback()
            failed_users += 1
            print(f"[ERR] student_id={student_id} cf={cf_handle} error={e}", file=sys.stderr)

        time.sleep(args.sleep)

    conn.close()

    print(
        f"[DONE] total_users={total_users} ok={ok_users} failed={failed_users} "
        f"contest_rows={total_contests} daily_rows={total_daily}"
    )


if __name__ == "__main__":
    main()
