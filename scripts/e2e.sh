#!/bin/bash

set -ux
IFS=$'\n\t'

TEST="$1"
OUT_PATH="/tmp/cni-e2e"
LOG_PATH="$OUT_PATH/plugins/e2e/results"

mkdir -p "$LOG_PATH"

ginkgo run -v "$TEST" | tee "$LOG_PATH/e2e.log"
STATUS=$?

tar -C "$OUT_PATH" -cvzf "$OUT_PATH/results.tar.gz" plugins/

exit $STATUS