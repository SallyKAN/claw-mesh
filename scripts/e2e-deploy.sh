#!/usr/bin/env bash
set -euo pipefail

# ============================================================
# claw-mesh e2e deploy & test script
# Run from Mac Mini to build, deploy, and test both machines.
# ============================================================

# --- Configuration ---
MAC_IP="192.168.3.171"
LINUX_IP="192.168.3.29"
LINUX_USER="snape"
LINUX_SSH="${LINUX_USER}@${LINUX_IP}"
LINUX_PROJECT_DIR="/home/snape/claude-code-projects/claw-mesh"
LINUX_BIN_DIR="${LINUX_PROJECT_DIR}/bin"

COORD_PORT=9180
TOKEN="e2e-test-token"
PROJECT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
BIN_DIR="${PROJECT_DIR}/bin"

# --- Colors ---
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
CYAN='\033[0;36m'
NC='\033[0m'

COORD_PID=""
TESTS_PASSED=0
TESTS_FAILED=0

# --- Helpers ---
info()  { echo -e "${CYAN}[INFO]${NC}  $*"; }
ok()    { echo -e "${GREEN}[OK]${NC}    $*"; }
warn()  { echo -e "${YELLOW}[WARN]${NC}  $*"; }
fail()  { echo -e "${RED}[FAIL]${NC}  $*"; }

cleanup() {
    info "Cleaning up..."
    # Stop coordinator
    if [[ -n "$COORD_PID" ]] && kill -0 "$COORD_PID" 2>/dev/null; then
        kill "$COORD_PID" 2>/dev/null || true
        wait "$COORD_PID" 2>/dev/null || true
    fi
    # Stop node on Linux
    ssh -o ConnectTimeout=5 "$LINUX_SSH" "pkill -f 'claw-mesh join' 2>/dev/null || true" 2>/dev/null || true
    # Stop any local coordinator
    pkill -f "claw-mesh up" 2>/dev/null || true
    info "Cleanup done."
}

trap cleanup EXIT

wait_for_port() {
    local host=$1 port=$2 timeout=${3:-15}
    local elapsed=0
    while ! nc -z "$host" "$port" 2>/dev/null; do
        sleep 0.5
        elapsed=$((elapsed + 1))
        if [[ $elapsed -ge $((timeout * 2)) ]]; then
            return 1
        fi
    done
    return 0
}

assert_contains() {
    local label=$1 haystack=$2 needle=$3
    if echo "$haystack" | grep -q "$needle"; then
        ok "$label"
        TESTS_PASSED=$((TESTS_PASSED + 1))
    else
        fail "$label (expected '$needle')"
        echo "  Got: $haystack"
        TESTS_FAILED=$((TESTS_FAILED + 1))
    fi
}

# ============================================================
# Step 1: Build
# ============================================================
info "Step 1: Building binaries..."

cd "$PROJECT_DIR"

# Mac (darwin/arm64)
CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 \
    go build -ldflags "-s -w -X main.version=e2e" -o "${BIN_DIR}/claw-mesh" ./cmd/claw-mesh
ok "Built darwin/arm64 → bin/claw-mesh"

# Linux (linux/amd64)
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags "-s -w -X main.version=e2e" -o "${BIN_DIR}/claw-mesh-linux" ./cmd/claw-mesh
ok "Built linux/amd64  → bin/claw-mesh-linux"

# ============================================================
# Step 2: Deploy to Linux
# ============================================================
info "Step 2: Deploying to Linux (${LINUX_SSH})..."

ssh -o ConnectTimeout=5 "$LINUX_SSH" "mkdir -p ${LINUX_BIN_DIR}"
scp -o ConnectTimeout=5 "${BIN_DIR}/claw-mesh-linux" "${LINUX_SSH}:${LINUX_BIN_DIR}/claw-mesh"
ssh -o ConnectTimeout=5 "$LINUX_SSH" "chmod +x ${LINUX_BIN_DIR}/claw-mesh"
ok "Deployed binary to ${LINUX_SSH}:${LINUX_BIN_DIR}/claw-mesh"

# ============================================================
# Step 3: Kill old processes
# ============================================================
info "Step 3: Stopping old processes..."

pkill -f "claw-mesh up" 2>/dev/null || true
ssh -o ConnectTimeout=5 "$LINUX_SSH" "pkill -f 'claw-mesh join' 2>/dev/null || true" 2>/dev/null || true
sleep 1
ok "Old processes stopped"

# ============================================================
# Step 4: Start coordinator (Mac)
# ============================================================
info "Step 4: Starting coordinator on Mac (port ${COORD_PORT})..."

"${BIN_DIR}/claw-mesh" up --port "$COORD_PORT" --token "$TOKEN" --allow-private &
COORD_PID=$!

if wait_for_port "127.0.0.1" "$COORD_PORT" 10; then
    ok "Coordinator running (PID ${COORD_PID})"
else
    fail "Coordinator failed to start within 10s"
    exit 1
fi

# ============================================================
# Step 5: Start node (Linux via SSH)
# ============================================================
info "Step 5: Starting node on Linux..."

ssh -o ConnectTimeout=5 "$LINUX_SSH" "
    cd ${LINUX_PROJECT_DIR}
    export no_proxy='${MAC_IP}'
    nohup ./bin/claw-mesh join http://${MAC_IP}:${COORD_PORT} \
        --name linux-box --tags linux,docker --token ${TOKEN} \
        > /tmp/claw-mesh-node.log 2>&1 &
    echo \$!
"

# Wait for node to register
sleep 3

# ============================================================
# Step 6: Verify
# ============================================================
info "Step 6: Running verification tests..."

# Test 1: Node list should contain linux-box
NODES=$(curl -sf "http://127.0.0.1:${COORD_PORT}/api/v1/nodes" 2>/dev/null || echo "")
assert_contains "Node list returns data" "$NODES" "linux-box"
assert_contains "Node is online" "$NODES" '"status":"online"'

# Test 2: Send a message via auto-route
ROUTE_RESP=$(curl -sf -X POST "http://127.0.0.1:${COORD_PORT}/api/v1/route" \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer ${TOKEN}" \
    -d '{"content":"hello from e2e test","source":"e2e-script"}' 2>/dev/null || echo "")
assert_contains "Auto-route returns response" "$ROUTE_RESP" "hello from e2e test"
assert_contains "Response has node_id" "$ROUTE_RESP" "node_id"

# Test 3: Dashboard is accessible
DASH_STATUS=$(curl -sf -o /dev/null -w "%{http_code}" "http://127.0.0.1:${COORD_PORT}/" 2>/dev/null || echo "000")
if [[ "$DASH_STATUS" == "200" ]]; then
    ok "Dashboard accessible (HTTP 200)"
    TESTS_PASSED=$((TESTS_PASSED + 1))
else
    fail "Dashboard returned HTTP ${DASH_STATUS}"
    TESTS_FAILED=$((TESTS_FAILED + 1))
fi

# ============================================================
# Step 7: Summary
# ============================================================
echo ""
echo "============================================"
if [[ $TESTS_FAILED -eq 0 ]]; then
    echo -e "  ${GREEN}ALL TESTS PASSED${NC} (${TESTS_PASSED}/${TESTS_PASSED})"
else
    echo -e "  ${RED}SOME TESTS FAILED${NC} (${TESTS_PASSED} passed, ${TESTS_FAILED} failed)"
fi
echo "============================================"
echo ""

# Exit with failure if any test failed
[[ $TESTS_FAILED -eq 0 ]]
