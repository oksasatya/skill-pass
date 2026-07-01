-- +goose Up
CREATE TABLE webhook_outbox (
    id           BIGSERIAL PRIMARY KEY,
    chain_id     BIGINT NOT NULL,
    tx_hash      TEXT NOT NULL,
    token_id     TEXT NOT NULL,
    payload      JSONB NOT NULL,
    enqueued_at  TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (chain_id, tx_hash, token_id)
);

-- Partial index: the sweep query (WHERE enqueued_at IS NULL) only ever touches unenqueued
-- rows, which stay a small fraction of the table as it grows — O(log k) via this index,
-- not an O(n) scan of the whole outbox history.
CREATE INDEX idx_webhook_outbox_unenqueued ON webhook_outbox (id) WHERE enqueued_at IS NULL;

-- +goose Down
DROP TABLE webhook_outbox;
