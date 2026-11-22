-- CreateTable
CREATE TABLE `registration_invitations` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `public_id` CHAR(36) NOT NULL,
    `facility_id` BIGINT UNSIGNED NOT NULL,
    `token_hash` BINARY(32) NOT NULL,
    `check_in_date` DATE NOT NULL,
    `check_out_date` DATE NOT NULL,
    `booking_reference` VARCHAR(255) NULL,
    `note` VARCHAR(255) NULL,
    `expires_at` DATETIME(6) NOT NULL,
    `used_at` DATETIME(6) NULL,
    `registration_id` BIGINT UNSIGNED NULL,
    `revoked_at` DATETIME(6) NULL,
    `created_by` BIGINT UNSIGNED NOT NULL,
    `created_at` DATETIME(6) NOT NULL,
    `updated_at` DATETIME(6) NOT NULL,

    UNIQUE INDEX `public_id`(`public_id`),
    UNIQUE INDEX `token_hash`(`token_hash`),
    UNIQUE INDEX `registration_id`(`registration_id`),
    INDEX `idx_invitation_facility_created`(`facility_id`, `created_at`),
    INDEX `fk_invitation_creator`(`created_by`),
    PRIMARY KEY (`id`)
) DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

-- AddForeignKey
ALTER TABLE `registration_invitations` ADD CONSTRAINT `fk_invitation_facility` FOREIGN KEY (`facility_id`) REFERENCES `facilities`(`id`) ON DELETE RESTRICT ON UPDATE RESTRICT;

-- AddForeignKey
ALTER TABLE `registration_invitations` ADD CONSTRAINT `fk_invitation_registration` FOREIGN KEY (`registration_id`) REFERENCES `guest_registrations`(`id`) ON DELETE RESTRICT ON UPDATE RESTRICT;

-- AddForeignKey
ALTER TABLE `registration_invitations` ADD CONSTRAINT `fk_invitation_creator` FOREIGN KEY (`created_by`) REFERENCES `admin_users`(`id`) ON DELETE RESTRICT ON UPDATE RESTRICT;
