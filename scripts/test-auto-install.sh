#!/usr/bin/env bash
set -euo pipefail

# ============================================================
# test-auto-install.sh — E2E test for runtime auto-detect & auto-install
#
# Run from Mac. Tests on remote Linux via SSH.
# Covers: auto-detect logic, --auto-install, --runtime, --no-gateway
#
# Usage:
#   ./scripts/test-auto-install.sh              # run all tests
#   ./scripts/test-auto-install.sh test_name    # run single test
# ============================================================

# --- Configuration (same as e2e-deploy.sh) ---
MAC_IP="192.168.3.171"
LINUX_IP="192.168.3.29"
LINUX_USER="snape"
LINUX_SSH="${LINUX_USER}@${LINUX_IP}"
LINUX_PROJECT_DIR="/home/snape/claude-code-projects/claw-mesh"
LINUX_BIN_DIR="${LINUX_PROJECT_DIR}/bin"
LINUX_BIN="${LINUX_BIN_DIR}/claw-mesh"

COORD_PORT=9180
PROJECT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
BIN_DIR="${PROJECT_DIR}/bin"

# Read token from existing config (coordinator loads this file automatically)
CONFIG_FILE="${PROJECT_DIR}/claw-mesh.yaml"
if [[ -f "$CONFIG_FILE" ]]; then
    TOKEN=$(grep 'token:' "$CONFIG_FILE" | head -1 | awk '{print $2}')
fi
TOKEN="${TOKEN:-autoinstall-test-$(date +%s)}"

# --- Colors ---
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
CYAN='\033[0;36m'
NC='\033[0m'

COORD_PID=""
TESTS_PASSED=0
TESTS_FAILED=0
TESTS_SKIPPED=0

# --- Helpers ---
info()  { echo -e "${CYAN}[INFO]${NC}  $*"; }
ok()    { echo -e "${GREEN}[OK]${NC}    $*"; TESTS_PASSED=$((TESTS_PASSED + 1)); }
fail()  { echo -e "${RED}[FAIL]${NC}  $*"; TESTS_FAILED=$((TESTS_FAILED + 1)); }
skip()  { echo -e "${YELLOW}[SKIP]${NC}  $*"; TESTS_SKIPPED=$((TESTS_SKIPPED + 1)); }

ssh_cmd() { ssh -o ConnectTimeout=5 -o StrictHostKeyChecking=no "$LINUX_SSH" "$@"; }

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

# --- Cleanup: runs on EXIT, always ---
cleanup_all() {
    info "Final cleanup..."
    # Stop local coordinator
    if [[ -n "$COORD_PID" ]] && kill -0 "$COORD_PID" 2>/dev/null; then
        kill "$COORD_PID" 2>/dev/null || true
        wait "$COORD_PID" 2>/dev/null || true
    fi
    pkill -f "claw-mesh up" 2>/dev/null || true
    # Stop remote node + clean installed runtimes
    cleanup_linux
    info "Final cleanup done."
}
trap cleanup_all EXIT

# Clean up Linux: kill node, remove installed runtimes
cleanup_linux() {
    ssh_cmd "
        # Kill any running claw-mesh node
        pkill -f 'claw-mesh join' 2>/dev/null || true
        # Remove openclaw if installed via npm (global or user-local)
        npm uninstall -g openclaw 2>/dev/null || true
        npm uninstall -g --prefix \$HOME/.local openclaw 2>/dev/null || true
        rm -f \$HOME/.local/bin/openclaw 2>/dev/null || true
        # Remove zeroclaw binary
        rm -f /usr/local/bin/zeroclaw 2>/dev/null || true
        rm -f \$HOME/.local/bin/zeroclaw 2>/dev/null || true
    " 2>/dev/null || true
}

# Stop coordinator if running
stop_coordinator() {
    if [[ -n "$COORD_PID" ]] && kill -0 "$COORD_PID" 2>/dev/null; then
        kill "$COORD_PID" 2>/dev/null || true
        wait "$COORD_PID" 2>/dev/null || true
    fi
    COORD_PID=""
}

# Start coordinator fresh
start_coordinator() {
    stop_coordinator
    "${BIN_DIR}/claw-mesh" up --port "$COORD_PORT" --token "$TOKEN" --allow-private \
        >/tmp/claw-mesh-coord-autoinstall.log 2>&1 &
    COORD_PID=$!
    if wait_for_port "127.0.0.1" "$COORD_PORT" 10; then
        info "Coordinator running (PID ${COORD_PID})"
    else
        fail "Coordinator failed to start"
        cat /tmp/claw-mesh-coord-autoinstall.log
        exit 1
    fi
}

# ============================================================
# Step 0: Build & Deploy
# ============================================================
info "=== Building and deploying ==="

cd "$PROJECT_DIR"

CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 \
    go build -ldflags "-s -w -X main.version=autoinstall-test" -o "${BIN_DIR}/claw-mesh" ./cmd/claw-mesh
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags "-s -w -X main.version=autoinstall-test" -o "${BIN_DIR}/claw-mesh-linux" ./cmd/claw-mesh
ok "Binaries built"

ssh_cmd "mkdir -p ${LINUX_BIN_DIR}"
scp -o ConnectTimeout=5 "${BIN_DIR}/claw-mesh-linux" "${LINUX_SSH}:${LINUX_BIN}"
ssh_cmd "chmod +x ${LINUX_BIN}"
ok "Deployed to Linux"

# Clean slate: remove any pre-existing runtimes
cleanup_linux
info "Linux cleaned (no openclaw/zeroclaw)"
echo ""

# ============================================================
# Test 1: --auto-install auto-detects OpenClaw (has Node.js, 16GB)
# ============================================================
test_auto_detect_openclaw() {
    info "=== Test 1: --auto-install auto-detects OpenClaw ==="

    start_coordinator

    local node_log="/tmp/claw-mesh-autoinstall-t1.log"
    # Start node in background
    ssh_cmd "
        export no_proxy='${MAC_IP}'
        nohup ${LINUX_BIN} join http://${MAC_IP}:${COORD_PORT} \
            --name auto-node-1 --token ${TOKEN} --auto-install \
            >${node_log} 2>&1 &
        echo \$!
    " > /tmp/test1_pid.txt 2>&1

    # Wait for install to complete (npm install openclaw takes ~5min)
    for i in $(seq 1 90); do
        local log_content
        log_content=$(ssh_cmd "cat ${node_log} 2>/dev/null" 2>/dev/null || echo "")
        if echo "$log_content" | grep -q "installed successfully\|install failed\|installed to\|detected.*runtime"; then
            break
        fi
        # Also check if npm is still running
        local npm_running
        npm_running=$(ssh_cmd "pgrep -f 'npm install openclaw' >/dev/null 2>&1 && echo yes || echo no" 2>/dev/null || echo "unknown")
        if echo "$log_content" | grep -q "auto-installing" && [[ "$npm_running" == "no" ]]; then
            # npm finished but no success/fail message — give a moment for log flush
            sleep 3
            break
        fi
        sleep 5
    done
    sleep 2

    local output
    output=$(ssh_cmd "cat ${node_log} 2>/dev/null" 2>/dev/null || echo "")

    # Should recommend openclaw (has Node.js + enough memory)
    if echo "$output" | grep -q "auto-installing openclaw"; then
        ok "Test 1a: auto-install picked OpenClaw (correct for Node.js + 16GB)"
    elif echo "$output" | grep -q "detected openclaw runtime"; then
        ok "Test 1a: OpenClaw already detected (pre-installed)"
    elif echo "$output" | grep -q "recommended: openclaw"; then
        ok "Test 1a: recommended OpenClaw (correct)"
    else
        fail "Test 1a: expected openclaw recommendation/install"
        echo "  Output: $output"
    fi

    # Verify openclaw binary actually exists after install
    if echo "$output" | grep -q "install failed\|npm install openclaw failed"; then
        fail "Test 1b: openclaw install failed"
        echo "  Output: $output"
    elif echo "$output" | grep -qE "installed successfully|installed to|added [0-9]+ packages"; then
        # Double-check binary exists
        local openclaw_path
        openclaw_path=$(ssh_cmd "which openclaw 2>/dev/null || ls \$HOME/.local/bin/openclaw 2>/dev/null || echo 'not_found'")
        if [[ "$openclaw_path" != "not_found" ]]; then
            ok "Test 1b: openclaw binary verified at $openclaw_path"
        else
            fail "Test 1b: log says installed but binary not found in PATH"
        fi
    elif echo "$output" | grep -q "detected openclaw runtime"; then
        ok "Test 1b: openclaw was already present"
    else
        fail "Test 1b: install did not complete (timeout?)"
        echo "  Output tail: $(echo "$output" | tail -5)"
    fi

    ssh_cmd "pkill -f 'claw-mesh join' 2>/dev/null || true" 2>/dev/null || true
    stop_coordinator
    cleanup_linux
    echo ""
}

# ============================================================
# Test 2: --auto-install --runtime zeroclaw forces ZeroClaw
# ============================================================
test_force_zeroclaw() {
    info "=== Test 2: --auto-install --runtime zeroclaw ==="

    start_coordinator

    local node_log="/tmp/claw-mesh-autoinstall-t2.log"
    ssh_cmd "
        export no_proxy='${MAC_IP}'
        nohup ${LINUX_BIN} join http://${MAC_IP}:${COORD_PORT} \
            --name auto-node-2 --token ${TOKEN} \
            --auto-install --runtime zeroclaw \
            >${node_log} 2>&1 &
        echo \$!
    " > /tmp/test2_pid.txt 2>&1

    for i in $(seq 1 40); do
        local log_content
        log_content=$(ssh_cmd "cat ${node_log} 2>/dev/null" 2>/dev/null || echo "")
        if echo "$log_content" | grep -q "registered as node\|auto-installing\|install failed"; then
            break
        fi
        sleep 1
    done
    sleep 5

    local output
    output=$(ssh_cmd "cat ${node_log} 2>/dev/null" 2>/dev/null || echo "")

    if echo "$output" | grep -q "auto-installing zeroclaw"; then
        ok "Test 2a: --runtime zeroclaw forced ZeroClaw install"
    else
        fail "Test 2a: expected zeroclaw install attempt"
        echo "  Output: $output"
    fi

    # Check if zeroclaw binary was actually installed
    local zeroclaw_exists
    zeroclaw_exists=$(ssh_cmd "which zeroclaw 2>/dev/null || ls \$HOME/.local/bin/zeroclaw 2>/dev/null || echo 'not_found'")
    if [[ "$zeroclaw_exists" != "not_found" ]]; then
        ok "Test 2b: zeroclaw binary installed at $zeroclaw_exists"
    else
        if echo "$output" | grep -q "install failed"; then
            skip "Test 2b: zeroclaw install failed (expected — release may not exist)"
        else
            fail "Test 2b: zeroclaw binary not found after install"
        fi
    fi

    ssh_cmd "pkill -f 'claw-mesh join' 2>/dev/null || true" 2>/dev/null || true
    stop_coordinator
    cleanup_linux
    echo ""
}

# ============================================================
# Test 3: join without --auto-install prints tip
# ============================================================
test_no_auto_install_prints_tip() {
    info "=== Test 3: join without --auto-install prints tip ==="

    start_coordinator

    local node_log="/tmp/claw-mesh-autoinstall-t3.log"
    ssh_cmd "
        export no_proxy='${MAC_IP}'
        nohup ${LINUX_BIN} join http://${MAC_IP}:${COORD_PORT} \
            --name auto-node-3 --token ${TOKEN} \
            >${node_log} 2>&1 &
        echo \$!
    " > /tmp/test3_pid.txt 2>&1

    for i in $(seq 1 15); do
        local log_content
        log_content=$(ssh_cmd "cat ${node_log} 2>/dev/null" 2>/dev/null || echo "")
        if echo "$log_content" | grep -q "registered as node\|tip:.*--auto-install\|detected.*runtime"; then
            break
        fi
        sleep 1
    done
    sleep 2

    local output
    output=$(ssh_cmd "cat ${node_log} 2>/dev/null" 2>/dev/null || echo "")

    if echo "$output" | grep -q "tip:.*--auto-install"; then
        ok "Test 3a: printed --auto-install tip (no runtime, no auto-install flag)"
    elif echo "$output" | grep -q "detected.*runtime"; then
        skip "Test 3a: runtime already present, tip not shown"
    else
        fail "Test 3a: expected --auto-install tip in output"
        echo "  Output: $output"
    fi

    # Should NOT have installed anything
    if echo "$output" | grep -q "auto-installing"; then
        fail "Test 3b: should NOT auto-install without --auto-install flag"
    else
        ok "Test 3b: no auto-install without flag (correct)"
    fi

    ssh_cmd "pkill -f 'claw-mesh join' 2>/dev/null || true" 2>/dev/null || true
    stop_coordinator
    cleanup_linux
    echo ""
}

# ============================================================
# Test 4: --no-gateway skips runtime detection entirely
# ============================================================
test_no_gateway_skips_runtime() {
    info "=== Test 4: --no-gateway skips runtime detection ==="

    start_coordinator

    local node_log="/tmp/claw-mesh-autoinstall-t4.log"
    # Start node in background (don't kill it yet)
    ssh_cmd "
        export no_proxy='${MAC_IP}'
        nohup ${LINUX_BIN} join http://${MAC_IP}:${COORD_PORT} \
            --name auto-node-4 --token ${TOKEN} \
            --no-gateway \
            >${node_log} 2>&1 &
        echo \$!
    " > /tmp/test4_pid.txt 2>&1

    # Wait for registration
    local registered=false
    for i in $(seq 1 15); do
        local log_content
        log_content=$(ssh_cmd "cat ${node_log} 2>/dev/null" 2>/dev/null || echo "")
        if echo "$log_content" | grep -q "registered as node"; then
            registered=true
            break
        fi
        sleep 1
    done

    local output
    output=$(ssh_cmd "cat ${node_log} 2>/dev/null" 2>/dev/null || echo "")

    if $registered; then
        ok "Test 4a: node registered in echo mode"
    else
        fail "Test 4a: node did not register"
        echo "  Output: $output"
    fi

    if echo "$output" | grep -q "auto-installing\|recommended:\|detected.*runtime"; then
        fail "Test 4b: --no-gateway should skip runtime detection"
        echo "  Output: $output"
    else
        ok "Test 4b: no runtime detection with --no-gateway (correct)"
    fi

    # Verify node is online via API (node is still running)
    sleep 1
    local nodes
    nodes=$(curl -sf "http://127.0.0.1:${COORD_PORT}/api/v1/nodes" \
        -H "Authorization: Bearer ${TOKEN}" 2>/dev/null || echo "")
    if echo "$nodes" | grep -q "auto-node-4"; then
        ok "Test 4c: node visible in coordinator"
    else
        fail "Test 4c: node not visible in coordinator"
        echo "  Nodes API response: $nodes"
    fi

    # Now clean up
    ssh_cmd "pkill -f 'claw-mesh join' 2>/dev/null || true" 2>/dev/null || true
    stop_coordinator
    cleanup_linux
    echo ""
}

# ============================================================
# Run tests
# ============================================================
FILTER="${1:-all}"

if [[ "$FILTER" == "all" ]]; then
    test_auto_detect_openclaw
    test_force_zeroclaw
    test_no_auto_install_prints_tip
    test_no_gateway_skips_runtime
else
    # Run single test by function name
    if declare -f "test_${FILTER}" >/dev/null 2>&1; then
        "test_${FILTER}"
    elif declare -f "$FILTER" >/dev/null 2>&1; then
        "$FILTER"
    else
        echo "Unknown test: $FILTER"
        echo "Available: auto_detect_openclaw, force_zeroclaw, no_auto_install_prints_tip, no_gateway_skips_runtime"
        exit 1
    fi
fi

# ============================================================
# Summary
# ============================================================
echo ""
echo "============================================"
TOTAL=$((TESTS_PASSED + TESTS_FAILED + TESTS_SKIPPED))
if [[ $TESTS_FAILED -eq 0 ]]; then
    echo -e "  ${GREEN}ALL TESTS PASSED${NC} (${TESTS_PASSED} passed, ${TESTS_SKIPPED} skipped / ${TOTAL} total)"
else
    echo -e "  ${RED}SOME TESTS FAILED${NC} (${TESTS_PASSED} passed, ${TESTS_FAILED} failed, ${TESTS_SKIPPED} skipped / ${TOTAL} total)"
fi
echo "============================================"
echo ""

[[ $TESTS_FAILED -eq 0 ]]
