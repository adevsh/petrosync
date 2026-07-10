
-- =============================================================================
-- SECTION 32 — TELEGRAM LINK TOKENS
-- =============================================================================

-- name: CreateTelegramLinkToken :one
INSERT INTO telegram_link_tokens (user_id, token, expires_at)
VALUES ($1, $2, NOW() + INTERVAL '48 hours')
RETURNING *;

-- name: GetValidTelegramLinkToken :one
-- Returns token only if unused and not expired.
SELECT tlt.*, u.username, u.full_name, u.telegram_user_id
FROM telegram_link_tokens tlt
JOIN users u ON u.id = tlt.user_id
WHERE tlt.token      = $1
  AND tlt.used_at    IS NULL
  AND tlt.expires_at >  NOW();

-- name: UseTelegramLinkToken :one
UPDATE telegram_link_tokens
SET used_at = NOW()
WHERE token   = $1
  AND used_at IS NULL
RETURNING *;

-- name: DeleteExpiredTelegramLinkTokens :execrows
-- Called by cron worker nightly.
DELETE FROM telegram_link_tokens
WHERE expires_at < NOW()
  AND used_at IS NOT NULL;

-- name: ListActiveTokensForUser :many
SELECT *
FROM telegram_link_tokens
WHERE user_id    = $1
  AND used_at    IS NULL
  AND expires_at > NOW()
ORDER BY created_at DESC;
