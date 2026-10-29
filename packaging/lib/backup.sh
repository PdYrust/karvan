# Copies of the data folder taken by an update (D23) and put back by a
# rollback across a schema change (D24).
# shellcheck shell=sh
BACKUP_DIR=/var/lib/karvan-backups
# take_backup copies $DATA_DIR to $BACKUP_DIR/<$1>-<UTC time>/ while the
# service is stopped, keeps the two newest and prints the new one's name.
# Each step is checked, since set -e does not hold where it is called.
take_backup() {
	install -d -m 0700 -o root -g root "$BACKUP_DIR" && chmod 0700 "$BACKUP_DIR" || return 1
	name=$1-$(date -u +%Y%m%dT%H%M%SZ)
	rm -rf "$BACKUP_DIR/.new"
	if ! cp -a "$DATA_DIR" "$BACKUP_DIR/.new"; then
		rm -rf "$BACKUP_DIR/.new"
		return 1
	fi
	chmod 0700 "$BACKUP_DIR/.new" && rm -rf "${BACKUP_DIR:?}/$name" && mv "$BACKUP_DIR/.new" "$BACKUP_DIR/$name" || return 1
	prune_backups
	echo "$name"
}
# backups prints "<UTC time> <name>" for every backup, oldest first. The
# time is the part after the last dash, so it sorts as text.
backups() {
	for d in "$BACKUP_DIR"/*; do
		[ -d "$d" ] || continue
		n=${d##*/}
		case ${n##*-} in
		[0-9][0-9][0-9][0-9][0-9][0-9][0-9][0-9]T[0-9][0-9][0-9][0-9][0-9][0-9]Z) echo "${n##*-} $n" ;;
		esac
	done | sort
}
prune_backups() {
	backups | sed '$d' | sed '$d' | while read -r _ n; do
		rm -rf "${BACKUP_DIR:?}/$n"
	done
}
# backup_of prints the newest backup taken while version $1 was running.
backup_of() {
	backups | while read -r _ n; do
		[ "${n%-*}" = "$1" ] && echo "$n"
	done | tail -n 1
}
# backup_time prints the time in backup name $1 for people to read.
backup_time() {
	echo "${1##*-}" | sed 's/^\(....\)\(..\)\(..\)T\(..\)\(..\)\(..\)Z$/\1-\2-\3 \4:\5:\6 UTC/'
}
# restore_backup replaces $DATA_DIR with a copy of backup $1. The backup
# itself stays. Run it with the service stopped.
restore_backup() {
	rm -rf "$DATA_DIR.restore"
	cp -a "$BACKUP_DIR/$1" "$DATA_DIR.restore"
	chmod 0700 "$DATA_DIR.restore"
	rm -rf "$DATA_DIR.old"
	mv "$DATA_DIR" "$DATA_DIR.old"
	mv "$DATA_DIR.restore" "$DATA_DIR"
	rm -rf "$DATA_DIR.old"
}
