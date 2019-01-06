#!/bin/bash

export URLS='a b c d e f g h i j k l m n o p'

check () {
  for URL in ${URLS}; do
    if [ $(curl -m 3 -sf -o /dev/null -w "%{http_code}" "http://whoami-${URL}.localtest.me") -eq "200" ]; then
      curl -k -sf -o /dev/null -w "%{time_total}\t%{http_code}\n" https://whoami-${URL}.localtest.me/login?[1-1000] &
    else
      MISSING="${MISSING} ${URL}"
    fi
  done
  wait
}
echo "Performing checks ..."
time check
echo "*** ${MISSING} not available ***"
