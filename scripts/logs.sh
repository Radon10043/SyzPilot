#!/bin/bash

# this script is used to find logs related to a specific crash from a directory
# of syzkaller crash logs
#
# usage:
#   bash scripts/logs.sh <CRASH_PATH> <INTEREST_CRASH_FILE>

CRASH_PATH=$(realpath $1)
INTEREST_CRASH=$(realpath $2)

logs=()
while IFS= read -r title; do
    desc=$(find $CRASH_PATH -name description -exec grep -l "$title" {} +)
    new_logs=$(find $(dirname $desc) -name "log*")
    logs+=("${new_logs[@]}")
done < "$INTEREST_CRASH"

printf "%s\n" "${logs[@]}" | sort