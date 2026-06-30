package domain

import "fmt"

// ValidateTokenID ensures s is a non-empty string of decimal digits representing
// a non-negative integer. Arbitrarily large (uint256) — validated as a digit string.
func ValidateTokenID(s string) error {
	if len(s) == 0 {
		return fmt.Errorf("%w: empty", ErrInvalidTokenID)
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return fmt.Errorf("%w: non-digit character %q", ErrInvalidTokenID, c)
		}
	}
	return nil
}
