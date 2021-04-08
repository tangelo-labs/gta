#!/usr/bin/env bash
########################################################################
# Description: Detect regressions by comparing two gta binaries; an old
#              version and a new version. To use this script, compile
#              the old version of gta and the new version of gta.
#
#              This script will compare the results and timing of the
#              two gta binaries against the last 100 merges in the repo
#              in the current directory.
########################################################################
if [[ -z "$1" || -z "$2" ]]; then
  printf "usage: ${BASH_SOURCE[0]} OLD_GTA NEW_GTA\n" >&2
  exit 1
fi

set -u -o pipefail

gta_old_bin="$1"
gta_new_bin="$2"
count=100

if [[ "$(uname)" == "Darwin" ]];
then
  time=gtime
fi

branch=$(git branch --show-current)

function restore_branch() {
  git checkout "${branch}"
}

trap restore_branch EXIT

for commit in $(git log --first-parent --merges --pretty=format:"%H" | head -n "$count")
do
    printf "Merge commit: $commit" >&2
    results_dir="results/$commit"
    mkdir -p "$results_dir"

    git checkout "$commit^2" >/dev/null 2>&1

    changed_files="$results_dir/changed.txt"
    git diff --name-only --no-renames -r "$commit"^1..."$commit" | sed "s|^|$(git rev-parse --show-toplevel)/|" >"$changed_files"
    files_changed=$(wc -l "$changed_files" | tr -s ' ' | cut -d ' ' -f 2)
    printf " [# files changed: $files_changed]" >&2

    gta_old_result="$results_dir/gta.txt"
    export GO111MODULE=off
    gta_old_time=$(/usr/bin/env "$time" -f '%es' "$gta_old_bin" -changed-files "$changed_files" -include do/doge/,do/exp/,do/services/,do/teams/,do/tools/ -tags integration 2>&1 >"$gta_old_result")
    gta_new_result="$results_dir/gta2.txt"
    export GO111MODULE=on
    gta_new_time=$(/usr/bin/env "$time" -f '%es' "$gta_new_bin" -changed-files "$changed_files" -include do/doge/,do/exp/,do/services/,do/teams/,do/tools/ -tags integration 2>&1 >"$gta_new_result")
    printf " [old: ${gta_old_time}]" >&2
    printf " [new: ${gta_new_time}]\n" >&2
    diff "$gta_old_result" "$gta_new_result" | sed "s/^/  /"
done
