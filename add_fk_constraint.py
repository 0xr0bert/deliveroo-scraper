#!/usr/bin/env python
import sqlite3

add_fk_cmds = open("sqlite/add_fk_constraint.sql", "r").read()

conn = sqlite3.connect("deliveroo.db")
cur = conn.cursor()
cur.executescript(add_fk_cmds)

conn.commit()
cur.close()
conn.close()
