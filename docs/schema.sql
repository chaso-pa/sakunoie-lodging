-- MariaDB baseline schema for Saku no Ie MVP.
-- Review legal requirements, operational retention and encryption implementation before production.
-- UTC convention: application writes DATETIME(6) values as UTC.
CREATE TABLE facilities (
 id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
 code VARCHAR(64) NOT NULL UNIQUE,
 name VARCHAR(255) NOT NULL,
 timezone VARCHAR(64) NOT NULL DEFAULT 'Asia/Tokyo',
 max_guests INT UNSIGNED NOT NULL,
 is_active BOOLEAN NOT NULL DEFAULT TRUE,
 created_at DATETIME(6) NOT NULL,
 updated_at DATETIME(6) NOT NULL,
 CONSTRAINT chk_facility_capacity CHECK (max_guests >= 1)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE admin_users (
 id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
 firebase_uid VARCHAR(128) NOT NULL UNIQUE,
 created_at DATETIME(6) NOT NULL,
 updated_at DATETIME(6) NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE admin_facility_access (
 admin_user_id BIGINT UNSIGNED NOT NULL,
 facility_id BIGINT UNSIGNED NOT NULL,
 role VARCHAR(32) NOT NULL,
 created_at DATETIME(6) NOT NULL,
 PRIMARY KEY (admin_user_id, facility_id),
 CONSTRAINT fk_afa_admin FOREIGN KEY (admin_user_id) REFERENCES admin_users(id),
 CONSTRAINT fk_afa_facility FOREIGN KEY (facility_id) REFERENCES facilities(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE guest_registrations (
 id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
 public_id CHAR(36) CHARACTER SET ascii COLLATE ascii_bin NOT NULL UNIQUE,
 facility_id BIGINT UNSIGNED NOT NULL,
 check_in_date DATE NOT NULL,
 check_out_date DATE NOT NULL,
 estimated_arrival_time TIME NULL,
 planned_guest_count INT UNSIGNED NOT NULL,
 capacity_snapshot INT UNSIGNED NOT NULL,
 booking_source VARCHAR(50) NULL,
 booking_reference VARCHAR(255) NULL,
 terms_version VARCHAR(64) NOT NULL,
 privacy_version VARCHAR(64) NOT NULL,
 terms_accepted_at DATETIME(6) NOT NULL,
 privacy_accepted_at DATETIME(6) NOT NULL,
 submitted_at DATETIME(6) NOT NULL,
 identity_verified_at DATETIME(6) NULL,
 identity_verified_by BIGINT UNSIGNED NULL,
 created_at DATETIME(6) NOT NULL,
 updated_at DATETIME(6) NOT NULL,
 deleted_at DATETIME(6) NULL,
 CONSTRAINT fk_registration_facility FOREIGN KEY (facility_id) REFERENCES facilities(id),
 CONSTRAINT fk_registration_verifier FOREIGN KEY (identity_verified_by) REFERENCES admin_users(id),
 CONSTRAINT chk_registration_dates CHECK (check_out_date > check_in_date),
 CONSTRAINT chk_registration_counts CHECK (planned_guest_count >= 1 AND capacity_snapshot >= 1),
 INDEX idx_registration_facility_dates (facility_id, check_in_date, check_out_date),
 INDEX idx_registration_facility_created (facility_id, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE guests (
 id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
 public_id CHAR(36) CHARACTER SET ascii COLLATE ascii_bin NOT NULL UNIQUE,
 registration_id BIGINT UNSIGNED NOT NULL,
 is_representative BOOLEAN NOT NULL DEFAULT FALSE,
 full_name VARCHAR(255) NOT NULL,
 name_kana VARCHAR(255) NULL,
 address TEXT NOT NULL,
 phone VARCHAR(50) NULL,
 email VARCHAR(255) NULL,
 has_japan_address BOOLEAN NOT NULL,
 nationality VARCHAR(100) NULL,
 passport_number_ciphertext VARBINARY(1024) NULL,
 passport_number_last4 VARCHAR(4) NULL,
 created_at DATETIME(6) NOT NULL,
 updated_at DATETIME(6) NOT NULL,
 CONSTRAINT fk_guest_registration FOREIGN KEY (registration_id) REFERENCES guest_registrations(id),
 INDEX idx_guest_registration (registration_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE registration_access_tokens (
 id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
 registration_id BIGINT UNSIGNED NOT NULL,
 token_hash BINARY(32) NOT NULL UNIQUE,
 expires_at DATETIME(6) NOT NULL,
 revoked_at DATETIME(6) NULL,
 last_used_at DATETIME(6) NULL,
 created_at DATETIME(6) NOT NULL,
 CONSTRAINT fk_token_registration FOREIGN KEY (registration_id) REFERENCES guest_registrations(id),
 INDEX idx_token_registration (registration_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE audit_logs (
 id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
 facility_id BIGINT UNSIGNED NOT NULL,
 registration_id BIGINT UNSIGNED NULL,
 actor_type VARCHAR(32) NOT NULL,
 actor_admin_user_id BIGINT UNSIGNED NULL,
 action VARCHAR(100) NOT NULL,
 changed_fields_json JSON NULL,
 created_at DATETIME(6) NOT NULL,
 CONSTRAINT fk_audit_facility FOREIGN KEY (facility_id) REFERENCES facilities(id),
 CONSTRAINT fk_audit_registration FOREIGN KEY (registration_id) REFERENCES guest_registrations(id),
 CONSTRAINT fk_audit_admin FOREIGN KEY (actor_admin_user_id) REFERENCES admin_users(id),
 INDEX idx_audit_facility_time (facility_id, created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE survey_responses (
 id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
 facility_id BIGINT UNSIGNED NOT NULL,
 survey_version VARCHAR(64) NOT NULL,
 answers_json JSON NOT NULL,
 submitted_at DATETIME(6) NOT NULL,
 created_at DATETIME(6) NOT NULL,
 CONSTRAINT fk_survey_facility FOREIGN KEY (facility_id) REFERENCES facilities(id),
 INDEX idx_survey_facility_time (facility_id, submitted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE api_idempotency_keys (
 id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
 key_hash BINARY(32) NOT NULL UNIQUE,
 request_hash BINARY(32) NOT NULL,
 response_json JSON NULL,
 response_status SMALLINT UNSIGNED NULL,
 created_at DATETIME(6) NOT NULL,
 expires_at DATETIME(6) NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE import_batches (
 id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
 source_type VARCHAR(64) NOT NULL,
 source_checksum CHAR(64) NOT NULL,
 imported_at DATETIME(6) NOT NULL,
 UNIQUE KEY uq_import_batch (source_type, source_checksum)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE import_rows (
 id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
 batch_id BIGINT UNSIGNED NOT NULL,
 source_row_number INT UNSIGNED NOT NULL,
 outcome VARCHAR(32) NOT NULL,
 registration_id BIGINT UNSIGNED NULL,
 error_code VARCHAR(100) NULL,
 CONSTRAINT fk_import_batch FOREIGN KEY (batch_id) REFERENCES import_batches(id),
 CONSTRAINT fk_import_registration FOREIGN KEY (registration_id) REFERENCES guest_registrations(id),
 UNIQUE KEY uq_import_row (batch_id, source_row_number)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
