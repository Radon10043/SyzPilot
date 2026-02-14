#!/bin/bash

# this script is used to find logs related to a specific crash from a directory
# of syzkaller crash logs
#
# usage:
#   bash scripts/logs.sh -d <CRASH_DIR> -i <INTEREST_CRASH>

set -euo pipefail

# required args
CRASH_DIR=
INTEREST_CRASH=

print_help() {
    echo "usage: $0 [ARGS]"
    echo "  args (required):"
    echo "    -d,--dir <DIR>            path to the directory containing syzkaller crash logs"
    echo "    -i,--interest <INTEREST>  path to the crash file that contains the titles of the crashes we are interested in, one title per line"
    echo "  args (optional):"
    echo "    -h, --help                print help message"
}

while [[ $# -gt 0 ]]; do
    case $1 in
    -d | --dir)
        CRASH_DIR=$(realpath $2)
        shift
        shift
        ;;
    -i | --interest)
        INTEREST_CRASH=$(realpath $2)
        shift
        shift
        ;;
    -h | --help)
        print_help
        exit 0
        ;;
    *)
        echo "unknown arg: $1"
        exit 1
        ;;
    esac
done

logs=()
while IFS= read -r title; do
    desc=$(find $CRASH_DIR -name description -exec grep -l "$title" {} +)
    new_logs=$(find $(dirname $desc) -name "log*")
    logs+=("${new_logs[@]}")
done < "$INTEREST_CRASH"

printf "%s\n" "${logs[@]}" | sort
