-- UpsertCertificate inserts or updates a certificate row by token_id.
-- Idempotent: re-processing the same event updates canonical tx/block (reorg guard).
-- name: UpsertCertificate :one
INSERT INTO certificates (
    token_id, owner_address, title, recipient_name, issuer_name,
    description, metadata_uri, issued_at, chain_id, tx_hash,
    log_index, block_number, block_hash, created_at, updated_at
) VALUES (
    $1, $2, $3, $4, $5,
    $6, $7, $8, $9, $10,
    $11, $12, $13, now(), now()
)
ON CONFLICT (token_id) DO UPDATE SET
    owner_address  = EXCLUDED.owner_address,
    title          = EXCLUDED.title,
    recipient_name = EXCLUDED.recipient_name,
    issuer_name    = EXCLUDED.issuer_name,
    description    = EXCLUDED.description,
    metadata_uri   = EXCLUDED.metadata_uri,
    issued_at      = EXCLUDED.issued_at,
    chain_id       = EXCLUDED.chain_id,
    tx_hash        = EXCLUDED.tx_hash,
    log_index      = EXCLUDED.log_index,
    block_number   = EXCLUDED.block_number,
    block_hash     = EXCLUDED.block_hash,
    updated_at     = now()
RETURNING *;

-- GetCertificateByTokenID fetches a certificate by its primary key.
-- name: GetCertificateByTokenID :one
SELECT * FROM certificates WHERE token_id = $1;

-- ListCertificates returns certificates via keyset pagination (token_id DESC).
-- Passing NULL cursor returns the first page.
-- name: ListCertificates :many
SELECT * FROM certificates
WHERE ($1::numeric IS NULL OR token_id < $1)
ORDER BY token_id DESC
LIMIT $2;

-- ListCertificatesByOwner returns certificates for a given owner via keyset pagination.
-- Uses idx_certificates_owner_token (owner_address, token_id DESC).
-- name: ListCertificatesByOwner :many
SELECT * FROM certificates
WHERE owner_address = $1
  AND ($2::numeric IS NULL OR token_id < $2)
ORDER BY token_id DESC
LIMIT $3;

-- SearchCertificates full-text searches title/issuer_name/recipient_name with keyset pagination.
-- ponytail: ILIKE scan; add pg_trgm GIN index when n grows
-- name: SearchCertificates :many
SELECT * FROM certificates
WHERE (title ILIKE '%' || $1 || '%'
    OR issuer_name ILIKE '%' || $1 || '%'
    OR recipient_name ILIKE '%' || $1 || '%')
  AND ($2::numeric IS NULL OR token_id < $2)
ORDER BY token_id DESC
LIMIT $3;

-- SearchCertificatesByOwner searches title/issuer_name/recipient_name for a given owner with keyset pagination.
-- Uses idx_certificates_owner_token (owner_address, token_id DESC).
-- ponytail: ILIKE scan; add pg_trgm GIN index when n grows
-- name: SearchCertificatesByOwner :many
SELECT * FROM certificates
WHERE owner_address = $1
  AND (title ILIKE '%' || $2 || '%'
    OR issuer_name ILIKE '%' || $2 || '%'
    OR recipient_name ILIKE '%' || $2 || '%')
  AND ($3::numeric IS NULL OR token_id < $3)
ORDER BY token_id DESC
LIMIT $4;

-- CountCertificates returns the total number of certificates (for indexer status).
-- name: CountCertificates :one
SELECT count(*) FROM certificates;

-- GetIndexerState fetches the singleton indexer progress row.
-- name: GetIndexerState :one
SELECT * FROM indexer_state WHERE id = 1;

-- UpsertIndexerState inserts or updates the singleton indexer progress row.
-- name: UpsertIndexerState :one
INSERT INTO indexer_state (id, chain_id, last_processed_block, last_processed_hash, updated_at)
VALUES (1, $1, $2, $3, now())
ON CONFLICT (id) DO UPDATE SET
    chain_id             = EXCLUDED.chain_id,
    last_processed_block = EXCLUDED.last_processed_block,
    last_processed_hash  = EXCLUDED.last_processed_hash,
    updated_at           = now()
RETURNING *;

-- DeleteCertificatesFromBlock removes all certificates at or above the given block number
-- (chain-scoped) — used by reorg reconcile to roll back the confirmation window.
-- name: DeleteCertificatesFromBlock :exec
DELETE FROM certificates WHERE chain_id = $1 AND block_number >= $2;
