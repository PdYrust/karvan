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
DROP TABLE IF EXISTS `admin_usage_logs`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `admin_usage_logs` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `admin_id` int(11) DEFAULT NULL,
  `used_traffic_at_reset` bigint(20) NOT NULL,
  `reset_at` datetime DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `admin_id` (`admin_id`),
  CONSTRAINT `admin_usage_logs_ibfk_1` FOREIGN KEY (`admin_id`) REFERENCES `admins` (`id`)
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
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `username` varchar(34) DEFAULT NULL,
  `hashed_password` varchar(128) DEFAULT NULL,
  `created_at` datetime DEFAULT NULL,
  `is_sudo` tinyint(1) DEFAULT 0,
  `password_reset_at` datetime DEFAULT NULL,
  `telegram_id` bigint(20) DEFAULT NULL,
  `discord_webhook` varchar(1024) DEFAULT NULL,
  `users_usage` bigint(20) NOT NULL DEFAULT 0,
  PRIMARY KEY (`id`),
  UNIQUE KEY `ix_admins_username` (`username`)
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
INSERT INTO `alembic_version` VALUES ('2b231de97dc3');
/*!40000 ALTER TABLE `alembic_version` ENABLE KEYS */;
UNLOCK TABLES;
DROP TABLE IF EXISTS `exclude_inbounds_association`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `exclude_inbounds_association` (
  `proxy_id` int(11) DEFAULT NULL,
  `inbound_tag` varchar(256) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL,
  KEY `proxy_id` (`proxy_id`),
  KEY `exclude_inbounds_association_ibfk_1` (`inbound_tag`),
  CONSTRAINT `exclude_inbounds_association_ibfk_1` FOREIGN KEY (`inbound_tag`) REFERENCES `inbounds` (`tag`),
  CONSTRAINT `exclude_inbounds_association_ibfk_2` FOREIGN KEY (`proxy_id`) REFERENCES `proxies` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_uca1400_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

LOCK TABLES `exclude_inbounds_association` WRITE;
/*!40000 ALTER TABLE `exclude_inbounds_association` DISABLE KEYS */;
/*!40000 ALTER TABLE `exclude_inbounds_association` ENABLE KEYS */;
UNLOCK TABLES;
DROP TABLE IF EXISTS `hosts`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `hosts` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `remark` varchar(256) NOT NULL,
  `address` varchar(256) NOT NULL,
  `port` int(11) DEFAULT NULL,
  `inbound_tag` varchar(256) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL,
  `sni` varchar(1000) DEFAULT NULL,
  `host` varchar(1000) DEFAULT NULL,
  `security` enum('inbound_default','none','tls') NOT NULL DEFAULT 'inbound_default',
  `alpn` enum('h3','h3,h2','h3,h2,http/1.1','none','h2','http/1.1','h2,http/1.1') NOT NULL,
  `fingerprint` enum('none','chrome','firefox','safari','ios','android','edge','360','qq','random','randomized') NOT NULL DEFAULT 'none',
  `allowinsecure` tinyint(1) DEFAULT NULL,
  `is_disabled` tinyint(1) DEFAULT NULL,
  `path` varchar(256) DEFAULT NULL,
  `mux_enable` tinyint(1) NOT NULL DEFAULT 0,
  `fragment_setting` varchar(100) DEFAULT NULL,
  `random_user_agent` tinyint(1) NOT NULL DEFAULT 0,
  `noise_setting` varchar(2000) DEFAULT NULL,
  `use_sni_as_host` tinyint(1) NOT NULL DEFAULT 0,
  PRIMARY KEY (`id`),
  KEY `hosts_ibfk_1` (`inbound_tag`),
  CONSTRAINT `hosts_ibfk_1` FOREIGN KEY (`inbound_tag`) REFERENCES `inbounds` (`tag`)
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
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `tag` varchar(256) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `ix_inbounds_tag` (`tag`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_uca1400_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

LOCK TABLES `inbounds` WRITE;
/*!40000 ALTER TABLE `inbounds` DISABLE KEYS */;
/*!40000 ALTER TABLE `inbounds` ENABLE KEYS */;
UNLOCK TABLES;
DROP TABLE IF EXISTS `jwt`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `jwt` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
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
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `user_id` int(11) NOT NULL,
  `data_limit` bigint(20) NOT NULL,
  `expire` int(11) DEFAULT NULL,
  `add_remaining_traffic` tinyint(1) NOT NULL DEFAULT 0,
  `fire_on_either` tinyint(1) NOT NULL DEFAULT 0,
  PRIMARY KEY (`id`),
  KEY `user_id` (`user_id`),
  CONSTRAINT `next_plans_ibfk_1` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_uca1400_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

LOCK TABLES `next_plans` WRITE;
/*!40000 ALTER TABLE `next_plans` DISABLE KEYS */;
/*!40000 ALTER TABLE `next_plans` ENABLE KEYS */;
UNLOCK TABLES;
DROP TABLE IF EXISTS `node_usages`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `node_usages` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `created_at` datetime NOT NULL,
  `node_id` int(11) DEFAULT NULL,
  `uplink` bigint(20) DEFAULT NULL,
  `downlink` bigint(20) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `created_at` (`created_at`,`node_id`),
  KEY `node_id` (`node_id`),
  CONSTRAINT `node_usages_ibfk_1` FOREIGN KEY (`node_id`) REFERENCES `nodes` (`id`)
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
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `created_at` datetime NOT NULL,
  `user_id` int(11) DEFAULT NULL,
  `node_id` int(11) DEFAULT NULL,
  `used_traffic` bigint(20) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `created_at` (`created_at`,`user_id`,`node_id`),
  KEY `node_id` (`node_id`),
  KEY `user_id` (`user_id`),
  CONSTRAINT `node_user_usages_ibfk_1` FOREIGN KEY (`node_id`) REFERENCES `nodes` (`id`),
  CONSTRAINT `node_user_usages_ibfk_2` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`)
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
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `name` varchar(256) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL,
  `address` varchar(256) NOT NULL,
  `port` int(11) NOT NULL,
  `api_port` int(11) NOT NULL,
  `status` enum('connected','connecting','error','disabled') NOT NULL,
  `last_status_change` datetime DEFAULT NULL,
  `message` varchar(1024) DEFAULT NULL,
  `created_at` datetime DEFAULT NULL,
  `uplink` bigint(20) DEFAULT NULL,
  `downlink` bigint(20) DEFAULT NULL,
  `xray_version` varchar(32) DEFAULT NULL,
  `usage_coefficient` float NOT NULL DEFAULT 1,
  PRIMARY KEY (`id`),
  UNIQUE KEY `name` (`name`)
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
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `user_id` int(11) DEFAULT NULL,
  `type` enum('expiration_date','data_usage') NOT NULL,
  `expires_at` datetime DEFAULT NULL,
  `created_at` datetime DEFAULT NULL,
  `threshold` int(11) DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `user_id` (`user_id`),
  CONSTRAINT `notification_reminders_ibfk_1` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_uca1400_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

LOCK TABLES `notification_reminders` WRITE;
/*!40000 ALTER TABLE `notification_reminders` DISABLE KEYS */;
/*!40000 ALTER TABLE `notification_reminders` ENABLE KEYS */;
UNLOCK TABLES;
DROP TABLE IF EXISTS `proxies`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `proxies` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `user_id` int(11) DEFAULT NULL,
  `type` enum('VMess','VLESS','Trojan','Shadowsocks') NOT NULL,
  `settings` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL CHECK (json_valid(`settings`)),
  PRIMARY KEY (`id`),
  KEY `user_id` (`user_id`),
  CONSTRAINT `proxies_ibfk_1` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_uca1400_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

LOCK TABLES `proxies` WRITE;
/*!40000 ALTER TABLE `proxies` DISABLE KEYS */;
/*!40000 ALTER TABLE `proxies` ENABLE KEYS */;
UNLOCK TABLES;
DROP TABLE IF EXISTS `system`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `system` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `uplink` bigint(20) DEFAULT NULL,
  `downlink` bigint(20) DEFAULT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_uca1400_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

LOCK TABLES `system` WRITE;
/*!40000 ALTER TABLE `system` DISABLE KEYS */;
INSERT INTO `system` VALUES (1,0,0);
/*!40000 ALTER TABLE `system` ENABLE KEYS */;
UNLOCK TABLES;
DROP TABLE IF EXISTS `template_inbounds_association`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `template_inbounds_association` (
  `user_template_id` int(11) DEFAULT NULL,
  `inbound_tag` varchar(256) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL,
  KEY `user_template_id` (`user_template_id`),
  KEY `template_inbounds_association_ibfk_1` (`inbound_tag`),
  CONSTRAINT `template_inbounds_association_ibfk_1` FOREIGN KEY (`inbound_tag`) REFERENCES `inbounds` (`tag`),
  CONSTRAINT `template_inbounds_association_ibfk_2` FOREIGN KEY (`user_template_id`) REFERENCES `user_templates` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_uca1400_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

LOCK TABLES `template_inbounds_association` WRITE;
/*!40000 ALTER TABLE `template_inbounds_association` DISABLE KEYS */;
/*!40000 ALTER TABLE `template_inbounds_association` ENABLE KEYS */;
UNLOCK TABLES;
DROP TABLE IF EXISTS `tls`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `tls` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `key` varchar(4096) NOT NULL,
  `certificate` varchar(2048) NOT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_uca1400_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

LOCK TABLES `tls` WRITE;
/*!40000 ALTER TABLE `tls` DISABLE KEYS */;
/*!40000 ALTER TABLE `tls` ENABLE KEYS */;
UNLOCK TABLES;
DROP TABLE IF EXISTS `user_templates`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8mb4 */;
CREATE TABLE `user_templates` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `name` varchar(64) NOT NULL,
  `data_limit` bigint(20) DEFAULT NULL,
  `expire_duration` bigint(20) DEFAULT NULL,
  `username_prefix` varchar(20) DEFAULT NULL,
  `username_suffix` varchar(20) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `name` (`name`)
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
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `user_id` int(11) DEFAULT NULL,
  `used_traffic_at_reset` bigint(20) NOT NULL,
  `reset_at` datetime DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `user_id` (`user_id`),
  CONSTRAINT `user_usage_logs_ibfk_1` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`)
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
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `username` varchar(34) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL,
  `status` enum('on_hold','active','limited','expired','disabled') NOT NULL,
  `used_traffic` bigint(20) DEFAULT NULL,
  `data_limit` bigint(20) DEFAULT NULL,
  `expire` int(11) DEFAULT NULL,
  `created_at` datetime DEFAULT current_timestamp(),
  `admin_id` int(11) DEFAULT NULL,
  `data_limit_reset_strategy` enum('no_reset','day','week','month','year') NOT NULL DEFAULT 'no_reset',
  `sub_revoked_at` datetime DEFAULT NULL,
  `note` varchar(500) DEFAULT NULL,
  `sub_updated_at` datetime DEFAULT NULL,
  `sub_last_user_agent` varchar(512) DEFAULT NULL,
  `online_at` datetime DEFAULT NULL,
  `edit_at` datetime DEFAULT NULL,
  `on_hold_timeout` datetime DEFAULT NULL,
  `on_hold_expire_duration` bigint(20) DEFAULT NULL,
  `auto_delete_in_days` int(11) DEFAULT NULL,
  `last_status_change` datetime DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `ix_users_username` (`username`),
  KEY `fk_users_admin_id_admins` (`admin_id`),
  CONSTRAINT `fk_users_admin_id_admins` FOREIGN KEY (`admin_id`) REFERENCES `admins` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_uca1400_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

LOCK TABLES `users` WRITE;
/*!40000 ALTER TABLE `users` DISABLE KEYS */;
/*!40000 ALTER TABLE `users` ENABLE KEYS */;
UNLOCK TABLES;
/*!40103 SET TIME_ZONE=@OLD_TIME_ZONE */;

/*!40101 SET SQL_MODE=@OLD_SQL_MODE */;
/*!40014 SET FOREIGN_KEY_CHECKS=@OLD_FOREIGN_KEY_CHECKS */;
/*!40014 SET UNIQUE_CHECKS=@OLD_UNIQUE_CHECKS */;
/*!40101 SET CHARACTER_SET_CLIENT=@OLD_CHARACTER_SET_CLIENT */;
/*!40101 SET CHARACTER_SET_RESULTS=@OLD_CHARACTER_SET_RESULTS */;
/*!40101 SET COLLATION_CONNECTION=@OLD_COLLATION_CONNECTION */;
/*M!100616 SET NOTE_VERBOSITY=@OLD_NOTE_VERBOSITY */;

