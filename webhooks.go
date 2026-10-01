package novasend

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

var hexPattern = regexp.MustCompile(`^[0-9a-fA-F]{64}$`)

const maxWebhookSize = 1 << 20

// ── Webhook payload ──────────────────────────────────────────────────────────

type WebhookPayload struct {
	ID                   string                 `json:"id"`
	Reference            string                 `json:"reference"`
	Type                 string                 `json:"type"`
	Status               string                 `json:"status"`
	Amount               int                    `json:"amount"`
	Fee                  int                    `json:"fees"`
	Currency             string                 `json:"currency"`
	Customer             *CustomerInfo          `json:"customer,omitempty"`
	CreatedAt            string                 `json:"createdAt"`
	PaymentURL           string                 `json:"paymentUrl,omitempty"`
	ChargedAmount        int                    `json:"chargedAmount"`
	ConfirmationStatus   string                 `json:"confirmationStatus"`
	ConfirmationRequired bool                   `json:"confirmationRequired"`
	Failure              map[string]interface{} `json:"failure"`
}

// ── Webhooks ─────────────────────────────────────────────────────────────────

type Webhooks struct {
	secret string
}

func NewWebhooks(secret string) (*Webhooks, error) {
	s := strings.TrimSpace(secret)
	if s == "" {
		return nil, fmt.Errorf("novasend: webhook secret is required")
	}
	return &Webhooks{secret: s}, nil
}

func (w *Webhooks) VerifySignature(payload []byte, signature string) (*WebhookPayload, error) {
	if len(payload) > maxWebhookSize {
		return nil, fmt.Errorf("novasend: webhook payload is too large")
	}
	sig := strings.TrimSpace(signature)
	if sig == "" {
		return nil, fmt.Errorf("novasend: missing signature header")
	}
	if !hexPattern.MatchString(sig) {
		return nil, fmt.Errorf("novasend: invalid signature format (expected 64 hex chars)")
	}

	mac := hmac.New(sha256.New, []byte(w.secret))
	mac.Write(payload)
	provided, err := hex.DecodeString(sig)
	if err != nil || !hmac.Equal(mac.Sum(nil), provided) {
		return nil, fmt.Errorf("novasend: invalid webhook signature")
	}

	var event WebhookPayload
	if err := json.Unmarshal(payload, &event); err != nil {
		return nil, fmt.Errorf("novasend: failed to parse webhook payload: %w", err)
	}
	if strings.TrimSpace(event.Reference) == "" || strings.TrimSpace(event.Status) == "" {
		return nil, fmt.Errorf("novasend: invalid webhook payload")
	}
	return &event, nil
}
