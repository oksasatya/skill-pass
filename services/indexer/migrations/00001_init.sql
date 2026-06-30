-- +goose Up
CREATE TABLE certificates (
    token_id        NUMERIC(78,0) PRIMARY KEY,
    owner_address   TEXT        NOT NULL,          -- recipient, store lowercased hex
    title           TEXT        NOT NULL,
    recipient_name  TEXT        NOT NULL,
    issuer_name     TEXT        NOT NULL,
    description     TEXT        NOT NULL DEFAULT '',
    metadata_uri    TEXT        NOT NULL DEFAULT '',
    issued_at       TIMESTAMPTZ NOT NULL,          -- from event issuedAt (block.timestamp)
    chain_id        BIGINT      NOT NULL,
    tx_hash         TEXT        NOT NULL,
    log_index       BIGINT      NOT NULL,
    block_number    BIGINT      NOT NULL,
    block_hash      TEXT        NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (chain_id, tx_hash, log_index)          -- event-level idempotency / reorg guard
);
CREATE INDEX idx_certificates_owner_token ON certificates (owner_address, token_id DESC);
CREATE INDEX idx_certificates_issuer_name ON certificates (issuer_name);

CREATE TABLE indexer_state (
    id                   INTEGER     PRIMARY KEY DEFAULT 1 CHECK (id = 1), -- singleton
    chain_id             BIGINT      NOT NULL,
    last_processed_block BIGINT      NOT NULL DEFAULT 0,
    last_processed_hash  TEXT        NOT NULL DEFAULT '',
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE indexer_state;
DROP TABLE certificates;
