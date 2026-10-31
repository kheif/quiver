#!/bin/sh
# Installs quiver from its GitHub release.
#
#   curl -fsSL https://raw.githubusercontent.com/kheif/quiver/main/install.sh | sh
#
# QUIVER_VERSION    release to install, such as v1.0.0 (default: the latest)
# QUIVER_INSTALL_DIR  where to put the binary (default: ~/.local/bin)
# QUIVER_DOWNLOAD_URL folder with the archives and checksums.txt, for mirrors
#                     and tests (default: the GitHub release)
set -eu

repo="kheif/quiver"
version="${QUIVER_VERSION:-latest}"
dir="${QUIVER_INSTALL_DIR:-$HOME/.local/bin}"

fail() {
	echo "quiver install: $*" >&2
	exit 1
}

case "$(uname -s)" in
Darwin) os=darwin ;;
Linux) os=linux ;;
*) fail "unsupported system $(uname -s); on Windows, install in PowerShell with: irm https://raw.githubusercontent.com/kheif/quiver/main/install.ps1 | iex" ;;
esac
case "$(uname -m)" in
x86_64 | amd64) arch=amd64 ;;
arm64 | aarch64) arch=arm64 ;;
*) fail "unsupported CPU $(uname -m)" ;;
esac

if [ -n "${QUIVER_DOWNLOAD_URL:-}" ]; then
	base="$QUIVER_DOWNLOAD_URL"
elif [ "$version" = latest ]; then
	base="https://github.com/$repo/releases/latest/download"
else
	base="https://github.com/$repo/releases/download/$version"
fi
archive="quiver_${os}_${arch}.tar.gz"

if command -v sha256sum >/dev/null 2>&1; then
	sha256() { sha256sum "$1" | cut -d ' ' -f 1; }
elif command -v shasum >/dev/null 2>&1; then
	sha256() { shasum -a 256 "$1" | cut -d ' ' -f 1; }
else
	fail "sha256sum or shasum is needed to check the download"
fi

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

echo "Downloading $archive ($version)..."
curl -fsSL -o "$tmp/$archive" "$base/$archive" || fail "could not download $base/$archive"
curl -fsSL -o "$tmp/checksums.txt" "$base/checksums.txt" || fail "could not download $base/checksums.txt"

want="$(awk -v f="$archive" '$2 == f { print $1 }' "$tmp/checksums.txt")"
[ -n "$want" ] || fail "$archive is not in checksums.txt"
[ "$(sha256 "$tmp/$archive")" = "$want" ] || fail "checksum mismatch for $archive; the download is broken or was changed"

tar -xzf "$tmp/$archive" -C "$tmp" quiver
mkdir -p "$dir"
# Move, not copy over, so a running quiver keeps its old file.
mv -f "$tmp/quiver" "$dir/quiver"
chmod 755 "$dir/quiver"

echo "Installed $("$dir/quiver" --version) to $dir/quiver"
case ":$PATH:" in
*":$dir:"*) ;;
*)
	echo
	echo "$dir is not on your PATH. Add this line to your shell profile (~/.zshrc or ~/.bashrc):"
	echo "  export PATH=\"$dir:\$PATH\""
	;;
esac
echo
echo "Next: quiver mcp install"
