#!/usr/bin/env bash
set -euo pipefail

repo_root="$(git rev-parse --show-toplevel)"
cd "$repo_root"

blocked_name="$(printf '\345\221\250\345\244\247\346\267\273')"

if git grep -n --cached -- "$blocked_name"; then
  echo
  echo "Blocked: staged changes still contain the old teacher name. Replace it with 张老师 before committing." >&2
  exit 1
fi

