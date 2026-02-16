#!/usr/bin/env bash
# test_persistence.sh — Acceptance test for P0-2: routing rule persistence.
#
# Validates: add rule → restart coordinator → rule survives.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"
source "$SCRIPT_DIR/lib/assert.sh"

TMPDIR_TEST="$(mktemp -d)"
BINARY="$TMPDIR_TEST/claw-mesh"
COORD_LOG="$TMPDIR_TEST/coordinator.log"
COORD_PID=""
TOKEN="persist-test-$(date +%s)"

pick_port() {
  python3 -c "import socket; s=socket.socket(); s.bind(('127.0.0.1',0)); print(s.getsockname()[1]); s.close()"
}

COORD_PORT="$(pick_port)"

cleanup() {
  if [[ -n "$COORD_PID" ]] && kill -0 "$COORD_PID" 2>/dev/null; then
    kill "$COORD_PID" 2>/dev/null || true
    wait "$COORD_PID" 2>/dev/null || true
  fi
  rm -rf "$TMPDIR_TEST"
}
trap cleanup EXIT

echo "=== P0-2: Persistence Test ==="
echo "  coordinator port: $COORD_PORT"
echo ""

# --- Step 1: Build ---
echo "--- Step 1: Build binary ---"
(cd "$ROOT_DIR" && go build -o "$BINARY" ./cmd/claw-mesh)
echo "Binary built."

# --- Step 2: Start coordinator ---
echo "--- Step 2: Start coordinator ---"
"$BINARY" up --port "$COORD_PORT" --token "$TOKEN" --allow-private >"$COORD_LOG" 2>&1 &
COORD_PID=$!
if ! wait_for_log "$COORD_LOG" "coordinator listening" 10; then
  _fail "coordinator did not start"
fi
_pass "coordinator started"

COORD_URL="http://127.0.0.1:${COORD_PORT}"

# --- Step 3: Add a routing rule ---
echo "--- Step 3: Add routing rule ---"
ADD_OUT=$(curl -s -X POST "$COORD_URL/api/v1/rules" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"match":{"requires_os":"linux"},"target":"linux-box"}')
echo "$ADD_OUT"
RULE_ID=$(echo "$ADD_OUT" | python3 -c "import sys,json; print(json.load(sys.stdin)['id'])" 2>/dev/null || echo "")
if [[ -z "$RULE_ID" ]]; then
  _fail "could not extract rule ID"
fi
_pass "rule added: $RULE_ID"

# Verify rule exists
LIST_OUT=$(curl -s "$COORD_URL/api/v1/rules" -H "Authorization: Bearer $TOKEN")
assert_contains_str "$LIST_OUT" "linux-box"
_pass "rule visible in list"

# --- Step 4: Stop coordinator ---
echo "--- Step 4: Stop coordinator (SIGTERM) ---"
kill "$COORD_PID" 2>/dev/null || true
wait "$COORD_PID" 2>/dev/null || true
COORD_PID=""
sleep 1
_pass "coordinator stopped"

# --- Step 5: Restart coordinator ---
echo "--- Step 5: Restart coordinator ---"
COORD_PORT2="$(pick_port)"
COORD_LOG2="$TMPDIR_TEST/coordinator2.log"
"$BINARY" up --port "$COORD_PORT2" --token "$TOKEN" --allow-private >"$COORD_LOG2" 2>&1 &
COORD_PID=$!
if ! wait_for_log "$COORD_LOG2" "coordinator listening" 10; then
  _fail "coordinator did not restart"
fi
_pass "coordinator restarted on port $COORD_PORT2"

COORD_URL2="http://127.0.0.1:${COORD_PORT2}"

# --- Step 6: Verify rule persisted ---
echo "--- Step 6: Verify rule persisted ---"
LIST_OUT2=$(curl -s "$COORD_URL2/api/v1/rules" -H "Authorization: Bearer $TOKEN")
echo "$LIST_OUT2"
assert_contains_str "$LIST_OUT2" "linux-box"
assert_contains_str "$LIST_OUT2" "$RULE_ID"
_pass "rule survived restart"

# --- Step 7: Verify no nodes (nodes are ephemeral) ---
echo "--- Step 7: Verify no nodes after restart ---"
NODES_OUT=$("$BINARY" nodes --coordinator "$COORD_URL2" --token "$TOKEN" 2>&1 || true)
echo "$NODES_OUT"
# Should show no nodes or empty list
if echo "$NODES_OUT" | grep -q "online"; then
  _fail "nodes should not persist across restarts"
fi
_pass "no nodes after restart (correct)"

echo ""
echo "=== ALL PERSISTENCE TESTS PASSED ==="
exit 0
