#!/bin/bash
ps -eaf|grep geth|grep -v attach|grep -v grep |grep $1|tr -s " " " "|cut -d" " -f3 | xargs kill
