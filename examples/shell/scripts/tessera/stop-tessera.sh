#!/bin/bash
ps -eaf|grep tessera|grep -v attach|grep -v grep |grep tessera-config-09-$1.json|tr -s " " " "|cut -d" " -f3 | xargs kill
