from dataclasses import dataclass
from datetime import datetime
from typing import Dict


@dataclass
class ContestRecord:
    student_id: str
    platform: str  # "CF" / "AC"
    contest_id: str
    name: str
    date: datetime
    rank: int
    old_rating: int
    new_rating: int
    rating_change: int
    performance: int


@dataclass
class DailyTrainingStats:
    student_id: str
    date: datetime  # normalized to 00:00:00

    # CF
    cf_new_total: int
    cf_new: Dict[int, int]  # rating -> count

    # AC
    ac_new_total: int
    ac_new_range: Dict[str, int]  # "0-399" etc