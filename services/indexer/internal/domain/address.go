package domain

import (
	"fmt"
	"strings"
)

const (
	hexPrefix   = "0x"
	addrHexLen  = 40 // 20 bytes * 2 hex chars
	zeroAddrHex = "0x0000000000000000000000000000000000000000"
)

// Address is a normalized EVM address value object (always lowercase 0x…).
type Address struct {
	hex string // lowercased "0x" + 40 hex chars
}

// NewAddress parses and normalizes an EVM address string.
// Trims whitespace, lowercases, requires 0x prefix + exactly 40 hex chars.
func NewAddress(s string) (Address, error) {
	s = strings.TrimSpace(strings.ToLower(s))
	if !strings.HasPrefix(s, hexPrefix) {
		return Address{}, fmt.Errorf("%w: missing 0x prefix", ErrInvalidAddress)
	}
	body := s[len(hexPrefix):]
	if len(body) != addrHexLen {
		return Address{}, fmt.Errorf("%w: want 40 hex chars, got %d", ErrInvalidAddress, len(body))
	}
	for _, c := range body {
		if !isHexRune(c) {
			return Address{}, fmt.Errorf("%w: non-hex character %q", ErrInvalidAddress, c)
		}
	}
	return Address{hex: s}, nil
}

// String returns the canonical lowercase 0x… form.
func (a Address) String() string { return a.hex }

// IsZero returns true when the address is the zero (null) address.
func (a Address) IsZero() bool { return a.hex == zeroAddrHex || a.hex == "" }

func isHexRune(c rune) bool {
	return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')
}
