import argparse
import json
import sys
from datetime import datetime, timedelta, timezone

from service import fetch_range

UTC_PLUS_8 = timezone(timedelta(hours=8))


def parse_date(s: str) -> datetime:
    return datetime.strptime(s, "%Y-%m-%d")


def build_output(student_id, from_dt, to_dt, contests, daily):
    return {
        "student_id": student_id,
        "from": from_dt.strftime("%Y-%m-%d"),
        "to": to_dt.strftime("%Y-%m-%d"),
        "contest_records": [
            {
                "student_id": c.student_id,
                "platform": c.platform,
                "contest_id": c.contest_id,
                "name": c.name,
                "date": c.date.replace(tzinfo=UTC_PLUS_8).isoformat(),
                "rank": c.rank,
                "old_rating": c.old_rating,
                "new_rating": c.new_rating,
                "rating_change": c.rating_change,
                "performance": c.performance,
            }
            for c in contests
        ],
        "daily_stats": [
            {
                "student_id": d.student_id,
                "date": d.date.replace(tzinfo=UTC_PLUS_8).isoformat(),
                "cf_new_total": d.cf_new_total,
                "cf_new_undefined": d.cf_new_undefined,
                "cf_new": d.cf_new,
                "ac_new_total": d.ac_new_total,
                "ac_new_undefined": d.ac_new_undefined,
                "ac_new_range": d.ac_new_range,
            }
            for d in daily
        ],
    }


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--student", required=True)
    ap.add_argument("--cf", default="")
    ap.add_argument("--ac", default="")
    ap.add_argument("--from", dest="from_date", required=True)
    ap.add_argument("--to", dest="to_date", required=True)
    args = ap.parse_args()

    try:
        from_dt = parse_date(args.from_date)
        to_dt = parse_date(args.to_date).replace(
            hour=23, minute=59, second=59
        )

        contests, daily = fetch_range(
            student_id=args.student,
            cf_handle=args.cf or None,
            ac_handle=args.ac or None,
            from_dt=from_dt,
            to_dt=to_dt,
        )

        result = build_output(
            args.student, from_dt, to_dt, contests, daily
        )

        print(json.dumps(result, ensure_ascii=False))
    except Exception as e:
        print(str(e), file=sys.stderr)
        sys.exit(1)


if __name__ == "__main__":
    main()
