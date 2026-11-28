#!/bin/sh
# usage: g.sh <a|b> <Method> '<json>'   (runs grpcurl in the client container)
case "$1" in a) H=172.31.77.11;; b) H=172.31.77.12;; *) H=$1;; esac
M=$2; shift 2
case "$M" in Health) M=grpc.health.v1.Health/Check;; *) M=supkvapi.v1.SupKVService/$M;; esac
docker exec supkv-lab-client-1 /opt/bin/grpcurl -plaintext -d "${1:-{\}}" $H:7901 $M 2>&1
