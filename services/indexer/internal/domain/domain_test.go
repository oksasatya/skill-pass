package domain_test

import (
	"errors"
	"testing"
	"time"

	"github.com/oksasatya/skillpass/services/indexer/internal/domain"
)

// ── Address ──────────────────────────────────────────────────────────────────

func TestNewAddress(t *testing.T) {
	t.Run("valid lowercase", func(t *testing.T) {
		a, err := domain.NewAddress("0xabcdef1234567890abcdef1234567890abcdef12")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if a.String() != "0xabcdef1234567890abcdef1234567890abcdef12" {
			t.Fatalf("wrong string: %q", a.String())
		}
	})

	t.Run("valid mixed-case normalizes equal", func(t *testing.T) {
		lower, _ := domain.NewAddress("0xabcdef1234567890abcdef1234567890abcdef12")
		upper, err := domain.NewAddress("0xABCDEF1234567890ABCDEF1234567890ABCDEF12")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if lower != upper {
			t.Fatalf("addresses should be equal after normalization: %q vs %q", lower, upper)
		}
	})

	t.Run("empty string", func(t *testing.T) {
		_, err := domain.NewAddress("")
		if !errors.Is(err, domain.ErrInvalidAddress) {
			t.Fatalf("expected ErrInvalidAddress, got %v", err)
		}
	})

	t.Run("wrong length — too short", func(t *testing.T) {
		_, err := domain.NewAddress("0xabcdef")
		if !errors.Is(err, domain.ErrInvalidAddress) {
			t.Fatalf("expected ErrInvalidAddress, got %v", err)
		}
	})

	t.Run("wrong length — too long", func(t *testing.T) {
		_, err := domain.NewAddress("0x" + "ab" + "abcdef1234567890abcdef1234567890abcdef12")
		if !errors.Is(err, domain.ErrInvalidAddress) {
			t.Fatalf("expected ErrInvalidAddress, got %v", err)
		}
	})

	t.Run("non-hex chars", func(t *testing.T) {
		_, err := domain.NewAddress("0xzzzzzz1234567890abcdef1234567890abcdef12")
		if !errors.Is(err, domain.ErrInvalidAddress) {
			t.Fatalf("expected ErrInvalidAddress, got %v", err)
		}
	})

	t.Run("missing 0x prefix", func(t *testing.T) {
		_, err := domain.NewAddress("abcdef1234567890abcdef1234567890abcdef12")
		if !errors.Is(err, domain.ErrInvalidAddress) {
			t.Fatalf("expected ErrInvalidAddress, got %v", err)
		}
	})

	t.Run("zero address accepted", func(t *testing.T) {
		a, err := domain.NewAddress("0x0000000000000000000000000000000000000000")
		if err != nil {
			t.Fatalf("zero address should be valid: %v", err)
		}
		if a.String() != "0x0000000000000000000000000000000000000000" {
			t.Fatalf("wrong string: %q", a.String())
		}
	})
}

func TestAddressIsZero(t *testing.T) {
	t.Run("zero address is zero", func(t *testing.T) {
		a, _ := domain.NewAddress("0x0000000000000000000000000000000000000000")
		if !a.IsZero() {
			t.Fatal("expected IsZero() = true")
		}
	})

	t.Run("non-zero address", func(t *testing.T) {
		a, _ := domain.NewAddress("0xabcdef1234567890abcdef1234567890abcdef12")
		if a.IsZero() {
			t.Fatal("expected IsZero() = false")
		}
	})
}

// ── ValidateTokenID ───────────────────────────────────────────────────────────

func TestValidateTokenID(t *testing.T) {
	t.Run("valid zero", func(t *testing.T) {
		if err := domain.ValidateTokenID("0"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("valid simple", func(t *testing.T) {
		if err := domain.ValidateTokenID("12345"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("valid large 78-digit", func(t *testing.T) {
		big78 := "115792089237316195423570985008687907853269984665640564039457584007913129639935"
		if err := domain.ValidateTokenID(big78); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("empty string", func(t *testing.T) {
		if !errors.Is(domain.ValidateTokenID(""), domain.ErrInvalidTokenID) {
			t.Fatal("expected ErrInvalidTokenID for empty")
		}
	})

	t.Run("non-digit chars", func(t *testing.T) {
		if !errors.Is(domain.ValidateTokenID("abc"), domain.ErrInvalidTokenID) {
			t.Fatal("expected ErrInvalidTokenID for non-digit")
		}
	})

	t.Run("negative", func(t *testing.T) {
		if !errors.Is(domain.ValidateTokenID("-1"), domain.ErrInvalidTokenID) {
			t.Fatal("expected ErrInvalidTokenID for negative")
		}
	})

	t.Run("decimal", func(t *testing.T) {
		if !errors.Is(domain.ValidateTokenID("1.5"), domain.ErrInvalidTokenID) {
			t.Fatal("expected ErrInvalidTokenID for decimal")
		}
	})

	t.Run("leading plus", func(t *testing.T) {
		if !errors.Is(domain.ValidateTokenID("+1"), domain.ErrInvalidTokenID) {
			t.Fatal("expected ErrInvalidTokenID for leading +")
		}
	})
}

// ── Certificate.Validate ──────────────────────────────────────────────────────

func validCert(t *testing.T) domain.Certificate {
	t.Helper()
	owner, _ := domain.NewAddress("0xabcdef1234567890abcdef1234567890abcdef12")
	return domain.Certificate{
		TokenID:     "42",
		Owner:       owner,
		Title:       "Go Expert",
		IssuerName:  "Hacktiv8",
		IssuedAt:    time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		ChainID:     1,
		TxHash:      "0xdeadbeef",
		LogIndex:    0,
		BlockNumber: 100,
		BlockHash:   "0xblockhash",
	}
}

func TestCertificateValidate(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		c := validCert(t)
		if err := c.Validate(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("zero owner", func(t *testing.T) {
		c := validCert(t)
		zero, _ := domain.NewAddress("0x0000000000000000000000000000000000000000")
		c.Owner = zero
		if !errors.Is(c.Validate(), domain.ErrInvalidCertificate) {
			t.Fatal("expected ErrInvalidCertificate for zero owner")
		}
	})

	t.Run("empty title", func(t *testing.T) {
		c := validCert(t)
		c.Title = ""
		if !errors.Is(c.Validate(), domain.ErrInvalidCertificate) {
			t.Fatal("expected ErrInvalidCertificate for empty title")
		}
	})

	t.Run("empty issuer name", func(t *testing.T) {
		c := validCert(t)
		c.IssuerName = ""
		if !errors.Is(c.Validate(), domain.ErrInvalidCertificate) {
			t.Fatal("expected ErrInvalidCertificate for empty issuer")
		}
	})

	t.Run("zero issued at", func(t *testing.T) {
		c := validCert(t)
		c.IssuedAt = time.Time{}
		if !errors.Is(c.Validate(), domain.ErrInvalidCertificate) {
			t.Fatal("expected ErrInvalidCertificate for zero IssuedAt")
		}
	})

	t.Run("empty tx hash", func(t *testing.T) {
		c := validCert(t)
		c.TxHash = ""
		if !errors.Is(c.Validate(), domain.ErrInvalidCertificate) {
			t.Fatal("expected ErrInvalidCertificate for empty TxHash")
		}
	})

	t.Run("empty block hash", func(t *testing.T) {
		c := validCert(t)
		c.BlockHash = ""
		if !errors.Is(c.Validate(), domain.ErrInvalidCertificate) {
			t.Fatal("expected ErrInvalidCertificate for empty BlockHash")
		}
	})

	t.Run("negative block number", func(t *testing.T) {
		c := validCert(t)
		c.BlockNumber = -1
		if !errors.Is(c.Validate(), domain.ErrInvalidCertificate) {
			t.Fatal("expected ErrInvalidCertificate for negative BlockNumber")
		}
	})

	t.Run("invalid token id", func(t *testing.T) {
		c := validCert(t)
		c.TokenID = "-5"
		if !errors.Is(c.Validate(), domain.ErrInvalidCertificate) {
			t.Fatal("expected ErrInvalidCertificate for bad TokenID")
		}
	})
}

// ── NewIndexedCertificate ─────────────────────────────────────────────────────

func TestNewIndexedCertificate(t *testing.T) {
	owner, _ := domain.NewAddress("0xabcdef1234567890abcdef1234567890abcdef12")

	goodLog := domain.IssuedLog{
		TokenID:     "42",
		BlockNumber: 100,
		BlockHash:   "0xblockhash",
		TxHash:      "0xdeadbeef",
		LogIndex:    3,
	}
	goodData := domain.OnchainCertificate{
		TokenID:       "42",
		Owner:         owner,
		Title:         "Go Expert",
		RecipientName: "Oksa",
		IssuerName:    "Hacktiv8",
		Description:   "desc",
		MetadataURI:   "ipfs://Qm",
		IssuedAt:      time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	t.Run("happy path — provenance from log, names from data", func(t *testing.T) {
		cert, err := domain.NewIndexedCertificate(goodLog, goodData, 1)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cert.TokenID != "42" {
			t.Errorf("TokenID: want 42, got %q", cert.TokenID)
		}
		if cert.TxHash != goodLog.TxHash {
			t.Errorf("TxHash: want %q, got %q", goodLog.TxHash, cert.TxHash)
		}
		if int64(goodLog.LogIndex) != cert.LogIndex {
			t.Errorf("LogIndex: want %d, got %d", goodLog.LogIndex, cert.LogIndex)
		}
		if int64(goodLog.BlockNumber) != cert.BlockNumber {
			t.Errorf("BlockNumber: want %d, got %d", goodLog.BlockNumber, cert.BlockNumber)
		}
		if cert.BlockHash != goodLog.BlockHash {
			t.Errorf("BlockHash: want %q, got %q", goodLog.BlockHash, cert.BlockHash)
		}
		if cert.Title != goodData.Title {
			t.Errorf("Title: want %q, got %q", goodData.Title, cert.Title)
		}
		if cert.RecipientName != goodData.RecipientName {
			t.Errorf("RecipientName: want %q, got %q", goodData.RecipientName, cert.RecipientName)
		}
		if cert.IssuerName != goodData.IssuerName {
			t.Errorf("IssuerName: want %q, got %q", goodData.IssuerName, cert.IssuerName)
		}
		if cert.Owner != owner {
			t.Errorf("Owner mismatch")
		}
		if cert.ChainID != 1 {
			t.Errorf("ChainID: want 1, got %d", cert.ChainID)
		}
	})

	t.Run("zero owner propagates ErrInvalidCertificate", func(t *testing.T) {
		zeroOwner, _ := domain.NewAddress("0x0000000000000000000000000000000000000000")
		badData := goodData
		badData.Owner = zeroOwner
		_, err := domain.NewIndexedCertificate(goodLog, badData, 1)
		if !errors.Is(err, domain.ErrInvalidCertificate) {
			t.Fatalf("expected ErrInvalidCertificate, got %v", err)
		}
	})

	t.Run("empty title propagates ErrInvalidCertificate", func(t *testing.T) {
		badData := goodData
		badData.Title = ""
		_, err := domain.NewIndexedCertificate(goodLog, badData, 1)
		if !errors.Is(err, domain.ErrInvalidCertificate) {
			t.Fatalf("expected ErrInvalidCertificate, got %v", err)
		}
	})

	t.Run("bad token id propagates ErrInvalidCertificate", func(t *testing.T) {
		badLog := goodLog
		badLog.TokenID = "abc"
		_, err := domain.NewIndexedCertificate(badLog, goodData, 1)
		if !errors.Is(err, domain.ErrInvalidCertificate) {
			t.Fatalf("expected ErrInvalidCertificate, got %v", err)
		}
	})

	t.Run("empty tx hash propagates ErrInvalidCertificate", func(t *testing.T) {
		badLog := goodLog
		badLog.TxHash = ""
		_, err := domain.NewIndexedCertificate(badLog, goodData, 1)
		if !errors.Is(err, domain.ErrInvalidCertificate) {
			t.Fatalf("expected ErrInvalidCertificate, got %v", err)
		}
	})
}
