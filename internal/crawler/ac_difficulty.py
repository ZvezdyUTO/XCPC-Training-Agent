import requests

_DIFFICULTY_CACHE = None


def load_ac_difficulty() -> dict:
    """
    https://kenkoooo.com/atcoder/resources/problem-models.json
    返回 {problem_id: {"difficulty": ...}, ...}
    """
    global _DIFFICULTY_CACHE
    if _DIFFICULTY_CACHE is not None:
        return _DIFFICULTY_CACHE

    try:
        r = requests.get(
            "https://kenkoooo.com/atcoder/resources/problem-models.json",
            timeout=20,
        )
        r.raise_for_status()
        _DIFFICULTY_CACHE = r.json()
    except Exception:
        _DIFFICULTY_CACHE = {}

    return _DIFFICULTY_CACHE