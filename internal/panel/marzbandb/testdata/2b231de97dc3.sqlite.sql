PRAGMA foreign_keys=OFF;
BEGIN TRANSACTION;
CREATE TABLE alembic_version (
	version_num VARCHAR(32) NOT NULL, 
	CONSTRAINT alembic_version_pkc PRIMARY KEY (version_num)
);
INSERT INTO alembic_version VALUES('2b231de97dc3');
CREATE TABLE system (
	id INTEGER NOT NULL, 
	uplink BIGINT, 
	downlink BIGINT, 
	PRIMARY KEY (id)
);
INSERT INTO system VALUES(1,0,0);
CREATE TABLE jwt (
	id INTEGER NOT NULL, 
	secret_key VARCHAR(64) NOT NULL, 
	PRIMARY KEY (id)
);
CREATE TABLE proxies (
	id INTEGER NOT NULL, 
	user_id INTEGER, 
	type VARCHAR(11) NOT NULL, 
	settings JSON NOT NULL, 
	PRIMARY KEY (id), 
	FOREIGN KEY(user_id) REFERENCES users (id)
);
CREATE TABLE admins (
	id INTEGER NOT NULL, 
	username VARCHAR(34), 
	hashed_password VARCHAR(128), 
	created_at DATETIME, is_sudo BOOLEAN DEFAULT (0), password_reset_at DATETIME, telegram_id BIGINT, discord_webhook VARCHAR(1024), users_usage BIGINT DEFAULT '0' NOT NULL, 
	PRIMARY KEY (id)
);
CREATE TABLE inbounds (
	id INTEGER NOT NULL, 
	tag VARCHAR(256) NOT NULL, 
	PRIMARY KEY (id)
);
CREATE TABLE exclude_inbounds_association (
	proxy_id INTEGER, 
	inbound_tag VARCHAR(256), 
	FOREIGN KEY(inbound_tag) REFERENCES inbounds (tag), 
	FOREIGN KEY(proxy_id) REFERENCES proxies (id)
);
CREATE TABLE user_usage_logs (
	id INTEGER NOT NULL, 
	user_id INTEGER, 
	used_traffic_at_reset BIGINT NOT NULL, 
	reset_at DATETIME, 
	PRIMARY KEY (id), 
	FOREIGN KEY(user_id) REFERENCES users (id)
);
CREATE TABLE template_inbounds_association (
	user_template_id INTEGER, 
	inbound_tag VARCHAR(256), 
	FOREIGN KEY(inbound_tag) REFERENCES inbounds (tag), 
	FOREIGN KEY(user_template_id) REFERENCES user_templates (id)
);
CREATE TABLE node_usages (
	id INTEGER NOT NULL, 
	created_at DATETIME NOT NULL, 
	node_id INTEGER, 
	uplink BIGINT, 
	downlink BIGINT, 
	PRIMARY KEY (id), 
	FOREIGN KEY(node_id) REFERENCES nodes (id), 
	UNIQUE (created_at, node_id)
);
CREATE TABLE node_user_usages (
	id INTEGER NOT NULL, 
	created_at DATETIME NOT NULL, 
	user_id INTEGER, 
	node_id INTEGER, 
	used_traffic BIGINT, 
	PRIMARY KEY (id), 
	FOREIGN KEY(node_id) REFERENCES nodes (id), 
	FOREIGN KEY(user_id) REFERENCES users (id), 
	UNIQUE (created_at, user_id, node_id)
);
CREATE TABLE notification_reminders (
	id INTEGER NOT NULL, 
	user_id INTEGER, 
	type VARCHAR(15) NOT NULL, 
	expires_at DATETIME, 
	created_at DATETIME, threshold INTEGER, 
	PRIMARY KEY (id), 
	FOREIGN KEY(user_id) REFERENCES users (id)
);
CREATE TABLE tls (
	id INTEGER NOT NULL, 
	"key" VARCHAR(4096) NOT NULL, 
	certificate VARCHAR(2048) NOT NULL, 
	PRIMARY KEY (id)
);
CREATE TABLE IF NOT EXISTS "nodes" (
	id INTEGER NOT NULL, 
	name VARCHAR(256), 
	address VARCHAR(256) NOT NULL, 
	port INTEGER NOT NULL, 
	api_port INTEGER NOT NULL, 
	status VARCHAR(10) NOT NULL, 
	last_status_change DATETIME, 
	message VARCHAR(1024), 
	created_at DATETIME, 
	uplink BIGINT, 
	downlink BIGINT, 
	xray_version VARCHAR(32), 
	usage_coefficient FLOAT DEFAULT (1.0) NOT NULL, 
	PRIMARY KEY (id), 
	UNIQUE (name)
);
CREATE TABLE IF NOT EXISTS "user_templates" (
	id INTEGER NOT NULL, 
	name VARCHAR(64) NOT NULL, 
	data_limit INTEGER, 
	expire_duration INTEGER, 
	username_prefix VARCHAR(20), 
	username_suffix VARCHAR(20), 
	PRIMARY KEY (id), 
	UNIQUE (name)
);
CREATE TABLE IF NOT EXISTS "users" (
	id INTEGER NOT NULL, 
	username VARCHAR(34), 
	status VARCHAR(8) NOT NULL, 
	used_traffic BIGINT, 
	data_limit BIGINT, 
	expire INTEGER, 
	created_at DATETIME DEFAULT (CURRENT_TIMESTAMP), 
	admin_id INTEGER, 
	data_limit_reset_strategy VARCHAR(8) DEFAULT 'no_reset' NOT NULL, 
	sub_revoked_at DATETIME, 
	note VARCHAR(500), 
	sub_updated_at DATETIME, 
	sub_last_user_agent VARCHAR(64), 
	online_at DATETIME, 
	edit_at DATETIME, 
	on_hold_timeout DATETIME, 
	on_hold_expire_duration BIGINT, auto_delete_in_days INTEGER, last_status_change DATETIME, 
	PRIMARY KEY (id), 
	CONSTRAINT fk_users_admin_id_admins FOREIGN KEY(admin_id) REFERENCES admins (id)
);
CREATE TABLE next_plans (
	id INTEGER NOT NULL, 
	user_id INTEGER NOT NULL, 
	data_limit BIGINT NOT NULL, 
	expire INTEGER, 
	add_remaining_traffic BOOLEAN DEFAULT '0' NOT NULL, 
	fire_on_either BOOLEAN DEFAULT '0' NOT NULL, 
	PRIMARY KEY (id), 
	FOREIGN KEY(user_id) REFERENCES users (id)
);
CREATE TABLE admin_usage_logs (
	id INTEGER NOT NULL, 
	admin_id INTEGER, 
	used_traffic_at_reset BIGINT NOT NULL, 
	reset_at DATETIME, 
	PRIMARY KEY (id), 
	FOREIGN KEY(admin_id) REFERENCES admins (id)
);
CREATE TABLE IF NOT EXISTS "hosts" (
	id INTEGER NOT NULL, 
	remark VARCHAR(256) NOT NULL, 
	address VARCHAR(256) NOT NULL, 
	port INTEGER, 
	inbound_tag VARCHAR(256) NOT NULL, 
	sni VARCHAR(1000), 
	host VARCHAR(1000), 
	security VARCHAR(15) DEFAULT 'inbound_default' NOT NULL, 
	alpn VARCHAR(14) DEFAULT 'none' NOT NULL, 
	fingerprint VARCHAR(10) DEFAULT 'none' NOT NULL, 
	allowinsecure BOOLEAN, 
	is_disabled BOOLEAN, 
	path VARCHAR(256), 
	mux_enable BOOLEAN DEFAULT '0' NOT NULL, 
	fragment_setting VARCHAR(100), 
	random_user_agent BOOLEAN DEFAULT '0' NOT NULL, 
	noise_setting VARCHAR(2000), use_sni_as_host BOOLEAN DEFAULT '0' NOT NULL, 
	PRIMARY KEY (id), 
	FOREIGN KEY(inbound_tag) REFERENCES inbounds (tag)
);
CREATE UNIQUE INDEX ix_admins_username ON admins (username);
CREATE UNIQUE INDEX ix_inbounds_tag ON inbounds (tag);
CREATE UNIQUE INDEX ix_users_username ON users (username);
COMMIT;
