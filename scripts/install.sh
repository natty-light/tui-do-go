#!/usr/bin/env bash
set -euo pipefail

# Installer for tui-do: copies a suitable binary into a directory on your PATH
# Usage:
#   scripts/install.sh [--prefix <dir>] [--force]
#
# Defaults to installing into ~/.local/bin if available, otherwise /usr/local/bin.
# If no prebuilt binary for your platform is found in dist/, it will build from source.

APP_NAME="tui-do"
PROJECT_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
DIST_DIR="${PROJECT_ROOT}/dist"
PREFIX=""
FORCE="0"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --prefix)
      PREFIX="$2"; shift 2;;
    --force)
      FORCE="1"; shift;;
    -h|--help)
      echo "Usage: $0 [--prefix <dir>] [--force]"; exit 0;;
    *)
      echo "Unknown option: $1"; exit 1;;
  esac
done

uname_s="$(uname -s 2>/dev/null || echo unknown)"
uname_m="$(uname -m 2>/dev/null || echo unknown)"
case "$uname_s" in
  Darwin) os="darwin";;
  Linux)  os="linux";;
  MINGW*|MSYS*|CYGWIN*|Windows_NT) os="windows";;
  *) os="unknown";;
esac

case "$uname_m" in
  x86_64|amd64) arch="amd64";;
  arm64|aarch64) arch="arm64";;
  *) arch="$uname_m";;
esac

# Decide install directory
choose_prefix() {
  if [[ -n "$PREFIX" ]]; then
    echo "$PREFIX"; return
  fi
  if [[ -d "$HOME/.local/bin" || -w "$HOME" ]]; then
    echo "$HOME/.local/bin"; return
  fi
  echo "/usr/local/bin"
}

prefix="$(choose_prefix)"
mkdir -p "$prefix"

# Figure out source binary path or build it
src_bin=""

# Prefer existing platform-specific dist binary
if [[ -d "$DIST_DIR" ]]; then
  # Find the newest version subdir
  latest_dir="$(ls -1t "$DIST_DIR" 2>/dev/null | head -n1 || true)"
  if [[ -n "$latest_dir" ]]; then
    case "$os" in
      windows) cand="${DIST_DIR}/${latest_dir}/${APP_NAME}-${os}-${arch}.exe" ;;
      *)       cand="${DIST_DIR}/${latest_dir}/${APP_NAME}-${os}-${arch}" ;;
    esac
    if [[ -f "$cand" ]]; then
      src_bin="$cand"
    fi
  fi
fi

# If not found, check project root compiled binary
if [[ -z "$src_bin" ]]; then
  case "$os" in
    windows) proj_bin="${PROJECT_ROOT}/${APP_NAME}.exe" ;;
    *)       proj_bin="${PROJECT_ROOT}/${APP_NAME}" ;;
  esac
  if [[ -f "$proj_bin" ]]; then
    src_bin="$proj_bin"
  fi
fi

# If still not found, try building from source for current platform
if [[ -z "$src_bin" ]]; then
  echo "No prebuilt binary found; building from source for ${os}/${arch}..."
  export CGO_ENABLED=0
  case "$os" in
    windows) out_name="${APP_NAME}.exe" ;;
    *)       out_name="${APP_NAME}" ;;
  esac
  ( cd "$PROJECT_ROOT" && GOOS="${os}" GOARCH="${arch}" go build -trimpath -ldflags "-s -w" -o "$out_name" ./ )
  src_bin="${PROJECT_ROOT}/${out_name}"
fi

# Destination path
case "$os" in
  windows) dest="${prefix}/${APP_NAME}.exe" ;;
  *)       dest="${prefix}/${APP_NAME}" ;;
esac

# If destination exists
if [[ -f "$dest" && "$FORCE" != "1" ]]; then
  echo "Found existing ${dest}. Use --force to overwrite."
  exit 1
fi

# Copy and make executable
cp "$src_bin" "$dest"
chmod +x "$dest" || true

echo "Installed $APP_NAME to: $dest"

# Ensure the install directory is on PATH; attempt to add automatically if missing
need_export_msg=""
case ":$PATH:" in
  *:"$prefix":*) :;;
  *) need_export_msg="yes";;
esac

if [[ -n "$need_export_msg" ]]; then
  shell_name="$(basename "${SHELL:-}")"
  echo "${prefix} not detected on PATH. Attempting to add it for ${shell_name}..."

  if [[ "$shell_name" == "fish" ]]; then
    # Prefer fish_add_path (Fish 3.2+). This persists via universal vars.
    if command -v fish >/dev/null 2>&1 && fish -c 'functions -q fish_add_path' >/dev/null 2>&1; then
      if fish -c "contains -- $prefix \$fish_user_paths; or contains -- $prefix \$PATH"; then
        echo "Fish: ${prefix} already present in fish paths."
      else
        if fish -c "fish_add_path -U '$prefix'"; then
          echo "Fish: Added ${prefix} to PATH using fish_add_path (universal var)."
        else
          echo "Fish: Failed to run fish_add_path; falling back to config.fish edit."
          mkdir -p "$HOME/.config/fish"
          cfg="$HOME/.config/fish/config.fish"
          line="set -gx PATH $prefix \$PATH"
          if [[ ! -f "$cfg" ]] || ! grep -qsF "$line" "$cfg"; then
            printf "\n# Added by tui-do installer\n%s\n" "$line" >> "$cfg"
          fi
        fi
      fi
    else
      # Older fish: add to PATH in config.fish and set universal var if possible
      if command -v fish >/dev/null 2>&1; then
        if fish -c "contains -- $prefix \$PATH"; then
          echo "Fish: ${prefix} already present in PATH."
        else
          fish -c "set -Ux fish_user_paths '$prefix' \$fish_user_paths" && echo "Fish: Added ${prefix} via fish_user_paths."
          mkdir -p "$HOME/.config/fish"
          cfg="$HOME/.config/fish/config.fish"
          line="set -gx PATH $prefix \$PATH"
          if [[ ! -f "$cfg" ]] || ! grep -qsF "$line" "$cfg"; then
            printf "\n# Added by tui-do installer\n%s\n" "$line" >> "$cfg"
          fi
        fi
      else
        echo "Fish shell not found but SHELL indicates fish; please ensure ${prefix} is on PATH."
      fi
    fi
    echo "Restart your terminal or run: exec fish"
  else
    # POSIX shells / bash / zsh
    profile=""
    case "$shell_name" in
      zsh) profile="$HOME/.zshrc" ;;
      bash) profile="$HOME/.bashrc" ;;
      *) profile="$HOME/.profile" ;;
    esac
    mkdir -p "$(dirname "$profile")"
    line="export PATH=\"$prefix:\$PATH\""
    if [[ -f "$profile" ]] && grep -qsF "$line" "$profile"; then
      echo "${profile} already contains PATH entry for ${prefix}."
    else
      printf "\n# Added by tui-do installer\n%s\n" "$line" >> "$profile"
      echo "Added ${prefix} to PATH in ${profile}."
    fi
    echo "Reload your shell (e.g., 'source ${profile}' or restart) for changes to take effect."
  fi
fi

printf "\nVerify:\n"
echo "  which ${APP_NAME}  # should print ${dest}"
echo "  ${APP_NAME}        # run it"
