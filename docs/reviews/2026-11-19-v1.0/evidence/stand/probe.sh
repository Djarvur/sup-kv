#!/bin/sh
# probe.sh <host> <n> : n rapid Set+Health probes with ms timestamps (runs inside the client container)
H=$1; N=$2
i=0
while [ $i -lt $N ]; do
  T=$(date +%s.%N | cut -c1-14)
  S=$(/opt/bin/grpcurl -plaintext -max-time 1 -d '{"key":"gate/probe","value":"eA=="}' $H:7901 supkvapi.v1.SupKVService/Set 2>&1 | tr '\n' ' ' | sed 's/"version": "[^"]*"/OK/' | cut -c1-90)
  HS=$(/opt/bin/grpcurl -plaintext -max-time 1 -d '{}' $H:7901 grpc.health.v1.Health/Check 2>&1 | tr -d '\n {}"' | cut -c1-50)
  G=$(/opt/bin/grpcurl -plaintext -max-time 1 -d '{"key":"demo/hello"}' $H:7901 supkvapi.v1.SupKVService/Get 2>&1 | tr -d '\n ' | cut -c1-28)
  echo "$T set=[$S] health=[$HS] get=[$G]"
  i=$((i+1))
done
