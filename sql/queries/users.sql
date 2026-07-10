
-- =============================================================================
-- SECTION 12 — USERS
-- =============================================================================

-- name: GetUser :one
SELECT id, username, full_name, telegram_user_id,
       telegram_linked_at, force_password_change, active,
       last_login_at, created_at, updated_at
FROM users
WHERE id = $1;

-- name: GetUserByUsername :one
-- Used during login — includes password_hash for bcrypt comparison.
SELECT *
FROM users
WHERE username = $1;

-- name: GetUserByTelegramID :one
SELECT id, username, full_name, telegram_user_id,
       telegram_linked_at, force_password_change, active,
       last_login_at, created_at, updated_at
FROM users
WHERE telegram_user_id = $1
  AND active = TRUE;

-- name: ListUsers :many
SELECT id, username, full_name, telegram_user_id,
       telegram_linked_at, force_password_change, active,
       last_login_at, created_at, updated_at
FROM users
ORDER BY full_name;

-- name: ListActiveUsers :many
SELECT id, username, full_name, telegram_user_id,
       telegram_linked_at, force_password_change, active,
       last_login_at, created_at, updated_at
FROM users
WHERE active = TRUE
ORDER BY full_name;

-- name: CreateUser :one
INSERT INTO users (username, password_hash, full_name, force_password_change)
VALUES ($1, $2, $3, $4)
RETURNING id, username, full_name, telegram_user_id,
          telegram_linked_at, force_password_change, active,
          last_login_at, created_at, updated_at;

-- name: UpdateUserPassword :exec
UPDATE users
SET
    password_hash          = $2,
    force_password_change  = FALSE,
    updated_at             = NOW()
WHERE id = $1;

-- name: SetForcePasswordChange :exec
-- Called by admin password reset flow before sending temp password via Telegram.
UPDATE users
SET force_password_change = TRUE, updated_at = NOW()
WHERE id = $1;

-- name: LinkTelegramAccount :exec
UPDATE users
SET
    telegram_user_id   = $2,
    telegram_linked_at = NOW(),
    updated_at         = NOW()
WHERE id = $1;

-- name: UnlinkTelegramAccount :exec
UPDATE users
SET
    telegram_user_id   = NULL,
    telegram_linked_at = NULL,
    updated_at         = NOW()
WHERE id = $1;

-- name: RecordUserLogin :exec
UPDATE users
SET last_login_at = NOW(), updated_at = NOW()
WHERE id = $1;

-- name: DeactivateUser :exec
UPDATE users
SET active = FALSE, updated_at = NOW()
WHERE id = $1;
-- name: GetUserPasswordHash :one
SELECT password_hash FROM users WHERE id = $1;
