#!/bin/bash

if [ -z $MAILLIST_DB_NAME ]; then
	echo "MAILLIST_DB_NAME must be set to the desired database name"
	exit 1
fi

if [ -z $MAILLIST_DB_USER ]; then
	echo "MAILLIST_DB_USER must be set to the desired database username"
	exit 1
fi

DB=$MAILLIST_DB_NAME
USER=$MAILLIST_DB_USER
PW=$MAILLIST_DB_PW

mysql -u$USER -p$PW -e "drop database if exists $DB; create database $DB;"
mysql -u$USER -p$PW $DB <attendly_email_service.sql

db-migrate
