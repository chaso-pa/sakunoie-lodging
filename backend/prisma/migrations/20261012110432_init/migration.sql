-- CreateTable
CREATE TABLE `members` (
    `id` VARCHAR(191) NOT NULL,
    `nickname` VARCHAR(30) NOT NULL,
    `avatar_url` VARCHAR(500) NULL,
    `preferred_language` VARCHAR(5) NOT NULL DEFAULT 'ja',
    `transfer_code_hash` VARCHAR(255) NOT NULL,
    `session_version` INTEGER NOT NULL DEFAULT 1,
    `created_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    `updated_at` DATETIME(3) NOT NULL,

    UNIQUE INDEX `uq_members_transfer_code_hash`(`transfer_code_hash`),
    PRIMARY KEY (`id`)
) DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

-- CreateTable
CREATE TABLE `sessions` (
    `token` CHAR(43) NOT NULL,
    `data` BLOB NOT NULL,
    `expiry` TIMESTAMP(6) NOT NULL,

    INDEX `idx_sessions_expiry`(`expiry`),
    PRIMARY KEY (`token`)
) DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

-- CreateTable
CREATE TABLE `spots` (
    `id` VARCHAR(191) NOT NULL,
    `name` VARCHAR(100) NOT NULL,
    `emoji` VARCHAR(10) NOT NULL,
    `area` VARCHAR(20) NOT NULL,
    `category` VARCHAR(30) NOT NULL,
    `description` TEXT NOT NULL,
    `is_sample` BOOLEAN NOT NULL DEFAULT false,
    `created_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    `updated_at` DATETIME(3) NOT NULL,

    PRIMARY KEY (`id`)
) DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

-- CreateTable
CREATE TABLE `posts` (
    `id` VARCHAR(191) NOT NULL,
    `member_id` VARCHAR(191) NOT NULL,
    `spot_id` VARCHAR(191) NULL,
    `caption` TEXT NOT NULL,
    `language` VARCHAR(5) NOT NULL DEFAULT 'ja',
    `weather` VARCHAR(20) NULL,
    `temperature` DOUBLE NULL,
    `visited_at` DATETIME(3) NULL,
    `is_sample` BOOLEAN NOT NULL DEFAULT false,
    `created_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
    `updated_at` DATETIME(3) NOT NULL,

    INDEX `idx_posts_member_id`(`member_id`),
    INDEX `idx_posts_spot_id`(`spot_id`),
    INDEX `idx_posts_created_at`(`created_at`),
    PRIMARY KEY (`id`)
) DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

-- CreateTable
CREATE TABLE `post_media` (
    `id` VARCHAR(191) NOT NULL,
    `post_id` VARCHAR(191) NOT NULL,
    `media_url` VARCHAR(500) NOT NULL,
    `media_type` VARCHAR(10) NOT NULL DEFAULT 'photo',
    `sort_order` INTEGER NOT NULL DEFAULT 0,
    `created_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),

    INDEX `idx_post_media_post_id`(`post_id`),
    PRIMARY KEY (`id`)
) DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

-- CreateTable
CREATE TABLE `post_tags` (
    `id` VARCHAR(191) NOT NULL,
    `post_id` VARCHAR(191) NOT NULL,
    `tag` VARCHAR(50) NOT NULL,

    INDEX `idx_post_tags_post_id`(`post_id`),
    PRIMARY KEY (`id`)
) DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

-- CreateTable
CREATE TABLE `likes` (
    `id` VARCHAR(191) NOT NULL,
    `member_id` VARCHAR(191) NOT NULL,
    `post_id` VARCHAR(191) NOT NULL,
    `created_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),

    UNIQUE INDEX `likes_member_id_post_id_key`(`member_id`, `post_id`),
    PRIMARY KEY (`id`)
) DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

-- CreateTable
CREATE TABLE `bookmarks` (
    `id` VARCHAR(191) NOT NULL,
    `member_id` VARCHAR(191) NOT NULL,
    `post_id` VARCHAR(191) NOT NULL,
    `created_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),

    UNIQUE INDEX `bookmarks_member_id_post_id_key`(`member_id`, `post_id`),
    PRIMARY KEY (`id`)
) DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

-- CreateTable
CREATE TABLE `reviews` (
    `id` VARCHAR(191) NOT NULL,
    `member_id` VARCHAR(191) NOT NULL,
    `spot_id` VARCHAR(191) NOT NULL,
    `stars` TINYINT NOT NULL,
    `text` TEXT NOT NULL,
    `is_sample` BOOLEAN NOT NULL DEFAULT false,
    `created_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),

    INDEX `idx_reviews_spot_id`(`spot_id`),
    PRIMARY KEY (`id`)
) DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

-- CreateTable
CREATE TABLE `gps_visits` (
    `id` VARCHAR(191) NOT NULL,
    `member_id` VARCHAR(191) NOT NULL,
    `spot_id` VARCHAR(191) NULL,
    `place_name` VARCHAR(100) NOT NULL,
    `emoji` VARCHAR(10) NOT NULL,
    `area` VARCHAR(20) NOT NULL,
    `visited_date` DATE NOT NULL,
    `weather` VARCHAR(20) NULL,
    `temperature` DOUBLE NULL,
    `note` TEXT NULL,
    `is_public` BOOLEAN NOT NULL DEFAULT true,
    `exact_public` BOOLEAN NOT NULL DEFAULT false,
    `latitude` DOUBLE NULL,
    `longitude` DOUBLE NULL,
    `created_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),

    INDEX `idx_gps_visits_member_id`(`member_id`),
    PRIMARY KEY (`id`)
) DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

-- CreateTable
CREATE TABLE `ai_spots` (
    `id` VARCHAR(191) NOT NULL,
    `name` VARCHAR(100) NOT NULL,
    `purposes` VARCHAR(200) NOT NULL,
    `area` VARCHAR(50) NOT NULL,
    `kind` VARCHAR(30) NOT NULL,
    `drive_min` INTEGER NOT NULL,
    `transit_min` INTEGER NOT NULL,
    `stay_min` INTEGER NOT NULL,
    `genres` VARCHAR(200) NOT NULL,
    `weathers` VARCHAR(100) NOT NULL,
    `description` TEXT NOT NULL,
    `map_query` VARCHAR(200) NOT NULL,
    `search_url` VARCHAR(500) NOT NULL,

    PRIMARY KEY (`id`)
) DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

-- CreateTable
CREATE TABLE `post_translations` (
    `id` VARCHAR(191) NOT NULL,
    `source_text_hash` CHAR(64) NOT NULL,
    `source_lang` VARCHAR(5) NOT NULL,
    `target_lang` VARCHAR(5) NOT NULL,
    `translated_text` TEXT NOT NULL,
    `provider` VARCHAR(20) NOT NULL,
    `created_at` DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),

    UNIQUE INDEX `post_translations_source_text_hash_source_lang_target_lang_key`(`source_text_hash`, `source_lang`, `target_lang`),
    PRIMARY KEY (`id`)
) DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

-- AddForeignKey
ALTER TABLE `posts` ADD CONSTRAINT `posts_member_id_fkey` FOREIGN KEY (`member_id`) REFERENCES `members`(`id`) ON DELETE RESTRICT ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE `posts` ADD CONSTRAINT `posts_spot_id_fkey` FOREIGN KEY (`spot_id`) REFERENCES `spots`(`id`) ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE `post_media` ADD CONSTRAINT `post_media_post_id_fkey` FOREIGN KEY (`post_id`) REFERENCES `posts`(`id`) ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE `post_tags` ADD CONSTRAINT `post_tags_post_id_fkey` FOREIGN KEY (`post_id`) REFERENCES `posts`(`id`) ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE `likes` ADD CONSTRAINT `likes_member_id_fkey` FOREIGN KEY (`member_id`) REFERENCES `members`(`id`) ON DELETE RESTRICT ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE `likes` ADD CONSTRAINT `likes_post_id_fkey` FOREIGN KEY (`post_id`) REFERENCES `posts`(`id`) ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE `bookmarks` ADD CONSTRAINT `bookmarks_member_id_fkey` FOREIGN KEY (`member_id`) REFERENCES `members`(`id`) ON DELETE RESTRICT ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE `bookmarks` ADD CONSTRAINT `bookmarks_post_id_fkey` FOREIGN KEY (`post_id`) REFERENCES `posts`(`id`) ON DELETE CASCADE ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE `reviews` ADD CONSTRAINT `reviews_member_id_fkey` FOREIGN KEY (`member_id`) REFERENCES `members`(`id`) ON DELETE RESTRICT ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE `reviews` ADD CONSTRAINT `reviews_spot_id_fkey` FOREIGN KEY (`spot_id`) REFERENCES `spots`(`id`) ON DELETE RESTRICT ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE `gps_visits` ADD CONSTRAINT `gps_visits_member_id_fkey` FOREIGN KEY (`member_id`) REFERENCES `members`(`id`) ON DELETE RESTRICT ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE `gps_visits` ADD CONSTRAINT `gps_visits_spot_id_fkey` FOREIGN KEY (`spot_id`) REFERENCES `spots`(`id`) ON DELETE SET NULL ON UPDATE CASCADE;
