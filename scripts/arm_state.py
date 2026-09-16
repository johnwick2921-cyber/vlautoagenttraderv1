"""Canonical arm-state queries for read-only watches and audit scripts.

The predicate is emitted by Go; Python owns no state list. Import this module
from the repository's scripts directory. It never opens the live database.
"""
from functools import lru_cache
from contextlib import closing
from pathlib import Path
import sqlite3
import subprocess


@lru_cache(maxsize=1)
def terminal_arm_sql():
    root = Path(__file__).resolve().parents[1]
    return subprocess.check_output(
        ["go", "run", "./cmd/arm-state-sql", "-terminal"], cwd=root, text=True
    ).strip()


def nonterminal_arm_sql():
    return "NOT (" + terminal_arm_sql() + ")"


@lru_cache(maxsize=256)
def is_terminal_arm_state(state):
    with closing(sqlite3.connect(":memory:")) as db:
        db.execute("PRAGMA query_only=ON")
        return bool(db.execute(
            "SELECT " + terminal_arm_sql() + " FROM (SELECT ? AS state)", (state,)
        ).fetchone()[0])
