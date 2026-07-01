// Package usecase holds notify's pure business logic -- signing webhook payloads. It
// imports nothing from adapter or third-party HTTP/asynq packages (hexagonal-lite).
package usecase

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
)

// SignPayload returns the hex-encoded HMAC-SHA256 of body using secret -- the value sent
// in the X-SkillPass-Signature header (as "sha256=<hex>"), mirroring the GitHub/Stripe
// webhook-signing convention. Pure function -- O(len(body)).
func SignPayload(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body) //nolint:errcheck // hash.Hash.Write never returns an error
	return hex.EncodeToString(mac.Sum(nil))
}
