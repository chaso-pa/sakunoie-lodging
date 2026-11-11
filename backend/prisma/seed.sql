-- Facility seed. Idempotent: re-running keeps exactly one `saku-no-ie` row
-- and never overwrites values an operator changed later (capacity changes
-- must be explicit, contract §4.1).
-- VERIFY: max_guests = 4 is a placeholder until the host confirms the
-- facility's licensed capacity (open question Q2).
INSERT INTO `facilities` (`code`, `name`, `timezone`, `max_guests`, `is_active`, `created_at`, `updated_at`)
VALUES ('saku-no-ie', '佐久の家', 'Asia/Tokyo', 4, TRUE, UTC_TIMESTAMP(6), UTC_TIMESTAMP(6))
ON DUPLICATE KEY UPDATE `code` = `code`;
