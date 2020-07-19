-- --------------------------------------------------------
-- Host:                         127.0.0.1
-- Server version:               PostgreSQL 12.2 (Ubuntu 12.2-4) on x86_64-pc-linux-gnu, compiled by gcc (Ubuntu 9.3.0-8ubuntu1) 9.3.0, 64-bit
-- Server OS:
-- HeidiSQL Version:             11.0.0.5919
-- --------------------------------------------------------

/*!40101 SET @OLD_CHARACTER_SET_CLIENT=@@CHARACTER_SET_CLIENT */;
/*!40101 SET NAMES  */;
/*!40014 SET @OLD_FOREIGN_KEY_CHECKS=@@FOREIGN_KEY_CHECKS, FOREIGN_KEY_CHECKS=0 */;
/*!40101 SET @OLD_SQL_MODE=@@SQL_MODE, SQL_MODE='NO_AUTO_VALUE_ON_ZERO' */;

-- Dumping structure for table public.group
CREATE TABLE IF NOT EXISTS "group" (
	"id" UUID NOT NULL,
	"parent_id" UUID NOT NULL,
	"name" BYTEA NOT NULL,
	"flags" INTEGER NOT NULL DEFAULT '0',
	"key" BYTEA NOT NULL,
	UNIQUE INDEX "group_unique_name" ("name"),
	PRIMARY KEY ("id", "parent_id"),
	UNIQUE INDEX "group_id_uindex" ("id"),
	UNIQUE INDEX "group_key_uindex" ("key"),
	INDEX "group_flags_index" ("flags")
);

-- Data exporting was unselected.

-- Dumping structure for table public.group_assets
CREATE TABLE IF NOT EXISTS "group_assets" (
	"group_id" UUID NOT NULL,
	"asset_id" UUID NOT NULL,
	"asset_kind" SMALLINT NOT NULL DEFAULT '0',
	PRIMARY KEY ("group_id", "asset_id", "asset_kind"),
	INDEX "group_assets_asset_id_index" ("asset_id"),
	INDEX "group_assets_group_id_index" ("group_id"),
	CONSTRAINT "group_assets_group_id_fk" FOREIGN KEY ("group_id") REFERENCES "public"."group" ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

-- Data exporting was unselected.

/*!40101 SET SQL_MODE=IFNULL(@OLD_SQL_MODE, '') */;
/*!40014 SET FOREIGN_KEY_CHECKS=IF(@OLD_FOREIGN_KEY_CHECKS IS NULL, 1, @OLD_FOREIGN_KEY_CHECKS) */;
/*!40101 SET CHARACTER_SET_CLIENT=@OLD_CHARACTER_SET_CLIENT */;
