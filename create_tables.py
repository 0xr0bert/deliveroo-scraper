#!/usr/bin/env python
import sqlite3

create_table_cmds = open("sqlite/create_tables.sql", "r").read()

conn = sqlite3.connect("deliveroo.db")
cur = conn.cursor()
cur.executescript(create_table_cmds)

postcodes = open("postcodes.txt", "r").readlines()
postcodes_insert_vals = list((x.strip(),) for x in postcodes)

cur.executemany(
    "INSERT INTO customers (postcode) VALUES (?)",
    postcodes_insert_vals
)

conn.commit()
cur.close()
conn.close()
