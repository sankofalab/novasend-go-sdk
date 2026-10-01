package novasend

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
)

func TestClientRejectsUnsafeConfigAndInvalidInput(t *testing.T) {
	if _, err := NewClient(Config{APIKey: "a", SecretKey: "b", BaseURL: "http://api.example.com"}); err == nil {
		t.Fatal("expected insecure base URL to be rejected")
	}
	client, err := NewClient(Config{APIKey: "a", SecretKey: "b", BaseURL: "http://localhost:3000"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.CreateWalletPayment(context.Background(), nil); err == nil {
		t.Fatal("expected nil request to be rejected")
	}
	if err := validatePhone("+2250102030405", "XX"); err == nil {
		t.Fatal("expected unsupported country to be rejected")
	}
	if err := validateProvider("UNKNOWN"); err == nil {
		t.Fatal("expected unsupported provider to be rejected")
	}
}

func TestWebhookAcceptsUppercaseSignatureAndValidatesPayload(t *testing.T) {
	webhooks, err := NewWebhooks("secret")
	if err != nil {
		t.Fatal(err)
	}
	payload := []byte(`{"reference":"ref","status":"processed"}`)
	mac := hmac.New(sha256.New, []byte("secret"))
	mac.Write(payload)
	signature := strings.ToUpper(hex.EncodeToString(mac.Sum(nil)))
	if _, err := webhooks.VerifySignature(payload, signature); err != nil {
		t.Fatalf("expected valid uppercase signature: %v", err)
	}

	invalid := []byte(`{"reference":"ref"}`)
	mac = hmac.New(sha256.New, []byte("secret"))
	mac.Write(invalid)
	if _, err := webhooks.VerifySignature(invalid, hex.EncodeToString(mac.Sum(nil))); err == nil {
		t.Fatal("expected payload without status to be rejected")
	}
}

func TestWebhookRejectsOversizedPayload(t *testing.T) {
	webhooks, err := NewWebhooks("secret")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := webhooks.VerifySignature([]byte(strings.Repeat("x", maxWebhookSize+1)), strings.Repeat("0", 64)); err == nil {
		t.Fatal("expected oversized payload to be rejected")
	}
}
