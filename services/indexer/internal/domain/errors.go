package domain

import "errors"

// Sentinel errors for caller branching via errors.Is.
var (
	ErrNotFound            = errors.New("not found")
	ErrInvalidAddress      = errors.New("invalid address")
	ErrInvalidTokenID      = errors.New("invalid token id")
	ErrInvalidCertificate  = errors.New("invalid certificate")
)
