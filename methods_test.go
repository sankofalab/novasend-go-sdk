package novasend

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMethodsUseDocumentedHTTPContract(t *testing.T) {
	const reference = "merchant-ref"
	const phone = "+2250101010101"
	tests := []struct {
		name, method, path, bodyKey, idempotencyKey string
		call                                        func(*Client) error
	}{
		{"wallet payment", "POST", "/v1/payin/sessions", "amount", reference, func(c *Client) error {
			_, err := c.CreateWalletPayment(context.Background(), &WalletPaymentRequest{Reference: reference, SandboxScenario: "completed", Amount: 300, MSISDN: phone, Country: "CI"})
			return err
		}},
		{"wallet transfer", "POST", "/v1/payout", "amount", reference, func(c *Client) error {
			_, err := c.CreateWalletTransfer(context.Background(), &WalletTransferRequest{Reference: reference, SandboxScenario: "completed", Amount: 500, MSISDN: phone, Country: "CI"})
			return err
		}},
		{"direct payment", "POST", "/v1/direct/payin", "payin", reference, func(c *Client) error {
			_, err := c.CreateDirectPayment(context.Background(), &DirectPaymentRequest{Reference: reference, SandboxScenario: "completed", Payin: DirectPayinData{Amount: 300, MSISDN: phone, Country: "CI", Provider: "WAVE"}})
			return err
		}},
		{"direct transfer", "POST", "/v1/direct/payout", "payout", reference, func(c *Client) error {
			_, err := c.CreateDirectTransfer(context.Background(), &DirectTransferRequest{Reference: reference, SandboxScenario: "completed", Payout: DirectPayoutData{Amount: 500, MSISDN: phone, Country: "CI", Provider: "WAVE"}})
			return err
		}},
		{"payment status", "GET", "/v1/payin/merchant-ref", "", "", func(c *Client) error {
			_, err := c.GetPaymentStatus(context.Background(), reference)
			return err
		}},
		{"wallet transfer status", "GET", "/v1/payout/merchant-ref", "", "", func(c *Client) error {
			_, err := c.GetWalletTransferStatus(context.Background(), reference)
			return err
		}},
		{"direct transfer status", "GET", "/v1/direct/payout/merchant-ref", "", "", func(c *Client) error {
			_, err := c.GetDirectTransferStatus(context.Background(), reference)
			return err
		}},
		{"refund", "POST", "/v1/payin/merchant-ref/refund", "externalReference", "refund-ref", func(c *Client) error {
			_, err := c.RefundPayment(context.Background(), reference, &RefundRequest{ExternalReference: "refund-ref", Reason: "requested", SandboxScenario: "completed"})
			return err
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != tt.method || r.URL.Path != tt.path {
					t.Errorf("request = %s %s, want %s %s", r.Method, r.URL.Path, tt.method, tt.path)
				}
				wantAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte("key:secret"))
				if r.Header.Get("Authorization") != wantAuth || r.Header.Get("X-Idempotency-Key") != tt.idempotencyKey {
					t.Errorf("incorrect auth or idempotency headers")
				}
				if tt.bodyKey != "" {
					var body map[string]json.RawMessage
					if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
						t.Errorf("decode request: %v", err)
					}
					if _, ok := body[tt.bodyKey]; !ok || string(body["sandboxScenario"]) != `"completed"` {
						t.Errorf("incorrect request body: %s", body)
					}
				}
				w.Header().Set("Content-Type", "application/json")
				if tt.name == "refund" {
					_, _ = w.Write([]byte(`{"reference":"refund-ref","transaction":{"processedAt":null}}`))
				} else {
					_, _ = w.Write([]byte(`{"reference":"merchant-ref","amount":300,"fee":7,"chargedAmount":307}`))
				}
			}))
			defer server.Close()
			client, err := NewClient(Config{APIKey: "key", SecretKey: "secret", BaseURL: server.URL})
			if err != nil {
				t.Fatal(err)
			}
			if err := tt.call(client); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestClientRejectsRedirectsAndInvalidScenario(t *testing.T) {
	redirected := false
	target := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { redirected = true }))
	defer target.Close()
	server := httptest.NewServer(http.RedirectHandler(target.URL, http.StatusFound))
	defer server.Close()
	client, err := NewClient(Config{APIKey: "key", SecretKey: "secret", BaseURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.GetPaymentStatus(context.Background(), "merchant-ref")
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusFound || redirected {
		t.Fatalf("redirect should return APIError without being followed: %v", err)
	}
	_, err = client.CreateWalletPayment(context.Background(), &WalletPaymentRequest{Reference: "merchant-ref", SandboxScenario: "invalid"})
	if err == nil {
		t.Fatal("invalid sandbox scenario should be rejected")
	}
}
