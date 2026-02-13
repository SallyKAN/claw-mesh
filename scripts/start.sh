#!/usr/bin/env bash
set -euo pipefail

# ============================================================
# claw-mesh start — launch coordinator + remote node
# Processes run in background. Use scripts/stop.sh to stop.
# ============================================================

# --- Configuration ---
MAC_IP="192.168.3.171"
LINUX_IP="192.168.3.29"
LINUX_USER="snape"
LINUX_SSH="${LINUX_USER}@${LINUX_IP}"
LINUX_PROJECT_DIR="/home/snape/claude-code-projects/claw-mesh"

COORD_PORT=9180
TOKEN="e2e-test-token"
PROJECT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
BIN_DIR="${PROJECT_DIR}/bin"
PID_DIR="${PROJECT_DIR}/.pids"
LOG_DIR="${PROJECT_DIR}/.logs"

GREEN='\033[0;32m'
CYAN='\033[0;36m'
RED='\033[0;31m'
NC='\033[0m'

info() { echo -e "${CYAN}[INFO]${NC}  $*"; }
ok()   { echo -e "${GREEN}[OK]${NC}    $*"; }
fail() { echo -e "${RED}[FAIL]${NC}  $*"; }

wait_for_port() {
    local host=$1 port=$2 timeout=${3:-10}
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

# --- Setup ---
mkdir -p "$PID_DIR" "$LOG_DIR"

# --- Stop any existing processes first ---
if [[ -f "${PROJECT_DIR}/scripts/stop.sh" ]]; then
    bash "${PROJECT_DIR}/scripts/stop.sh" 2>/dev/null || true
fi

# --- Start coordinator ---
info "Starting coordinator on :${COORD_PORT}..."
"${BIN_DIR}/claw-mesh" up --port "$COORD_PORT" --token "$TOKEN" --allow-private \
    > "${LOG_DIR}/coordinator.log" 2>&1 &
echo $! > "${PID_DIR}/coordinator.pid"

if wait_for_port "127.0.0.1" "$COORD_PORT" 10; then
    ok "Coordinator running (PID $(cat "${PID_DIR}/coordinator.pid"))"
    ok "Dashboard: http://127.0.0.1:${COORD_PORT}/"
else
    fail "Coordinator failed to start. Check ${LOG_DIR}/coordinator.log"
    exit 1
fi

# --- Start node on Linux ---
info "Starting node on Linux (${LINUX_SSH})..."
REMOTE_PID=$(ssh -o ConnectTimeout=5 "$LINUX_SSH" "
    cd ${LINUX_PROJECT_DIR}
    export no_proxy='${MAC_IP}'
    nohup ./bin/claw-mesh join http://${MAC_IP}:${COORD_PORT} \
        --name linux-box --tags linux,docker --token ${TOKEN} \
        > /tmp/claw-mesh-node.log 2>&1 &
    echo \$!
")
echo "$REMOTE_PID" > "${PID_DIR}/node-linux.pid"

sleep 2

# Verify node registered
NODES=$(curl -sf "http://127.0.0.1:${COORD_PORT}/api/v1/nodes" 2>/dev/null || echo "")
if echo "$NODES" | grep -q "linux-box"; then
    ok "Node linux-box registered (remote PID ${REMOTE_PID})"
else
    fail "Node failed to register. Check remote log: ssh ${LINUX_SSH} cat /tmp/claw-mesh-node.log"
    exit 1
fi

echo ""
info "Mesh is running. Logs:"
info "  Coordinator: ${LOG_DIR}/coordinator.log"
info "  Node:        ssh ${LINUX_SSH} cat /tmp/claw-mesh-node.log"
info "Stop with: ./scripts/stop.sh"
