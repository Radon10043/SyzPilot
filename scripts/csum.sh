#!/bin/bash

# this script is used to summarize the titles of the crashes from a directory
# of syzkaller crash logs. A csv file will be generated with status and title.
# If the csv file already exists, the script will only add new crashes into
# the csv file.
#
# usage:
#   bash scripts/csum.sh -d <CRASH_DIR> -o <OUTPUT_CSV>

set -euo pipefail

# required args
CRASH_DIR=
OUTPUT_CSV=

print_help() {
    echo "usage: $0 [ARGS]"
    echo "  args (required):"
    echo "    -d, --dir <DIR>        path to the directory containing syzkaller crash logs"
    echo "    -o, --output <OUTPUT>  path to the output csv file"
    echo "  args (optional):"
    echo "    -h, --help             print help message"
}

# Argument parsing
while [[ $# -gt 0 ]]; do
    case $1 in
    -d | --dir)
        # check if dir exists immediately
        if [[ ! -d "$2" ]]; then
            echo "Error: Directory '$2' does not exist."
            exit 1
        fi
        CRASH_DIR=$(realpath "$2")
        shift 2
        ;;
    -o | --output)
        # use -m for realpath to allow non-existent files, or simply use the path as provided
        # if realpath -m is not supported (e.g. on macOS), just use the raw path or $(readlink -f $2)
        if command -v realpath >/dev/null 2>&1; then
            OUTPUT_CSV=$(realpath -m "$2")
        else
            OUTPUT_CSV="$2"
        fi
        shift 2
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

# validation
if [[ -z "$CRASH_DIR" ]] || [[ -z "$OUTPUT_CSV" ]]; then
    echo "Error: Missing required arguments."
    print_help
    exit 1
fi

# read existing CSV file if exists
declare -A EXISTING_TITLES
if [ -f "$OUTPUT_CSV" ]; then
    # IFS=, splits by comma.
    while IFS=, read -r status title; do
        # remove surrounding quotes from the title read from CSV
        # "${title%\"}" removes trailing quote, "${...#\"}" removes leading quote
        clean_title="${title%\"}"
        clean_title="${clean_title#\"}"
        EXISTING_TITLES["$clean_title"]=$status
    done < <(tail -n +2 "$OUTPUT_CSV") # skip header
fi

# Find all crashes and summarize titles
TEMP_CSV="${OUTPUT_CSV}.tmp"

{
    echo "status,title"
    find "$CRASH_DIR" -name description | while read -r desc; do
        # Extract title from the first line of description file
        if [[ -f "$desc" ]]; then
            title=$(head -n 1 "$desc")
            if [[ -n "${EXISTING_TITLES[$title]:-}" ]]; then
                status=${EXISTING_TITLES[$title]}
            else
                status="unknown"
            fi
            echo "$status,\"$title\""
        fi
    done | sort
} > "$TEMP_CSV"

mv "$TEMP_CSV" "$OUTPUT_CSV"
echo "Done. Summary saved to $OUTPUT_CSV"
