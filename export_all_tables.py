#!/usr/bin/env python
import sqlite3
import subprocess
from pathlib import Path

SQLITE_DB_LOC = "deliveroo.db"
OUTPUT_DIR = "output"

Path(OUTPUT_DIR).mkdir(parents=True, exist_ok=True)

conn = sqlite3.connect(SQLITE_DB_LOC)
cur = conn.cursor()

cur.execute(
    """
    SELECT tbl_name FROM sqlite_master
    WHERE type='table' and tbl_name not like 'sqlite_%'
    """
)

while row := cur.fetchone():
    table_name = row[0]
    sqlite_input = "\n".join([
        ".headers on",
        ".mode csv",
        f".output {OUTPUT_DIR}/{table_name}.csv",
        f"SELECT * FROM {table_name}"
    ]).encode("UTF-8")

    subprocess.run(["sqlite3", SQLITE_DB_LOC], input=sqlite_input)

cur.close()
conn.close()
