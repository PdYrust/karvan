# Where Karvan's files go and the steps that put them there.
# shellcheck shell=sh

LIB_DIR=/usr/local/lib/karvan
BIN_LINK=/usr/local/bin/karvan
DATA_DIR=/var/lib/karvan
ETC_DIR=/etc/karvan
CONFIG=$ETC_DIR/config.json
UNIT=/etc/systemd/system/karvan.service

# install_version copies the archive folder $1 to $LIB_DIR/$2. It is built
# next to the target and moved in at the end, so a failed copy leaves no
# half-filled version folder behind.
install_version() {
	src=$1
	dst=$LIB_DIR/$2
	tmp=$LIB_DIR/.new-$2
	install -d -m 0755 -o root -g root "$LIB_DIR"
	rm -rf "$tmp"
	install -d -m 0755 -o root -g root "$tmp" "$tmp/lib"
	install -m 0755 -o root -g root "$src/karvan" "$src/install.sh" "$src/get.sh" "$tmp/"
	install -m 0644 -o root -g root "$src"/lib/*.sh "$tmp/lib/"
	for f in karvan.service README.md LICENSE; do
		[ -f "$src/$f" ] && install -m 0644 -o root -g root "$src/$f" "$tmp/"
	done
	rm -rf "$dst"
	mv "$tmp" "$dst"
}

# switch_current points $LIB_DIR/current at version $1 in one rename, and
# /usr/local/bin/karvan at the binary behind it.
switch_current() {
	ln -sfn "$1" "$LIB_DIR/.current.new"
	mv -T "$LIB_DIR/.current.new" "$LIB_DIR/current"
	ln -sfn "$LIB_DIR/current/karvan" "$BIN_LINK"
}

# version_of prints the version link $1 (current or previous) points at.
version_of() {
	readlink "$LIB_DIR/$1" 2>/dev/null || true
}
# set_previous points $LIB_DIR/previous at version $1.
set_previous() {
	ln -sfn "$1" "$LIB_DIR/.previous.new"
	mv -T "$LIB_DIR/.previous.new" "$LIB_DIR/previous"
}
# prune_versions deletes every version folder but current and previous.
prune_versions() {
	keep_cur=$(version_of current)
	keep_prev=$(version_of previous)
	for d in "$LIB_DIR"/*; do
		if [ ! -d "$d" ] || [ -L "$d" ]; then
			continue
		fi
		v=${d##*/}
		[ "$v" = "$keep_cur" ] || [ "$v" = "$keep_prev" ] || rm -rf "$d"
	done
}
# schema_of prints the store schema version the binary of version $1
# migrates to (karvan version -json).
schema_of() {
	"$LIB_DIR/$1/karvan" version -json 2>/dev/null | sed -n 's/.*"schema":\([0-9]*\).*/\1/p'
}
# make_dirs creates the data and config folders for root only and writes a
# config file with the defaults when there is none.
make_dirs() {
	install -d -m 0700 -o root -g root "$DATA_DIR" "$ETC_DIR"
	chmod 0700 "$DATA_DIR" "$ETC_DIR"
	if [ ! -f "$CONFIG" ]; then
		(
			umask 077
			cat >"$CONFIG" <<-EOF
				{
				  "dataDir": "$DATA_DIR",
				  "keyFile": "$ETC_DIR/secret.key",
				  "logLevel": "info",
				  "panelControl": "auto",
				  "acmeHttpPort": 80
				}
			EOF
		)
	fi
	chmod 0600 "$CONFIG"
}

# put_unit copies the unit of the current version in place.
put_unit() {
	install -m 0644 -o root -g root "$LIB_DIR/current/karvan.service" "$UNIT"
	systemctl daemon-reload
}

# remove_path deletes $1 when it is there, and says so when it is not.
remove_path() {
	if [ -e "$1" ] || [ -L "$1" ]; then
		rm -rf "$1"
	else
		detail "$1 was already gone" >&2
	fi
}
