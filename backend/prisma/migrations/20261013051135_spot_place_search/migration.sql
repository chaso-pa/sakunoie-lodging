/*
  Warnings:

  - A unique constraint covering the columns `[external_key]` on the table `spots` will be added. If there are existing duplicate values, this will fail.

*/
-- AlterTable
ALTER TABLE `spots` ADD COLUMN `address` VARCHAR(300) NULL,
    ADD COLUMN `created_by_member_id` VARCHAR(191) NULL,
    ADD COLUMN `external_id` VARCHAR(512) NULL,
    ADD COLUMN `external_key` CHAR(64) NULL,
    ADD COLUMN `latitude` DOUBLE NULL,
    ADD COLUMN `longitude` DOUBLE NULL,
    ADD COLUMN `map_url` VARCHAR(500) NULL,
    ADD COLUMN `original_name` VARCHAR(200) NULL,
    ADD COLUMN `source` VARCHAR(10) NOT NULL DEFAULT 'seed';

-- CreateIndex
CREATE INDEX `idx_spots_created_by_member_id` ON `spots`(`created_by_member_id`);

-- CreateIndex
CREATE INDEX `idx_spots_source` ON `spots`(`source`);

-- CreateIndex
CREATE UNIQUE INDEX `uq_spots_external_key` ON `spots`(`external_key`);

-- AddForeignKey
ALTER TABLE `spots` ADD CONSTRAINT `spots_created_by_member_id_fkey` FOREIGN KEY (`created_by_member_id`) REFERENCES `members`(`id`) ON DELETE SET NULL ON UPDATE CASCADE;
