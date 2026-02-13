#!/usr/bin/env bash
set -euo pipefail

# ============================================================
# claw-mesh stop — stop coordinator + remote node
# ============================================================

LINUX_IP="192.168.3.29"
LINUX_USER="snape"
LINUX_SSH="${LINUX_USER}@${LINUX_IP}"

PROJECT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
PID_DIR="${PROJECT_DIR}/.pids"

GREEN='\033[0;32m'
CYAN='\033[0;36m'
NC='\033[0m'

info() { echo -e "${CYAN}[INFO]${NC}  $*"; }
ok()   { echo -e "${GREEN}[OK]${NC}    $*"; }

# --- Stop node on Linux ---
info "Stopping node on Linux..."
if [[ -f "${PID_DIR}/node-linux.pid" ]]; then
    REMOTE_PID=$(cat "${PID_DIR}/node-linux.pid")
    ssh -o ConnectTimeout=5 "$LINUX_SSH" "kill $REMOTE_PID 2>/dev/null || true" 2>/dev/null || true
    rm -f "${PID_DIR}/node-linux.pid"
fi
ssh -o ConnectTimeout=5 "$LINUX_SSH" "pkill -f 'claw-mesh join' 2>/dev/null || true" 2>/dev/null || true
ok "Node stopped"

# --- Stop coordinator ---
info "Stopping coordinator..."
if [[ -f "${PID_DIR}/coordinator.pid" ]]; then
    COORD_PID=$(cat "${PID_DIR}/coordinator.pid")
    if kill -0 "$COORD_PID" 2>/dev/null; then
        kill "$COORD_PID" 2>/dev/null || true
        wait "$COORD_PID" 2>/dev/null || true
    fi
    rm -f "${PID_DIR}/coordinator.pid"
fi
pkill -f "claw-mesh up" 2>/dev/null || true
ok "Coordinator stopped"

info "Mesh stopped."
