#!/bin/bash
set -eo pipefail

# Start a four-node dYdX v4 localnet natively on Ubuntu without Docker or cosmovisor.
# Reuse helpers from testing/genesis.sh without modifying source files.
# Replace container-specific behavior from local.sh with native equivalents,
# separate ports for each node, and local process management.
#
# Usage:
#   ./local-native.sh init     # Initialize four validator homes and shared genesis
#   ./local-native.sh start    # Start all four nodes in the background
#   ./local-native.sh stop     # Stop all nodes
#   ./local-native.sh status   # Show node status
#   ./local-native.sh logs <moniker>   # Follow logs for alice, bob, carl, or dave
#   ./local-native.sh clean     # Delete all chain data after confirmation
#
# Environment overrides:
#   CHAIN_HOME      Node data root. Default: $HOME/dydx-localnet
#   DYDX_BIN        dydxprotocold path. Defaults to PATH, then protocol/build/dydxprotocold
#   ENABLE_ORACLE   Set to 1 to enable oracle or 0 to disable it. Default: 1
#   SLINKY_ADDR     Slinky address. Default: localhost:8080
#   ETH_RPC_ENDPOINT  Bridge Ethereum RPC endpoint. Empty disables the bridge daemon
#   LOG_LEVEL       Log level. Default: info

CHAIN_ID="localdydxprotocol"
CHAIN_HOME="${CHAIN_HOME:-$HOME/dydx-localnet}"
WORK_DIR="${WORK_DIR:-$CHAIN_HOME/.work}"
LOG_DIR="$CHAIN_HOME/.logs"
STATE_FILE="$CHAIN_HOME/.state"
DYDX_BIN="${DYDX_BIN:-dydxprotocold}"
ENABLE_ORACLE="${ENABLE_ORACLE:-1}"
SLINKY_ADDR="${SLINKY_ADDR:-localhost:8080}"
ETH_RPC_ENDPOINT="${ETH_RPC_ENDPOINT:-}"
LOG_LEVEL="${LOG_LEVEL:-info}"

# Script and repository paths
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TESTING_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"                       # testing/
REPO_PROTO_DIR="$(cd "$TESTING_DIR/.." && pwd)"                    # protocol/
EXCHANGE_CONFIG_SRC="$REPO_PROTO_DIR/daemons/pricefeed/client/constants/testdata"
DELAYMSG_SRC="$TESTING_DIR/delaymsg_config"
GENESIS_SH="$TESTING_DIR/genesis.sh"

# Add the Go bin directory to PATH because dasel may be installed there
if command -v go >/dev/null 2>&1; then
	export PATH="$PATH:$(go env GOPATH)/bin"
fi

# --- Node identities aligned with local.sh for reproducible IDs---
declare -a MONIKERS=(alice bob carl dave)
declare -A NODE_ID=(
	[alice]="17e5e45691f0d01449c84fd4ae87279578cdd7ec"
	[bob]="b69182310be02559483e42c77b7b104352713166"
	[carl]="47539956aaa8e624e0f1d926040e54908ad0eb44"
	[dave]="5882428984d83b03d0c907c1f0af343534987052"
)
declare -A NODE_KEY=(
	[alice]="8EGQBxfGMcRfH0C45UTedEG5Xi3XAcukuInLUqFPpskjp1Ny0c5XvwlKevAwtVvkwoeYYQSe0geQG/cF3GAcUA=="
	[bob]="3OZf5HenMmeTncJY40VJrNYKIKcXoILU5bkYTLzTJvewowU2/iV2+8wSlGOs9LoKdl0ODfj8UutpMhLn5cORlw=="
	[carl]="tWV4uEya9Xvmm/kwcPTnEQIV1ZHqiqUTN/jLPHhIBq7+g/5AEXInokWUGM0shK9+BPaTPTNlzv7vgE8smsFg4w=="
	[dave]="++C3kWgFAs7rUfwAHB7Ffrv43muPg0wTD2/UtSPFFkhtobooIqc78UiotmrT8onuT1jg8/wFPbSjhnKRThTRZg=="
)
declare -A MNEMONIC=(
	[alice]="merge panther lobster crazy road hollow amused security before critic about cliff exhibit cause coyote talent happy where lion river tobacco option coconut small"
	[bob]="color habit donor nurse dinosaur stable wonder process post perfect raven gold census inside worth inquiry mammal panic olive toss shadow strong name drum"
	[carl]="school artefact ghost shop exchange slender letter debris dose window alarm hurt whale tiger find found island what engine ketchup globe obtain glory manage"
	[dave]="switch boring kiss cash lizard coconut romance hurry sniff bus accident zone chest height merit elevator furnace eagle fetch quit toward steak mystery nest"
)
declare -a TEST_ACCOUNTS=(
	"dydx199tqg4wdlnu4qjlxchpd7seg454937hjrknju4" # alice
	"dydx10fx7sy6ywd5senxae9dwytf8jxek3t2gcen2vs" # bob
	"dydx1fjg6zp6vv8t9wvy4lps03r5l4g7tkjw9wvmh70" # carl
	"dydx1wau5mja7j7zdavtfq9lu7ejef05hm6ffenlcsn" # dave
)
declare -a FAUCET_ACCOUNTS=(
	"dydx1nzuttarf5k2j0nug5yzhr6p74t9avehn9hlh8m" # main faucet
)
declare -a VAULT_ACCOUNTS=(
	"dydx1c0m5x87llaunl5sgv3q5vd7j5uha26d2q2r2q0" # BTC vault
	"dydx14rplxdyycc6wxmgl8fggppgq4774l70zt6phkw" # ETH vault
)
declare -a VAULT_NUMBERS=(0 1)

# --- Distinct ports for multiple nodes on one host---
declare -A P2P_PORT=(  [alice]=26656 [bob]=26666 [carl]=26676 [dave]=26686 )
declare -A RPC_PORT=(  [alice]=26657 [bob]=26667 [carl]=26677 [dave]=26687 )
declare -A GRPC_PORT=( [alice]=9090  [bob]=9092  [carl]=9094  [dave]=9096  )
declare -A API_PORT=(  [alice]=1317  [bob]=1318  [carl]=1319  [dave]=1320  )
declare -A TMPROM_PORT=( [alice]=26660 [bob]=26661 [carl]=26662 [dave]=26663 )

# Connect all four nodes as persistent peers through loopback addresses
PERSISTENT_PEERS=""
for m in "${MONIKERS[@]}"; do
	entry="${NODE_ID[$m]}@127.0.0.1:${P2P_PORT[$m]}"
	if [[ -z "$PERSISTENT_PEERS" ]]; then
		PERSISTENT_PEERS="$entry"
	else
		PERSISTENT_PEERS="$PERSISTENT_PEERS,$entry"
	fi
done

log() { echo "[local-native] $*"; }
die() { echo "[local-native] error: $*" >&2; exit 1; }

resolve_binary() {
	if command -v "$DYDX_BIN" >/dev/null 2>&1; then
		DYDX_BIN="$(command -v "$DYDX_BIN")"
	elif [[ -x "$REPO_PROTO_DIR/build/dydxprotocold" ]]; then
		DYDX_BIN="$REPO_PROTO_DIR/build/dydxprotocold"
		log "dydxprotocold was not found in PATH; using $DYDX_BIN"
	else
		die "dydxprotocold was not found. Run 'make build' or 'make install', or set DYDX_BIN."
	fi
	log "Using binary: $DYDX_BIN"
}

install_prerequisites() {
	# Install jq from Ubuntu packages when needed
	if ! command -v jq >/dev/null 2>&1; then
		log "Installing jq..."
		sudo apt-get update -y && sudo apt-get install -y jq
	fi
	# Install dasel with Go when needed
	if ! command -v dasel >/dev/null 2>&1; then
		log "Installing dasel with go install..."
		go install github.com/tomwright/dasel/v2/cmd/dasel@latest
	fi
	command -v jq   >/dev/null 2>&1 || die "jq is unavailable after installation."
	command -v dasel >/dev/null 2>&1 || die "dasel is unavailable after installation."
}

# Prepare links used by the default path resolution in genesis.sh
build_work_dir() {
	mkdir -p "$WORK_DIR"
	ln -sfn "$EXCHANGE_CONFIG_SRC" "$WORK_DIR/exchange_config"
	ln -sfn "$DELAYMSG_SRC"        "$WORK_DIR/delaymsg_config"
}

# Configure Slinky through localhost, or disable oracle integration
native_use_slinky() {
	local config_folder="$1"
	if [[ "$ENABLE_ORACLE" == "1" ]]; then
		dasel put -t bool   -f "$config_folder/app.toml" 'oracle.enabled'                -v true
		dasel put -t string -f "$config_folder/app.toml" 'oracle.oracle_address'         -v "$SLINKY_ADDR"
	else
		dasel put -t bool   -f "$config_folder/app.toml" 'oracle.enabled'                -v false
	fi
}

# Match local.sh settings while using native host addresses
edit_config() {
	local config_folder="$1"
	dasel put -t bool   -f "$config_folder/config.toml" '.p2p.pex'                       -v 'false'
	dasel put -t string -f "$config_folder/config.toml" '.consensus.timeout_commit'     -v '5s'
	# Permit loopback addresses and multiple peers from the same host
	dasel put -t bool   -f "$config_folder/config.toml" '.p2p.addr_book_strict'          -v 'false'
	dasel put -t bool   -f "$config_folder/config.toml" '.p2p.allow_duplicate_ip'       -v 'true'
	dasel put -t bool   -f "$config_folder/app.toml"    '.oracle.metrics_enabled'        -v 'true'
	dasel put -t string -f "$config_folder/app.toml"    '.oracle.prometheus_server_address' -v 'localhost:8001'
}

# Configure persistent peers and distinct ports for each local node
apply_port_config() {
	local moniker="$1"
	local home_dir="$CHAIN_HOME/.$moniker"
	local cfg="$home_dir/config"

	dasel put -t string -f "$cfg/config.toml" '.p2p.persistent_peers' -v "$PERSISTENT_PEERS"
	if [[ "$moniker" != "alice" ]]; then
		dasel put -t string -f "$cfg/config.toml" '.rpc.laddr'                       -v "tcp://0.0.0.0:${RPC_PORT[$moniker]}"
		dasel put -t string -f "$cfg/config.toml" '.p2p.laddr'                        -v "tcp://0.0.0.0:${P2P_PORT[$moniker]}"
		dasel put -t string -f "$cfg/config.toml" '.instrumentation.prometheus_listen_addr' -v "localhost:${TMPROM_PORT[$moniker]}"
		dasel put -t string -f "$cfg/app.toml"    '.grpc.address'                     -v "0.0.0.0:${GRPC_PORT[$moniker]}"
		dasel put -t string -f "$cfg/app.toml"    '.api.address'                      -v "tcp://0.0.0.0:${API_PORT[$moniker]}"
	else
		# Write the default ports explicitly for alice as well
		dasel put -t string -f "$cfg/config.toml" '.rpc.laddr'                       -v "tcp://0.0.0.0:${RPC_PORT[$moniker]}"
		dasel put -t string -f "$cfg/config.toml" '.p2p.laddr'                        -v "tcp://0.0.0.0:${P2P_PORT[$moniker]}"
		dasel put -t string -f "$cfg/config.toml" '.instrumentation.prometheus_listen_addr' -v "localhost:${TMPROM_PORT[$moniker]}"
		dasel put -t string -f "$cfg/app.toml"    '.grpc.address'                     -v "0.0.0.0:${GRPC_PORT[$moniker]}"
		dasel put -t string -f "$cfg/app.toml"    '.api.address'                      -v "tcp://0.0.0.0:${API_PORT[$moniker]}"
	fi
}

# Initialize four validator homes and a shared genesis.
create_validators() {
	local gentx_dir="$WORK_DIR/gentx"
	rm -rf "$gentx_dir"; mkdir -p "$gentx_dir"

	for m in "${MONIKERS[@]}"; do
		local home_dir="$CHAIN_HOME/.$m"
		local cfg="$home_dir/config"
		mkdir -p "$home_dir"

		log "[$m] init ..."
		"$DYDX_BIN" init "$m" -o --chain-id="$CHAIN_ID" --home "$home_dir"

		log "[$m] generating deterministic validator key..."
		"$DYDX_BIN" tendermint gen-priv-key --home "$home_dir" --mnemonic "${MNEMONIC[$m]}"

		# Replace the random node key with a deterministic key for stable peer IDs
		local new_file
		new_file=$(jq ".priv_key.value = \"${NODE_KEY[$m]}\"" "$cfg/node_key.json")
		printf '%s' "$new_file" > "$cfg/node_key.json"

		edit_config "$cfg"
		native_use_slinky "$cfg"

		# edit_genesis must run before add-genesis-account
		edit_genesis "$cfg" "${TEST_ACCOUNTS[*]}" "${FAUCET_ACCOUNTS[*]}" "${VAULT_ACCOUNTS[*]}" "${VAULT_NUMBERS[*]}" "" "" "" ""
		update_genesis_use_test_volatile_market "$cfg"
		update_genesis_complete_bridge_delay "$cfg" "30"

		log "[$m] restoring keyring..."
		printf '%s\n' "${MNEMONIC[$m]}" | "$DYDX_BIN" keys add "$m" --recover --keyring-backend=test --home "$home_dir"

		log "[$m] adding genesis accounts..."
		for acct in "${TEST_ACCOUNTS[@]}"; do
			"$DYDX_BIN" add-genesis-account "$acct" "100000000000000000$USDC_DENOM,$TESTNET_VALIDATOR_NATIVE_TOKEN_BALANCE$NATIVE_TOKEN" --home "$home_dir"
		done
		for acct in "${FAUCET_ACCOUNTS[@]}"; do
			"$DYDX_BIN" add-genesis-account "$acct" "900000000000000000$USDC_DENOM,$TESTNET_VALIDATOR_NATIVE_TOKEN_BALANCE$NATIVE_TOKEN" --home "$home_dir"
		done

		log "[$m] generating gentx..."
		"$DYDX_BIN" gentx "$m" "$TESTNET_VALIDATOR_SELF_DELEGATE_AMOUNT$NATIVE_TOKEN" \
			--moniker="$m" --keyring-backend=test --chain-id="$CHAIN_ID" --home "$home_dir"

		cp -a "$cfg/gentx/." "$gentx_dir/"
	done

	# Collect gentxs in alice's home and build the final genesis
	local first_home="$CHAIN_HOME/.${MONIKERS[0]}"
	local first_cfg="$first_home/config"
	rm -rf "$first_cfg/gentx"; mkdir -p "$first_cfg/gentx"
	cp -r "$gentx_dir" "$first_cfg/"
	log "Collecting gentxs and building final genesis..."
	"$DYDX_BIN" collect-gentxs --home "$first_home"

	# Distribute the final genesis to the remaining nodes
	for m in "${MONIKERS[@]}"; do
		[[ "$m" == "${MONIKERS[0]}" ]] && continue
		local cfg="$CHAIN_HOME/.$m/config"
		rm -f "$cfg/genesis.json"
		cp "$first_cfg/genesis.json" "$cfg/genesis.json"
	done

	# Apply peer and port settings after distributing genesis
	for m in "${MONIKERS[@]}"; do
		apply_port_config "$m"
	done
}

cmd_init() {
	resolve_binary
	install_prerequisites
	build_work_dir

	log "Using work directory: $WORK_DIR"
	# Source genesis.sh from WORK_DIR so its default relative paths resolve
	# through the links prepared above.
	pushd "$WORK_DIR" >/dev/null
	# shellcheck disable=SC1091
	source "$GENESIS_SH"
	create_validators
	popd >/dev/null

	log "Initialization complete."
	log "  Node homes: $CHAIN_HOME/.{alice,bob,carl,dave}"
	log "  RPC(alice): http://127.0.0.1:26657  GRPC: 127.0.0.1:9090  API: http://127.0.0.1:1317"
	log "Next step: $0 start"
}

start_one() {
	local m="$1"
	local home_dir="$CHAIN_HOME/.$m"
	local log_file="$LOG_DIR/$m.log"
	[[ -d "$home_dir" ]] || die "Node $m is not initialized. Run 'init' first."

	local extra=()
	extra+=(--log_level "$LOG_LEVEL")
	extra+=(--max-daemon-unhealthy-seconds 4294967295) # Keep local daemon outages from stopping block production
	if [[ -n "$ETH_RPC_ENDPOINT" ]]; then
		extra+=(--bridge-daemon-eth-rpc-endpoint "$ETH_RPC_ENDPOINT")
	else
		extra+=(--bridge-daemon-enabled=false) # Avoid startup panic when no Ethereum RPC endpoint is configured
	fi

	log "Starting $m (home=$home_dir)..."
	nohup "$DYDX_BIN" start --home "$home_dir" "${extra[@]}" >"$log_file" 2>&1 &
	local pid=$!
	echo "$m $pid" >>"$STATE_FILE"
	log "  $m pid=$pid, log: $log_file"
}

cmd_start() {
	resolve_binary
	mkdir -p "$LOG_DIR"
	if [[ -f "$STATE_FILE" ]]; then
		log "Existing process state found; stopping old processes..."
		cmd_stop
	fi
	: >"$STATE_FILE"
	for m in "${MONIKERS[@]}"; do
		start_one "$m"
		sleep 1
	done
	log "All nodes started. Use '$0 status' for status and '$0 logs alice' for logs."
	log "If Slinky is unavailable at localhost:8080, oracle errors are expected but blocks can continue."
}

cmd_stop() {
	[[ -f "$STATE_FILE" ]] || { log "No process state found."; return; }
	while read -r m pid; do
		if kill -0 "$pid" 2>/dev/null; then
			log "Stopping $m (pid=$pid)..."
			kill "$pid" 2>/dev/null || true
		fi
	done <"$STATE_FILE"
	# Clean up matching processes if the state file is incomplete
	if command -v pkill >/dev/null 2>&1; then
		pkill -f "$DYDX_BIN start --home $CHAIN_HOME" 2>/dev/null || true
	fi
	rm -f "$STATE_FILE"
	log "Stopped."
}

cmd_status() {
	if [[ -f "$STATE_FILE" ]]; then
		while read -r m pid; do
			if kill -0 "$pid" 2>/dev/null; then
				printf '  %-6s RUNNING  pid=%-8s rpc=127.0.0.1:%s\n' "$m" "$pid" "${RPC_PORT[$m]}"
			else
				printf '  %-6s STOPPED pid=%-8s\n' "$m" "$pid"
			fi
		done <"$STATE_FILE"
	else
		log "No process state found."
	fi
}

cmd_logs() {
	local m="${1:-alice}"
	local log_file="$LOG_DIR/$m.log"
	[[ -f "$log_file" ]] || die "Log file does not exist: $log_file"
	log "Following $m logs. Press Ctrl-C to stop."
	tail -n 100 -f "$log_file"
}

cmd_clean() {
	read -rp "Delete all chain data under $CHAIN_HOME? [y/N] " ans
	[[ "$ans" == "y" || "$ans" == "Y" ]] || { log "Cancelled."; return; }
	if [[ -f "$STATE_FILE" ]]; then cmd_stop; fi
	rm -rf "$CHAIN_HOME"
	log "Removed $CHAIN_HOME."
}

usage() {
	cat <<'EOF'
Usage: ./local-native.sh <command>

  init            Initialize four validator homes and shared genesis
  start           Start all four nodes in the background
  stop            Stop all nodes
  status          Show node status
  logs <moniker>  Follow logs for alice, bob, carl, or dave
  clean           Delete all chain data after confirmation

Environment variables:
  CHAIN_HOME        Node data root. Default: $HOME/dydx-localnet
  DYDX_BIN          dydxprotocold path. Defaults to PATH, then protocol/build/dydxprotocold
  ENABLE_ORACLE     Set to 1 to enable oracle or 0 to disable it. Default: 1
  SLINKY_ADDR       Slinky address. Default: localhost:8080
  ETH_RPC_ENDPOINT  Bridge Ethereum RPC endpoint. Empty disables the bridge daemon
  LOG_LEVEL         Log level. Default: info
EOF
	exit 1
}

main() {
	local cmd="${1:-}"
	shift || true
	case "$cmd" in
		init)   cmd_init ;;
		start)  cmd_start ;;
		stop)   cmd_stop ;;
		status) cmd_status ;;
		logs)   cmd_logs "$@" ;;
		clean)  cmd_clean ;;
		*)      usage ;;
	esac
}

main "$@"
