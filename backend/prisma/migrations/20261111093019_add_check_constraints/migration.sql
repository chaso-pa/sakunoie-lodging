-- CHECK constraints that the Prisma schema cannot express.
-- Created with `prisma migrate dev --create-only` and filled by hand. This is
-- the one sanctioned exception to "do not hand-edit migrations" (plan
-- Decision E, CLAUDE.md §13).

-- From docs/schema.sql.
ALTER TABLE `facilities`
    ADD CONSTRAINT `chk_facility_capacity` CHECK (`max_guests` >= 1);

ALTER TABLE `guest_registrations`
    ADD CONSTRAINT `chk_registration_dates` CHECK (`check_out_date` > `check_in_date`),
    ADD CONSTRAINT `chk_registration_counts` CHECK (`planned_guest_count` >= 1 AND `capacity_snapshot` >= 1);

-- docs/schema.sql declares these columns as JSON. In MariaDB, JSON is an alias
-- for LONGTEXT with a json_valid() CHECK. The Prisma schema models them as
-- LONGTEXT, so the CHECK is added here. NULL passes json_valid().
ALTER TABLE `audit_logs`
    ADD CONSTRAINT `chk_audit_changed_fields_json` CHECK (json_valid(`changed_fields_json`));

ALTER TABLE `survey_responses`
    ADD CONSTRAINT `chk_survey_answers_json` CHECK (json_valid(`answers_json`));

ALTER TABLE `api_idempotency_keys`
    ADD CONSTRAINT `chk_idempotency_response_json` CHECK (json_valid(`response_json`));
