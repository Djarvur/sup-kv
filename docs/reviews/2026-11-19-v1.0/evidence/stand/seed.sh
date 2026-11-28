#!/bin/sh
# a service seeding its build-time default as soon as the node accepts writes
i=0
while [ $i -lt 200 ]; do
  R=$(/opt/bin/grpcurl -plaintext -max-time 1 -d '{"key":"cfg/color","value":"ZGVmYXVsdA==","create_if_absent":true}' $1:7901 supkvapi.v1.SupKVService/Set 2>&1 | tr -d '\n ')
  case "$R" in *Unavailable*|*dial*|*refused*|*deadline*) ;; *) echo "attempt $i -> $R"; exit 0;; esac
  i=$((i+1))
done
echo "gave up"
