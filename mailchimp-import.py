#!/usr/bin/python3

import csv
import sys
import io
import pymysql
import datetime

accountemail = "sendgrid@eventarc.com"
listname = 'imported-list ' + datetime.datetime.now().strftime('%Y-%m-%d %H:%M:%S')

conn = pymysql.connect(unix_socket='/run/mysqld/mysqld.sock', user='tt', passwd='tt', db='attendly_email_service')
cur = conn.cursor()

cur.execute("select id from account where email=%s", (accountemail))
accountid= cur.fetchone()[0]

cur.execute("insert into list (account_id, name, status) values (%s, %s, 'active')", (accountid, listname))
listid = cur.lastrowid

with io.TextIOWrapper(sys.stdin.buffer, encoding='latin_1') as csvfile:
    f = csv.reader(csvfile, delimiter=',')
    for row in f:
        email,firstname,lastname = row[0], row[1], row[2]
        f = cur.execute("select id from subscriber where email=%s and account_id=%s", (email, accountid))
        if f != 0:
            subscriberid = cur.fetchone()[0]
        else:
            cur.execute("insert into subscriber (account_id, first_name, last_name, email, status) values (%s, %s, %s, %s, %s)",
                    (accountid, firstname, lastname, email, 'active'))
            subscriberid = cur.lastrowid
        cur.execute("insert into list_subscriber (list_id, subscriber_id, status) values (%s, %s, %s)",
                (listid, subscriberid, 'active'))
        conn.commit()

cur.close()
conn.close()
