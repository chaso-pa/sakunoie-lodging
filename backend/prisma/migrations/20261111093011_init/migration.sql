-- CreateTable
CREATE TABLE `facilities` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `code` VARCHAR(64) NOT NULL,
    `name` VARCHAR(255) NOT NULL,
    `timezone` VARCHAR(64) NOT NULL DEFAULT 'Asia/Tokyo',
    `max_guests` INTEGER UNSIGNED NOT NULL,
    `is_active` BOOLEAN NOT NULL DEFAULT true,
    `created_at` DATETIME(6) NOT NULL,
    `updated_at` DATETIME(6) NOT NULL,

    UNIQUE INDEX `code`(`code`),
    PRIMARY KEY (`id`)
) DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

-- CreateTable
CREATE TABLE `admin_users` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `firebase_uid` VARCHAR(128) NOT NULL,
    `created_at` DATETIME(6) NOT NULL,
    `updated_at` DATETIME(6) NOT NULL,

    UNIQUE INDEX `firebase_uid`(`firebase_uid`),
    PRIMARY KEY (`id`)
) DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

-- CreateTable
CREATE TABLE `admin_facility_access` (
    `admin_user_id` BIGINT UNSIGNED NOT NULL,
    `facility_id` BIGINT UNSIGNED NOT NULL,
    `role` VARCHAR(32) NOT NULL,
    `created_at` DATETIME(6) NOT NULL,

    INDEX `fk_afa_facility`(`facility_id`),
    PRIMARY KEY (`admin_user_id`, `facility_id`)
) DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

-- CreateTable
CREATE TABLE `guest_registrations` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `public_id` CHAR(36) NOT NULL,
    `facility_id` BIGINT UNSIGNED NOT NULL,
    `check_in_date` DATE NOT NULL,
    `check_out_date` DATE NOT NULL,
    `estimated_arrival_time` TIME(0) NULL,
    `planned_guest_count` INTEGER UNSIGNED NOT NULL,
    `capacity_snapshot` INTEGER UNSIGNED NOT NULL,
    `booking_source` VARCHAR(50) NULL,
    `booking_reference` VARCHAR(255) NULL,
    `terms_version` VARCHAR(64) NOT NULL,
    `privacy_version` VARCHAR(64) NOT NULL,
    `terms_accepted_at` DATETIME(6) NOT NULL,
    `privacy_accepted_at` DATETIME(6) NOT NULL,
    `submitted_at` DATETIME(6) NOT NULL,
    `identity_verified_at` DATETIME(6) NULL,
    `identity_verified_by` BIGINT UNSIGNED NULL,
    `created_at` DATETIME(6) NOT NULL,
    `updated_at` DATETIME(6) NOT NULL,
    `deleted_at` DATETIME(6) NULL,

    UNIQUE INDEX `public_id`(`public_id`),
    INDEX `idx_registration_facility_dates`(`facility_id`, `check_in_date`, `check_out_date`),
    INDEX `idx_registration_facility_created`(`facility_id`, `created_at`),
    INDEX `fk_registration_verifier`(`identity_verified_by`),
    PRIMARY KEY (`id`)
) DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

-- CreateTable
CREATE TABLE `guests` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `public_id` CHAR(36) NOT NULL,
    `registration_id` BIGINT UNSIGNED NOT NULL,
    `is_representative` BOOLEAN NOT NULL DEFAULT false,
    `full_name` VARCHAR(255) NOT NULL,
    `name_kana` VARCHAR(255) NULL,
    `address` TEXT NOT NULL,
    `phone` VARCHAR(50) NULL,
    `email` VARCHAR(255) NULL,
    `has_japan_address` BOOLEAN NOT NULL,
    `nationality` VARCHAR(100) NULL,
    `passport_number_ciphertext` VARBINARY(1024) NULL,
    `passport_number_last4` VARCHAR(4) NULL,
    `created_at` DATETIME(6) NOT NULL,
    `updated_at` DATETIME(6) NOT NULL,

    UNIQUE INDEX `public_id`(`public_id`),
    INDEX `idx_guest_registration`(`registration_id`),
    PRIMARY KEY (`id`)
) DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

-- CreateTable
CREATE TABLE `registration_access_tokens` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `registration_id` BIGINT UNSIGNED NOT NULL,
    `token_hash` BINARY(32) NOT NULL,
    `expires_at` DATETIME(6) NOT NULL,
    `revoked_at` DATETIME(6) NULL,
    `last_used_at` DATETIME(6) NULL,
    `created_at` DATETIME(6) NOT NULL,

    UNIQUE INDEX `token_hash`(`token_hash`),
    INDEX `idx_token_registration`(`registration_id`),
    PRIMARY KEY (`id`)
) DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

-- CreateTable
CREATE TABLE `audit_logs` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `facility_id` BIGINT UNSIGNED NOT NULL,
    `registration_id` BIGINT UNSIGNED NULL,
    `actor_type` VARCHAR(32) NOT NULL,
    `actor_admin_user_id` BIGINT UNSIGNED NULL,
    `action` VARCHAR(100) NOT NULL,
    `changed_fields_json` LONGTEXT NULL,
    `created_at` DATETIME(6) NOT NULL,

    INDEX `idx_audit_facility_time`(`facility_id`, `created_at`),
    INDEX `fk_audit_registration`(`registration_id`),
    INDEX `fk_audit_admin`(`actor_admin_user_id`),
    PRIMARY KEY (`id`)
) DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

-- CreateTable
CREATE TABLE `survey_responses` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `facility_id` BIGINT UNSIGNED NOT NULL,
    `survey_version` VARCHAR(64) NOT NULL,
    `answers_json` LONGTEXT NOT NULL,
    `submitted_at` DATETIME(6) NOT NULL,
    `created_at` DATETIME(6) NOT NULL,

    INDEX `idx_survey_facility_time`(`facility_id`, `submitted_at`),
    PRIMARY KEY (`id`)
) DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

-- CreateTable
CREATE TABLE `api_idempotency_keys` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `key_hash` BINARY(32) NOT NULL,
    `request_hash` BINARY(32) NOT NULL,
    `response_json` LONGTEXT NULL,
    `response_status` SMALLINT UNSIGNED NULL,
    `created_at` DATETIME(6) NOT NULL,
    `expires_at` DATETIME(6) NOT NULL,

    UNIQUE INDEX `key_hash`(`key_hash`),
    PRIMARY KEY (`id`)
) DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

-- CreateTable
CREATE TABLE `import_batches` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `source_type` VARCHAR(64) NOT NULL,
    `source_checksum` CHAR(64) NOT NULL,
    `imported_at` DATETIME(6) NOT NULL,

    UNIQUE INDEX `uq_import_batch`(`source_type`, `source_checksum`),
    PRIMARY KEY (`id`)
) DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

-- CreateTable
CREATE TABLE `import_rows` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `batch_id` BIGINT UNSIGNED NOT NULL,
    `source_row_number` INTEGER UNSIGNED NOT NULL,
    `outcome` VARCHAR(32) NOT NULL,
    `registration_id` BIGINT UNSIGNED NULL,
    `error_code` VARCHAR(100) NULL,

    INDEX `fk_import_registration`(`registration_id`),
    UNIQUE INDEX `uq_import_row`(`batch_id`, `source_row_number`),
    PRIMARY KEY (`id`)
) DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

-- AddForeignKey
ALTER TABLE `admin_facility_access` ADD CONSTRAINT `fk_afa_admin` FOREIGN KEY (`admin_user_id`) REFERENCES `admin_users`(`id`) ON DELETE RESTRICT ON UPDATE RESTRICT;

-- AddForeignKey
ALTER TABLE `admin_facility_access` ADD CONSTRAINT `fk_afa_facility` FOREIGN KEY (`facility_id`) REFERENCES `facilities`(`id`) ON DELETE RESTRICT ON UPDATE RESTRICT;

-- AddForeignKey
ALTER TABLE `guest_registrations` ADD CONSTRAINT `fk_registration_facility` FOREIGN KEY (`facility_id`) REFERENCES `facilities`(`id`) ON DELETE RESTRICT ON UPDATE RESTRICT;

-- AddForeignKey
ALTER TABLE `guest_registrations` ADD CONSTRAINT `fk_registration_verifier` FOREIGN KEY (`identity_verified_by`) REFERENCES `admin_users`(`id`) ON DELETE RESTRICT ON UPDATE RESTRICT;

-- AddForeignKey
ALTER TABLE `guests` ADD CONSTRAINT `fk_guest_registration` FOREIGN KEY (`registration_id`) REFERENCES `guest_registrations`(`id`) ON DELETE RESTRICT ON UPDATE RESTRICT;

-- AddForeignKey
ALTER TABLE `registration_access_tokens` ADD CONSTRAINT `fk_token_registration` FOREIGN KEY (`registration_id`) REFERENCES `guest_registrations`(`id`) ON DELETE RESTRICT ON UPDATE RESTRICT;

-- AddForeignKey
ALTER TABLE `audit_logs` ADD CONSTRAINT `fk_audit_facility` FOREIGN KEY (`facility_id`) REFERENCES `facilities`(`id`) ON DELETE RESTRICT ON UPDATE RESTRICT;

-- AddForeignKey
ALTER TABLE `audit_logs` ADD CONSTRAINT `fk_audit_registration` FOREIGN KEY (`registration_id`) REFERENCES `guest_registrations`(`id`) ON DELETE RESTRICT ON UPDATE RESTRICT;

-- AddForeignKey
ALTER TABLE `audit_logs` ADD CONSTRAINT `fk_audit_admin` FOREIGN KEY (`actor_admin_user_id`) REFERENCES `admin_users`(`id`) ON DELETE RESTRICT ON UPDATE RESTRICT;

-- AddForeignKey
ALTER TABLE `survey_responses` ADD CONSTRAINT `fk_survey_facility` FOREIGN KEY (`facility_id`) REFERENCES `facilities`(`id`) ON DELETE RESTRICT ON UPDATE RESTRICT;

-- AddForeignKey
ALTER TABLE `import_rows` ADD CONSTRAINT `fk_import_batch` FOREIGN KEY (`batch_id`) REFERENCES `import_batches`(`id`) ON DELETE RESTRICT ON UPDATE RESTRICT;

-- AddForeignKey
ALTER TABLE `import_rows` ADD CONSTRAINT `fk_import_registration` FOREIGN KEY (`registration_id`) REFERENCES `guest_registrations`(`id`) ON DELETE RESTRICT ON UPDATE RESTRICT;
