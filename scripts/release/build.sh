#!/bin/sh
# Builds the release files for VERSION into OUT (default dist/): the web
# panel, karvan_<version>_linux_<arch>.tar.gz for amd64 and arm64 (D16),
# get.sh with the version stamped in, SHA256SUMS over all of them, and
# image/linux_<arch>/karvan, the build context of packaging/docker/Dockerfile.
# The archives are packed the same way every time: sorted names, root owner,
# the commit time as file time and no gzip name or time.
set -eu
ROOT=$(cd "$(dirname "$0")/../.." && pwd)
VERSION=${VERSION:?set VERSION, for example VERSION=0.5.0-beta}
VERSION=${VERSION#v}
case $VERSION in
[0-9]*[!0-9A-Za-z.+-]* | [!0-9]*) echo "not a version: $VERSION" >&2 && exit 1 ;;
esac
OUT=${OUT:-$ROOT/dist}
mtime=$(git -C "$ROOT" log -1 --format=%ct)
rm -rf "$OUT"
mkdir -p "$OUT/image"
[ -n "${SKIP_WEB:-}" ] || make -s -C "$ROOT" web
sed "s/^VERSION=.*/VERSION=$VERSION/" "$ROOT/packaging/get.sh" >"$OUT/get.sh"
chmod 0755 "$OUT/get.sh"
for arch in amd64 arm64; do
	name=karvan_${VERSION}_linux_$arch
	dir=$OUT/$name
	mkdir -p "$dir/lib" "$OUT/image/linux_$arch"
	GOOS=linux GOARCH=$arch make -s -C "$ROOT" build-go VERSION="$VERSION" BIN="$dir/karvan"
	cp "$dir/karvan" "$OUT/image/linux_$arch/karvan"
	cp "$OUT/get.sh" "$ROOT/packaging/install.sh" "$ROOT/packaging/karvan.service" "$dir/"
	cp "$ROOT"/packaging/lib/*.sh "$dir/lib/"
	cp "$ROOT/README.md" "$ROOT/LICENSE" "$dir/"
	chmod 0755 "$dir/install.sh"
	tar --sort=name --owner=0 --group=0 --numeric-owner --mode=u+rwX,go+rX,go-w --mtime="@$mtime" \
		-cf - -C "$OUT" "$name" | gzip -n -9 >"$OUT/$name.tar.gz"
	rm -rf "$dir"
done
(cd "$OUT" && sha256sum karvan_"${VERSION}"_linux_*.tar.gz get.sh >SHA256SUMS)
echo "built $VERSION in $OUT:"
(cd "$OUT" && ls -l -- *.tar.gz get.sh SHA256SUMS)
