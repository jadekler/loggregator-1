#!/usr/bin/env bash
set -ex

for i in `seq 1 $NUM_APPS`; do
    rm -f output-$i.txt
    cf logs drainspinner-$i > output-$i.txt 2>&1 &
done;

sleep 30 #wait 30 seconds for socket connection

echo "Begin the hammer"
for i in `seq 1 $NUM_APPS`; do
    domain=$(cf app drainspinner-$i | grep urls | awk '{print $2}')
    curl "$domain?cycles=${CYCLES}&delay=${DELAY_US}us" &> /dev/null
done;

sleep 25
