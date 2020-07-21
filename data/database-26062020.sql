-- --------------------------------------------------------
-- Host:                         127.0.0.1
-- Server version:               8.0.20-0ubuntu0.20.04.1 - (Ubuntu)
-- Server OS:                    Linux
-- HeidiSQL Version:             11.0.0.5919
-- --------------------------------------------------------

/*!40101 SET @OLD_CHARACTER_SET_CLIENT=@@CHARACTER_SET_CLIENT */;
/*!40101 SET NAMES utf8 */;
/*!50503 SET NAMES utf8mb4 */;
/*!40014 SET @OLD_FOREIGN_KEY_CHECKS=@@FOREIGN_KEY_CHECKS, FOREIGN_KEY_CHECKS=0 */;
/*!40101 SET @OLD_SQL_MODE=@@SQL_MODE, SQL_MODE='NO_AUTO_VALUE_ON_ZERO' */;

-- Dumping structure for table hometown.accesspolicy
CREATE TABLE IF NOT EXISTS `accesspolicy` (
  `id` int unsigned NOT NULL AUTO_INCREMENT,
  `parent_id` int unsigned NOT NULL,
  `owner_id` int unsigned NOT NULL COMMENT 'owner user ID',
  `key` varchar(32) NOT NULL,
  `object_type` varchar(50) NOT NULL COMMENT 'is a type name of an object (i.e.: user)',
  `object_id` int unsigned NOT NULL COMMENT 'is an object ID (depends on the scope)',
  `flags` tinyint unsigned NOT NULL DEFAULT '0',
  PRIMARY KEY (`id`),
  UNIQUE KEY `key` (`key`),
  KEY `parent_id` (`parent_id`),
  KEY `owner_id` (`owner_id`),
  KEY `object_id` (`object_id`),
  KEY `id_checksum` (`id`),
  KEY `object_type` (`object_type`),
  KEY `flags` (`flags`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

-- Data exporting was unselected.

-- Dumping structure for table hometown.accesspolicy_roster
CREATE TABLE IF NOT EXISTS `accesspolicy_roster` (
  `policy_id` int unsigned NOT NULL,
  `subject_kind` tinyint unsigned NOT NULL COMMENT 'typical kinds: everyone, user, group, role_group, etc...',
  `subject_id` int unsigned NOT NULL COMMENT 'represents an ID of a user, group, role_group, etc...; 0 if not required',
  `access` bigint unsigned NOT NULL COMMENT 'bitmask which represents the accesspolicy rights to a policy for the subject of its kind',
  `access_explained` text NOT NULL COMMENT 'a human-readable conjunction of comma-separated accesspolicy names for this given context namespace',
  PRIMARY KEY (`policy_id`,`subject_kind`,`subject_id`),
  KEY `access_right` (`access`),
  KEY `policy_id` (`policy_id`),
  KEY `subject_kind` (`subject_kind`),
  KEY `subject_id` (`subject_id`),
  KEY `subject_id_access` (`subject_id`,`access`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

-- Data exporting was unselected.

-- Dumping structure for table hometown.group
CREATE TABLE IF NOT EXISTS `group` (
  `id` int unsigned NOT NULL AUTO_INCREMENT,
  `parent_id` int unsigned NOT NULL,
  `is_default` tinyint(1) NOT NULL DEFAULT '0',
  `kind` tinyint NOT NULL,
  `key` varchar(50) NOT NULL,
  `name` varchar(100) NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `key` (`key`),
  KEY `parent_id` (`parent_id`),
  KEY `kind` (`kind`),
  KEY `name` (`name`),
  KEY `is_default` (`is_default`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

-- Data exporting was unselected.

-- Dumping structure for table hometown.group_users
CREATE TABLE IF NOT EXISTS `group_users` (
  `group_id` int unsigned NOT NULL,
  `user_id` int unsigned NOT NULL,
  PRIMARY KEY (`group_id`,`user_id`),
  KEY `FK_group_users_group` (`group_id`),
  KEY `FK_group_users_user` (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

-- Data exporting was unselected.

-- Dumping structure for table hometown.password
CREATE TABLE IF NOT EXISTS `password` (
  `kind` tinyint unsigned NOT NULL,
  `owner_id` int unsigned NOT NULL,
  `hash` binary(60) NOT NULL,
  `is_change_required` tinyint(1) NOT NULL DEFAULT '0',
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  `expire_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`kind`,`owner_id`),
  KEY `created_at` (`created_at`),
  KEY `updated_at` (`updated_at`),
  KEY `expire_at` (`expire_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

-- Data exporting was unselected.

-- Dumping structure for table hometown.token
CREATE TABLE IF NOT EXISTS `token` (
  `kind` smallint unsigned NOT NULL DEFAULT '0',
  `token` binary(64) NOT NULL,
  `payload` blob NOT NULL,
  `checkin_total` smallint NOT NULL DEFAULT '0',
  `checkin_remainder` smallint NOT NULL DEFAULT '0',
  `created_at` timestamp NOT NULL,
  `expire_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`token`),
  KEY `kind` (`kind`),
  KEY `c_remainder` (`checkin_remainder`),
  KEY `c_total` (`checkin_total`),
  KEY `created_at` (`created_at`),
  KEY `expire_at` (`expire_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

-- Data exporting was unselected.

-- Dumping structure for table hometown.user
CREATE TABLE IF NOT EXISTS `user` (
  `id` int NOT NULL AUTO_INCREMENT,
  `ulid` binary(16) NOT NULL,
  `username` varchar(30) CHARACTER SET utf8 COLLATE utf8_unicode_ci NOT NULL,
  `display_name` varchar(50) CHARACTER SET utf8 COLLATE utf8_unicode_ci NOT NULL,
  `last_login_at` timestamp NULL DEFAULT NULL,
  `last_login_ip` varchar(45) CHARACTER SET utf8 COLLATE utf8_unicode_ci NOT NULL,
  `last_login_failed_at` timestamp NULL DEFAULT NULL,
  `last_login_failed_ip` varchar(45) CHARACTER SET utf8 COLLATE utf8_unicode_ci NOT NULL,
  `last_login_attempts` tinyint unsigned NOT NULL DEFAULT '0',
  `is_suspended` tinyint(1) NOT NULL DEFAULT '0',
  `suspension_reason` varchar(255) CHARACTER SET utf8 COLLATE utf8_unicode_ci NOT NULL DEFAULT '',
  `suspended_at` timestamp NULL DEFAULT NULL,
  `suspension_expires_at` timestamp NULL DEFAULT NULL,
  `suspended_by_id` int NOT NULL DEFAULT '0',
  `checksum` bigint unsigned NOT NULL DEFAULT '0',
  `confirmed_at` timestamp NULL DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT NULL,
  `created_by_id` int NOT NULL DEFAULT '0',
  `updated_at` timestamp NULL DEFAULT NULL,
  `updated_by_id` int NOT NULL DEFAULT '0',
  `deleted_at` timestamp NULL DEFAULT NULL,
  `deleted_by_id` int NOT NULL DEFAULT '0',
  PRIMARY KEY (`id`),
  UNIQUE KEY `username` (`username`),
  UNIQUE KEY `display_name` (`display_name`),
  KEY `ulid` (`ulid`),
  KEY `deleted_at` (`deleted_at`),
  KEY `created_at` (`created_at`),
  KEY `updated_at` (`updated_at`),
  KEY `is_suspended` (`is_suspended`),
  KEY `checksum` (`checksum`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_unicode_ci ROW_FORMAT=DYNAMIC;

-- Data exporting was unselected.

-- Dumping structure for table hometown.user_email
CREATE TABLE IF NOT EXISTS `user_email` (
  `user_id` int unsigned NOT NULL,
  `addr` varchar(255) NOT NULL,
  `is_primary` tinyint(1) NOT NULL DEFAULT '0',
  `created_at` timestamp NULL DEFAULT NULL,
  `confirmed_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`user_id`,`addr`),
  UNIQUE KEY `user_id_is_primary` (`user_id`,`is_primary`),
  UNIQUE KEY `addr` (`addr`),
  KEY `confirmed_at` (`confirmed_at`),
  KEY `created_at` (`created_at`),
  KEY `updated_at` (`updated_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- Data exporting was unselected.

-- Dumping structure for table hometown.user_phone
CREATE TABLE IF NOT EXISTS `user_phone` (
  `user_id` int unsigned NOT NULL,
  `number` varchar(15) NOT NULL,
  `is_primary` tinyint(1) NOT NULL DEFAULT '0',
  `created_at` timestamp NULL DEFAULT NULL,
  `confirmed_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`user_id`,`number`),
  UNIQUE KEY `number` (`number`),
  UNIQUE KEY `user_id_is_primary` (`user_id`,`is_primary`),
  KEY `confirmed_at` (`confirmed_at`),
  KEY `created_at` (`created_at`),
  KEY `updated_at` (`updated_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- Data exporting was unselected.

-- Dumping structure for table hometown.user_profile
CREATE TABLE IF NOT EXISTS `user_profile` (
  `user_id` int unsigned NOT NULL,
  `firstname` varchar(50) NOT NULL,
  `middlename` varchar(50) NOT NULL,
  `lastname` varchar(50) NOT NULL,
  `language` char(2) NOT NULL,
  `checksum` bigint unsigned NOT NULL DEFAULT '0',
  `created_at` timestamp NULL DEFAULT NULL,
  `updated_at` timestamp NULL DEFAULT NULL,
  PRIMARY KEY (`user_id`),
  KEY `updated_at` (`updated_at`),
  KEY `checksum` (`checksum`),
  KEY `created_at` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- Data exporting was unselected.

/*!40101 SET SQL_MODE=IFNULL(@OLD_SQL_MODE, '') */;
/*!40014 SET FOREIGN_KEY_CHECKS=IF(@OLD_FOREIGN_KEY_CHECKS IS NULL, 1, @OLD_FOREIGN_KEY_CHECKS) */;
/*!40101 SET CHARACTER_SET_CLIENT=@OLD_CHARACTER_SET_CLIENT */;
