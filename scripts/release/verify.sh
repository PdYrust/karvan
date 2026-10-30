#!/bin/sh
# Checks the release files in DIR (default dist/) against their SHA256SUMS:
# every archive must be listed exactly once and match. Exit 0 when
# all match, 1 otherwise.
set -eu
DIR=${1:-$(cd "$(dirname "$0")/../.." && pwd)/dist}
cd "$DIR"
[ -f SHA256SUMS ] || { echo "no SHA256SUMS in $DIR" >&2 && exit 1; }
status=0
for f in karvan_*_linux_*.tar.gz; do
	[ -f "$f" ] || { echo "missing: $f" >&2 && status=1 && continue; }
	n=$(awk -v f="$f" '$2 == f { n++ } END { print n + 0 }' SHA256SUMS)
	[ "$n" = 1 ] || { echo "SHA256SUMS lists $f $n times" >&2 && status=1; }
done
sha256sum -c --strict --quiet SHA256SUMS || status=1
[ $status = 0 ] && echo "SHA256SUMS: all files match"
exit $status
