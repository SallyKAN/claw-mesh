# Quick Start

Get a two-node mesh running in under 5 minutes.

## Prerequisites

- Two machines (or two terminals on one machine for testing)
- Go 1.22+ (for building from source) or use the install script

## 1. Install

```bash
# One-liner install (macOS / Linux)
curl -fsSL https://raw.githubusercontent.com/SallyKAN/claw-mesh/main/install.sh | sh

# Or with Go
go install github.com/SallyKAN/claw-mesh/cmd/claw-mesh@latest

# Or build from source
git clone https://github.com/SallyKAN/claw-mesh.git
cd claw-mesh && make build
```

## 2. Start the coordinator

On your main machine:

```bash
claw-mesh init                # generates claw-mesh.yaml with a random token
claw-mesh up --allow-private  # start coordinator on :9180
```

The web dashboard is now available at `http://localhost:9180`.

## 3. Join a node

On another machine (or another terminal):

```bash
claw-mesh join http://<coordinator-ip>:9180 \
  --name linux-gpu \
  --tags gpu,docker \
  --token <your-token> \
  --auto-install
```

`--auto-install` will detect and install the best OpenClaw runtime for the machine.

## 4. Verify

```bash
claw-mesh status    # mesh overview
claw-mesh nodes     # list connected nodes
```

## 5. Send a message

```bash
# Auto-route — coordinator picks the best node
claw-mesh send --auto "What tools do you have available?"

# Target a specific node
claw-mesh send --node linux-gpu "Run the Docker build"
```

## Next steps

- [Nodes & Capabilities](/guide/nodes) — understand how capability detection works
- [Routing](/guide/routing) — configure routing rules
- [Configuration](/reference/configuration) — full config file reference
- [CLI Reference](/reference/cli) — all commands and flags
