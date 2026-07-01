package usecase_test

import (
	"testing"

	"github.com/oksasatya/skillpass/services/notify/internal/usecase"
)

func TestSignPayload_MatchesKnownHMAC(t *testing.T) {
	// Ground truth computed independently:
	//   printf '%s' '{"event":"certificate.issued"}' | openssl dgst -sha256 -hmac "test-secret"
	got := usecase.SignPayload("test-secret", []byte(`{"event":"certificate.issued"}`))
	want := "9eafe280c2c64ecd9cb6342b5e415dfde6aa92b787bcaab89440b1f9bc2d532f"
	if got != want {
		t.Fatalf("SignPayload = %q, want %q", got, want)
	}
}

func TestSignPayload_DifferentSecrets_ProduceDifferentSignatures(t *testing.T) {
	body := []byte(`{"event":"certificate.issued"}`)
	sig1 := usecase.SignPayload("secret-a", body)
	sig2 := usecase.SignPayload("secret-b", body)
	if sig1 == sig2 {
		t.Fatal("different secrets must produce different signatures")
	}
}

func TestSignPayload_DifferentBodies_ProduceDifferentSignatures(t *testing.T) {
	sig1 := usecase.SignPayload("secret", []byte(`{"a":1}`))
	sig2 := usecase.SignPayload("secret", []byte(`{"a":2}`))
	if sig1 == sig2 {
		t.Fatal("different bodies must produce different signatures")
	}
}

func TestSignPayload_SameInput_IsDeterministic(t *testing.T) {
	body := []byte(`{"event":"certificate.issued"}`)
	sig1 := usecase.SignPayload("secret", body)
	sig2 := usecase.SignPayload("secret", body)
	if sig1 != sig2 {
		t.Fatal("same secret+body must always produce the same signature")
	}
}
