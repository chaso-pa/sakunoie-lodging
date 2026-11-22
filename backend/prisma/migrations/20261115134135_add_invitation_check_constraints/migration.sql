-- CHECK constraint of registration_invitations that the Prisma schema cannot
-- express. Created with `prisma migrate dev --create-only` and filled by hand,
-- following `20261111093019_add_check_constraints` (CLAUDE.md §13, CHECK
-- constraints).

-- The host sets the stay dates of an invitation; the registration it
-- authorizes must use exactly these dates, so they obey the same rule as
-- `chk_registration_dates`.
ALTER TABLE `registration_invitations`
    ADD CONSTRAINT `chk_invitation_dates` CHECK (`check_out_date` > `check_in_date`);
