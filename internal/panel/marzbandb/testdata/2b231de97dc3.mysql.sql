
/*!40101 SET @OLD_CHARACTER_SET_CLIENT=@@CHARACTER_SET_CLIENT */;
/*!40101 SET @OLD_CHARACTER_SET_RESULTS=@@CHARACTER_SET_RESULTS */;
/*!40101 SET @OLD_COLLATION_CONNECTION=@@COLLATION_CONNECTION */;
/*!50503 SET NAMES utf8mb4 */;
/*!40103 SET @OLD_TIME_ZONE=@@TIME_ZONE */;
/*!40103 SET TIME_ZONE='+00:00' */;
/*!40014 SET @OLD_UNIQUE_CHECKS=@@UNIQUE_CHECKS, UNIQUE_CHECKS=0 */;
/*!40014 SET @OLD_FOREIGN_KEY_CHECKS=@@FOREIGN_KEY_CHECKS, FOREIGN_KEY_CHECKS=0 */;
/*!40101 SET @OLD_SQL_MODE=@@SQL_MODE, SQL_MODE='NO_AUTO_VALUE_ON_ZERO' */;
/*!40111 SET @OLD_SQL_NOTES=@@SQL_NOTES, SQL_NOTES=0 */;
DROP TABLE IF EXISTS `admin_usage_logs`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `admin_usage_logs` (
  `id` int NOT NULL AUTO_INCREMENT,
  `admin_id` int DEFAULT NULL,
  `used_traffic_at_reset` bigint NOT NULL,
  `reset_at` datetime DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `admin_id` (`admin_id`),
  CONSTRAINT `admin_usage_logs_ibfk_1` FOREIGN KEY (`admin_id`) REFERENCES `admins` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

LOCK TABLES `admin_usage_logs` WRITE;
/*!40000 ALTER TABLE `admin_usage_logs` DISABLE KEYS */;
/*!40000 ALTER TABLE `admin_usage_logs` ENABLE KEYS */;
UNLOCK TABLES;
DROP TABLE IF EXISTS `admins`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `admins` (
  `id` int NOT NULL AUTO_INCREMENT,
  `username` varchar(34) DEFAULT NULL,
  `hashed_password` varchar(128) DEFAULT NULL,
  `created_at` datetime DEFAULT NULL,
  `is_sudo` tinyint(1) DEFAULT '0',
  `password_reset_at` datetime DEFAULT NULL,
  `telegram_id` bigint DEFAULT NULL,
  `discord_webhook` varchar(1024) DEFAULT NULL,
  `users_usage` bigint NOT NULL DEFAULT '0',
  PRIMARY KEY (`id`),
  UNIQUE KEY `ix_admins_username` (`username`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

LOCK TABLES `admins` WRITE;
/*!40000 ALTER TABLE `admins` DISABLE KEYS */;
/*!40000 ALTER TABLE `admins` ENABLE KEYS */;
UNLOCK TABLES;
DROP TABLE IF EXISTS `alembic_version`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `alembic_version` (
  `version_num` varchar(32) NOT NULL,
  PRIMARY KEY (`version_num`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

LOCK TABLES `alembic_version` WRITE;
/*!40000 ALTER TABLE `alembic_version` DISABLE KEYS */;
INSERT INTO `alembic_version` VALUES ('2b231de97dc3');
/*!40000 ALTER TABLE `alembic_version` ENABLE KEYS */;
UNLOCK TABLES;
DROP TABLE IF EXISTS `exclude_inbounds_association`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `exclude_inbounds_association` (
  `proxy_id` int DEFAULT NULL,
  `inbound_tag` varchar(256) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL,
  KEY `proxy_id` (`proxy_id`),
  KEY `exclude_inbounds_association_ibfk_1` (`inbound_tag`),
  CONSTRAINT `exclude_inbounds_association_ibfk_1` FOREIGN KEY (`inbound_tag`) REFERENCES `inbounds` (`tag`),
  CONSTRAINT `exclude_inbounds_association_ibfk_2` FOREIGN KEY (`proxy_id`) REFERENCES `proxies` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

LOCK TABLES `exclude_inbounds_association` WRITE;
/*!40000 ALTER TABLE `exclude_inbounds_association` DISABLE KEYS */;
/*!40000 ALTER TABLE `exclude_inbounds_association` ENABLE KEYS */;
UNLOCK TABLES;
DROP TABLE IF EXISTS `hosts`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `hosts` (
  `id` int NOT NULL AUTO_INCREMENT,
  `remark` varchar(256) NOT NULL,
  `address` varchar(256) NOT NULL,
  `port` int DEFAULT NULL,
  `inbound_tag` varchar(256) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL,
  `sni` varchar(1000) DEFAULT NULL,
  `host` varchar(1000) DEFAULT NULL,
  `security` enum('inbound_default','none','tls') NOT NULL DEFAULT 'inbound_default',
  `alpn` enum('h3','h3,h2','h3,h2,http/1.1','none','h2','http/1.1','h2,http/1.1') NOT NULL,
  `fingerprint` enum('none','chrome','firefox','safari','ios','android','edge','360','qq','random','randomized') NOT NULL DEFAULT 'none',
  `allowinsecure` tinyint(1) DEFAULT NULL,
  `is_disabled` tinyint(1) DEFAULT NULL,
  `path` varchar(256) DEFAULT NULL,
  `mux_enable` tinyint(1) NOT NULL DEFAULT '0',
  `fragment_setting` varchar(100) DEFAULT NULL,
  `random_user_agent` tinyint(1) NOT NULL DEFAULT '0',
  `noise_setting` varchar(2000) DEFAULT NULL,
  `use_sni_as_host` tinyint(1) NOT NULL DEFAULT '0',
  PRIMARY KEY (`id`),
  KEY `hosts_ibfk_1` (`inbound_tag`),
  CONSTRAINT `hosts_ibfk_1` FOREIGN KEY (`inbound_tag`) REFERENCES `inbounds` (`tag`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

LOCK TABLES `hosts` WRITE;
/*!40000 ALTER TABLE `hosts` DISABLE KEYS */;
/*!40000 ALTER TABLE `hosts` ENABLE KEYS */;
UNLOCK TABLES;
DROP TABLE IF EXISTS `inbounds`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `inbounds` (
  `id` int NOT NULL AUTO_INCREMENT,
  `tag` varchar(256) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `ix_inbounds_tag` (`tag`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

LOCK TABLES `inbounds` WRITE;
/*!40000 ALTER TABLE `inbounds` DISABLE KEYS */;
/*!40000 ALTER TABLE `inbounds` ENABLE KEYS */;
UNLOCK TABLES;
DROP TABLE IF EXISTS `jwt`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `jwt` (
  `id` int NOT NULL AUTO_INCREMENT,
  `secret_key` varchar(64) NOT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

LOCK TABLES `jwt` WRITE;
/*!40000 ALTER TABLE `jwt` DISABLE KEYS */;
INSERT INTO `jwt` VALUES (1,'e0e3cc998e5a702d12e157a8870ec7cc4c3a6973800cb62b1454a2552d995510');
/*!40000 ALTER TABLE `jwt` ENABLE KEYS */;
UNLOCK TABLES;
DROP TABLE IF EXISTS `next_plans`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `next_plans` (
  `id` int NOT NULL AUTO_INCREMENT,
  `user_id` int NOT NULL,
  `data_limit` bigint NOT NULL,
  `expire` int DEFAULT NULL,
  `add_remaining_traffic` tinyint(1) NOT NULL DEFAULT '0',
  `fire_on_either` tinyint(1) NOT NULL DEFAULT '0',
  PRIMARY KEY (`id`),
  KEY `user_id` (`user_id`),
  CONSTRAINT `next_plans_ibfk_1` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

LOCK TABLES `next_plans` WRITE;
/*!40000 ALTER TABLE `next_plans` DISABLE KEYS */;
/*!40000 ALTER TABLE `next_plans` ENABLE KEYS */;
UNLOCK TABLES;
DROP TABLE IF EXISTS `node_usages`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `node_usages` (
  `id` int NOT NULL AUTO_INCREMENT,
  `created_at` datetime NOT NULL,
  `node_id` int DEFAULT NULL,
  `uplink` bigint DEFAULT NULL,
  `downlink` bigint DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `created_at` (`created_at`,`node_id`),
  KEY `node_id` (`node_id`),
  CONSTRAINT `node_usages_ibfk_1` FOREIGN KEY (`node_id`) REFERENCES `nodes` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

LOCK TABLES `node_usages` WRITE;
/*!40000 ALTER TABLE `node_usages` DISABLE KEYS */;
/*!40000 ALTER TABLE `node_usages` ENABLE KEYS */;
UNLOCK TABLES;
DROP TABLE IF EXISTS `node_user_usages`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `node_user_usages` (
  `id` int NOT NULL AUTO_INCREMENT,
  `created_at` datetime NOT NULL,
  `user_id` int DEFAULT NULL,
  `node_id` int DEFAULT NULL,
  `used_traffic` bigint DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `created_at` (`created_at`,`user_id`,`node_id`),
  KEY `node_id` (`node_id`),
  KEY `user_id` (`user_id`),
  CONSTRAINT `node_user_usages_ibfk_1` FOREIGN KEY (`node_id`) REFERENCES `nodes` (`id`),
  CONSTRAINT `node_user_usages_ibfk_2` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

LOCK TABLES `node_user_usages` WRITE;
/*!40000 ALTER TABLE `node_user_usages` DISABLE KEYS */;
/*!40000 ALTER TABLE `node_user_usages` ENABLE KEYS */;
UNLOCK TABLES;
DROP TABLE IF EXISTS `nodes`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `nodes` (
  `id` int NOT NULL AUTO_INCREMENT,
  `name` varchar(256) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL,
  `address` varchar(256) NOT NULL,
  `port` int NOT NULL,
  `api_port` int NOT NULL,
  `status` enum('connected','connecting','error','disabled') NOT NULL,
  `last_status_change` datetime DEFAULT NULL,
  `message` varchar(1024) DEFAULT NULL,
  `created_at` datetime DEFAULT NULL,
  `uplink` bigint DEFAULT NULL,
  `downlink` bigint DEFAULT NULL,
  `xray_version` varchar(32) DEFAULT NULL,
  `usage_coefficient` float NOT NULL DEFAULT '1',
  PRIMARY KEY (`id`),
  UNIQUE KEY `name` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

LOCK TABLES `nodes` WRITE;
/*!40000 ALTER TABLE `nodes` DISABLE KEYS */;
/*!40000 ALTER TABLE `nodes` ENABLE KEYS */;
UNLOCK TABLES;
DROP TABLE IF EXISTS `notification_reminders`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `notification_reminders` (
  `id` int NOT NULL AUTO_INCREMENT,
  `user_id` int DEFAULT NULL,
  `type` enum('expiration_date','data_usage') NOT NULL,
  `expires_at` datetime DEFAULT NULL,
  `created_at` datetime DEFAULT NULL,
  `threshold` int DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `user_id` (`user_id`),
  CONSTRAINT `notification_reminders_ibfk_1` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

LOCK TABLES `notification_reminders` WRITE;
/*!40000 ALTER TABLE `notification_reminders` DISABLE KEYS */;
/*!40000 ALTER TABLE `notification_reminders` ENABLE KEYS */;
UNLOCK TABLES;
DROP TABLE IF EXISTS `proxies`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `proxies` (
  `id` int NOT NULL AUTO_INCREMENT,
  `user_id` int DEFAULT NULL,
  `type` enum('VMess','VLESS','Trojan','Shadowsocks') NOT NULL,
  `settings` json NOT NULL,
  PRIMARY KEY (`id`),
  KEY `user_id` (`user_id`),
  CONSTRAINT `proxies_ibfk_1` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

LOCK TABLES `proxies` WRITE;
/*!40000 ALTER TABLE `proxies` DISABLE KEYS */;
/*!40000 ALTER TABLE `proxies` ENABLE KEYS */;
UNLOCK TABLES;
DROP TABLE IF EXISTS `system`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `system` (
  `id` int NOT NULL AUTO_INCREMENT,
  `uplink` bigint DEFAULT NULL,
  `downlink` bigint DEFAULT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

LOCK TABLES `system` WRITE;
/*!40000 ALTER TABLE `system` DISABLE KEYS */;
INSERT INTO `system` VALUES (1,0,0);
/*!40000 ALTER TABLE `system` ENABLE KEYS */;
UNLOCK TABLES;
DROP TABLE IF EXISTS `template_inbounds_association`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `template_inbounds_association` (
  `user_template_id` int DEFAULT NULL,
  `inbound_tag` varchar(256) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL,
  KEY `user_template_id` (`user_template_id`),
  KEY `template_inbounds_association_ibfk_1` (`inbound_tag`),
  CONSTRAINT `template_inbounds_association_ibfk_1` FOREIGN KEY (`inbound_tag`) REFERENCES `inbounds` (`tag`),
  CONSTRAINT `template_inbounds_association_ibfk_2` FOREIGN KEY (`user_template_id`) REFERENCES `user_templates` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

LOCK TABLES `template_inbounds_association` WRITE;
/*!40000 ALTER TABLE `template_inbounds_association` DISABLE KEYS */;
/*!40000 ALTER TABLE `template_inbounds_association` ENABLE KEYS */;
UNLOCK TABLES;
DROP TABLE IF EXISTS `tls`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `tls` (
  `id` int NOT NULL AUTO_INCREMENT,
  `key` varchar(4096) NOT NULL,
  `certificate` varchar(2048) NOT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

LOCK TABLES `tls` WRITE;
/*!40000 ALTER TABLE `tls` DISABLE KEYS */;
INSERT INTO `tls` VALUES (1,'-----BEGIN PRIVATE KEY-----\nMIIJRAIBADANBgkqhkiG9w0BAQEFAASCCS4wggkqAgEAAoICAQC/MR3IKzLpfKE3\nrxA3LRxYgapZceE+XbvuG6lf5pqppFWnmNA6Nv+JORYRR/IWSelwBngKYpAJZn4m\nN84JUqKLrNbGrIxe/GqXwDvSeAxZfO6wNrbEwVHucuUyKetmV6229mHfC9muFPO9\nHdvDEbcQyR1/2VjOQcRH69fSuub77V+57Q7gvuAYbBNzpdjGtFfqA+Hxth0i/K3D\nvyJxnI5T9jcZYj04jphEkCSPJiVWbD75dFYKLO/OCACTzG4ad0qMxTFlOvEwtX4w\nXhZKvcCDwt9b+UHeDwBJBTD9hyndF+24DnV1MqxZo1zK2mqZPYFeW3+N1gGzlBFa\naRjkCIWj+5FdB73AuNebTOO5NRQVbaRFJlZ3YnftDKbnsJoEKSYi43/A+Rod2jVU\nR0PsGz66h0AyBTZ675y5vWA96JEOLFrYztLpX/ukzkecsDirwdxpWnqF/x/dLfEt\nW4Ltp+NIXpbKfnixAMxX2vOKT9yY1GrQsP8Tn/Fax69xNukst73WZ7PY41A63xKF\nHnZ6hDvmVaBMO+80rdHZLG+ptRFRox2Nqe+8n44sjI8qKBw64O6fM6SbbXFb69ty\nsqcEtLBQaSF79XSuxFJv3FbNYHhgtvy0lmJFCWKVW7IkJihadT/XPpiUYDh1BDNA\n2qxsOZ/Nf+6ztpwu6FXdpeSFJxdzcwIDAQABAoICAEmIpIG2S7CNpJv8EuqFyR8p\nT9FzAUSxXHRp4yzlTprsf50kIwIg4QtwD1K6LxvvdX8Gneiwm+FhsCcW04G2bJn3\n8esbtilaP8z5vIhob6eDa3IOuSDXkaF6QziV8tm74y45eM8GicRwfoZAE6PlDlnA\nYD4p5AQAO3PF+44rIGfIlka2LEC4aQw7lQenw8Y42o9AywfiGTa5pENjIeY8ZmlY\nizySEiXMgjfskIMWEso4e8gJvIXRXrIJVd+Cgt5e/ydBOIxMrTMkutHzffNlI8QJ\nkUqYhYLYli1/d6Yq9Cj66Q0fUoGu6abYM/o5I0Ko6zbVTivJIYJl8n6paU9bD7qk\nw95NltYaSy9UpTdNgQ8ZoobipNp9xb289mjqk7goefqhmg4J+8G2EfkPzEgb0k5m\nzSY18QwlSVe5Foz9OGEVMS/pifmfYSvyhfArHPDTMwztd0c6mh2TyVrLPSyBb+4l\noLrTMzlFi/GD6WE0fQJhqz12gn3bKFehVM0fSrjB9c+/fCjF8wkppbLTMm4XyWcK\nEZx4fl+bfB+xG80XH6S0XsYkw+//bSNpSPterD1O2tI+sOQC9wJll8Ix2pbV3JAT\n3Ma/2qBH7Df5ldeydSbJUcmxkvbA1nQ5sHY93yi9IH8h4dVDirdjX/HJhPw3MDsQ\nsLqN2WMW93KTXwUKsdfhAoIBAQD5S+QShgQJMJFsHwJ1XmfiRS/jhXvm2lk2Drmr\nVx86O2gGKy5+u6BEaqcA/D6LrthwCw0oFeVkY48LwU5TrdHX1YALQRUeiAEyi3B+\ndeBqCAaWSM40GREking8n5X4gWosSk32e2kCHo/6c2ivaeUTOweqXrCPYonEWqbd\nODJWgCLdEcJWkBy63oJiS/hNRLlJxIUOuuA/OcVpX5VfC6R1Yq9LgPPbqAeeFtZK\nL/KnYCrIcsVbnAwFb+66NdYusYSJuxn9xvggk2xL985XII3bjO1sqiKBsI+w9WgN\nTT4+kB8Fh0V2z1YZW/gy13GCOI9s3GwKC+/KhR05/PNJyyabAoIBAQDEVT6a79Vf\n8/df+xdeY2KlDOSUfv0z/alCzF03kbhYVonAvlf6LZ8aS04h7VQP9DR3qBADDvjj\ntdXdKxeZi69n0KzPab2SsbN0jZvsZwCPjGWGDkkhkKL8ZEObkTie8eiz3TWUdflb\nWl3JHJFeKnD660JgRD7FA2HyaArNJl08BcNpr+CzLxMGAoY619fuiSy6CFvX9s1F\n84M+1TDtQzSFgY++Trkhh1Y0pWo8Dj66S+t0phk5g4UIIyoCglv1S1XAHB0PiCyh\nTbYNj0QteHlGEnOVP2SwtvKCvBXgIPq8SlD9hYFQUIh1+AlJIGxSurmLq2d6T4Vf\nzkIB+nTR/sgJAoIBAQDAAX+L9MQu/8VBbTDLjQad84PKzQh86TdEt1qxaMyYYSDh\nf2i0G3Z4u1wdGipiEyyIPzrG4PZot3V5LE1TjdDNV5bjFCFb/oxNja/69nb9s2i/\nXxvSbGwTpMKujotHdwV7T090w+BAzaE+2JyMXhcdFJ32zi9WgCJzpQ5X2+yJSu+P\nLRps6Vn3Y2EwrCp1+LRs6HtCuRP2Hly5p/C335OvBW7VzGAFkgkr7ZLmWL+fwiWx\ndRQkVq+7kFZh/Q0VsYvPONfv6cs569HBQP8fg+c+bfB/fRpOoNpLBPg7Jn8GWKJL\n87/AnjhGF9u17NFz5nQw+CQOKNtrM9Mq0nPfJiR5AoIBAQCj7k1szu6JxMn7kUss\n260lBQPi3bmLbLSkesEmwfx7fh5dGlGZKfib2t7tQx2aiJw6gNCFzT2R3GUq221E\nHrch5sI0GSVZ1Bh0LD1SAZsdQOwQJ+N2LC96sDknLvVo+bIEZKL1SCAh3+1JHBpg\n1u0pBcCRYspKdA2XernyG6Njl6hXmavz0U2ACiGMBmdc25rF5m7Rf1hcnPwsPZQx\nt5imFM0x6Md5sGgRY0C1zmsf4H5EEuAi/qbYaYwNNk2UHggGdTLqohgpwxpMbiBO\nGzafhI6PwrPPPaJ1+JgHYWIaT68aUBvuNKAMaENQlMr9tL1yWnb3UGib2zuQLVDX\nenYJAoIBAQCxCLkbyjuLmz4aprWhUZlz1MFbSbFsuSoleAO0OKXVzv2KXBuAxuoZ\nkBfZfsaWsrZDp9955YzbhPSg7bOjgydxa19MLwIrZhvEltxxGW4L99Ab7xEH5S1R\nusRiX3RSqAarwPTlTAyc16HsOrFzgZJX4iN3cFbxYCQByAE3615xIS3UrkOpZatA\nS2gKYD+0V2rsprnRFYUSIDdUsi616o8H68rLWpQ3DiBWPd04V3ic61a7okKjk/68\nNFlBEK5QDed7UvfufqzkbpjwV3O86/DoYZs6XKr6wwKsq7IlDtPZsxwqSTO8yBeH\nlrIMULEII3dBknlZceX/IPGPJ2KhcEDq\n-----END PRIVATE KEY-----\n','-----BEGIN CERTIFICATE-----\nMIIEnDCCAoQCAQAwDQYJKoZIhvcNAQENBQAwEzERMA8GA1UEAwwIR296YXJnYWgw\nIBcNMjYxMDE2MTM0MDA3WhgPMjEyNjA5MjIxMzQwMDdaMBMxETAPBgNVBAMMCEdv\nemFyZ2FoMIICIjANBgkqhkiG9w0BAQEFAAOCAg8AMIICCgKCAgEAvzEdyCsy6Xyh\nN68QNy0cWIGqWXHhPl277hupX+aaqaRVp5jQOjb/iTkWEUfyFknpcAZ4CmKQCWZ+\nJjfOCVKii6zWxqyMXvxql8A70ngMWXzusDa2xMFR7nLlMinrZlettvZh3wvZrhTz\nvR3bwxG3EMkdf9lYzkHER+vX0rrm++1fue0O4L7gGGwTc6XYxrRX6gPh8bYdIvyt\nw78icZyOU/Y3GWI9OI6YRJAkjyYlVmw++XRWCizvzggAk8xuGndKjMUxZTrxMLV+\nMF4WSr3Ag8LfW/lB3g8ASQUw/Ycp3RftuA51dTKsWaNcytpqmT2BXlt/jdYBs5QR\nWmkY5AiFo/uRXQe9wLjXm0zjuTUUFW2kRSZWd2J37Qym57CaBCkmIuN/wPkaHdo1\nVEdD7Bs+uodAMgU2eu+cub1gPeiRDixa2M7S6V/7pM5HnLA4q8HcaVp6hf8f3S3x\nLVuC7afjSF6Wyn54sQDMV9rzik/cmNRq0LD/E5/xWsevcTbpLLe91mez2ONQOt8S\nhR52eoQ75lWgTDvvNK3R2SxvqbURUaMdjanvvJ+OLIyPKigcOuDunzOkm21xW+vb\ncrKnBLSwUGkhe/V0rsRSb9xWzWB4YLb8tJZiRQlilVuyJCYoWnU/1z6YlGA4dQQz\nQNqsbDmfzX/us7acLuhV3aXkhScXc3MCAwEAATANBgkqhkiG9w0BAQ0FAAOCAgEA\ntqPf+op91YkkYFaxQe/I6PjMnKBrqHqYRyLKR0CitXv2A/F8lzG0aRjIIegkr5WM\npgo9abYJPY+uPy5GMJsQAm53liVJEU4h+k0Papx/slURZMIX8gcFMk/h56ONDmW5\nxzMbFtQlWY9Rp1Z7FQCjYd8bCM1tDLJIj+BXESGx2UldRZqejJVaj8Vtve4jieFl\nHhB6wOEJwzUA7uhTD7rOYSNlNb3Jf7bQN7LA4ppP35YD5YziC/Xul1pI2ac2AaBa\ni2Q9toKg/QD0T3zya9orBcQSJE39SKQ4I0thp30tOY3i4Ksw4pStaXgU8VwfgDnm\nsrmFJ5qh5MVfjWKFbmYSLIP/fBikOfBZyOw2EFRzIzqI7sUX9B52XG3l/0vj2w1n\nqqK7Rcq0cJn5glkKHO0lVZbbB/b/91dYSpqlTvLcqgHJlCPzRvt5rvx/L+e8BYBY\nsaSqPWK5EOr1E6MHyz4yllDop0HpI14nPfVLpQ35LZB99Su8pKFldeVovZoO44l9\nTST94ipIT+OaRq4T2FGH/IuJZ9zAeZXQiwOmaEV5xxwGSmkLrbzfCFQ0RlHikC7S\nwU7wFD9PeI1ormJT83KZxxdMohw76hqhu0sX0LFaM45iKkV0TpeiJZT+MesuJCmk\nPTg9wi2GeloNkechxdDtKRKw+i0JtHFszARVc9SYTRw=\n-----END CERTIFICATE-----\n');
/*!40000 ALTER TABLE `tls` ENABLE KEYS */;
UNLOCK TABLES;
DROP TABLE IF EXISTS `user_templates`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `user_templates` (
  `id` int NOT NULL AUTO_INCREMENT,
  `name` varchar(64) NOT NULL,
  `data_limit` bigint DEFAULT NULL,
  `expire_duration` bigint DEFAULT NULL,
  `username_prefix` varchar(20) DEFAULT NULL,
  `username_suffix` varchar(20) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `name` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

LOCK TABLES `user_templates` WRITE;
/*!40000 ALTER TABLE `user_templates` DISABLE KEYS */;
/*!40000 ALTER TABLE `user_templates` ENABLE KEYS */;
UNLOCK TABLES;
DROP TABLE IF EXISTS `user_usage_logs`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `user_usage_logs` (
  `id` int NOT NULL AUTO_INCREMENT,
  `user_id` int DEFAULT NULL,
  `used_traffic_at_reset` bigint NOT NULL,
  `reset_at` datetime DEFAULT NULL,
  PRIMARY KEY (`id`),
  KEY `user_id` (`user_id`),
  CONSTRAINT `user_usage_logs_ibfk_1` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
/*!40101 SET character_set_client = @saved_cs_client */;

LOCK TABLES `user_usage_logs` WRITE;
/*!40000 ALTER TABLE `user_usage_logs` DISABLE KEYS */;
/*!40000 ALTER TABLE `user_usage_logs` ENABLE KEYS */;
UNLOCK TABLES;
DROP TABLE IF EXISTS `users`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `users` (
  `id` int NOT NULL AUTO_INCREMENT,
  `username` varchar(34) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL,
  `status` enum('on_hold','active','limited','expired','disabled') NOT NULL,
  `used_traffic` bigint DEFAULT NULL,
  `data_limit` bigint DEFAULT NULL,
  `expire` int DEFAULT NULL,
  `created_at` datetime DEFAULT (now()),
  `admin_id` int DEFAULT NULL,
  `data_limit_reset_strategy` enum('no_reset','day','week','month','year') NOT NULL DEFAULT 'no_reset',
  `sub_revoked_at` datetime DEFAULT NULL,
  `note` varchar(500) DEFAULT NULL,
  `sub_updated_at` datetime DEFAULT NULL,
  `sub_last_user_agent` varchar(512) DEFAULT NULL,
  `online_at` datetime DEFAULT NULL,
  `edit_at` datetime DEFAULT NULL,
  `on_hold_timeout` datetime DEFAULT NULL,
  `on_hold_expire_duration` bigint DEFAULT NULL,
  `auto_delete_in_days` int DEFAULT NULL,
  `last_status_change` datetime DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `ix_users_username` (`username`),
  KEY `fk_users_admin_id_admins` (`admin_id`),
  CONSTRAINT `fk_users_admin_id_admins` FOREIGN KEY (`admin_id`) REFERENCES `admins` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
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
/*!40111 SET SQL_NOTES=@OLD_SQL_NOTES */;

