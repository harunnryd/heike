#!/usr/bin/env sh
set -eu

REPO="${HEIKE_REPO:-harunnryd/heike}"
REQUESTED_VERSION="${HEIKE_VERSION:-latest}"
INSTALL_DIR="${HEIKE_INSTALL_DIR:-/usr/local/bin}"
VERIFY_CHECKSUM="${HEIKE_VERIFY_CHECKSUM:-1}"
RELEASE_BASE_URL="${HEIKE_RELEASE_BASE_URL:-https://github.com/${REPO}/releases/download}"

require_cmd() {
	command -v "$1" >/dev/null 2>&1 || {
		echo "error: required command not found: $1" >&2
		exit 1
	}
}

detect_os() {
	os_name="$(uname -s | tr '[:upper:]' '[:lower:]')"
	case "$os_name" in
	linux)
		echo "linux"
		;;
	darwin)
		echo "darwin"
		;;
	*)
		echo "error: unsupported OS: $os_name" >&2
		exit 1
		;;
	esac
}

detect_arch() {
	arch_name="$(uname -m)"
	case "$arch_name" in
	x86_64|amd64)
		echo "amd64"
		;;
	arm64|aarch64)
		echo "arm64"
		;;
	*)
		echo "error: unsupported architecture: $arch_name" >&2
		exit 1
		;;
	esac
}

resolve_tag() {
	if [ "$REQUESTED_VERSION" != "latest" ]; then
		echo "$REQUESTED_VERSION"
		return
	fi

	latest_json="$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest")"
	tag="$(printf '%s' "$latest_json" | sed -n 's/.*"tag_name":[[:space:]]*"\([^"]*\)".*/\1/p' | head -n1)"
	if [ -z "$tag" ]; then
		echo "error: failed to resolve latest release tag for ${REPO}" >&2
		exit 1
	fi
	echo "$tag"
}

resolve_install_dir() {
	target="$INSTALL_DIR"
	if [ ! -d "$target" ]; then
		mkdir -p "$target" 2>/dev/null || true
	fi

	if [ -w "$target" ]; then
		echo "$target"
		return
	fi

	fallback="${HOME}/.local/bin"
	mkdir -p "$fallback"
	echo "$fallback"
}

compute_sha256() {
	file="$1"
	if command -v sha256sum >/dev/null 2>&1; then
		sha256sum "$file" | awk '{print $1}'
		return
	fi
	if command -v shasum >/dev/null 2>&1; then
		shasum -a 256 "$file" | awk '{print $1}'
		return
	fi

	echo "error: missing sha256 tool (need sha256sum or shasum)" >&2
	exit 1
}

verify_archive_checksum() {
	checksums_file="$1"
	archive_file="$2"
	asset_name="$3"

	expected="$(awk -v file="$asset_name" '$2 == file {print $1}' "$checksums_file" | head -n1)"
	if [ -z "$expected" ]; then
		echo "error: checksum entry for ${asset_name} not found" >&2
		exit 1
	fi

	actual="$(compute_sha256 "$archive_file")"
	if [ "$expected" != "$actual" ]; then
		echo "error: checksum mismatch for ${asset_name}" >&2
		exit 1
	fi
}

require_cmd curl
require_cmd tar

os="$(detect_os)"
arch="$(detect_arch)"
tag="$(resolve_tag)"
asset="heike_${os}_${arch}.tar.gz"
archive_url="${RELEASE_BASE_URL}/${tag}/${asset}"
checksums_url="${RELEASE_BASE_URL}/${tag}/checksums.txt"

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT INT HUP

archive_path="${tmpdir}/${asset}"
checksums_path="${tmpdir}/checksums.txt"

echo "==> Downloading ${asset} (${tag})"
curl -fsSL "$archive_url" -o "$archive_path"

if [ "$VERIFY_CHECKSUM" = "1" ]; then
	echo "==> Verifying checksum"
	curl -fsSL "$checksums_url" -o "$checksums_path"
	verify_archive_checksum "$checksums_path" "$archive_path" "$asset"
fi

echo "==> Extracting"
tar -xzf "$archive_path" -C "$tmpdir"

if [ ! -f "${tmpdir}/heike" ]; then
	echo "error: release archive did not contain expected 'heike' binary" >&2
	exit 1
fi

target_dir="$(resolve_install_dir)"
target_bin="${target_dir}/heike"

echo "==> Installing to ${target_bin}"
cp "${tmpdir}/heike" "$target_bin"
chmod +x "$target_bin"

echo "==> Installed successfully"
echo "Run: heike version"
if [ "$target_dir" = "${HOME}/.local/bin" ]; then
	echo "Note: add ${HOME}/.local/bin to your PATH if needed."
fi
