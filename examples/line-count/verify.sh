#!/bin/sh
set -eu
sample=$(mktemp)
trap 'rm -f "$sample"' EXIT
printf 'one\ntwo\nthree\n' > "$sample"
test "$(sh "$RIVET_SCRIPT" "$sample")" = 3
