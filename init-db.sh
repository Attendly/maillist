#!/bin/bash

DB=attendly_email_service
USERNAME=tt
PASSWORD=tt

mysql -u$USERNAME -p$PASSWORD -e "drop database if exists $DB; create database $DB;"
mysql -u$USERNAME -p$PASSWORD $DB <attendly_email_service.sql
