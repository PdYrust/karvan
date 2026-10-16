/*M!999999\- enable the sandbox mode */ 

/*!40101 SET @OLD_CHARACTER_SET_CLIENT=@@CHARACTER_SET_CLIENT */;
/*!40101 SET @OLD_CHARACTER_SET_RESULTS=@@CHARACTER_SET_RESULTS */;
/*!40101 SET @OLD_COLLATION_CONNECTION=@@COLLATION_CONNECTION */;
/*!40101 SET NAMES utf8mb4 */;
/*!40103 SET @OLD_TIME_ZONE=@@TIME_ZONE */;
/*!40103 SET TIME_ZONE='+00:00' */;
/*!40014 SET @OLD_UNIQUE_CHECKS=@@UNIQUE_CHECKS, UNIQUE_CHECKS=0 */;
/*!40014 SET @OLD_FOREIGN_KEY_CHECKS=@@FOREIGN_KEY_CHECKS, FOREIGN_KEY_CHECKS=0 */;
/*!40101 SET @OLD_SQL_MODE=@@SQL_MODE, SQL_MODE='NO_AUTO_VALUE_ON_ZERO' */;
/*M!100616 SET @OLD_NOTE_VERBOSITY=@@NOTE_VERBOSITY, NOTE_VERBOSITY=0 */;
DROP TABLE IF EXISTS `admin_notification_reminders`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `admin_notification_reminders` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  `created_at` datetime NOT NULL,
  `admin_id` bigint(20) NOT NULL,
  `type` enum('expiration_date','data_usage') NOT NULL,
  `threshold` int(11) NOT NULL DEFAULT 0,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uq_admin_notification_reminders_admin_id_type_threshold` (`admin_id`,`type`,`threshold`),
  KEY `ix_admin_notification_reminders_admin_id_type` (`admin_id`,`type`),
  CONSTRAINT `fk_admin_notification_reminders_admin_id_admins` FOREIGN KEY (`admin_id`) REFERENCES `admins` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_uca1400_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

LOCK TABLES `admin_notification_reminders` WRITE;
/*!40000 ALTER TABLE `admin_notification_reminders` DISABLE KEYS */;
/*!40000 ALTER TABLE `admin_notification_reminders` ENABLE KEYS */;
UNLOCK TABLES;
DROP TABLE IF EXISTS `admin_roles`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `admin_roles` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  `name` varchar(64) NOT NULL,
  `is_owner` tinyint(1) NOT NULL DEFAULT 0,
  `permissions` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL CHECK (json_valid(`permissions`)),
  `limits` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL CHECK (json_valid(`limits`)),
  `features` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL CHECK (json_valid(`features`)),
  `access` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL CHECK (json_valid(`access`)),
  `created_at` datetime NOT NULL,
  `disabled_when_limited` tinyint(1) NOT NULL DEFAULT 0,
  `disconnect_users_when_limited` tinyint(1) NOT NULL DEFAULT 1,
  `hwid` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL CHECK (json_valid(`hwid`)),
  `disconnect_users_when_disabled` tinyint(1) NOT NULL DEFAULT 1,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uq_admin_roles_name` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_uca1400_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

LOCK TABLES `admin_roles` WRITE;
/*!40000 ALTER TABLE `admin_roles` DISABLE KEYS */;
INSERT INTO `admin_roles` VALUES (1,'owner',1,'{\"users\": {\"create\": true, \"read\": {\"scope\": 2}, \"read_simple\": {\"scope\": 2}, \"update\": {\"scope\": 2}, \"delete\": {\"scope\": 2}, \"reset_usage\": {\"scope\": 2}, \"revoke_sub\": {\"scope\": 2}, \"set_owner\": true, \"activate_next_plan\": {\"scope\": 2}}, \"admins\": {\"create\": true, \"read\": true, \"read_simple\": true, \"update\": true, \"delete\": true, \"reset_usage\": true}, \"nodes\": {\"create\": true, \"read\": true, \"read_simple\": true, \"update\": true, \"delete\": true, \"reconnect\": true, \"update_core\": true, \"logs\": true, \"stats\": true}, \"groups\": {\"create\": true, \"read\": true, \"read_simple\": true, \"update\": true, \"delete\": true}, \"hosts\": {\"create\": true, \"read\": true, \"update\": true}, \"templates\": {\"create\": true, \"read\": true, \"read_simple\": true, \"update\": true, \"delete\": true}, \"client_templates\": {\"create\": true, \"read\": true, \"read_simple\": true, \"update\": true, \"delete\": true}, \"cores\": {\"create\": true, \"read\": true, \"read_simple\": true, \"update\": true, \"delete\": true}, \"settings\": {\"read\": true, \"read_general\": true, \"update\": true}, \"system\": {\"read\": true}, \"hwids\": {\"read\": true, \"delete\": true}, \"admin_roles\": {\"create\": true, \"read\": true, \"read_simple\": true, \"update\": true, \"delete\": true}, \"api_keys\": {\"create\": true, \"read\": {\"scope\": 2}, \"read_simple\": {\"scope\": 2}, \"update\": {\"scope\": 2}, \"delete\": {\"scope\": 2}}}','{\"max_users\": null, \"data_limit_min\": null, \"data_limit_max\": null, \"expire_min\": null, \"expire_max\": null, \"min_hwid_per_user\": null, \"max_hwid_per_user\": null}','{\"can_use_reset_strategy\": true, \"can_use_next_plan\": true}','{\"require_template\": false, \"allowed_template_ids\": null, \"allowed_group_ids\": null}','2026-10-16 13:53:15',0,1,'{\"enabled\": true, \"forced\": false, \"fallback_limit\": null, \"min_limit\": null, \"max_limit\": null, \"mode\": \"use_global\"}',1);
INSERT INTO `admin_roles` VALUES (2,'administrator',0,'{\"users\": {\"create\": true, \"read\": {\"scope\": 2}, \"read_simple\": {\"scope\": 2}, \"update\": {\"scope\": 2}, \"delete\": {\"scope\": 2}, \"reset_usage\": {\"scope\": 2}, \"revoke_sub\": {\"scope\": 2}, \"set_owner\": true, \"activate_next_plan\": {\"scope\": 2}}, \"admins\": {\"create\": true, \"read\": true, \"read_simple\": true, \"update\": true, \"delete\": true, \"reset_usage\": true}, \"nodes\": {\"create\": true, \"read\": true, \"read_simple\": true, \"update\": true, \"delete\": true, \"reconnect\": true, \"update_core\": true, \"logs\": true, \"stats\": true}, \"groups\": {\"create\": true, \"read\": true, \"read_simple\": true, \"update\": true, \"delete\": true}, \"hosts\": {\"create\": true, \"read\": true, \"update\": true}, \"templates\": {\"create\": true, \"read\": true, \"read_simple\": true, \"update\": true, \"delete\": true}, \"client_templates\": {\"create\": true, \"read\": true, \"read_simple\": true, \"update\": true, \"delete\": true}, \"cores\": {\"create\": true, \"read\": true, \"read_simple\": true, \"update\": true, \"delete\": true}, \"settings\": {\"read\": true, \"read_general\": true, \"update\": true}, \"system\": {\"read\": true}, \"hwids\": {\"read\": true, \"delete\": true}, \"admin_roles\": {\"read\": true, \"read_simple\": true}, \"api_keys\": {\"create\": true, \"read\": {\"scope\": 2}, \"read_simple\": {\"scope\": 2}, \"update\": {\"scope\": 2}, \"delete\": {\"scope\": 2}}}','{\"max_users\": null, \"data_limit_min\": null, \"data_limit_max\": null, \"expire_min\": null, \"expire_max\": null, \"min_hwid_per_user\": null, \"max_hwid_per_user\": null}','{\"can_use_reset_strategy\": true, \"can_use_next_plan\": true}','{\"require_template\": false, \"allowed_template_ids\": null, \"allowed_group_ids\": null}','2026-10-16 13:53:15',0,1,'{\"enabled\": true, \"forced\": false, \"fallback_limit\": null, \"min_limit\": null, \"max_limit\": null, \"mode\": \"use_global\"}',1);
INSERT INTO `admin_roles` VALUES (3,'operator',0,'{\"users\": {\"create\": true, \"read\": {\"scope\": 1}, \"read_simple\": {\"scope\": 1}, \"update\": {\"scope\": 1}, \"delete\": {\"scope\": 1}, \"reset_usage\": {\"scope\": 1}, \"revoke_sub\": {\"scope\": 1}, \"set_owner\": null, \"activate_next_plan\": {\"scope\": 1}}, \"groups\": {\"create\": false, \"read\": false, \"read_simple\": true, \"update\": false, \"delete\": false}, \"templates\": {\"create\": false, \"read\": true, \"read_simple\": true, \"update\": false, \"delete\": false}, \"system\": {\"read\": true}, \"settings\": {\"read\": false, \"read_general\": true, \"update\": false}, \"hwids\": {\"read\": true, \"delete\": true}, \"api_keys\": {\"read\": {\"scope\": 1}, \"read_simple\": {\"scope\": 1}, \"update\": {\"scope\": 1}, \"delete\": {\"scope\": 1}}}','{\"max_users\": null, \"data_limit_min\": null, \"data_limit_max\": null, \"expire_min\": null, \"expire_max\": null, \"min_hwid_per_user\": null, \"max_hwid_per_user\": null}','{\"can_use_reset_strategy\": true, \"can_use_next_plan\": true}','{\"require_template\": false, \"allowed_template_ids\": null, \"allowed_group_ids\": null}','2026-10-16 13:53:15',0,1,'{\"enabled\": true, \"forced\": false, \"fallback_limit\": null, \"min_limit\": null, \"max_limit\": null, \"mode\": \"use_global\"}',1);
/*!40000 ALTER TABLE `admin_roles` ENABLE KEYS */;
UNLOCK TABLES;
DROP TABLE IF EXISTS `admin_usage_logs`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `admin_usage_logs` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  `admin_id` bigint(20) NOT NULL,
  `used_traffic_at_reset` bigint(20) NOT NULL,
  `reset_at` datetime NOT NULL,
  PRIMARY KEY (`id`),
  KEY `fk_admin_usage_logs_admin_id_admins` (`admin_id`),
  CONSTRAINT `fk_admin_usage_logs_admin_id_admins` FOREIGN KEY (`admin_id`) REFERENCES `admins` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_uca1400_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

LOCK TABLES `admin_usage_logs` WRITE;
/*!40000 ALTER TABLE `admin_usage_logs` DISABLE KEYS */;
/*!40000 ALTER TABLE `admin_usage_logs` ENABLE KEYS */;
UNLOCK TABLES;
DROP TABLE IF EXISTS `admins`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `admins` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  `username` varchar(34) NOT NULL,
  `hashed_password` varchar(128) NOT NULL,
  `created_at` datetime NOT NULL,
  `password_reset_at` datetime DEFAULT NULL,
  `telegram_id` bigint(20) DEFAULT NULL,
  `discord_webhook` varchar(1024) DEFAULT NULL,
  `used_traffic` bigint(20) NOT NULL DEFAULT 0,
  `sub_template` varchar(1024) DEFAULT NULL,
  `sub_domain` varchar(256) DEFAULT NULL,
  `profile_title` varchar(512) DEFAULT NULL,
  `support_url` varchar(1024) DEFAULT NULL,
  `notification_enable` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL CHECK (json_valid(`notification_enable`)),
  `note` varchar(500) DEFAULT NULL,
  `role_id` bigint(20) NOT NULL,
  `permission_overrides` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL CHECK (json_valid(`permission_overrides`)),
  `status` enum('active','disabled','limited') NOT NULL,
  `data_limit` bigint(20) DEFAULT NULL,
  `last_status_change` datetime DEFAULT NULL,
  `custom_variables` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL CHECK (json_valid(`custom_variables`)),
  PRIMARY KEY (`id`),
  UNIQUE KEY `ix_admins_username` (`username`),
  KEY `fk_admins_role_id_admin_roles` (`role_id`),
  CONSTRAINT `fk_admins_role_id_admin_roles` FOREIGN KEY (`role_id`) REFERENCES `admin_roles` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_uca1400_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

LOCK TABLES `admins` WRITE;
/*!40000 ALTER TABLE `admins` DISABLE KEYS */;
/*!40000 ALTER TABLE `admins` ENABLE KEYS */;
UNLOCK TABLES;
DROP TABLE IF EXISTS `alembic_version`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `alembic_version` (
  `version_num` varchar(32) NOT NULL,
  PRIMARY KEY (`version_num`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_uca1400_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

LOCK TABLES `alembic_version` WRITE;
/*!40000 ALTER TABLE `alembic_version` DISABLE KEYS */;
INSERT INTO `alembic_version` VALUES ('48a6bcb8bba1');
/*!40000 ALTER TABLE `alembic_version` ENABLE KEYS */;
UNLOCK TABLES;
DROP TABLE IF EXISTS `api_keys`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `api_keys` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  `admin_id` bigint(20) NOT NULL,
  `name` varchar(128) NOT NULL,
  `note` varchar(512) DEFAULT NULL,
  `key_hash` varchar(128) NOT NULL,
  `api_key_trimmed` varchar(16) NOT NULL,
  `permissions` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL CHECK (json_valid(`permissions`)),
  `created_at` datetime NOT NULL,
  `expire_date` datetime DEFAULT NULL,
  `revoked_at` datetime DEFAULT NULL,
  `status` enum('active','disabled') NOT NULL DEFAULT 'active',
  `inherit_permissions` tinyint(1) NOT NULL DEFAULT 1,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uq_api_keys_key_hash` (`key_hash`),
  UNIQUE KEY `uq_api_keys_admin_id` (`admin_id`,`name`),
  KEY `ix_api_keys_admin_id` (`admin_id`),
  KEY `ix_api_keys_created_at` (`created_at`),
  KEY `ix_api_keys_expire_date` (`expire_date`),
  CONSTRAINT `fk_api_keys_admin_id_admins` FOREIGN KEY (`admin_id`) REFERENCES `admins` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_uca1400_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

LOCK TABLES `api_keys` WRITE;
/*!40000 ALTER TABLE `api_keys` DISABLE KEYS */;
/*!40000 ALTER TABLE `api_keys` ENABLE KEYS */;
UNLOCK TABLES;
DROP TABLE IF EXISTS `client_templates`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `client_templates` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  `name` varchar(64) NOT NULL,
  `template_type` varchar(32) NOT NULL,
  `content` text NOT NULL,
  `is_default` tinyint(1) NOT NULL DEFAULT 0,
  `is_system` tinyint(1) NOT NULL DEFAULT 0,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uq_client_templates_template_type` (`template_type`,`name`),
  KEY `ix_client_templates_template_type` (`template_type`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_uca1400_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

LOCK TABLES `client_templates` WRITE;
/*!40000 ALTER TABLE `client_templates` DISABLE KEYS */;
INSERT INTO `client_templates` VALUES (1,'Default Clash Subscription','clash_subscription','mode: rule\nmixed-port: 7890\nipv6: true\n\ntun:\n  enable: true\n  stack: mixed\n  dns-hijack:\n    - \"any:53\"\n  auto-route: true\n  auto-detect-interface: true\n  strict-route: true\n\ndns:\n  enable: true\n  listen: :1053\n  ipv6: true\n  nameserver:\n    - \'https://1.1.1.1/dns-query#PROXY\'\n  proxy-server-nameserver:\n    - \'8.8.8.8\'\n    - \'1.1.1.1\'\n\nsniffer:\n  enable: true\n  override-destination: true\n  sniff:\n    HTTP:\n      ports: [80, 8080-8880]\n    TLS:\n      ports: [443, 8443]\n    QUIC:\n      ports: [443, 8443]\n\n{{ conf | except(\"proxy-groups\", \"port\", \"mode\", \"rules\") | yaml }}\n\nproxy-groups:\n- name: \'PROXY\'\n  type: \'select\'\n  proxies:\n  - \'Fastest\'\n  {{ proxy_remarks | yaml | indent(2) }}\n\n- name: \'Fastest\'\n  type: \'url-test\'\n  proxies:\n  {{ proxy_remarks | yaml | indent(2) }}\n\nrules:\n  - MATCH,PROXY',1,1);
INSERT INTO `client_templates` VALUES (2,'Default Xray Subscription','xray_subscription','{\n  \"log\": {\n    \"access\": \"\",\n    \"error\": \"\",\n    \"loglevel\": \"warning\"\n  },\n \"policy\": {\n    \"system\": {\n        \"statsOutboundDownlink\": true,\n        \"statsOutboundUplink\": true\n    },\n	\"levels\": {\n     \"8\": {\n       \"connIdle\": 300,\n       \"downlinkOnly\": 1,\n       \"handshake\": 4,\n       \"uplinkOnly\": 1\n     }\n   }\n   },\n  \"inbounds\": [\n    {\n      \"tag\": \"socks\",\n      \"port\": 10808,\n      \"listen\": \"0.0.0.0\",\n      \"protocol\": \"socks\",\n      \"sniffing\": {\n        \"enabled\": true,\n        \"destOverride\": [\n          \"http\",\n          \"tls\"\n        ],\n        \"routeOnly\": false\n      },\n      \"settings\": {\n        \"auth\": \"noauth\",\n        \"udp\": true,\n        \"allowTransparent\": false\n      }\n    },\n    {\n      \"tag\": \"http\",\n      \"port\": 10809,\n      \"listen\": \"0.0.0.0\",\n      \"protocol\": \"http\",\n      \"sniffing\": {\n        \"enabled\": true,\n        \"destOverride\": [\n          \"http\",\n          \"tls\"\n        ],\n        \"routeOnly\": false\n      },\n      \"settings\": {\n        \"auth\": \"noauth\",\n        \"udp\": true,\n        \"allowTransparent\": false\n      }\n    }\n  ],\n  \"outbounds\": [\n      {\n        \"protocol\": \"freedom\",\n        \"tag\": \"DIRECT\"\n      },\n      {\n        \"protocol\": \"blackhole\",\n        \"tag\": \"BLOCK\"\n      }\n    ],\n  \"dns\": {\n    \"servers\": [\n      \"1.1.1.1\",\n      \"8.8.8.8\"\n    ]\n  },\n  \"routing\": {\n    \"domainStrategy\": \"AsIs\",\n    \"rules\": []\n  }\n}',1,1);
INSERT INTO `client_templates` VALUES (3,'Default Singbox Subscription','singbox_subscription','{\n  \"log\": {\n    \"level\": \"warn\",\n    \"timestamp\": false\n  },\n  \"dns\": {\n    \"servers\": [\n      {\n        \"type\": \"udp\",\n        \"tag\": \"dns-remote\",\n        \"server\": \"1.1.1.2\",\n        \"detour\": \"proxy\"\n      },\n      {\n        \"type\": \"local\",\n        \"tag\": \"dns-local\"\n      }\n    ],\n    \"final\": \"dns-remote\"\n  },\n  \"inbounds\": [\n    {\n      \"type\": \"tun\",\n      \"tag\": \"tun-in\",\n      \"interface_name\": \"sing-tun\",\n      \"address\": [\n        \"172.19.0.1/30\",\n        \"fdfe:dcba:9876::1/126\"\n      ],\n      \"auto_route\": true,\n      \"route_exclude_address\": [\n        \"192.168.0.0/16\",\n        \"10.0.0.0/8\",\n        \"169.254.0.0/16\",\n        \"172.16.0.0/12\",\n        \"fe80::/10\",\n        \"fc00::/7\"\n      ]\n    }\n  ],\n  \"outbounds\": [\n    {\n      \"type\": \"selector\",\n      \"tag\": \"proxy\",\n      \"outbounds\": null,\n      \"interrupt_exist_connections\": true\n    },\n    {\n      \"type\": \"urltest\",\n      \"tag\": \"Best Latency\",\n      \"outbounds\": null\n    },\n    {\n      \"type\": \"direct\",\n      \"tag\": \"direct\"\n    }\n  ],\n  \"route\": {\n    \"rules\": [\n      {\n        \"inbound\": \"tun-in\",\n        \"action\": \"sniff\"\n      },\n      {\n        \"protocol\": \"dns\",\n        \"action\": \"hijack-dns\"\n      }\n    ],\n    \"final\": \"proxy\",\n    \"auto_detect_interface\": true,\n    \"override_android_vpn\": true\n  },\n  \"experimental\": {\n    \"cache_file\": {\n      \"enabled\": true,\n      \"store_dns\": true\n    }\n  }\n}',1,1);
INSERT INTO `client_templates` VALUES (4,'Default User-Agent Template','user_agent','{\n  \"list\":[\n    \"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.0.0 Safari/537.36\",\n    \"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36\",\n    \"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36\",\n    \"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.0.0 Safari/537.36 Edg/125.0.0.0\",\n    \"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.0.0 Safari/537.36\",\n    \"Mozilla/5.0 (iPhone; CPU iPhone OS 17_4_1 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.4.1 Mobile/15E148 Safari/604.1\",\n    \"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/106.0.0.0 Safari/537.36 PageSpeedPlus/1.0.0\",\n    \"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36 Edg/124.0.0.0\",\n    \"Mozilla/5.0 (Linux; Android 10; K) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.0.0 Mobile Safari/537.36\",\n    \"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:126.0) Gecko/20100101 Firefox/126.0\",\n    \"Mozilla/5.0 (Linux; Android 10; K) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Mobile Safari/537.36\",\n    \"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/123.0.0.0 Safari/537.36\",\n    \"Mozilla/5.0 (iPhone; CPU iPhone OS 17_4_1 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/15E148\",\n    \"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.4.1 Safari/605.1.15\",\n    \"Mozilla/5.0 (Macintosh; Intel Mac OS X 11.6; rv:92.0) Gecko/20100101 Firefox/92.0\",\n    \"Mozilla/5.0 (iPhone; CPU iPhone OS 16_6_1 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.6 Mobile/15E148 Safari/604.1\",\n    \"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36\",\n    \"Mozilla/5.0 (Linux; Android 10; K) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.0.0 Safari/537.36\",\n    \"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) obsidian/1.4.14 Chrome/114.0.5735.289 Electron/25.8.1 Safari/537.36\",\n    \"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/123.0.0.0 Safari/537.36\",\n    \"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36\",\n    \"Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:126.0) Gecko/20100101 Firefox/126.0\",\n    \"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.5 Safari/605.1.15\",\n    \"Mozilla/5.0 (Linux; Android 10; K) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/123.0.0.0 Mobile Safari/537.36\",\n    \"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/116.0.0.0 Safari/537.36\",\n    \"Mozilla/5.0 (Linux; Android 10; K) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36\",\n    \"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/109.0.0.0 Safari/537.36\",\n    \"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.0.0 Safari/537.36\",\n    \"Mozilla/5.0 (Linux; Android 10; K) AppleWebKit/537.36 (KHTML, like Gecko) SamsungBrowser/25.0 Chrome/121.0.0.0 Mobile Safari/537.36\",\n    \"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:125.0) Gecko/20100101 Firefox/125.0\",\n    \"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36\",\n    \"Mozilla/5.0 (Linux; Android 10; K) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/121.0.0.0 Mobile Safari/537.36\",\n    \"Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:125.0) Gecko/20100101 Firefox/125.0\",\n    \"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) obsidian/1.4.16 Chrome/114.0.5735.289 Electron/25.8.1 Safari/537.36\",\n    \"Mozilla/5.0 (Linux; Android 10; K) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Mobile Safari/537.36\",\n    \"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36\",\n    \"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/89.0.4389.114 Safari/537.36\",\n    \"Mozilla/5.0 (iPhone; CPU iPhone OS 17_2_1 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.2 Mobile/15E148 Safari/604.1\",\n    \"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/123.0.0.0 Safari/537.36 Edg/123.0.0.0\",\n    \"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.6 Safari/605.1.15\",\n    \"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36\",\n    \"Mozilla/5.0 (Linux; Android 10; K) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Mobile Safari/537.36\",\n    \"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) obsidian/1.5.3 Chrome/114.0.5735.289 Electron/25.8.1 Safari/537.36\",\n    \"Mozilla/5.0 (Linux; Android 10; K) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/123.0.0.0 Safari/537.36\",\n    \"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.1 Safari/605.1.15\",\n    \"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/121.0.0.0 Safari/537.36\",\n    \"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/121.0.0.0 Safari/537.36\",\n    \"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36\",\n    \"Mozilla/5.0 (iPhone; CPU iPhone OS 17_0_1 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Mobile/15E148 Safari/604.1\",\n    \"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36 Edg/122.0.0.0\",\n    \"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.2.1 Safari/605.1.15\",\n    \"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/123.0.0.0 Safari/537.36\",\n    \"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) obsidian/1.5.12 Chrome/120.0.6099.283 Electron/28.2.3 Safari/537.36\",\n    \"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) obsidian/1.4.16 Chrome/114.0.5735.289 Electron/25.8.1 Safari/537.36\",\n    \"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/119.0.0.0 Safari/537.36\",\n    \"Mozilla/5.0 (Linux; Android 10; K) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/119.0.0.0 Mobile Safari/537.36\",\n    \"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/123.0.0.0 Safari/537.36 OPR/109.0.0.0\",\n    \"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/81.0.4044.129 Safari/537.36\",\n    \"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36\",\n    \"Mozilla/5.0 (Linux; Android 10; K) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/116.0.0.0 Mobile Safari/537.36\",\n    \"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.1 Safari/605.1.15\",\n    \"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.3.1 Safari/605.1.15\",\n    \"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) obsidian/1.4.13 Chrome/114.0.5735.289 Electron/25.8.1 Safari/537.36\",\n    \"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Safari/605.1.15\",\n    \"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/119.0.0.0 Safari/537.36\",\n    \"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.2 Safari/605.1.15\",\n    \"Mozilla/5.0 (Windows NT 6.1; Win64; x64; rv:109.0) Gecko/20100101 Firefox/115.0\",\n    \"Mozilla/5.0 (iPhone; CPU iPhone OS 17_5_1 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.5 Mobile/15E148 Safari/604.1\",\n    \"Mozilla/5.0 (iPhone; CPU iPhone OS 17_3 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.3 Mobile/15E148 Safari/604.1\",\n    \"Mozilla/5.0 (iPhone; CPU iPhone OS 16_1_1 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.1 Mobile/15E148 Safari/604.1\",\n    \"Mozilla/5.0 (iPhone; CPU iPhone OS 17_3_1 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.3.1 Mobile/15E148 Safari/604.1\",\n    \"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:124.0) Gecko/20100101 Firefox/124.0\",\n    \"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.3 Safari/605.1.15\",\n    \"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36\",\n    \"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) obsidian/1.4.13 Chrome/114.0.5735.289 Electron/25.8.1 Safari/537.36\",\n    \"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.4 Safari/605.1.15\",\n    \"Mozilla/5.0 (iPhone; CPU iPhone OS 16_6 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.6 Mobile/15E148 Safari/604.1\",\n    \"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/15.6.1 Safari/605.1.15\",\n    \"Mozilla/5.0 (Linux; Android 10; K) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36\",\n    \"Mozilla/5.0 (iPhone; CPU iPhone OS 17_1_1 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.1.1 Mobile/15E148 Safari/604.1\",\n    \"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36 Edg/120.0.0.0\",\n    \"Mozilla/5.0 (iPhone; CPU iPhone OS 17_5 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.5 Mobile/15E148 Safari/604.1\",\n    \"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) obsidian/1.5.12 Chrome/120.0.6099.283 Electron/28.2.3 Safari/537.36\",\n    \"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/117.0.0.0 Safari/537.36\",\n    \"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.1.2 Safari/605.1.15\",\n    \"Mozilla/5.0 (iPhone; CPU iPhone OS 16_2 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.2 Mobile/15E148 Safari/604.1\",\n    \"Mozilla/5.0 (Linux; Android 10; K) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/114.0.0.0 Mobile Safari/537.36\",\n    \"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.5 Safari/605.1.15\",\n    \"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/121.0.0.0 Safari/537.36 Edg/121.0.0.0\",\n    \"Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:124.0) Gecko/20100101 Firefox/124.0\",\n    \"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/118.0.0.0 Safari/537.36\",\n    \"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) obsidian/1.5.3 Chrome/114.0.5735.289 Electron/25.8.1 Safari/537.36\",\n    \"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/101.0.4951.67 Safari/537.36\",\n    \"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.3 Safari/605.1.15\",\n    \"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36\",\n    \"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/121.0.0.0 Safari/537.36\",\n    \"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36 Edg/124.0.0.0\",\n    \"Mozilla/5.0 (Linux; Android 10; K) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/117.0.0.0 Mobile Safari/537.36\",\n    \"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/114.0.0.0 Safari/537.36\",\n    \"Mozilla/5.0 (Linux; Android 10; K) AppleWebKit/537.36 (KHTML, like Gecko) SamsungBrowser/24.0 Chrome/117.0.0.0 Mobile Safari/537.36\"\n  ]\n}',1,1);
INSERT INTO `client_templates` VALUES (5,'Default gRPC User-Agent Template','grpc_user_agent','{\n    \"list\": [\n        \"grpc-dotnet/2.41.0 (.NET 6.0.1; CLR 6.0.1; net6.0; windows; x64)\",\n        \"grpc-dotnet/2.41.0 (.NET 6.0.0-preview.7.21377.19; CLR 6.0.0; net6.0; osx; x64)\",\n        \"grpc-dotnet/2.41.0 (Mono 6.12.0.140; CLR 4.0.30319; netstandard2.0; osx; x64)\",\n        \"grpc-dotnet/2.41.0 (.NET 6.0.0-rc.1.21380.1; CLR 6.0.0; net6.0; linux; arm64)\",\n        \"grpc-dotnet/2.41.0 (.NET 5.0.8; CLR 5.0.8; net5.0; linux; arm64)\",\n        \"grpc-dotnet/2.41.0 (.NET Core; CLR 3.1.4; netstandard2.1; linux; arm64)\",\n        \"grpc-dotnet/2.41.0 (.NET Framework; CLR 4.0.30319.42000; netstandard2.0; windows; x86)\",\n        \"grpc-dotnet/2.41.0 (.NET 6.0.0-rc.1.21380.1; CLR 6.0.0; net6.0; windows; x64)\",\n        \"grpc-python-asyncio/1.62.1 grpc-c/39.0.0 (linux; chttp2)\",\n        \"grpc-go/1.58.1\",\n        \"grpc-java-okhttp/1.55.1\",\n        \"grpc-node/1.7.1 grpc-c/1.7.1 (osx; chttp2)\",\n        \"grpc-node/1.24.2 grpc-c/8.0.0 (linux; chttp2; ganges)\",\n        \"grpc-c++/1.16.0 grpc-c/6.0.0 (linux; nghttp2; hw)\",\n        \"grpc-node/1.19.0 grpc-c/7.0.0 (linux; chttp2; gold)\",\n        \"grpc-ruby/1.62.0 grpc-c/39.0.0 (osx; chttp2)]\"\n    ]\n}',1,1);
/*!40000 ALTER TABLE `client_templates` ENABLE KEYS */;
UNLOCK TABLES;
DROP TABLE IF EXISTS `core_configs`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `core_configs` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  `created_at` datetime NOT NULL,
  `name` varchar(256) NOT NULL,
  `config` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL CHECK (json_valid(`config`)),
  `exclude_inbound_tags` varchar(2048) DEFAULT NULL,
  `fallbacks_inbound_tags` varchar(2048) DEFAULT NULL,
  `type` enum('xray','wg','mtproto','singbox') NOT NULL DEFAULT 'xray',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_uca1400_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

LOCK TABLES `core_configs` WRITE;
/*!40000 ALTER TABLE `core_configs` DISABLE KEYS */;
INSERT INTO `core_configs` VALUES (1,'2026-10-16 13:53:14','Default Core Config','{\"policy\": {\"levels\": {\"0\": {\"statsUserOnline\": true}}}, \"log\": {\"loglevel\": \"warning\"}, \"inbounds\": [{\"tag\": \"Shadowsocks TCP\", \"listen\": \"0.0.0.0\", \"port\": 1080, \"protocol\": \"shadowsocks\", \"settings\": {\"clients\": [], \"network\": \"tcp,udp\"}}], \"outbounds\": [{\"protocol\": \"freedom\", \"tag\": \"DIRECT\"}, {\"protocol\": \"blackhole\", \"tag\": \"BLOCK\"}], \"routing\": {\"rules\": [{\"ip\": [\"geoip:private\"], \"outboundTag\": \"BLOCK\", \"type\": \"field\"}]}}','','','xray');
/*!40000 ALTER TABLE `core_configs` ENABLE KEYS */;
UNLOCK TABLES;
DROP TABLE IF EXISTS `groups`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `groups` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  `name` varchar(64) NOT NULL,
  `is_disabled` tinyint(1) NOT NULL DEFAULT 0,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_uca1400_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

LOCK TABLES `groups` WRITE;
/*!40000 ALTER TABLE `groups` DISABLE KEYS */;
/*!40000 ALTER TABLE `groups` ENABLE KEYS */;
UNLOCK TABLES;
DROP TABLE IF EXISTS `hosts`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `hosts` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  `remark` varchar(256) NOT NULL,
  `address` varchar(256) NOT NULL,
  `port` int(11) DEFAULT NULL,
  `inbound_tag` varchar(256) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL,
  `sni` varchar(1000) DEFAULT NULL,
  `host` varchar(1000) DEFAULT NULL,
  `security` enum('inbound_default','none','tls') NOT NULL DEFAULT 'inbound_default',
  `alpn` varchar(14) DEFAULT NULL,
  `fingerprint` enum('none','chrome','firefox','safari','ios','android','edge','360','qq','random','randomized','randomizednoalpn','unsafe') NOT NULL DEFAULT 'none',
  `allowinsecure` tinyint(1) DEFAULT NULL,
  `is_disabled` tinyint(1) DEFAULT NULL,
  `path` varchar(256) DEFAULT NULL,
  `random_user_agent` tinyint(1) NOT NULL DEFAULT 0,
  `use_sni_as_host` tinyint(1) NOT NULL DEFAULT 0,
  `priority` int(11) NOT NULL,
  `http_headers` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL CHECK (json_valid(`http_headers`)),
  `transport_settings` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL CHECK (json_valid(`transport_settings`)),
  `mux_settings` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL CHECK (json_valid(`mux_settings`)),
  `noise_settings` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL CHECK (json_valid(`noise_settings`)),
  `fragment_settings` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL CHECK (json_valid(`fragment_settings`)),
  `status` varchar(60) DEFAULT NULL,
  `vless_route` varchar(4) DEFAULT NULL,
  `pinned_peer_cert_sha256` varchar(128) DEFAULT NULL,
  `verify_peer_cert_by_name` varchar(1000) DEFAULT NULL,
  `wireguard_overrides` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL CHECK (json_valid(`wireguard_overrides`)),
  `subscription_templates` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL CHECK (json_valid(`subscription_templates`)),
  `final_mask_settings` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL CHECK (json_valid(`final_mask_settings`)),
  `cipher_suites` varchar(1024) DEFAULT NULL,
  `ech` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL CHECK (json_valid(`ech`)),
  PRIMARY KEY (`id`),
  KEY `fk_hosts_inbound_tag_inbounds` (`inbound_tag`),
  CONSTRAINT `fk_hosts_inbound_tag_inbounds` FOREIGN KEY (`inbound_tag`) REFERENCES `inbounds` (`tag`) ON DELETE SET NULL ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_uca1400_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

LOCK TABLES `hosts` WRITE;
/*!40000 ALTER TABLE `hosts` DISABLE KEYS */;
/*!40000 ALTER TABLE `hosts` ENABLE KEYS */;
UNLOCK TABLES;
DROP TABLE IF EXISTS `inbounds`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `inbounds` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  `tag` varchar(256) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `ix_inbounds_tag` (`tag`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_uca1400_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

LOCK TABLES `inbounds` WRITE;
/*!40000 ALTER TABLE `inbounds` DISABLE KEYS */;
/*!40000 ALTER TABLE `inbounds` ENABLE KEYS */;
UNLOCK TABLES;
DROP TABLE IF EXISTS `inbounds_groups_association`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `inbounds_groups_association` (
  `inbound_id` bigint(20) NOT NULL,
  `group_id` bigint(20) NOT NULL,
  PRIMARY KEY (`inbound_id`,`group_id`),
  KEY `fk_inbounds_groups_association_group_id_groups` (`group_id`),
  CONSTRAINT `fk_inbounds_groups_association_group_id_groups` FOREIGN KEY (`group_id`) REFERENCES `groups` (`id`),
  CONSTRAINT `fk_inbounds_groups_association_inbound_id_inbounds` FOREIGN KEY (`inbound_id`) REFERENCES `inbounds` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_uca1400_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

LOCK TABLES `inbounds_groups_association` WRITE;
/*!40000 ALTER TABLE `inbounds_groups_association` DISABLE KEYS */;
/*!40000 ALTER TABLE `inbounds_groups_association` ENABLE KEYS */;
UNLOCK TABLES;
DROP TABLE IF EXISTS `jwt`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `jwt` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  `secret_key` varchar(64) NOT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_uca1400_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

LOCK TABLES `jwt` WRITE;
/*!40000 ALTER TABLE `jwt` DISABLE KEYS */;
/*!40000 ALTER TABLE `jwt` ENABLE KEYS */;
UNLOCK TABLES;
DROP TABLE IF EXISTS `next_plans`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `next_plans` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  `user_id` bigint(20) NOT NULL,
  `data_limit` bigint(20) NOT NULL,
  `expire` int(11) DEFAULT NULL,
  `add_remaining_traffic` tinyint(1) NOT NULL DEFAULT 0,
  `user_template_id` bigint(20) DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `ix_next_plans_user_template_id` (`user_template_id`),
  KEY `fk_next_plans_user_id_users` (`user_id`),
  CONSTRAINT `fk_next_plans_user_id_users` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE,
  CONSTRAINT `fk_next_plans_user_template_id_user_templates` FOREIGN KEY (`user_template_id`) REFERENCES `user_templates` (`id`) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_uca1400_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

LOCK TABLES `next_plans` WRITE;
/*!40000 ALTER TABLE `next_plans` DISABLE KEYS */;
/*!40000 ALTER TABLE `next_plans` ENABLE KEYS */;
UNLOCK TABLES;
DROP TABLE IF EXISTS `node_stats`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `node_stats` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  `created_at` datetime NOT NULL,
  `node_id` bigint(20) NOT NULL,
  `mem_total` bigint(20) NOT NULL,
  `mem_used` bigint(20) NOT NULL,
  `cpu_cores` int(11) NOT NULL,
  `cpu_usage` float NOT NULL,
  `incoming_bandwidth_speed` bigint(20) NOT NULL,
  `outgoing_bandwidth_speed` bigint(20) NOT NULL,
  PRIMARY KEY (`id`),
  KEY `fk_node_stats_node_id_nodes` (`node_id`),
  CONSTRAINT `fk_node_stats_node_id_nodes` FOREIGN KEY (`node_id`) REFERENCES `nodes` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_uca1400_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

LOCK TABLES `node_stats` WRITE;
/*!40000 ALTER TABLE `node_stats` DISABLE KEYS */;
/*!40000 ALTER TABLE `node_stats` ENABLE KEYS */;
UNLOCK TABLES;
DROP TABLE IF EXISTS `node_usage_reset_logs`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `node_usage_reset_logs` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  `created_at` datetime NOT NULL,
  `node_id` bigint(20) NOT NULL,
  `uplink` bigint(20) NOT NULL,
  `downlink` bigint(20) NOT NULL,
  PRIMARY KEY (`id`),
  KEY `ix_node_usage_reset_logs_node_id_created_at` (`node_id`,`created_at`),
  CONSTRAINT `fk_node_usage_reset_logs_node_id_nodes` FOREIGN KEY (`node_id`) REFERENCES `nodes` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_uca1400_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

LOCK TABLES `node_usage_reset_logs` WRITE;
/*!40000 ALTER TABLE `node_usage_reset_logs` DISABLE KEYS */;
/*!40000 ALTER TABLE `node_usage_reset_logs` ENABLE KEYS */;
UNLOCK TABLES;
DROP TABLE IF EXISTS `node_usages`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `node_usages` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  `created_at` datetime NOT NULL,
  `node_id` bigint(20) DEFAULT NULL,
  `uplink` bigint(20) NOT NULL,
  `downlink` bigint(20) NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uq_node_usages_created_at` (`created_at`,`node_id`),
  KEY `ix_node_usages_created_at` (`created_at`),
  KEY `fk_node_usages_node_id_nodes` (`node_id`),
  CONSTRAINT `fk_node_usages_node_id_nodes` FOREIGN KEY (`node_id`) REFERENCES `nodes` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_uca1400_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

LOCK TABLES `node_usages` WRITE;
/*!40000 ALTER TABLE `node_usages` DISABLE KEYS */;
/*!40000 ALTER TABLE `node_usages` ENABLE KEYS */;
UNLOCK TABLES;
DROP TABLE IF EXISTS `node_user_usages`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `node_user_usages` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  `created_at` datetime NOT NULL,
  `user_id` bigint(20) NOT NULL,
  `node_id` bigint(20) DEFAULT NULL,
  `used_traffic` bigint(20) NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uq_node_user_usages_created_at` (`created_at`,`user_id`,`node_id`),
  KEY `ix_node_user_usages_created_at` (`created_at`),
  KEY `ix_node_user_usages_node_id_created_at` (`node_id`,`created_at`),
  KEY `ix_node_user_usages_user_id_created_at` (`user_id`,`created_at`),
  CONSTRAINT `fk_node_user_usages_node_id_nodes` FOREIGN KEY (`node_id`) REFERENCES `nodes` (`id`) ON DELETE CASCADE,
  CONSTRAINT `fk_node_user_usages_user_id_users` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_uca1400_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

LOCK TABLES `node_user_usages` WRITE;
/*!40000 ALTER TABLE `node_user_usages` DISABLE KEYS */;
/*!40000 ALTER TABLE `node_user_usages` ENABLE KEYS */;
UNLOCK TABLES;
DROP TABLE IF EXISTS `nodes`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `nodes` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  `name` varchar(256) NOT NULL,
  `address` varchar(256) NOT NULL,
  `port` int(11) NOT NULL,
  `status` enum('connected','connecting','error','disabled','limited') NOT NULL,
  `last_status_change` datetime DEFAULT NULL,
  `message` varchar(1024) DEFAULT NULL,
  `created_at` datetime NOT NULL,
  `uplink` bigint(20) NOT NULL,
  `downlink` bigint(20) NOT NULL,
  `xray_version` varchar(32) DEFAULT NULL,
  `usage_coefficient` float NOT NULL DEFAULT 1,
  `node_version` varchar(32) DEFAULT NULL,
  `connection_type` enum('grpc','rest') NOT NULL DEFAULT 'grpc',
  `server_ca` varchar(2048) NOT NULL DEFAULT '',
  `keep_alive` int(11) NOT NULL DEFAULT 0,
  `core_config_id` bigint(20) DEFAULT NULL,
  `api_key` varchar(36) DEFAULT NULL,
  `data_limit` bigint(20) NOT NULL DEFAULT 0,
  `data_limit_reset_strategy` enum('no_reset','day','week','month','year') NOT NULL DEFAULT 'no_reset',
  `reset_time` int(11) NOT NULL DEFAULT -1,
  `default_timeout` int(11) NOT NULL DEFAULT 10,
  `internal_timeout` int(11) NOT NULL DEFAULT 15,
  `api_port` int(11) NOT NULL DEFAULT 62051,
  `proxy_url` varchar(256) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uq_nodes_name` (`name`),
  KEY `fk_nodes_core_config_id_core_configs` (`core_config_id`),
  CONSTRAINT `fk_nodes_core_config_id_core_configs` FOREIGN KEY (`core_config_id`) REFERENCES `core_configs` (`id`) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_uca1400_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

LOCK TABLES `nodes` WRITE;
/*!40000 ALTER TABLE `nodes` DISABLE KEYS */;
/*!40000 ALTER TABLE `nodes` ENABLE KEYS */;
UNLOCK TABLES;
DROP TABLE IF EXISTS `notification_reminders`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `notification_reminders` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  `user_id` bigint(20) NOT NULL,
  `type` enum('expiration_date','data_usage') NOT NULL,
  `expires_at` datetime DEFAULT NULL,
  `created_at` datetime NOT NULL,
  `threshold` int(11) DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `fk_notification_reminders_user_id_users` (`user_id`),
  CONSTRAINT `fk_notification_reminders_user_id_users` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_uca1400_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

LOCK TABLES `notification_reminders` WRITE;
/*!40000 ALTER TABLE `notification_reminders` DISABLE KEYS */;
/*!40000 ALTER TABLE `notification_reminders` ENABLE KEYS */;
UNLOCK TABLES;
DROP TABLE IF EXISTS `settings`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `settings` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  `telegram` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL CHECK (json_valid(`telegram`)),
  `webhook` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL CHECK (json_valid(`webhook`)),
  `notification_settings` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL CHECK (json_valid(`notification_settings`)),
  `notification_enable` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL CHECK (json_valid(`notification_enable`)),
  `subscription` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL CHECK (json_valid(`subscription`)),
  `general` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL CHECK (json_valid(`general`)),
  `hwid` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL CHECK (json_valid(`hwid`)),
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_uca1400_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

LOCK TABLES `settings` WRITE;
/*!40000 ALTER TABLE `settings` DISABLE KEYS */;
INSERT INTO `settings` VALUES (1,'{\"enable\": false, \"token\": null, \"webhook_url\": null, \"webhook_secret\": null, \"proxy_url\": null}','{\"enable\": false, \"webhooks\": [], \"days_left\": [3], \"usage_percent\": [80], \"timeout\": 180, \"recurrent\": 3, \"proxy_url\": null}','{\"notify_telegram\": false, \"notify_discord\": false, \"telegram_api_token\": null, \"telegram_chat_id\": null, \"telegram_topic_id\": null, \"discord_webhook_url\": null, \"proxy_url\": null, \"max_retries\": 3}','{\"admin\": {\"create\": true, \"modify\": true, \"delete\": true, \"reset_usage\": true, \"login\": true}, \"core\": {\"create\": true, \"modify\": true, \"delete\": true}, \"group\": {\"create\": true, \"modify\": true, \"delete\": true}, \"host\": {\"create\": true, \"modify\": true, \"delete\": true, \"modify_hosts\": true}, \"node\": {\"create\": true, \"modify\": true, \"delete\": true, \"connect\": true, \"error\": true}, \"user\": {\"create\": true, \"modify\": true, \"delete\": true, \"status_change\": true, \"reset_data_usage\": true, \"data_reset_by_next\": true, \"subscription_revoked\": true}, \"user_template\": {\"create\": true, \"modify\": true, \"delete\": true}, \"days_left\": true, \"percentage_reached\": true}','{\"url_prefix\": \"\", \"update_interval\": \"12\", \"support_url\": \"https://t.me/\", \"profile_title\": \"Subscription\", \"host_status_filter\": true, \"randomize_order\": false, \"rules\": [{\"pattern\": \"^(?:FlClashX?|Flowvy|[Cc]lash(?:-(?:[Vv]erge|nyanpasu)|X [Mm]eta|-?[Mm]eta)|[Kk]oala-[Cc]lash|[Mm](?:urge|ihomo)|prizrak-box|clash\\\\.meta)\", \"target\": \"clash_meta\"}, {\"pattern\": \"^([Cc]lash|[Ss]tash)\", \"target\": \"clash\"}, {\"pattern\": \"^(SFA|SFI|SFM|SFT|[Kk]aring|[Hh]iddify[Nn]ext)|.*[Ss]ing[\\\\-b]?ox.*\", \"target\": \"sing_box\"}, {\"pattern\": \"^(SS|SSR|SSD|SSS|Outline|Shadowsocks|SSconf)\", \"target\": \"outline\"}, {\"pattern\": \"^[Ii]n[Hh]ive\", \"target\": \"xray\"}, {\"pattern\": \"^.*\", \"target\": \"links_base64\"}], \"manual_sub_request\": {\"links\": true, \"links_base64\": true, \"xray\": true, \"sing_box\": true, \"clash\": true, \"clash_meta\": true, \"outline\": true}}','{\"default_flow\": \"\", \"default_method\": \"chacha20-ietf-poly1305\"}','{}');
/*!40000 ALTER TABLE `settings` ENABLE KEYS */;
UNLOCK TABLES;
DROP TABLE IF EXISTS `system`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `system` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  `uplink` bigint(20) NOT NULL,
  `downlink` bigint(20) NOT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_uca1400_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

LOCK TABLES `system` WRITE;
/*!40000 ALTER TABLE `system` DISABLE KEYS */;
INSERT INTO `system` VALUES (1,0,0);
/*!40000 ALTER TABLE `system` ENABLE KEYS */;
UNLOCK TABLES;
DROP TABLE IF EXISTS `temp_keys`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `temp_keys` (
  `key` varchar(36) NOT NULL,
  `action` varchar(32) NOT NULL,
  `expires_at` datetime NOT NULL,
  `used_at` datetime DEFAULT NULL,
  `used_by_ip` varchar(45) DEFAULT NULL,
  PRIMARY KEY (`key`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_uca1400_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

LOCK TABLES `temp_keys` WRITE;
/*!40000 ALTER TABLE `temp_keys` DISABLE KEYS */;
/*!40000 ALTER TABLE `temp_keys` ENABLE KEYS */;
UNLOCK TABLES;
DROP TABLE IF EXISTS `template_group_association`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `template_group_association` (
  `user_template_id` bigint(20) DEFAULT NULL,
  `group_id` bigint(20) DEFAULT NULL,
  KEY `fk_template_group_association_group_id_groups` (`group_id`),
  KEY `fk_template_group_association_user_template_id_user_templates` (`user_template_id`),
  CONSTRAINT `fk_template_group_association_group_id_groups` FOREIGN KEY (`group_id`) REFERENCES `groups` (`id`),
  CONSTRAINT `fk_template_group_association_user_template_id_user_templates` FOREIGN KEY (`user_template_id`) REFERENCES `user_templates` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_uca1400_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

LOCK TABLES `template_group_association` WRITE;
/*!40000 ALTER TABLE `template_group_association` DISABLE KEYS */;
/*!40000 ALTER TABLE `template_group_association` ENABLE KEYS */;
UNLOCK TABLES;
DROP TABLE IF EXISTS `user_hwids`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `user_hwids` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  `user_id` bigint(20) NOT NULL,
  `hwid` varchar(256) NOT NULL,
  `device_os` varchar(256) DEFAULT NULL,
  `os_version` varchar(128) DEFAULT NULL,
  `device_model` varchar(256) DEFAULT NULL,
  `created_at` datetime NOT NULL,
  `last_used_at` datetime NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uq_user_hwids_user_id` (`user_id`,`hwid`),
  KEY `ix_user_hwids_user_id` (`user_id`),
  KEY `ix_user_hwids_hwid` (`hwid`),
  KEY `ix_user_hwids_created_at` (`created_at`),
  KEY `ix_user_hwids_last_used_at` (`last_used_at`),
  CONSTRAINT `fk_user_hwids_user_id_users` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_uca1400_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

LOCK TABLES `user_hwids` WRITE;
/*!40000 ALTER TABLE `user_hwids` DISABLE KEYS */;
/*!40000 ALTER TABLE `user_hwids` ENABLE KEYS */;
UNLOCK TABLES;
DROP TABLE IF EXISTS `user_subscription_updates`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `user_subscription_updates` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  `user_id` bigint(20) NOT NULL,
  `created_at` datetime NOT NULL,
  `user_agent` varchar(512) NOT NULL,
  `ip` varchar(64) DEFAULT NULL,
  `hwid` varchar(256) DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `idx_user_subscription_updates_user_id` (`user_id`),
  KEY `idx_user_subscription_updates_user_created` (`user_id`,`created_at`),
  CONSTRAINT `fk_user_subscription_updates_user_id_users` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_uca1400_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

LOCK TABLES `user_subscription_updates` WRITE;
/*!40000 ALTER TABLE `user_subscription_updates` DISABLE KEYS */;
/*!40000 ALTER TABLE `user_subscription_updates` ENABLE KEYS */;
UNLOCK TABLES;
DROP TABLE IF EXISTS `user_templates`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `user_templates` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  `name` varchar(64) NOT NULL,
  `data_limit` bigint(20) NOT NULL,
  `expire_duration` bigint(20) NOT NULL,
  `username_prefix` varchar(20) DEFAULT NULL,
  `username_suffix` varchar(20) DEFAULT NULL,
  `extra_settings` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL CHECK (json_valid(`extra_settings`)),
  `status` enum('active','on_hold') NOT NULL DEFAULT 'active',
  `reset_usages` tinyint(1) NOT NULL DEFAULT 0,
  `on_hold_timeout` int(11) DEFAULT NULL,
  `data_limit_reset_strategy` enum('no_reset','day','week','month','year') NOT NULL DEFAULT 'no_reset',
  `is_disabled` tinyint(1) NOT NULL DEFAULT 0,
  `hwid_limit` bigint(20) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uq_user_templates_name` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_uca1400_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

LOCK TABLES `user_templates` WRITE;
/*!40000 ALTER TABLE `user_templates` DISABLE KEYS */;
/*!40000 ALTER TABLE `user_templates` ENABLE KEYS */;
UNLOCK TABLES;
DROP TABLE IF EXISTS `user_usage_logs`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `user_usage_logs` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  `user_id` bigint(20) DEFAULT NULL,
  `used_traffic_at_reset` bigint(20) NOT NULL,
  `reset_at` datetime NOT NULL,
  PRIMARY KEY (`id`),
  KEY `ix_user_usage_logs_user_id_reset_at` (`user_id`,`reset_at`),
  CONSTRAINT `fk_user_usage_logs_user_id_users` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_uca1400_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

LOCK TABLES `user_usage_logs` WRITE;
/*!40000 ALTER TABLE `user_usage_logs` DISABLE KEYS */;
/*!40000 ALTER TABLE `user_usage_logs` ENABLE KEYS */;
UNLOCK TABLES;
DROP TABLE IF EXISTS `users`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `users` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  `username` varchar(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL,
  `status` enum('active','disabled','limited','expired','on_hold') NOT NULL,
  `used_traffic` bigint(20) NOT NULL,
  `data_limit` bigint(20) DEFAULT NULL,
  `created_at` datetime NOT NULL DEFAULT current_timestamp(),
  `admin_id` bigint(20) DEFAULT NULL,
  `data_limit_reset_strategy` enum('no_reset','day','week','month','year') NOT NULL DEFAULT 'no_reset',
  `sub_revoked_at` datetime DEFAULT NULL,
  `note` varchar(500) DEFAULT NULL,
  `online_at` datetime DEFAULT NULL,
  `edit_at` datetime DEFAULT NULL,
  `on_hold_timeout` datetime DEFAULT NULL,
  `on_hold_expire_duration` bigint(20) DEFAULT NULL,
  `auto_delete_in_days` int(11) DEFAULT NULL,
  `last_status_change` datetime DEFAULT NULL,
  `expire` datetime DEFAULT NULL,
  `proxy_settings` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL CHECK (json_valid(`proxy_settings`)),
  `hwid_limit` bigint(20) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `ix_users_username` (`username`),
  KEY `idx_users_admin_online` (`admin_id`,`online_at`),
  KEY `idx_users_admin_status` (`admin_id`,`status`),
  KEY `idx_users_admin_created` (`admin_id`,`created_at`),
  CONSTRAINT `fk_users_admin_id_admins` FOREIGN KEY (`admin_id`) REFERENCES `admins` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_uca1400_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

LOCK TABLES `users` WRITE;
/*!40000 ALTER TABLE `users` DISABLE KEYS */;
/*!40000 ALTER TABLE `users` ENABLE KEYS */;
UNLOCK TABLES;
DROP TABLE IF EXISTS `users_groups_association`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `users_groups_association` (
  `user_id` bigint(20) NOT NULL,
  `groups_id` bigint(20) NOT NULL,
  PRIMARY KEY (`user_id`,`groups_id`),
  KEY `fk_users_groups_association_groups_id_groups` (`groups_id`),
  CONSTRAINT `fk_users_groups_association_groups_id_groups` FOREIGN KEY (`groups_id`) REFERENCES `groups` (`id`),
  CONSTRAINT `fk_users_groups_association_user_id_users` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_uca1400_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

LOCK TABLES `users_groups_association` WRITE;
/*!40000 ALTER TABLE `users_groups_association` DISABLE KEYS */;
/*!40000 ALTER TABLE `users_groups_association` ENABLE KEYS */;
UNLOCK TABLES;
DROP TABLE IF EXISTS `wireguard_subnets`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `wireguard_subnets` (
  `id` bigint(20) NOT NULL AUTO_INCREMENT,
  `network` varchar(64) NOT NULL,
  `next_offset` bigint(20) NOT NULL DEFAULT 1,
  `free_offsets` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL CHECK (json_valid(`free_offsets`)),
  PRIMARY KEY (`id`),
  UNIQUE KEY `uq_wireguard_subnets_network` (`network`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_uca1400_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

LOCK TABLES `wireguard_subnets` WRITE;
/*!40000 ALTER TABLE `wireguard_subnets` DISABLE KEYS */;
/*!40000 ALTER TABLE `wireguard_subnets` ENABLE KEYS */;
UNLOCK TABLES;
/*!40103 SET TIME_ZONE=@OLD_TIME_ZONE */;

/*!40101 SET SQL_MODE=@OLD_SQL_MODE */;
/*!40014 SET FOREIGN_KEY_CHECKS=@OLD_FOREIGN_KEY_CHECKS */;
/*!40014 SET UNIQUE_CHECKS=@OLD_UNIQUE_CHECKS */;
/*!40101 SET CHARACTER_SET_CLIENT=@OLD_CHARACTER_SET_CLIENT */;
/*!40101 SET CHARACTER_SET_RESULTS=@OLD_CHARACTER_SET_RESULTS */;
/*!40101 SET COLLATION_CONNECTION=@OLD_COLLATION_CONNECTION */;
/*M!100616 SET NOTE_VERBOSITY=@OLD_NOTE_VERBOSITY */;

