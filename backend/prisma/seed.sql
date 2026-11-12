-- Facility seed. Idempotent: re-running keeps exactly one `saku-no-ie` row
-- and never overwrites values an operator changed later (capacity changes
-- must be explicit, contract §4.1).
-- max_guests = 2 was confirmed by the owner (2026-11-12). Existing databases
-- are not changed by re-seeding; update their row by hand.
INSERT INTO `facilities` (`code`, `name`, `timezone`, `max_guests`, `is_active`, `created_at`, `updated_at`)
VALUES ('saku-no-ie', '佐久の家', 'Asia/Tokyo', 2, TRUE, UTC_TIMESTAMP(6), UTC_TIMESTAMP(6))
ON DUPLICATE KEY UPDATE `code` = `code`;
