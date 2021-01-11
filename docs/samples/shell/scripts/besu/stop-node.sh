#!/bin/sh
ps -eaf|grep "Dbesu" |grep -v grep |grep $1|tr -s " "|cut -f3 -d" " | xargs kill