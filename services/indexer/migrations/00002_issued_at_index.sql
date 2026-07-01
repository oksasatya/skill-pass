-- +goose Up
CREATE INDEX idx_certificates_issued_at ON certificates (issued_at);

-- +goose Down
DROP INDEX idx_certificates_issued_at;
