#!/bin/bash

set -e
# A simple performance "smoke test" for femebe using simpleproxy
#
# This simply:
#  1. Starts (and times) a simpleproxy
#  2. Runs a pgbench test
#  3. Repeates designated number of times
#  4. Munges the numbers

if [ ! -f perf-smoke.config ]
then
    cp perf-smoke.config.sample perf-smoke.config
fi

source perf-smoke.config

$PROXY_BUILD

for iter in $(seq $ITERATIONS)
do
    env time -v --quiet -o run-${iter}.time ./${PROXY_CMD} $PROXY_ARGS &
    # Give the pipeline time to kick everything off and
    # the proxy time to start listening
    sleep 1
    # A little ugly: we want to stop the proxy, but because it's
    # kicked off by go run, which is kicked off by time, which is
    # kicked off by env, we don't have a terribly principled way of
    # asking it to exit. Instead we find its pid (we assume it's the
    # only one running) and send it a SIGINT
    proxy_pid="$(pgrep $PROXY_CMD)"
    $PGB_CMD
    # This would normally be a SIGTERM, but this seems busted:
    # I can't get even a toy program to respond to that signal.
    # SIGINT works fine.
    kill -INT $proxy_pid
done

cp run-1.time run-summary.time

for iter in $(seq 2 $ITERATIONS)
do
    cp run-summary.time tmp
    paste tmp <(sed -e '1s/.*//'  -e's/.*://' run-${iter}.time) > run-summary.time
done
