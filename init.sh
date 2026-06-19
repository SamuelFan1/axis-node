#!/usr/bin/env bash

set -euo pipefail

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SERVICE_NAME="axis-node.service"
SERVICE_FILE_SOURCE="${PROJECT_ROOT}/deployments/systemd/axis-node.service"
SERVICE_FILE_TARGET="/etc/systemd/system/${SERVICE_NAME}"
BUILD_OUTPUT="${PROJECT_ROOT}/axis-node"
INSTALL_TARGET="/usr/local/bin/axis-node"
ENV_FILE="${PROJECT_ROOT}/.env"
ENV_EXAMPLE_FILE="${PROJECT_ROOT}/.env.example"
REGION_MAPPING_FILE="${PROJECT_ROOT}/../NetStone/NetStone/conf/server_region_mapping.yaml"
GO_TOOLCHAIN_BASE="/usr/local"
GO_TOOLCHAIN_DEFAULT_PATCH="${GO_TOOLCHAIN_DEFAULT_PATCH:-0}"
GO_CMD=""

if [[ ! -f "${PROJECT_ROOT}/README.md" ]]; then
  echo -e "${RED}Error:${NC} script must live in the axis-node project root."
  exit 1
fi

if [[ ! -f "${SERVICE_FILE_SOURCE}" ]]; then
  echo -e "${RED}Error:${NC} missing systemd unit: ${SERVICE_FILE_SOURCE}"
  exit 1
fi

if [[ ! -f "${ENV_FILE}" ]]; then
  if [[ -f "${ENV_EXAMPLE_FILE}" ]]; then
    echo -e "${YELLOW}.env not found, copying from .env.example...${NC}"
    cp "${ENV_EXAMPLE_FILE}" "${ENV_FILE}"
  else
    echo -e "${RED}Error:${NC} missing ${ENV_FILE} and ${ENV_EXAMPLE_FILE}."
    exit 1
  fi
fi

if [[ "${EUID}" -eq 0 ]]; then
  SUDO=""
else
  if ! command -v sudo >/dev/null 2>&1; then
    echo -e "${RED}Error:${NC} sudo is required when not running as root."
    exit 1
  fi
  SUDO="sudo"
fi

run_root() {
  if [[ -n "${SUDO}" ]]; then
    ${SUDO} "$@"
  else
    "$@"
  fi
}

go_required_version() {
  awk '$1 == "go" { print $2; exit }' "${PROJECT_ROOT}/go.mod"
}

go_version_number() {
  local go_bin="$1"
  GOTOOLCHAIN=local "${go_bin}" version 2>/dev/null | awk '{print $3}' | sed 's/^go//' || true
}

version_at_least() {
  local current="$1"
  local required="$2"
  python3 - "${current}" "${required}" <<'PY'
import sys

def parts(value):
    out = []
    for item in value.split("."):
        try:
            out.append(int(item))
        except ValueError:
            out.append(0)
    while len(out) < 3:
        out.append(0)
    return tuple(out[:3])

raise SystemExit(0 if parts(sys.argv[1]) >= parts(sys.argv[2]) else 1)
PY
}

go_download_arch() {
  local machine
  machine="$(uname -m)"
  case "${machine}" in
    x86_64|amd64)
      echo "amd64"
      ;;
    aarch64|arm64)
      echo "arm64"
      ;;
    *)
      echo -e "${RED}Error:${NC} unsupported Go architecture: ${machine}" >&2
      return 1
      ;;
  esac
}

go_install_version() {
  local required="$1"
  local dot_count
  dot_count="$(awk -F. '{print NF-1}' <<< "${required}")"
  if [[ "${dot_count}" -lt 2 ]]; then
    printf '%s.%s\n' "${required}" "${GO_TOOLCHAIN_DEFAULT_PATCH}"
  else
    printf '%s\n' "${required}"
  fi
}

ensure_go_toolchain() {
  local required install_version bundled_go system_go system_version arch url tmp_dir archive

  required="$(go_required_version)"
  if [[ -z "${required}" ]]; then
    echo -e "${RED}Error:${NC} go.mod does not declare a Go version."
    exit 1
  fi

  if command -v go >/dev/null 2>&1; then
    system_go="$(command -v go)"
    system_version="$(go_version_number "${system_go}")"
    if [[ -n "${system_version}" ]] && version_at_least "${system_version}" "${required}"; then
      GO_CMD="${system_go}"
      echo -e "${BLUE}Using system Go toolchain:${NC} $("${GO_CMD}" version)"
      return 0
    fi
  fi

  install_version="$(go_install_version "${required}")"
  bundled_go="${GO_TOOLCHAIN_BASE}/go${install_version}/bin/go"
  if [[ -x "${bundled_go}" ]]; then
    system_version="$(go_version_number "${bundled_go}")"
    if [[ -n "${system_version}" ]] && version_at_least "${system_version}" "${required}"; then
      GO_CMD="${bundled_go}"
      echo -e "${BLUE}Using local Go toolchain:${NC} $("${GO_CMD}" version)"
      return 0
    fi
  fi

  if ! command -v curl >/dev/null 2>&1; then
    echo -e "${RED}Error:${NC} curl is required to install Go ${install_version}."
    exit 1
  fi

  if ! command -v tar >/dev/null 2>&1; then
    echo -e "${RED}Error:${NC} tar is required to install Go ${install_version}."
    exit 1
  fi

  arch="$(go_download_arch)"
  url="https://go.dev/dl/go${install_version}.linux-${arch}.tar.gz"
  tmp_dir="$(mktemp -d)"
  archive="${tmp_dir}/go${install_version}.linux-${arch}.tar.gz"

  echo -e "${YELLOW}Installing Go ${install_version} for linux/${arch}...${NC}"
  curl -fL --retry 3 -o "${archive}" "${url}"
  run_root rm -rf "${GO_TOOLCHAIN_BASE}/go${install_version}"
  run_root mkdir -p "${GO_TOOLCHAIN_BASE}/go${install_version}"
  run_root tar -C "${GO_TOOLCHAIN_BASE}/go${install_version}" --strip-components=1 -xzf "${archive}"
  rm -rf "${tmp_dir}"

  GO_CMD="${bundled_go}"
  if [[ ! -x "${GO_CMD}" ]]; then
    echo -e "${RED}Error:${NC} installed Go binary not found: ${GO_CMD}"
    exit 1
  fi
  echo -e "${BLUE}Using installed Go toolchain:${NC} $("${GO_CMD}" version)"
}

detect_wt0_ipv4() {
  if ! command -v ip >/dev/null 2>&1; then
    return 1
  fi
  ip -o -4 addr show dev wt0 2>/dev/null | awk '{print $4}' | cut -d/ -f1 | head -n1
}

resolve_region_by_wt0_ip() {
  local wt0_ip="$1"
  python3 - "${ENV_FILE}" "${wt0_ip}" <<'PY'
import pathlib, sys

env_path = pathlib.Path(sys.argv[1])
wt0_ip = sys.argv[2].strip()
if not wt0_ip or not env_path.exists():
    raise SystemExit(0)

env = {}
for line in env_path.read_text().splitlines():
    line = line.strip()
    if not line or line.startswith("#") or "=" not in line:
        continue
    key, value = line.split("=", 1)
    env[key.strip()] = value.strip().strip('"').strip("'")

mapping = [
    ("north_america", "AXIS_WT0_REGION_NORTH_AMERICA_PREFIXES"),
    ("asia", "AXIS_WT0_REGION_ASIA_PREFIXES"),
    ("australia", "AXIS_WT0_REGION_AUSTRALIA_PREFIXES"),
    ("europe", "AXIS_WT0_REGION_EUROPE_PREFIXES"),
    ("middle_east", "AXIS_WT0_REGION_MIDDLE_EAST_PREFIXES"),
]

for region, key in mapping:
    raw = env.get(key, "")
    prefixes = [item.strip() for item in raw.split(",") if item.strip()]
    for prefix in prefixes:
        if wt0_ip.startswith(prefix):
            print(region)
            raise SystemExit(0)
PY
}

resolve_node_hostname() {
  if [[ -n "${HOST_HOSTNAME:-}" ]]; then
    printf '%s\n' "${HOST_HOSTNAME}"
    return 0
  fi
  if [[ -n "${HOSTNAME:-}" ]]; then
    printf '%s\n' "${HOSTNAME}"
    return 0
  fi
  if command -v hostname >/dev/null 2>&1; then
    hostname
    return 0
  fi
  return 1
}

extract_hostname_prefix() {
  local hostname_value="$1"
  printf '%s\n' "${hostname_value}" | awk -F- '{print toupper($1)}'
}

resolve_region_zone_by_prefix() {
  local prefix="$1"
  [[ -f "${REGION_MAPPING_FILE}" ]] || return 1

  awk -v prefix="${prefix}" '
    $1 == "prefix_map:" { in_map = 1; next }
    in_map && $0 ~ /^  [A-Z0-9]+:$/ {
      if (entry == prefix && region != "" && zone != "") {
        print region "|" zone
        exit
      }
      entry = $1
      sub(/:$/, "", entry)
      region = ""
      zone = ""
      next
    }
    in_map && entry == prefix && $1 == "axis_region:" {
      region = $2
      next
    }
    in_map && entry == prefix && $1 == "country_code:" {
      zone = $2
      next
    }
    END {
      if (entry == prefix && region != "" && zone != "") {
        print region "|" zone
      }
    }
  ' "${REGION_MAPPING_FILE}"
}

current_management_port() {
  if [[ ! -f "${ENV_FILE}" ]]; then
    echo "9090"
    return
  fi
  python3 - "${ENV_FILE}" <<'PY'
import pathlib, sys
env_path = pathlib.Path(sys.argv[1])
port = "9090"
for line in env_path.read_text().splitlines():
    line = line.strip()
    if not line or line.startswith("#") or "=" not in line:
        continue
    key, value = line.split("=", 1)
    if key.strip() != "AXIS_NODE_MANAGEMENT_ADDRESS":
        continue
    value = value.strip().strip('"').strip("'")
    if ":" in value:
        maybe_port = value.rsplit(":", 1)[1].strip()
        if maybe_port:
            port = maybe_port
    break
print(port)
PY
}

upsert_env_value() {
  local key="$1"
  local value="$2"
  python3 - "${ENV_FILE}" "${key}" "${value}" <<'PY'
import pathlib, sys
env_path = pathlib.Path(sys.argv[1])
target_key = sys.argv[2]
target_value = sys.argv[3]
lines = env_path.read_text().splitlines()
updated = False
out = []
for line in lines:
    stripped = line.strip()
    if stripped and not stripped.startswith("#") and "=" in line:
        key, _ = line.split("=", 1)
        if key.strip() == target_key:
            out.append(f"{target_key}={target_value}")
            updated = True
            continue
    out.append(line)
if not updated:
    if out and out[-1] != "":
        out.append("")
    out.append(f"{target_key}={target_value}")
env_path.write_text("\n".join(out) + "\n")
PY
}

sync_cloudflared_monitoring_config() {
  local cloudflared_unit="cloudflared.service"
  local monitor_unit="cloudflared-health-monitor.service"

  if systemctl cat "${cloudflared_unit}" >/dev/null 2>&1 &&
     systemctl cat "${monitor_unit}" >/dev/null 2>&1; then
    echo -e "${YELLOW}Detected Cloudflare Tunnel services; enabling axis-node cloudflared monitoring provider.${NC}"
    upsert_env_value "AXIS_NODE_MONITORING_CF_TUNNEL_ENABLED" "true"
    return 0
  fi

  echo -e "${YELLOW}Cloudflare Tunnel service units are incomplete; keeping AXIS_NODE_MONITORING_CF_TUNNEL_ENABLED unchanged.${NC}"
}

sync_region_zone_from_mapping() {
  local hostname_value prefix mapping region zone wt0_ip wt0_region

  wt0_ip="$(detect_wt0_ipv4 || true)"
  if [[ -n "${wt0_ip}" ]]; then
    wt0_region="$(resolve_region_by_wt0_ip "${wt0_ip}" || true)"
    if [[ -n "${wt0_region}" && "${wt0_region}" != "null" ]]; then
      echo -e "${YELLOW}Detected wt0 IPv4:${NC} ${wt0_ip}"
      echo -e "${YELLOW}Updating AXIS_NODE_REGION in .env to:${NC} ${wt0_region}"
      upsert_env_value "AXIS_NODE_REGION" "${wt0_region}"
    else
      echo -e "${BLUE}wt0 IPv4 detected but no AXIS_WT0_REGION_* prefix matched; falling back to hostname mapping for region/zone.${NC}"
    fi
  else
    echo -e "${BLUE}wt0 IPv4 not found; falling back to hostname mapping for region/zone.${NC}"
  fi

  if [[ ! -f "${REGION_MAPPING_FILE}" ]]; then
    echo -e "${BLUE}Region mapping file not found; keeping existing AXIS_NODE_ZONE in .env.${NC}"
    return 0
  fi

  hostname_value="$(resolve_node_hostname || true)"
  if [[ -z "${hostname_value}" ]]; then
    echo -e "${YELLOW}Hostname not resolved; keeping existing AXIS_NODE_ZONE in .env.${NC}"
    return 0
  fi

  prefix="$(extract_hostname_prefix "${hostname_value}")"
  if [[ -z "${prefix}" ]]; then
    echo -e "${YELLOW}Hostname prefix is empty for ${hostname_value}; keeping existing AXIS_NODE_ZONE in .env.${NC}"
    return 0
  fi

  mapping="$(resolve_region_zone_by_prefix "${prefix}" || true)"
  if [[ -z "${mapping}" ]]; then
    echo -e "${YELLOW}No region mapping found for hostname prefix ${prefix}; keeping existing AXIS_NODE_ZONE in .env.${NC}"
    return 0
  fi

  IFS='|' read -r region zone <<< "${mapping}"
  if [[ -z "${region}" || -z "${zone}" || "${region}" == "null" || "${zone}" == "null" ]]; then
    echo -e "${YELLOW}Incomplete region mapping for hostname prefix ${prefix}; keeping existing AXIS_NODE_ZONE in .env.${NC}"
    return 0
  fi

  echo -e "${YELLOW}Resolved hostname prefix:${NC} ${prefix} (hostname: ${hostname_value})"
  if [[ -z "${wt0_region:-}" ]]; then
    echo -e "${YELLOW}Updating AXIS_NODE_REGION in .env to:${NC} ${region}"
    upsert_env_value "AXIS_NODE_REGION" "${region}"
  fi
  echo -e "${YELLOW}Updating AXIS_NODE_ZONE in .env to:${NC} ${zone}"
  upsert_env_value "AXIS_NODE_ZONE" "${zone}"
}

echo -e "${CYAN}========================================${NC}"
echo -e "${CYAN}      axis-node Update Installer        ${NC}"
echo -e "${CYAN}========================================${NC}"

echo -e "${BLUE}Project root:${NC} ${PROJECT_ROOT}"

sync_region_zone_from_mapping

WT0_IPV4="$(detect_wt0_ipv4 || true)"
if [[ -n "${WT0_IPV4}" ]]; then
  MANAGEMENT_PORT="$(current_management_port)"
  NEW_MANAGEMENT_ADDRESS="${WT0_IPV4}:${MANAGEMENT_PORT}"
  echo -e "${YELLOW}Detected wt0 IPv4:${NC} ${WT0_IPV4}"
  echo -e "${YELLOW}Updating AXIS_NODE_MANAGEMENT_ADDRESS in .env to:${NC} ${NEW_MANAGEMENT_ADDRESS}"
  upsert_env_value "AXIS_NODE_MANAGEMENT_ADDRESS" "${NEW_MANAGEMENT_ADDRESS}"
else
  echo -e "${BLUE}wt0 IPv4 not found; keeping existing AXIS_NODE_MANAGEMENT_ADDRESS in .env.${NC}"
fi

if ! command -v systemctl >/dev/null 2>&1; then
  echo -e "${RED}Error:${NC} systemctl not found."
  exit 1
fi

sync_cloudflared_monitoring_config

ensure_go_toolchain

echo -e "${YELLOW}Building latest axis-node binary...${NC}"
(
  cd "${PROJECT_ROOT}"
  GOTOOLCHAIN=local "${GO_CMD}" build -o "${BUILD_OUTPUT}" ./cmd/axis-node
)

if run_root systemctl list-unit-files "${SERVICE_NAME}" >/dev/null 2>&1; then
  if run_root systemctl is-active --quiet "${SERVICE_NAME}"; then
    echo -e "${YELLOW}Stopping ${SERVICE_NAME} before replacing binary...${NC}"
    run_root systemctl stop "${SERVICE_NAME}"
  else
    echo -e "${BLUE}${SERVICE_NAME} is not active.${NC}"
  fi
else
  echo -e "${BLUE}${SERVICE_NAME} is not installed yet.${NC}"
fi

echo -e "${YELLOW}Installing binary to ${INSTALL_TARGET}...${NC}"
run_root install -m 0755 "${BUILD_OUTPUT}" "${INSTALL_TARGET}"

echo -e "${YELLOW}Installing systemd unit to ${SERVICE_FILE_TARGET}...${NC}"
run_root install -m 0644 "${SERVICE_FILE_SOURCE}" "${SERVICE_FILE_TARGET}"

echo -e "${YELLOW}Reloading systemd daemon...${NC}"
run_root systemctl daemon-reload

echo -e "${YELLOW}Enabling and starting ${SERVICE_NAME}...${NC}"
run_root systemctl enable --now "${SERVICE_NAME}"

echo -e "${YELLOW}Verifying ${SERVICE_NAME} status...${NC}"
if run_root systemctl is-active --quiet "${SERVICE_NAME}"; then
  echo -e "${GREEN}${SERVICE_NAME} is active.${NC}"
else
  echo -e "${RED}${SERVICE_NAME} failed to start.${NC}"
  run_root systemctl status "${SERVICE_NAME}" --no-pager || true
  exit 1
fi

echo -e "${GREEN}axis-node update applied successfully.${NC}"
echo -e "${BLUE}Binary:${NC} ${INSTALL_TARGET}"
echo -e "${BLUE}Unit:${NC} ${SERVICE_FILE_TARGET}"
echo -e "${BLUE}Time:${NC} $(date '+%Y-%m-%d %H:%M:%S')"
