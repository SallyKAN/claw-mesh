#!/usr/bin/env bash
# test_health_check.sh — P0-4: Active health check probing.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"
source "$SCRIPT_DIR/lib/assert.sh"

TMPDIR_TEST="$(mktemp -d)"
BINARY="$TMPDIR_TEST/claw-mesh"
COORD_LOG="$TMPDIR_TEST/coordinator.log"
NODE_LOG="$TMPDIR_TEST/node.log"
COORD_PID="" NODE_PID=""
TOKEN="health-test-$(date +%s)"

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

echo "=== P0-4: Active Health Check Test ==="

# --- Build ---
echo "--- Step 1: Build ---"
(cd "$ROOT_DIR" && go build -o "$BINARY" ./cmd/claw-mesh)
_pass "binary built"

# --- Start coordinator ---
echo "--- Step 2: Start coordinator ---"
"$BINARY" up --port "$COORD_PORT" --token "$TOKEN" --allow-private \
  --data-dir "$TMPDIR_TEST/data" >"$COORD_LOG" 2>&1 &
COORD_PID=$!
if ! wait_for_log "$COORD_LOG" "coordinator listening" 10; then
  _fail "coordinator did not start"
fi
_pass "coordinator started"

COORD_URL="http://127.0.0.1:${COORD_PORT}"

# --- Join node ---
echo "--- Step 3: Join node ---"
"$BINARY" join "$COORD_URL" \
  --name "health-node" \
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

# --- Verify /healthz endpoint ---
echo "--- Step 4: Verify /healthz endpoint ---"
HEALTHZ_OUT=$(curl -s -o /dev/null -w "%{http_code}" "http://127.0.0.1:${NODE_PORT}/healthz")
if [[ "$HEALTHZ_OUT" == "200" ]]; then
  _pass "/healthz returns 200"
else
  _fail "/healthz returned $HEALTHZ_OUT (expected 200)"
fi

# --- Kill node (simulate crash) ---
echo "--- Step 5: Kill node (SIGKILL, no deregister) ---"
kill -9 "$NODE_PID" 2>/dev/null || true
wait "$NODE_PID" 2>/dev/null || true
NODE_PID=""
_pass "node killed"

# --- Wait for coordinator to detect node offline ---
echo "--- Step 6: Wait for node to be marked offline ---"
# Health checker runs every 10s, 2 probe failures = ~20-30s
MAX_WAIT=60
ELAPSED=0
while [[ $ELAPSED -lt $MAX_WAIT ]]; do
  NODES_OUT=$("$BINARY" nodes --coordinator "$COORD_URL" --token "$TOKEN" 2>&1 || true)
  if echo "$NODES_OUT" | grep -q "offline"; then
    break
  fi
  sleep 3
  ELAPSED=$((ELAPSED + 3))
done

NODES_OUT=$("$BINARY" nodes --coordinator "$COORD_URL" --token "$TOKEN" 2>&1 || true)
echo "$NODES_OUT"
assert_contains_str "$NODES_OUT" "offline"
_pass "node marked offline after crash"

# --- Verify messages not routed to offline node ---
echo "--- Step 7: Verify no routing to offline node ---"
SEND_OUT=$("$BINARY" send "should fail" --auto \
  --coordinator "$COORD_URL" --token "$TOKEN" 2>&1 || true)
echo "$SEND_OUT"
if echo "$SEND_OUT" | grep -q "no online nodes\|503\|no available"; then
  _pass "message correctly rejected (no online nodes)"
else
  _fail "message should not route to offline node"
fi

echo ""
echo "=== ALL HEALTH CHECK TESTS PASSED ==="
exit 0
