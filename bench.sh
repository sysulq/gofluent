#!/usr/bin/env bash
i="0"

while [ $i -lt 100000 ]
do
  echo '{"data":"mydata","test":"mytest"}' >> /usr/local/ysec_agent/log/exec_log.json 
  i=$[$i+1]
done
