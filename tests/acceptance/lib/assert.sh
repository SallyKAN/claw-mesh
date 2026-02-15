#!/usr/bin/env bash
# assert.sh — helper functions for acceptance tests.

set -euo pipefail

# Colors (disabled if not a terminal).
if [[ -t 1 ]]; then
  GREEN='\033[0;32m'
  RED='\033[0;31m'
  RESET='\033[0m'
else
  GREEN='' RED='' RESET=''
fi

_pass() { echo -e "${GREEN}PASS${RESET}: $1"; }
_fail() { echo -e "${RED}FAIL${RESET}: $1"; exit 1; }

# assert_contains FILE PATTERN
#   Grep for PATTERN in FILE. Fails if not found.
assert_contains() {
  local file="$1" pattern="$2"
  if grep -qE "$pattern" "$file" 2>/dev/null; then
    _pass "file $(basename "$file") contains '$pattern'"
  else
    echo "--- file contents ---"
    cat "$file" 2>/dev/null || echo "(file not found)"
    echo "--- end ---"
    _fail "file $(basename "$file") does not contain '$pattern'"
  fi
}

# assert_contains_str STRING PATTERN
#   Check that STRING matches PATTERN (extended regex).
assert_contains_str() {
  local str="$1" pattern="$2"
  if echo "$str" | grep -qE "$pattern"; then
    _pass "string contains '$pattern'"
  else
    echo "--- string value ---"
    echo "$str"
    echo "--- end ---"
    _fail "string does not contain '$pattern'"
  fi
}

# assert_exit_code CMD EXPECTED_CODE
#   Run CMD and check its exit code matches EXPECTED_CODE.
assert_exit_code() {
  local cmd="$1" expected="$2"
  local actual=0
  eval "$cmd" >/dev/null 2>&1 || actual=$?
  if [[ "$actual" -eq "$expected" ]]; then
    _pass "exit code $actual == $expected for: $cmd"
  else
    _fail "exit code $actual != $expected for: $cmd"
  fi
}

# wait_for_log FILE PATTERN TIMEOUT_SECONDS
#   Poll FILE until PATTERN appears or timeout expires.
#   Returns 0 on match, 1 on timeout.
wait_for_log() {
  local file="$1" pattern="$2" timeout="${3:-10}"
  local deadline=$((SECONDS + timeout))
  while [[ $SECONDS -lt $deadline ]]; do
    if grep -qE "$pattern" "$file" 2>/dev/null; then
      return 0
    fi
    sleep 0.2
  done
  echo "TIMEOUT waiting for '$pattern' in $(basename "$file") (${timeout}s)"
  echo "--- file contents ---"
  cat "$file" 2>/dev/null || echo "(file not found)"
  echo "--- end ---"
  return 1
}
