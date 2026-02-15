#!/usr/bin/env bash
# test_e2e_single_node.sh — End-to-end acceptance test for single-node mesh.
#
# Validates: build → coordinator up → node join → send (--node & --auto) → cleanup.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"
source "$SCRIPT_DIR/lib/assert.sh"

# --- Temp dir & cleanup ---
TMPDIR_TEST="$(mktemp -d)"
COORD_LOG="$TMPDIR_TEST/coordinator.log"
NODE_LOG="$TMPDIR_TEST/node.log"
SEND_AUTO_OUT="$TMPDIR_TEST/send_auto.out"
SEND_NODE_OUT="$TMPDIR_TEST/send_node.out"
BINARY="$TMPDIR_TEST/claw-mesh"

COORD_PID="" NODE_PID=""

cleanup() {
  local pids=("$COORD_PID" "$NODE_PID")
  for pid in "${pids[@]}"; do
    if [[ -n "$pid" ]] && kill -0 "$pid" 2>/dev/null; then
      kill "$pid" 2>/dev/null || true
      wait "$pid" 2>/dev/null || true
    fi
  done
  rm -rf "$TMPDIR_TEST"
}
trap cleanup EXIT

# --- Pick random high ports ---
pick_port() {
  python3 -c "import socket; s=socket.socket(); s.bind(('127.0.0.1',0)); print(s.getsockname()[1]); s.close()"
}

COORD_PORT="$(pick_port)"
NODE_PORT="$(pick_port)"
TOKEN="test-token-$(date +%s)"

echo "=== E2E Single Node Test ==="
echo "  coordinator port: $COORD_PORT"
echo "  node handler port: $NODE_PORT"
echo "  token: $TOKEN"
echo ""

# --- Step 1: Build ---
echo "--- Step 1: Build binary ---"
(cd "$ROOT_DIR" && go build -o "$BINARY" ./cmd/claw-mesh)
echo "Binary built: $BINARY"
echo ""

# --- Step 2: Start coordinator ---
echo "--- Step 2: Start coordinator ---"
"$BINARY" up \
  --port "$COORD_PORT" \
  --token "$TOKEN" \
  --allow-private \
  --config /dev/null \
  >"$COORD_LOG" 2>&1 &
COORD_PID=$!

if ! wait_for_log "$COORD_LOG" "coordinator listening" 10; then
  _fail "coordinator did not start"
fi
_pass "coordinator started on port $COORD_PORT"
echo ""

COORD_URL="http://127.0.0.1:${COORD_PORT}"

# --- Step 3: Join a node ---
echo "--- Step 3: Join node ---"
"$BINARY" join "$COORD_URL" \
  --name "test-node" \
  --token "$TOKEN" \
  --listen "127.0.0.1:${NODE_PORT}" \
  --config /dev/null \
  >"$NODE_LOG" 2>&1 &
NODE_PID=$!

if ! wait_for_log "$NODE_LOG" "registered as node" 10; then
  echo "--- coordinator log ---"
  cat "$COORD_LOG"
  echo "--- node log ---"
  cat "$NODE_LOG"
  _fail "node did not register"
fi
_pass "node registered"

# Give heartbeat a moment to settle.
sleep 1

# Verify node appears in node list.
NODES_OUT="$("$BINARY" nodes --coordinator "$COORD_URL" --token "$TOKEN" --config /dev/null 2>&1)"
assert_contains_str "$NODES_OUT" "test-node"
assert_contains_str "$NODES_OUT" "online"
_pass "node visible in 'nodes' output"
echo ""

# --- Step 4: Send message with --auto ---
echo "--- Step 4: Send message (--auto) ---"
"$BINARY" send "hello from auto" \
  --auto \
  --coordinator "$COORD_URL" \
  --token "$TOKEN" \
  --config /dev/null \
  >"$SEND_AUTO_OUT" 2>&1 || true

cat "$SEND_AUTO_OUT"
assert_contains "$SEND_AUTO_OUT" "routed to node"
assert_contains "$SEND_AUTO_OUT" "Response:"
_pass "auto-route message succeeded"
echo ""

# --- Step 5: Send message with --node ---
echo "--- Step 5: Send message (--node test-node) ---"
"$BINARY" send "hello from node" \
  --node "test-node" \
  --coordinator "$COORD_URL" \
  --token "$TOKEN" \
  --config /dev/null \
  >"$SEND_NODE_OUT" 2>&1 || true

cat "$SEND_NODE_OUT"
assert_contains "$SEND_NODE_OUT" "routed to node"
assert_contains "$SEND_NODE_OUT" "Response:"
_pass "node-targeted message succeeded"
echo ""

# --- Step 6: Validate echo fallback content ---
echo "--- Step 6: Validate response content ---"
assert_contains "$SEND_AUTO_OUT" "hello from auto"
assert_contains "$SEND_NODE_OUT" "hello from node"
_pass "echo fallback returned message content"
echo ""

# --- Step 7: Status command ---
echo "--- Step 7: Status command ---"
STATUS_OUT="$("$BINARY" status --coordinator "$COORD_URL" --token "$TOKEN" --config /dev/null 2>&1)"
assert_contains_str "$STATUS_OUT" "1 online"
_pass "status shows 1 online node"
echo ""

echo "=== ALL TESTS PASSED ==="
exit 0
