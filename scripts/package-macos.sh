#!/bin/bash

set -euo pipefail

readonly project_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
readonly notary_profile="${AGRES_NOTARY_PROFILE:-notarytool}"

app_identity="${AGRES_APP_SIGN_IDENTITY:-}"
installer_identity="${AGRES_INSTALLER_SIGN_IDENTITY:-}"

usage() {
	cat <<'EOF'
Usage: ./scripts/package-macos.sh

Builds agres for the current macOS architecture, signs the executable and
installer with Developer ID, submits the PKG to Apple for notarization, and
staples the notarization ticket. The installer places agres in /usr/local/bin.

Environment variables:
  AGRES_APP_SIGN_IDENTITY        Developer ID Application identity
  AGRES_INSTALLER_SIGN_IDENTITY  Developer ID Installer identity
  AGRES_NOTARY_PROFILE           notarytool Keychain profile (default: notarytool)
EOF
}

die() {
	echo "Error: $*" >&2
	exit 1
}

resolve_identity() {
	local kind="$1" matches count
	matches=$(security find-identity -v 2>/dev/null |
		sed -n "s/.*\"\($kind: [^\"]*\)\".*/\1/p" | sort -u)
	count=$(printf '%s\n' "$matches" | sed '/^$/d' | wc -l | tr -d ' ')
	[ "$count" = "1" ] || die "有効な $kind 証明書を1件に特定できません。環境変数で指定してください。"
	printf '%s' "$matches"
}

while [ "$#" -gt 0 ]; do
	case "$1" in
		-h|--help) usage; exit 0 ;;
		*) usage >&2; die "unknown option: $1" ;;
	esac
done

[ "$(uname -s)" = "Darwin" ] || die "このスクリプトはmacOS専用です。"
for command_name in codesign go pkgbuild pkgutil security xcrun spctl; do
	command -v "$command_name" >/dev/null || die "$command_name が見つかりません。"
done

[ -n "$app_identity" ] || app_identity=$(resolve_identity "Developer ID Application")
[ -n "$installer_identity" ] || installer_identity=$(resolve_identity "Developer ID Installer")
xcrun notarytool history --keychain-profile "$notary_profile" >/dev/null 2>&1 ||
	die "notarytoolのKeychain profile '$notary_profile' を利用できません。"

cd "$project_dir"

version=$(sed -n 's/^var version = "\([^"]*\)".*/\1/p' main.go)
[ -n "$version" ] || die "main.go からバージョンを取得できません。"
[ "$(printf '%s\n' "$version" | wc -l | tr -d ' ')" = "1" ] ||
	die "main.go のバージョンを1件に特定できません。"

architecture=$(uname -m)
package_name="agres-$version-macos-$architecture.pkg"
package_path="$project_dir/dist/$package_name"
package_temp_dir=$(mktemp -d "/tmp/agres-pkg.XXXXXX")
trap 'rm -rf "$package_temp_dir"' EXIT HUP INT TERM
package_root="$package_temp_dir/root"
package_binary="$package_root/usr/local/bin/agres"
unsigned_package="$package_temp_dir/$package_name"

mkdir -p "$(dirname "$package_binary")"
echo "Building agres $version for macOS $architecture..."
CGO_ENABLED=0 go build -trimpath -ldflags '-s -w' -o "$package_binary" .

built_version=$("$package_binary" --version)
[ "$built_version" = "agres $version" ] ||
	die "ビルドしたagresのバージョンは '$built_version' です。期待値: 'agres $version'"

echo "Signing executable with: $app_identity"
codesign --force --timestamp --options runtime \
	--sign "$app_identity" "$package_binary"
codesign --verify --strict --verbose=2 "$package_binary"
codesign --display --verbose=2 "$package_binary" 2>&1 |
	sed -n '/^Identifier=/p;/^TeamIdentifier=/p;/^Runtime Version=/p'

pkgbuild \
	--root "$package_root" \
	--identifier "jp.fukuyori.agres.pkg" \
	--version "$version" \
	--ownership recommended \
	--install-location / \
	--sign "$installer_identity" \
	"$unsigned_package"

mkdir -p dist
mv -f "$unsigned_package" "$package_path"
pkgutil --check-signature "$package_path"

echo "Submitting to Apple notarization service..."
xcrun notarytool submit "$package_path" \
	--keychain-profile "$notary_profile" \
	--wait
xcrun stapler staple "$package_path"
xcrun stapler validate "$package_path"
spctl --assess --type install --verbose=4 "$package_path"

echo "Packaged and notarized: $package_path"
