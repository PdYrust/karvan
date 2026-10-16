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
INSERT INTO jwt VALUES(1,'72f3b496b047e41e34ce80e9dfeaa27d32d2b11e069ecc32b39634aaa59b5203');
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
INSERT INTO tls VALUES(1,replace('-----BEGIN PRIVATE KEY-----\n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\nz0MjHx3jNOqyEcr4xbSDAvjwaWKCsQ==\n-----END PRIVATE KEY-----\n','\n',char(10)),replace('-----BEGIN CERTIFICATE-----\nMIIEnDCCAoQCAQAwDQYJKoZIhvcNAQENBQAwEzERMA8GA1UEAwwIR296YXJnYWgw\nIBcNMjYxMDE2MTMzOTI0WhgPMjEyNjA5MjIxMzM5MjRaMBMxETAPBgNVBAMMCEdv\nemFyZ2FoMIICIjANBgkqhkiG9w0BAQEFAAOCAg8AMIICCgKCAgEAxHIJWSYLpORi\nVp5mGo3O7BFRwUkzQJqEbNl5DtEYag+8BKvG+wSUpoR+T+ygEpUh9dBqVR4hcmWk\n7W3Zl9OWy9394w2ghQpYpBPYyDPlv04bcQv9dUjyArCEoylaPq0SjPs5dZryMy70\nURTNYmdjduNZY80AIU6lnJVr4uvfF0b5MnLTnzjD9jJDhS1XPksRsxENON/67U9d\nB6Gdy8z5WCNdBFid+++0iRHqL1G/Ui/APpRPjw4AyjFboHn2AaLx9zUlDjb4sGMU\npyFoGeM7jGCruwVXnJkrWiAPO8WniPYCOt3r7oJGrc1SJwaT/SesEfoD6A/FepM/\nmlqM2IKVNwNf4PPLTxccbC2ySl1ioNVQiW3rYV/sRPRrAnZQPjvyq0BoJUvfvuJv\nBpUSb72XdQEW7XOpDJcnf9J2wOWbn6rGSjRIt6eqWOcGFQqD6O0MZMojZsKf2Wc/\nF2+YKT2/Am84ehiEv2v0AL06JKN62EitjitC25ICJ/V9IS++ElMQUNJcbs7IzvuD\nrItsWcaYLrfs/EFKJ/CMQjfeHxR0CCPJcUSXVzCIS0bY/Af3XqRAaJO4OtjBnaOD\nTmNNKh7p27Xf9iIVUfv3bxXPOobQlXKuMkQIMDyHRKvot+gtsMqWQT4e7N0gBkgF\nYAO1LQKx087sjxpI//f9kHawMpYoeaMCAwEAATANBgkqhkiG9w0BAQ0FAAOCAgEA\nof3KXiZk3RAXZ51R9xmFdPzF3hIcDc1KB1ri0e+bXv0YsYw34XNbkikyBZ9DFin5\nk8yw1WE6bBJDK1e2t4ghHcslFDA6UqAchl2m4b6M/QMI8J6tPCamOxEY1HsFA00c\n6YAY/3BguTgLDAKAH8Oa3YYbPrOGTl6n07/6aPZyrI2JvqyX2rwhso3GenyP9uDL\nwTYkKDHm75H6DyECzn9RPWa+p0sMlvdtv5PACE1delRzpyRJSCUFo9anzCMWkHEG\nEdy8yciwPukpsdaT3DcdtIWrzQaVaPbj4/PwIk1nhwdNwBPbvYzxGIGYzwZQP7cc\nE3TiYItUeG/PpZClSK52pE1qQjqJTaz6YQn3VlKSytOgvV0F2VhZdYNde9AENq+b\nnDto1PTrhqyBQPfJzXNGkksCf3LTIfaIebhX0fN+jawTxvApwEcXqwnNBEEgAfB9\nnqz/CdR1/s3a7kKcDje2tRy8QCquHK3ufR9MAtYSAnZj4AvJ0sHffvqDZiwSzAtS\nZajNVks2fblgHeXrWrXm09JxxyV6Fvz/5jhs4oW2f12cGdsvR9ykBaq5wAYgJW/D\nN/pEseAq+PTzh+0JLx6dAQUFhhaQr1mbsn58E4/8dgCchmcoN3X3Ug5FB0JWQ2CP\nvzuRCnenIRqQMm3S55NU2BY0NOxfnBqO1y0UqWWRm3g=\n-----END CERTIFICATE-----\n','\n',char(10)));
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
