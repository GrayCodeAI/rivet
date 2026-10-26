#!/bin/sh
set -eu
wc -l < "$1" | tr -d ' '
