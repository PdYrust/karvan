PRAGMA foreign_keys=OFF;
BEGIN TRANSACTION;
CREATE TABLE alembic_version (
	version_num VARCHAR(32) NOT NULL, 
	CONSTRAINT alembic_version_pkc PRIMARY KEY (version_num)
);
INSERT INTO alembic_version VALUES('48a6bcb8bba1');
CREATE TABLE jwt (
	id INTEGER NOT NULL, 
	secret_key VARCHAR(64) NOT NULL, 
	CONSTRAINT pk_jwt PRIMARY KEY (id)
);
CREATE TABLE inbounds (
	id INTEGER NOT NULL, 
	tag VARCHAR(256) NOT NULL, 
	CONSTRAINT pk_inbounds PRIMARY KEY (id)
);
CREATE TABLE inbounds_groups_association (
	inbound_id INTEGER NOT NULL, 
	group_id INTEGER NOT NULL, 
	CONSTRAINT pk_inbounds_groups_association PRIMARY KEY (inbound_id, group_id), 
	CONSTRAINT fk_inbounds_groups_association_group_id_groups FOREIGN KEY(group_id) REFERENCES groups (id), 
	CONSTRAINT fk_inbounds_groups_association_inbound_id_inbounds FOREIGN KEY(inbound_id) REFERENCES inbounds (id)
);
CREATE TABLE template_group_association (
	user_template_id INTEGER, 
	group_id INTEGER, 
	CONSTRAINT fk_template_group_association_group_id_groups FOREIGN KEY(group_id) REFERENCES groups (id), 
	CONSTRAINT fk_template_group_association_user_template_id_user_templates FOREIGN KEY(user_template_id) REFERENCES user_templates (id)
);
CREATE TABLE users_groups_association (
	user_id INTEGER NOT NULL, 
	groups_id INTEGER NOT NULL, 
	CONSTRAINT pk_users_groups_association PRIMARY KEY (user_id, groups_id), 
	CONSTRAINT fk_users_groups_association_groups_id_groups FOREIGN KEY(groups_id) REFERENCES groups (id), 
	CONSTRAINT fk_users_groups_association_user_id_users FOREIGN KEY(user_id) REFERENCES users (id)
);
CREATE TABLE IF NOT EXISTS "admin_usage_logs" (
	id INTEGER NOT NULL, 
	admin_id INTEGER NOT NULL, 
	used_traffic_at_reset BIGINT NOT NULL, 
	reset_at DATETIME NOT NULL, 
	CONSTRAINT pk_admin_usage_logs PRIMARY KEY (id), 
	CONSTRAINT fk_admin_usage_logs_admin_id_admins FOREIGN KEY(admin_id) REFERENCES admins (id)
);
CREATE TABLE IF NOT EXISTS "groups" (
	id INTEGER NOT NULL, 
	name VARCHAR(64) NOT NULL, 
	is_disabled BOOLEAN DEFAULT '0' NOT NULL, 
	CONSTRAINT pk_groups PRIMARY KEY (id)
);
CREATE TABLE IF NOT EXISTS "system" (
	id INTEGER NOT NULL, 
	uplink BIGINT NOT NULL, 
	downlink BIGINT NOT NULL, 
	CONSTRAINT pk_system PRIMARY KEY (id)
);
INSERT INTO system VALUES(1,0,0);
CREATE TABLE IF NOT EXISTS "user_templates" (
	id INTEGER NOT NULL, 
	name VARCHAR(64) NOT NULL, 
	data_limit BIGINT NOT NULL, 
	expire_duration BIGINT NOT NULL, 
	username_prefix VARCHAR(20), 
	username_suffix VARCHAR(20), extra_settings JSON, status VARCHAR(7) DEFAULT 'active' NOT NULL, reset_usages BOOLEAN DEFAULT '0' NOT NULL, on_hold_timeout INTEGER, data_limit_reset_strategy VARCHAR(8) DEFAULT 'no_reset' NOT NULL, is_disabled BOOLEAN DEFAULT '0' NOT NULL, hwid_limit BIGINT, 
	CONSTRAINT pk_user_templates PRIMARY KEY (id), 
	CONSTRAINT uq_user_templates_name UNIQUE (name)
);
CREATE TABLE core_configs (
	id INTEGER NOT NULL, 
	created_at DATETIME NOT NULL, 
	name VARCHAR(256) NOT NULL, 
	config JSON NOT NULL, 
	exclude_inbound_tags VARCHAR(2048), 
	fallbacks_inbound_tags VARCHAR(2048), type VARCHAR(7) DEFAULT 'xray' NOT NULL, 
	CONSTRAINT pk_core_configs PRIMARY KEY (id)
);
INSERT INTO core_configs VALUES(1,'2026-10-16 13:51:47.056762','Default Core Config','{"policy": {"levels": {"0": {"statsUserOnline": true}}}, "log": {"loglevel": "warning"}, "inbounds": [{"tag": "Shadowsocks TCP", "listen": "0.0.0.0", "port": 1080, "protocol": "shadowsocks", "settings": {"clients": [], "network": "tcp,udp"}}], "outbounds": [{"protocol": "freedom", "tag": "DIRECT"}, {"protocol": "blackhole", "tag": "BLOCK"}], "routing": {"rules": [{"ip": ["geoip:private"], "outboundTag": "BLOCK", "type": "field"}]}}','','','xray');
CREATE TABLE node_stats (
	id INTEGER NOT NULL, 
	created_at DATETIME NOT NULL, 
	node_id INTEGER NOT NULL, 
	mem_total BIGINT NOT NULL, 
	mem_used BIGINT NOT NULL, 
	cpu_cores INTEGER NOT NULL, 
	cpu_usage FLOAT NOT NULL, 
	incoming_bandwidth_speed BIGINT NOT NULL, 
	outgoing_bandwidth_speed BIGINT NOT NULL, 
	CONSTRAINT pk_node_stats PRIMARY KEY (id), 
	CONSTRAINT fk_node_stats_node_id_nodes FOREIGN KEY(node_id) REFERENCES nodes (id)
);
CREATE TABLE IF NOT EXISTS "users" (
	id INTEGER NOT NULL, 
	username VARCHAR(128) NOT NULL, 
	status VARCHAR(8) NOT NULL, 
	used_traffic BIGINT NOT NULL, 
	data_limit BIGINT, 
	created_at DATETIME DEFAULT (CURRENT_TIMESTAMP) NOT NULL, 
	admin_id INTEGER, 
	data_limit_reset_strategy VARCHAR(8) DEFAULT 'no_reset' NOT NULL, 
	sub_revoked_at DATETIME, 
	note VARCHAR(500), 
	online_at DATETIME, 
	edit_at DATETIME, 
	on_hold_timeout DATETIME, 
	on_hold_expire_duration BIGINT, 
	auto_delete_in_days INTEGER, 
	last_status_change DATETIME, 
	expire DATETIME, 
	proxy_settings JSON DEFAULT '{}' NOT NULL, hwid_limit BIGINT, 
	CONSTRAINT pk_users PRIMARY KEY (id), 
	CONSTRAINT fk_users_admin_id_admins FOREIGN KEY(admin_id) REFERENCES admins (id)
);
CREATE TABLE IF NOT EXISTS "nodes" (
	id INTEGER NOT NULL, 
	name VARCHAR(256) NOT NULL, 
	address VARCHAR(256) NOT NULL, 
	port INTEGER NOT NULL, 
	status VARCHAR(10) NOT NULL, 
	last_status_change DATETIME, 
	message VARCHAR(1024), 
	created_at DATETIME NOT NULL, 
	uplink BIGINT NOT NULL, 
	downlink BIGINT NOT NULL, 
	xray_version VARCHAR(32), 
	usage_coefficient FLOAT DEFAULT (1.0) NOT NULL, 
	node_version VARCHAR(32), 
	connection_type VARCHAR(4) DEFAULT 'grpc' NOT NULL, 
	server_ca VARCHAR(2048) DEFAULT ('') NOT NULL, 
	keep_alive INTEGER DEFAULT '0' NOT NULL, 
	core_config_id INTEGER, 
	api_key VARCHAR(36), 
	data_limit BIGINT DEFAULT 0 NOT NULL, data_limit_reset_strategy VARCHAR(8) DEFAULT 'no_reset' NOT NULL, reset_time INTEGER DEFAULT (-1) NOT NULL, default_timeout INTEGER DEFAULT (10) NOT NULL, internal_timeout INTEGER DEFAULT (15) NOT NULL, api_port INTEGER DEFAULT '62051' NOT NULL, proxy_url VARCHAR(256), 
	CONSTRAINT pk_nodes PRIMARY KEY (id), 
	CONSTRAINT nodes_ibfk_1 FOREIGN KEY(core_config_id) REFERENCES core_configs (id) ON DELETE SET NULL, 
	CONSTRAINT uq_nodes_name UNIQUE (name)
);
CREATE TABLE IF NOT EXISTS "next_plans" (
	id INTEGER NOT NULL, 
	user_id INTEGER NOT NULL, 
	data_limit BIGINT NOT NULL, 
	expire INTEGER, 
	add_remaining_traffic BOOLEAN DEFAULT '0' NOT NULL, 
	user_template_id INTEGER, 
	CONSTRAINT pk_next_plans PRIMARY KEY (id), 
	CONSTRAINT fk_next_plans_user_id_users FOREIGN KEY(user_id) REFERENCES users (id) ON DELETE CASCADE, 
	CONSTRAINT fk_next_plans_user_template_id_user_templates FOREIGN KEY(user_template_id) REFERENCES user_templates (id) ON DELETE SET NULL
);
CREATE TABLE IF NOT EXISTS "node_usage_reset_logs" (
	id INTEGER NOT NULL, 
	created_at DATETIME NOT NULL, 
	node_id INTEGER NOT NULL, 
	uplink BIGINT NOT NULL, 
	downlink BIGINT NOT NULL, 
	CONSTRAINT pk_node_usage_reset_logs PRIMARY KEY (id), 
	CONSTRAINT fk_node_usage_reset_logs_node_id_nodes FOREIGN KEY(node_id) REFERENCES nodes (id) ON DELETE CASCADE
);
CREATE TABLE IF NOT EXISTS "node_usages" (
	id INTEGER NOT NULL, 
	created_at DATETIME NOT NULL, 
	node_id INTEGER, 
	uplink BIGINT NOT NULL, 
	downlink BIGINT NOT NULL, 
	CONSTRAINT pk_node_usages PRIMARY KEY (id), 
	CONSTRAINT uq_node_usages_created_at UNIQUE (created_at, node_id), 
	CONSTRAINT fk_node_usages_node_id_nodes FOREIGN KEY(node_id) REFERENCES nodes (id) ON DELETE CASCADE
);
CREATE TABLE IF NOT EXISTS "node_user_usages" (
	id INTEGER NOT NULL, 
	created_at DATETIME NOT NULL, 
	user_id INTEGER NOT NULL, 
	node_id INTEGER, 
	used_traffic BIGINT NOT NULL, 
	CONSTRAINT pk_node_user_usages PRIMARY KEY (id), 
	CONSTRAINT uq_node_user_usages_created_at UNIQUE (created_at, user_id, node_id), 
	CONSTRAINT fk_node_user_usages_node_id_nodes FOREIGN KEY(node_id) REFERENCES nodes (id) ON DELETE CASCADE, 
	CONSTRAINT fk_node_user_usages_user_id_users FOREIGN KEY(user_id) REFERENCES users (id) ON DELETE CASCADE
);
CREATE TABLE IF NOT EXISTS "notification_reminders" (
	id INTEGER NOT NULL, 
	user_id INTEGER NOT NULL, 
	type VARCHAR(15) NOT NULL, 
	expires_at DATETIME, 
	created_at DATETIME NOT NULL, 
	threshold INTEGER, 
	CONSTRAINT pk_notification_reminders PRIMARY KEY (id), 
	CONSTRAINT fk_notification_reminders_user_id_users FOREIGN KEY(user_id) REFERENCES users (id) ON DELETE CASCADE
);
CREATE TABLE IF NOT EXISTS "user_subscription_updates" (
	id INTEGER NOT NULL, 
	user_id INTEGER NOT NULL, 
	created_at DATETIME NOT NULL, 
	user_agent VARCHAR(512) NOT NULL, ip VARCHAR(64), hwid VARCHAR(256), 
	CONSTRAINT pk_user_subscription_updates PRIMARY KEY (id), 
	CONSTRAINT fk_user_subscription_updates_user_id_users FOREIGN KEY(user_id) REFERENCES users (id) ON DELETE CASCADE
);
CREATE TABLE IF NOT EXISTS "user_usage_logs" (
	id INTEGER NOT NULL, 
	user_id INTEGER, 
	used_traffic_at_reset BIGINT NOT NULL, 
	reset_at DATETIME NOT NULL, 
	CONSTRAINT pk_user_usage_logs PRIMARY KEY (id), 
	CONSTRAINT fk_user_usage_logs_user_id_users FOREIGN KEY(user_id) REFERENCES users (id) ON DELETE CASCADE
);
CREATE TABLE IF NOT EXISTS "hosts" (
	id INTEGER NOT NULL, 
	remark VARCHAR(256) NOT NULL, 
	address VARCHAR(256) NOT NULL, 
	port INTEGER, 
	inbound_tag VARCHAR(256), 
	sni VARCHAR(1000), 
	host VARCHAR(1000), 
	security VARCHAR(15) DEFAULT 'inbound_default' NOT NULL, 
	alpn VARCHAR(14) DEFAULT 'none', 
	fingerprint VARCHAR(16) DEFAULT 'none' NOT NULL, 
	allowinsecure BOOLEAN, 
	is_disabled BOOLEAN, 
	path VARCHAR(256), 
	random_user_agent BOOLEAN DEFAULT '0' NOT NULL, 
	use_sni_as_host BOOLEAN DEFAULT '0' NOT NULL, 
	priority INTEGER NOT NULL, 
	http_headers JSON, 
	transport_settings JSON, 
	mux_settings JSON, 
	noise_settings JSON, 
	fragment_settings JSON, 
	status VARCHAR(60) DEFAULT (''), 
	vless_route VARCHAR(4), 
	pinned_peer_cert_sha256 VARCHAR(128), 
	verify_peer_cert_by_name VARCHAR(1000), wireguard_overrides JSON, subscription_templates JSON, final_mask_settings JSON, cipher_suites VARCHAR(1024), ech JSON, 
	CONSTRAINT pk_hosts PRIMARY KEY (id), 
	CONSTRAINT fk_hosts_inbound_tag_inbounds FOREIGN KEY(inbound_tag) REFERENCES inbounds (tag) ON DELETE SET NULL ON UPDATE CASCADE
);
CREATE TABLE client_templates (
	id INTEGER NOT NULL, 
	name VARCHAR(64) NOT NULL, 
	template_type VARCHAR(32) NOT NULL, 
	content TEXT NOT NULL, 
	is_default BOOLEAN DEFAULT '0' NOT NULL, 
	is_system BOOLEAN DEFAULT '0' NOT NULL, 
	CONSTRAINT pk_client_templates PRIMARY KEY (id), 
	CONSTRAINT uq_client_templates_template_type UNIQUE (template_type, name)
);
INSERT INTO client_templates VALUES(1,'Default Clash Subscription','clash_subscription',replace('mode: rule\nmixed-port: 7890\nipv6: true\n\ntun:\n  enable: true\n  stack: mixed\n  dns-hijack:\n    - "any:53"\n  auto-route: true\n  auto-detect-interface: true\n  strict-route: true\n\ndns:\n  enable: true\n  listen: :1053\n  ipv6: true\n  nameserver:\n    - ''https://1.1.1.1/dns-query#PROXY''\n  proxy-server-nameserver:\n    - ''8.8.8.8''\n    - ''1.1.1.1''\n\nsniffer:\n  enable: true\n  override-destination: true\n  sniff:\n    HTTP:\n      ports: [80, 8080-8880]\n    TLS:\n      ports: [443, 8443]\n    QUIC:\n      ports: [443, 8443]\n\n{{ conf | except("proxy-groups", "port", "mode", "rules") | yaml }}\n\nproxy-groups:\n- name: ''PROXY''\n  type: ''select''\n  proxies:\n  - ''Fastest''\n  {{ proxy_remarks | yaml | indent(2) }}\n\n- name: ''Fastest''\n  type: ''url-test''\n  proxies:\n  {{ proxy_remarks | yaml | indent(2) }}\n\nrules:\n  - MATCH,PROXY','\n',char(10)),1,1);
INSERT INTO client_templates VALUES(2,'Default Xray Subscription','xray_subscription',replace('{\n  "log": {\n    "access": "",\n    "error": "",\n    "loglevel": "warning"\n  },\n "policy": {\n    "system": {\n        "statsOutboundDownlink": true,\n        "statsOutboundUplink": true\n    },\n	"levels": {\n     "8": {\n       "connIdle": 300,\n       "downlinkOnly": 1,\n       "handshake": 4,\n       "uplinkOnly": 1\n     }\n   }\n   },\n  "inbounds": [\n    {\n      "tag": "socks",\n      "port": 10808,\n      "listen": "0.0.0.0",\n      "protocol": "socks",\n      "sniffing": {\n        "enabled": true,\n        "destOverride": [\n          "http",\n          "tls"\n        ],\n        "routeOnly": false\n      },\n      "settings": {\n        "auth": "noauth",\n        "udp": true,\n        "allowTransparent": false\n      }\n    },\n    {\n      "tag": "http",\n      "port": 10809,\n      "listen": "0.0.0.0",\n      "protocol": "http",\n      "sniffing": {\n        "enabled": true,\n        "destOverride": [\n          "http",\n          "tls"\n        ],\n        "routeOnly": false\n      },\n      "settings": {\n        "auth": "noauth",\n        "udp": true,\n        "allowTransparent": false\n      }\n    }\n  ],\n  "outbounds": [\n      {\n        "protocol": "freedom",\n        "tag": "DIRECT"\n      },\n      {\n        "protocol": "blackhole",\n        "tag": "BLOCK"\n      }\n    ],\n  "dns": {\n    "servers": [\n      "1.1.1.1",\n      "8.8.8.8"\n    ]\n  },\n  "routing": {\n    "domainStrategy": "AsIs",\n    "rules": []\n  }\n}','\n',char(10)),1,1);
INSERT INTO client_templates VALUES(3,'Default Singbox Subscription','singbox_subscription',replace('{\n  "log": {\n    "level": "warn",\n    "timestamp": false\n  },\n  "dns": {\n    "servers": [\n      {\n        "type": "udp",\n        "tag": "dns-remote",\n        "server": "1.1.1.2",\n        "detour": "proxy"\n      },\n      {\n        "type": "local",\n        "tag": "dns-local"\n      }\n    ],\n    "final": "dns-remote"\n  },\n  "inbounds": [\n    {\n      "type": "tun",\n      "tag": "tun-in",\n      "interface_name": "sing-tun",\n      "address": [\n        "172.19.0.1/30",\n        "fdfe:dcba:9876::1/126"\n      ],\n      "auto_route": true,\n      "route_exclude_address": [\n        "192.168.0.0/16",\n        "10.0.0.0/8",\n        "169.254.0.0/16",\n        "172.16.0.0/12",\n        "fe80::/10",\n        "fc00::/7"\n      ]\n    }\n  ],\n  "outbounds": [\n    {\n      "type": "selector",\n      "tag": "proxy",\n      "outbounds": null,\n      "interrupt_exist_connections": true\n    },\n    {\n      "type": "urltest",\n      "tag": "Best Latency",\n      "outbounds": null\n    },\n    {\n      "type": "direct",\n      "tag": "direct"\n    }\n  ],\n  "route": {\n    "rules": [\n      {\n        "inbound": "tun-in",\n        "action": "sniff"\n      },\n      {\n        "protocol": "dns",\n        "action": "hijack-dns"\n      }\n    ],\n    "final": "proxy",\n    "auto_detect_interface": true,\n    "override_android_vpn": true\n  },\n  "experimental": {\n    "cache_file": {\n      "enabled": true,\n      "store_dns": true\n    }\n  }\n}','\n',char(10)),1,1);
INSERT INTO client_templates VALUES(4,'Default User-Agent Template','user_agent',replace('{\n  "list":[\n    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.0.0 Safari/537.36",\n    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36",\n    "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36",\n    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.0.0 Safari/537.36 Edg/125.0.0.0",\n    "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.0.0 Safari/537.36",\n    "Mozilla/5.0 (iPhone; CPU iPhone OS 17_4_1 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.4.1 Mobile/15E148 Safari/604.1",\n    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/106.0.0.0 Safari/537.36 PageSpeedPlus/1.0.0",\n    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36 Edg/124.0.0.0",\n    "Mozilla/5.0 (Linux; Android 10; K) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.0.0 Mobile Safari/537.36",\n    "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:126.0) Gecko/20100101 Firefox/126.0",\n    "Mozilla/5.0 (Linux; Android 10; K) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Mobile Safari/537.36",\n    "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/123.0.0.0 Safari/537.36",\n    "Mozilla/5.0 (iPhone; CPU iPhone OS 17_4_1 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/15E148",\n    "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.4.1 Safari/605.1.15",\n    "Mozilla/5.0 (Macintosh; Intel Mac OS X 11.6; rv:92.0) Gecko/20100101 Firefox/92.0",\n    "Mozilla/5.0 (iPhone; CPU iPhone OS 16_6_1 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.6 Mobile/15E148 Safari/604.1",\n    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",\n    "Mozilla/5.0 (Linux; Android 10; K) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.0.0 Safari/537.36",\n    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) obsidian/1.4.14 Chrome/114.0.5735.289 Electron/25.8.1 Safari/537.36",\n    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/123.0.0.0 Safari/537.36",\n    "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36",\n    "Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:126.0) Gecko/20100101 Firefox/126.0",\n    "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.5 Safari/605.1.15",\n    "Mozilla/5.0 (Linux; Android 10; K) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/123.0.0.0 Mobile Safari/537.36",\n    "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/116.0.0.0 Safari/537.36",\n    "Mozilla/5.0 (Linux; Android 10; K) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36",\n    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/109.0.0.0 Safari/537.36",\n    "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.0.0 Safari/537.36",\n    "Mozilla/5.0 (Linux; Android 10; K) AppleWebKit/537.36 (KHTML, like Gecko) SamsungBrowser/25.0 Chrome/121.0.0.0 Mobile Safari/537.36",\n    "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:125.0) Gecko/20100101 Firefox/125.0",\n    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",\n    "Mozilla/5.0 (Linux; Android 10; K) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/121.0.0.0 Mobile Safari/537.36",\n    "Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:125.0) Gecko/20100101 Firefox/125.0",\n    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) obsidian/1.4.16 Chrome/114.0.5735.289 Electron/25.8.1 Safari/537.36",\n    "Mozilla/5.0 (Linux; Android 10; K) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Mobile Safari/537.36",\n    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36",\n    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/89.0.4389.114 Safari/537.36",\n    "Mozilla/5.0 (iPhone; CPU iPhone OS 17_2_1 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.2 Mobile/15E148 Safari/604.1",\n    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/123.0.0.0 Safari/537.36 Edg/123.0.0.0",\n    "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.6 Safari/605.1.15",\n    "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",\n    "Mozilla/5.0 (Linux; Android 10; K) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Mobile Safari/537.36",\n    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) obsidian/1.5.3 Chrome/114.0.5735.289 Electron/25.8.1 Safari/537.36",\n    "Mozilla/5.0 (Linux; Android 10; K) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/123.0.0.0 Safari/537.36",\n    "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.1 Safari/605.1.15",\n    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/121.0.0.0 Safari/537.36",\n    "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/121.0.0.0 Safari/537.36",\n    "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36",\n    "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0_1 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Mobile/15E148 Safari/604.1",\n    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36 Edg/122.0.0.0",\n    "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.2.1 Safari/605.1.15",\n    "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/123.0.0.0 Safari/537.36",\n    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) obsidian/1.5.12 Chrome/120.0.6099.283 Electron/28.2.3 Safari/537.36",\n    "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) obsidian/1.4.16 Chrome/114.0.5735.289 Electron/25.8.1 Safari/537.36",\n    "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/119.0.0.0 Safari/537.36",\n    "Mozilla/5.0 (Linux; Android 10; K) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/119.0.0.0 Mobile Safari/537.36",\n    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/123.0.0.0 Safari/537.36 OPR/109.0.0.0",\n    "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/81.0.4044.129 Safari/537.36",\n    "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",\n    "Mozilla/5.0 (Linux; Android 10; K) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/116.0.0.0 Mobile Safari/537.36",\n    "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.1 Safari/605.1.15",\n    "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.3.1 Safari/605.1.15",\n    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) obsidian/1.4.13 Chrome/114.0.5735.289 Electron/25.8.1 Safari/537.36",\n    "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Safari/605.1.15",\n    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/119.0.0.0 Safari/537.36",\n    "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.2 Safari/605.1.15",\n    "Mozilla/5.0 (Windows NT 6.1; Win64; x64; rv:109.0) Gecko/20100101 Firefox/115.0",\n    "Mozilla/5.0 (iPhone; CPU iPhone OS 17_5_1 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.5 Mobile/15E148 Safari/604.1",\n    "Mozilla/5.0 (iPhone; CPU iPhone OS 17_3 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.3 Mobile/15E148 Safari/604.1",\n    "Mozilla/5.0 (iPhone; CPU iPhone OS 16_1_1 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.1 Mobile/15E148 Safari/604.1",\n    "Mozilla/5.0 (iPhone; CPU iPhone OS 17_3_1 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.3.1 Mobile/15E148 Safari/604.1",\n    "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:124.0) Gecko/20100101 Firefox/124.0",\n    "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.3 Safari/605.1.15",\n    "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36",\n    "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) obsidian/1.4.13 Chrome/114.0.5735.289 Electron/25.8.1 Safari/537.36",\n    "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.4 Safari/605.1.15",\n    "Mozilla/5.0 (iPhone; CPU iPhone OS 16_6 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.6 Mobile/15E148 Safari/604.1",\n    "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/15.6.1 Safari/605.1.15",\n    "Mozilla/5.0 (Linux; Android 10; K) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36",\n    "Mozilla/5.0 (iPhone; CPU iPhone OS 17_1_1 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.1.1 Mobile/15E148 Safari/604.1",\n    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36 Edg/120.0.0.0",\n    "Mozilla/5.0 (iPhone; CPU iPhone OS 17_5 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.5 Mobile/15E148 Safari/604.1",\n    "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) obsidian/1.5.12 Chrome/120.0.6099.283 Electron/28.2.3 Safari/537.36",\n    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/117.0.0.0 Safari/537.36",\n    "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.1.2 Safari/605.1.15",\n    "Mozilla/5.0 (iPhone; CPU iPhone OS 16_2 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.2 Mobile/15E148 Safari/604.1",\n    "Mozilla/5.0 (Linux; Android 10; K) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/114.0.0.0 Mobile Safari/537.36",\n    "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.5 Safari/605.1.15",\n    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/121.0.0.0 Safari/537.36 Edg/121.0.0.0",\n    "Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:124.0) Gecko/20100101 Firefox/124.0",\n    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/118.0.0.0 Safari/537.36",\n    "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) obsidian/1.5.3 Chrome/114.0.5735.289 Electron/25.8.1 Safari/537.36",\n    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/101.0.4951.67 Safari/537.36",\n    "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.3 Safari/605.1.15",\n    "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36",\n    "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/121.0.0.0 Safari/537.36",\n    "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36 Edg/124.0.0.0",\n    "Mozilla/5.0 (Linux; Android 10; K) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/117.0.0.0 Mobile Safari/537.36",\n    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/114.0.0.0 Safari/537.36",\n    "Mozilla/5.0 (Linux; Android 10; K) AppleWebKit/537.36 (KHTML, like Gecko) SamsungBrowser/24.0 Chrome/117.0.0.0 Mobile Safari/537.36"\n  ]\n}','\n',char(10)),1,1);
INSERT INTO client_templates VALUES(5,'Default gRPC User-Agent Template','grpc_user_agent',replace('{\n    "list": [\n        "grpc-dotnet/2.41.0 (.NET 6.0.1; CLR 6.0.1; net6.0; windows; x64)",\n        "grpc-dotnet/2.41.0 (.NET 6.0.0-preview.7.21377.19; CLR 6.0.0; net6.0; osx; x64)",\n        "grpc-dotnet/2.41.0 (Mono 6.12.0.140; CLR 4.0.30319; netstandard2.0; osx; x64)",\n        "grpc-dotnet/2.41.0 (.NET 6.0.0-rc.1.21380.1; CLR 6.0.0; net6.0; linux; arm64)",\n        "grpc-dotnet/2.41.0 (.NET 5.0.8; CLR 5.0.8; net5.0; linux; arm64)",\n        "grpc-dotnet/2.41.0 (.NET Core; CLR 3.1.4; netstandard2.1; linux; arm64)",\n        "grpc-dotnet/2.41.0 (.NET Framework; CLR 4.0.30319.42000; netstandard2.0; windows; x86)",\n        "grpc-dotnet/2.41.0 (.NET 6.0.0-rc.1.21380.1; CLR 6.0.0; net6.0; windows; x64)",\n        "grpc-python-asyncio/1.62.1 grpc-c/39.0.0 (linux; chttp2)",\n        "grpc-go/1.58.1",\n        "grpc-java-okhttp/1.55.1",\n        "grpc-node/1.7.1 grpc-c/1.7.1 (osx; chttp2)",\n        "grpc-node/1.24.2 grpc-c/8.0.0 (linux; chttp2; ganges)",\n        "grpc-c++/1.16.0 grpc-c/6.0.0 (linux; nghttp2; hw)",\n        "grpc-node/1.19.0 grpc-c/7.0.0 (linux; chttp2; gold)",\n        "grpc-ruby/1.62.0 grpc-c/39.0.0 (osx; chttp2)]"\n    ]\n}','\n',char(10)),1,1);
CREATE TABLE user_hwids (
	id INTEGER NOT NULL, 
	user_id INTEGER NOT NULL, 
	hwid VARCHAR(256) NOT NULL, 
	device_os VARCHAR(256), 
	os_version VARCHAR(128), 
	device_model VARCHAR(256), 
	created_at DATETIME NOT NULL, 
	last_used_at DATETIME NOT NULL, 
	CONSTRAINT pk_user_hwids PRIMARY KEY (id), 
	CONSTRAINT fk_user_hwids_user_id_users FOREIGN KEY(user_id) REFERENCES users (id) ON DELETE CASCADE, 
	CONSTRAINT uq_user_hwids_user_id UNIQUE (user_id, hwid)
);
CREATE TABLE temp_keys (
	"key" VARCHAR(36) NOT NULL, 
	action VARCHAR(32) NOT NULL, 
	expires_at DATETIME NOT NULL, 
	used_at DATETIME, 
	used_by_ip VARCHAR(45), 
	CONSTRAINT pk_temp_keys PRIMARY KEY ("key")
);
CREATE TABLE IF NOT EXISTS "settings" (
	id INTEGER NOT NULL, 
	telegram JSON NOT NULL, 
	webhook JSON NOT NULL, 
	notification_settings JSON NOT NULL, 
	notification_enable JSON NOT NULL, 
	subscription JSON NOT NULL, 
	general JSON NOT NULL, 
	hwid JSON NOT NULL, 
	CONSTRAINT pk_settings PRIMARY KEY (id)
);
INSERT INTO settings VALUES(1,'{"enable": false, "token": null, "webhook_url": null, "webhook_secret": null, "proxy_url": null}','{"enable": false, "webhooks": [], "days_left": [3], "usage_percent": [80], "timeout": 180, "recurrent": 3, "proxy_url": null}','{"notify_telegram": false, "notify_discord": false, "telegram_api_token": null, "telegram_chat_id": null, "telegram_topic_id": null, "discord_webhook_url": null, "proxy_url": null, "max_retries": 3}','{"admin": {"create": true, "modify": true, "delete": true, "reset_usage": true, "login": true}, "core": {"create": true, "modify": true, "delete": true}, "group": {"create": true, "modify": true, "delete": true}, "host": {"create": true, "modify": true, "delete": true, "modify_hosts": true}, "node": {"create": true, "modify": true, "delete": true, "connect": true, "error": true}, "user": {"create": true, "modify": true, "delete": true, "status_change": true, "reset_data_usage": true, "data_reset_by_next": true, "subscription_revoked": true}, "user_template": {"create": true, "modify": true, "delete": true}, "days_left": true, "percentage_reached": true}','{"url_prefix": "", "update_interval": "12", "support_url": "https://t.me/", "profile_title": "Subscription", "host_status_filter": true, "randomize_order": false, "rules": [{"pattern": "^(?:FlClashX?|Flowvy|[Cc]lash(?:-(?:[Vv]erge|nyanpasu)|X [Mm]eta|-?[Mm]eta)|[Kk]oala-[Cc]lash|[Mm](?:urge|ihomo)|prizrak-box|clash\\.meta)", "target": "clash_meta"}, {"pattern": "^([Cc]lash|[Ss]tash)", "target": "clash"}, {"pattern": "^(SFA|SFI|SFM|SFT|[Kk]aring|[Hh]iddify[Nn]ext)|.*[Ss]ing[\\-b]?ox.*", "target": "sing_box"}, {"pattern": "^(SS|SSR|SSD|SSS|Outline|Shadowsocks|SSconf)", "target": "outline"}, {"pattern": "^[Ii]n[Hh]ive", "target": "xray"}, {"pattern": "^.*", "target": "links_base64"}], "manual_sub_request": {"links": true, "links_base64": true, "xray": true, "sing_box": true, "clash": true, "clash_meta": true, "outline": true}}','{"default_flow": "", "default_method": "chacha20-ietf-poly1305"}','{}');
CREATE TABLE IF NOT EXISTS "admins" (
	id INTEGER NOT NULL, 
	username VARCHAR(34) NOT NULL, 
	hashed_password VARCHAR(128) NOT NULL, 
	created_at DATETIME NOT NULL, 
	password_reset_at DATETIME, 
	telegram_id BIGINT, 
	discord_webhook VARCHAR(1024), 
	used_traffic BIGINT DEFAULT '0' NOT NULL, 
	sub_template VARCHAR(1024), 
	sub_domain VARCHAR(256), 
	profile_title VARCHAR(512), 
	support_url VARCHAR(1024), 
	notification_enable JSON, 
	note VARCHAR(500), 
	role_id INTEGER NOT NULL, 
	permission_overrides JSON, 
	status VARCHAR(8) DEFAULT 'active' NOT NULL, 
	data_limit BIGINT, 
	last_status_change DATETIME, custom_variables JSON, 
	CONSTRAINT pk_admins PRIMARY KEY (id), 
	CONSTRAINT fk_admins_role_id_admin_roles FOREIGN KEY(role_id) REFERENCES admin_roles (id)
);
CREATE TABLE IF NOT EXISTS "admin_roles" (
	id INTEGER NOT NULL, 
	name VARCHAR(64) NOT NULL, 
	is_owner BOOLEAN DEFAULT '0' NOT NULL, 
	permissions JSON NOT NULL, 
	limits JSON NOT NULL, 
	features JSON NOT NULL, 
	access JSON NOT NULL, 
	created_at DATETIME NOT NULL, 
	disabled_when_limited BOOLEAN DEFAULT '0' NOT NULL, 
	disconnect_users_when_limited BOOLEAN DEFAULT '1' NOT NULL, 
	hwid JSON NOT NULL, 
	disconnect_users_when_disabled BOOLEAN DEFAULT '1' NOT NULL, 
	CONSTRAINT pk_admin_roles PRIMARY KEY (id), 
	CONSTRAINT uq_admin_roles_name UNIQUE (name)
);
INSERT INTO admin_roles VALUES(1,'owner',1,'{"users": {"create": true, "read": {"scope": 2}, "read_simple": {"scope": 2}, "update": {"scope": 2}, "delete": {"scope": 2}, "reset_usage": {"scope": 2}, "revoke_sub": {"scope": 2}, "set_owner": true, "activate_next_plan": {"scope": 2}}, "admins": {"create": true, "read": true, "read_simple": true, "update": true, "delete": true, "reset_usage": true}, "nodes": {"create": true, "read": true, "read_simple": true, "update": true, "delete": true, "reconnect": true, "update_core": true, "logs": true, "stats": true}, "groups": {"create": true, "read": true, "read_simple": true, "update": true, "delete": true}, "hosts": {"create": true, "read": true, "update": true}, "templates": {"create": true, "read": true, "read_simple": true, "update": true, "delete": true}, "client_templates": {"create": true, "read": true, "read_simple": true, "update": true, "delete": true}, "cores": {"create": true, "read": true, "read_simple": true, "update": true, "delete": true}, "settings": {"read": true, "read_general": true, "update": true}, "system": {"read": true}, "hwids": {"read": true, "delete": true}, "admin_roles": {"create": true, "read": true, "read_simple": true, "update": true, "delete": true}, "api_keys": {"create": true, "read": {"scope": 2}, "read_simple": {"scope": 2}, "update": {"scope": 2}, "delete": {"scope": 2}}}','{"max_users": null, "data_limit_min": null, "data_limit_max": null, "expire_min": null, "expire_max": null, "min_hwid_per_user": null, "max_hwid_per_user": null}','{"can_use_reset_strategy": true, "can_use_next_plan": true}','{"require_template": false, "allowed_template_ids": null, "allowed_group_ids": null}','2026-10-16 13:51:48.964312',0,1,'{"enabled": true, "forced": false, "fallback_limit": null, "min_limit": null, "max_limit": null, "mode": "use_global"}',1);
INSERT INTO admin_roles VALUES(2,'administrator',0,'{"users": {"create": true, "read": {"scope": 2}, "read_simple": {"scope": 2}, "update": {"scope": 2}, "delete": {"scope": 2}, "reset_usage": {"scope": 2}, "revoke_sub": {"scope": 2}, "set_owner": true, "activate_next_plan": {"scope": 2}}, "admins": {"create": true, "read": true, "read_simple": true, "update": true, "delete": true, "reset_usage": true}, "nodes": {"create": true, "read": true, "read_simple": true, "update": true, "delete": true, "reconnect": true, "update_core": true, "logs": true, "stats": true}, "groups": {"create": true, "read": true, "read_simple": true, "update": true, "delete": true}, "hosts": {"create": true, "read": true, "update": true}, "templates": {"create": true, "read": true, "read_simple": true, "update": true, "delete": true}, "client_templates": {"create": true, "read": true, "read_simple": true, "update": true, "delete": true}, "cores": {"create": true, "read": true, "read_simple": true, "update": true, "delete": true}, "settings": {"read": true, "read_general": true, "update": true}, "system": {"read": true}, "hwids": {"read": true, "delete": true}, "admin_roles": {"read": true, "read_simple": true}, "api_keys": {"create": true, "read": {"scope": 2}, "read_simple": {"scope": 2}, "update": {"scope": 2}, "delete": {"scope": 2}}}','{"max_users": null, "data_limit_min": null, "data_limit_max": null, "expire_min": null, "expire_max": null, "min_hwid_per_user": null, "max_hwid_per_user": null}','{"can_use_reset_strategy": true, "can_use_next_plan": true}','{"require_template": false, "allowed_template_ids": null, "allowed_group_ids": null}','2026-10-16 13:51:48.964312',0,1,'{"enabled": true, "forced": false, "fallback_limit": null, "min_limit": null, "max_limit": null, "mode": "use_global"}',1);
INSERT INTO admin_roles VALUES(3,'operator',0,'{"users": {"create": true, "read": {"scope": 1}, "read_simple": {"scope": 1}, "update": {"scope": 1}, "delete": {"scope": 1}, "reset_usage": {"scope": 1}, "revoke_sub": {"scope": 1}, "set_owner": null, "activate_next_plan": {"scope": 1}}, "groups": {"create": false, "read": false, "read_simple": true, "update": false, "delete": false}, "templates": {"create": false, "read": true, "read_simple": true, "update": false, "delete": false}, "system": {"read": true}, "settings": {"read": false, "read_general": true, "update": false}, "hwids": {"read": true, "delete": true}, "api_keys": {"read": {"scope": 1}, "read_simple": {"scope": 1}, "update": {"scope": 1}, "delete": {"scope": 1}}}','{"max_users": null, "data_limit_min": null, "data_limit_max": null, "expire_min": null, "expire_max": null, "min_hwid_per_user": null, "max_hwid_per_user": null}','{"can_use_reset_strategy": true, "can_use_next_plan": true}','{"require_template": false, "allowed_template_ids": null, "allowed_group_ids": null}','2026-10-16 13:51:48.964312',0,1,'{"enabled": true, "forced": false, "fallback_limit": null, "min_limit": null, "max_limit": null, "mode": "use_global"}',1);
ANALYZE sqlite_schema;
CREATE TABLE api_keys (
	id INTEGER NOT NULL, 
	admin_id INTEGER NOT NULL, 
	name VARCHAR(128) NOT NULL, 
	note VARCHAR(512), 
	key_hash VARCHAR(128) NOT NULL, 
	api_key_trimmed VARCHAR(16) NOT NULL, 
	permissions JSON NOT NULL, 
	created_at DATETIME NOT NULL, 
	expire_date DATETIME, 
	revoked_at DATETIME, 
	status VARCHAR(8) DEFAULT 'active' NOT NULL, inherit_permissions BOOLEAN DEFAULT '1' NOT NULL, 
	CONSTRAINT pk_api_keys PRIMARY KEY (id), 
	CONSTRAINT fk_api_keys_admin_id_admins FOREIGN KEY(admin_id) REFERENCES admins (id) ON DELETE CASCADE, 
	CONSTRAINT uq_api_keys_key_hash UNIQUE (key_hash), 
	CONSTRAINT uq_api_keys_admin_id UNIQUE (admin_id, name)
);
CREATE TABLE wireguard_subnets (
	id INTEGER NOT NULL, 
	network VARCHAR(64) NOT NULL, 
	next_offset INTEGER DEFAULT '1' NOT NULL, 
	free_offsets JSON NOT NULL, 
	CONSTRAINT pk_wireguard_subnets PRIMARY KEY (id), 
	CONSTRAINT uq_wireguard_subnets_network UNIQUE (network)
);
CREATE TABLE IF NOT EXISTS "admin_notification_reminders" (
	id INTEGER NOT NULL, 
	created_at DATETIME NOT NULL, 
	admin_id BIGINT NOT NULL, 
	type VARCHAR(15) NOT NULL, 
	threshold INTEGER DEFAULT '0' NOT NULL, 
	CONSTRAINT pk_admin_notification_reminders PRIMARY KEY (id), 
	CONSTRAINT fk_admin_notification_reminders_admin_id_admins FOREIGN KEY(admin_id) REFERENCES admins (id) ON DELETE CASCADE, 
	CONSTRAINT uq_admin_notification_reminders_admin_id_type_threshold UNIQUE (admin_id, type, threshold)
);
CREATE UNIQUE INDEX ix_inbounds_tag ON inbounds (tag);
CREATE UNIQUE INDEX ix_users_username ON users (username);
CREATE INDEX ix_next_plans_user_template_id ON next_plans (user_template_id);
CREATE INDEX ix_node_usage_reset_logs_node_id_created_at ON node_usage_reset_logs (node_id, created_at);
CREATE INDEX ix_node_usages_created_at ON node_usages (created_at);
CREATE INDEX ix_node_user_usages_created_at ON node_user_usages (created_at);
CREATE INDEX ix_node_user_usages_node_id_created_at ON node_user_usages (node_id, created_at);
CREATE INDEX ix_node_user_usages_user_id_created_at ON node_user_usages (user_id, created_at);
CREATE INDEX ix_user_usage_logs_user_id_reset_at ON user_usage_logs (user_id, reset_at);
CREATE INDEX ix_client_templates_template_type ON client_templates (template_type);
CREATE INDEX ix_user_hwids_user_id ON user_hwids (user_id);
CREATE INDEX ix_user_hwids_hwid ON user_hwids (hwid);
CREATE INDEX ix_user_hwids_created_at ON user_hwids (created_at);
CREATE INDEX ix_user_hwids_last_used_at ON user_hwids (last_used_at);
CREATE UNIQUE INDEX ix_admins_username ON admins (username);
CREATE INDEX idx_user_subscription_updates_user_id ON user_subscription_updates (user_id);
CREATE INDEX idx_users_admin_online ON users (admin_id, online_at);
CREATE INDEX idx_users_admin_status ON users (admin_id, status);
CREATE INDEX idx_users_admin_created ON users (admin_id, created_at);
CREATE INDEX ix_api_keys_admin_id ON api_keys (admin_id);
CREATE INDEX ix_api_keys_created_at ON api_keys (created_at);
CREATE INDEX ix_api_keys_expire_date ON api_keys (expire_date);
CREATE INDEX ix_admin_notification_reminders_admin_id_type ON admin_notification_reminders (admin_id, type);
CREATE INDEX idx_user_subscription_updates_user_created ON user_subscription_updates (user_id, created_at);
COMMIT;
