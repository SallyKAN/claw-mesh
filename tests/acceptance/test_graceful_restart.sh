#!/usr/bin/env bash
# test_graceful_restart.sh — P0-3: Graceful restart + node auto-reconnect.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"
source "$SCRIPT_DIR/lib/assert.sh"

TMPDIR_TEST="$(mktemp -d)"
BINARY="$TMPDIR_TEST/claw-mesh"
COORD_LOG="$TMPDIR_TEST/coordinator.log"
COORD_LOG2="$TMPDIR_TEST/coordinator2.log"
NODE_LOG="$TMPDIR_TEST/node.log"
COORD_PID="" NODE_PID=""
TOKEN="restart-test-$(date +%s)"

pick_port() {
  python3 -c "import socket; s=socket.socket(); s.bind(('127.0.0.1',0)); print(s.getsockname()[1]); s.close()"
}

COORD_PORT="$(pick_port)"
NODE_PORT="$(pick_port)"

cleanup() {
  for pid in "$COORD_PID" "$NODE_PID"; do
    if [[ -n "$pid" ]] && kill -0 "$pid" 2>/dev/null; then
      kill "$pid" 2>/dev/null || true
      wait "$pid" 2>/dev/null || true
    fi
  done
  rm -rf "$TMPDIR_TEST"
}
trap cleanup EXIT

echo "=== P0-3: Graceful Restart + Auto-Reconnect Test ==="
echo "  coordinator port: $COORD_PORT"
echo "  node port: $NODE_PORT"
echo ""

# --- Step 1: Build ---
echo "--- Step 1: Build ---"
(cd "$ROOT_DIR" && go build -o "$BINARY" ./cmd/claw-mesh)
_pass "binary built"

# --- Step 2: Start coordinator ---
echo "--- Step 2: Start coordinator ---"
"$BINARY" up --port "$COORD_PORT" --token "$TOKEN" --allow-private \
  --data-dir "$TMPDIR_TEST/data" >"$COORD_LOG" 2>&1 &
COORD_PID=$!
if ! wait_for_log "$COORD_LOG" "coordinator listening" 10; then
  _fail "coordinator did not start"
fi
_pass "coordinator started"

COORD_URL="http://127.0.0.1:${COORD_PORT}"

# --- Step 3: Join node ---
echo "--- Step 3: Join node ---"
"$BINARY" join "$COORD_URL" \
  --name "reconnect-node" \
  --token "$TOKEN" \
  --listen "127.0.0.1:${NODE_PORT}" \
  --no-gateway \
  >"$NODE_LOG" 2>&1 &
NODE_PID=$!
if ! wait_for_log "$NODE_LOG" "registered as node" 10; then
  _fail "node did not register"
fi
_pass "node registered"
sleep 1

# --- Step 4: Graceful shutdown coordinator ---
echo "--- Step 4: SIGTERM coordinator ---"
kill "$COORD_PID" 2>/dev/null || true
wait "$COORD_PID" 2>/dev/null || true
COORD_PID=""
_pass "coordinator stopped gracefully"

# --- Step 5: Restart coordinator on SAME port ---
echo "--- Step 5: Restart coordinator ---"
"$BINARY" up --port "$COORD_PORT" --token "$TOKEN" --allow-private \
  --data-dir "$TMPDIR_TEST/data" >"$COORD_LOG2" 2>&1 &
COORD_PID=$!
if ! wait_for_log "$COORD_LOG2" "coordinator listening" 10; then
  _fail "coordinator did not restart"
fi
_pass "coordinator restarted on same port"

# --- Step 6: Wait for node to auto-reconnect ---
echo "--- Step 6: Wait for node auto-reconnect ---"
# Node heartbeat interval is 10s, 3 failures = 30s, then reconnect.
# But coordinator was down briefly, so first heartbeat after restart should fail,
# triggering reconnect within ~30-40s.
if ! wait_for_log "$NODE_LOG" "re-registered as node" 90; then
  echo "--- node log ---"
  cat "$NODE_LOG"
  echo "--- coordinator2 log ---"
  cat "$COORD_LOG2"
  _fail "node did not auto-reconnect"
fi
_pass "node auto-reconnected"

# --- Step 7: Verify node is online ---
echo "--- Step 7: Verify node online after reconnect ---"
sleep 2
NODES_OUT=$("$BINARY" nodes --coordinator "$COORD_URL" --token "$TOKEN" 2>&1 || true)
echo "$NODES_OUT"
assert_contains_str "$NODES_OUT" "reconnect-node"
assert_contains_str "$NODES_OUT" "online"
_pass "node visible and online"

# --- Step 8: Send message to verify full functionality ---
echo "--- Step 8: Send message after reconnect ---"
SEND_OUT=$("$BINARY" send "post-reconnect hello" \
  --auto --coordinator "$COORD_URL" --token "$TOKEN" 2>&1 || true)
echo "$SEND_OUT"
assert_contains_str "$SEND_OUT" "routed to node"
assert_contains_str "$SEND_OUT" "post-reconnect hello"
_pass "message routing works after reconnect"

echo ""
echo "=== ALL GRACEFUL RESTART TESTS PASSED ==="
exit 0
