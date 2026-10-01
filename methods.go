package novasend

import (
	"context"
	"encoding/json"
	"fmt"
)

// ── Request types ────────────────────────────────────────────────────────────

type PaymentAction struct {
	SuccessURL string `json:"successUrl"`
	FailureURL string `json:"failureUrl"`
}

type WalletPaymentRequest struct {
	Reference       string        `json:"reference"`
	SandboxScenario string        `json:"sandboxScenario,omitempty"`
	Amount          int           `json:"amount"`
	MSISDN          string        `json:"msisdn"`
	CustomerName    string        `json:"customerName"`
	Country         string        `json:"country"`
	Action          PaymentAction `json:"action"`
}

type DirectPayinData struct {
	Amount   int    `json:"amount"`
	MSISDN   string `json:"msisdn"`
	Provider string `json:"provider"`
	Country  string `json:"country"`
	OTP      string `json:"otp,omitempty"`
}

type DirectPaymentRequest struct {
	Reference       string          `json:"reference"`
	SandboxScenario string          `json:"sandboxScenario,omitempty"`
	CustomerName    string          `json:"customerName"`
	Payin           DirectPayinData `json:"payin"`
	Action          PaymentAction   `json:"action"`
}

type WalletTransferRequest struct {
	Reference       string `json:"reference"`
	SandboxScenario string `json:"sandboxScenario,omitempty"`
	Amount          int    `json:"amount"`
	MSISDN          string `json:"msisdn"`
	Country         string `json:"country"`
	CustomerName    string `json:"customerName"`
}

type DirectPayoutData struct {
	Amount   int    `json:"amount"`
	MSISDN   string `json:"msisdn"`
	Provider string `json:"provider"`
	Country  string `json:"country"`
}

type DirectTransferRequest struct {
	Reference       string           `json:"reference"`
	SandboxScenario string           `json:"sandboxScenario,omitempty"`
	CustomerName    string           `json:"customerName"`
	Payout          DirectPayoutData `json:"payout"`
}

type RefundRequest struct {
	Reason            string `json:"reason"`
	ExternalReference string `json:"externalReference"`
	SandboxScenario   string `json:"sandboxScenario,omitempty"`
}

// ── Response types ───────────────────────────────────────────────────────────

type CustomerInfo struct {
	Name        string `json:"name"`
	PhoneNumber string `json:"phoneNumber"`
}

type MobileMoneyProvider struct {
	Name string `json:"name"`
}

type MobileMoneyInfo struct {
	Provider MobileMoneyProvider `json:"provider"`
}

type PaymentResponse struct {
	ID                   string                 `json:"id"`
	Type                 string                 `json:"type"`
	Reference            string                 `json:"reference"`
	Status               string                 `json:"status"`
	ConfirmationRequired bool                   `json:"confirmationRequired"`
	ConfirmationStatus   string                 `json:"confirmationStatus"`
	PaymentURL           string                 `json:"paymentUrl,omitempty"`
	Customer             *CustomerInfo          `json:"customer,omitempty"`
	MobileMoney          *MobileMoneyInfo       `json:"mobileMoney,omitempty"`
	IsDirect             bool                   `json:"isDirect"`
	PayFee               bool                   `json:"payFee"`
	CreatedAt            string                 `json:"createdAt"`
	Amount               json.Number            `json:"amount"`
	Fee                  json.Number            `json:"fee"`
	ChargedAmount        json.Number            `json:"chargedAmount"`
	Currency             string                 `json:"currency"`
	Failure              map[string]interface{} `json:"failure"`
}

type RefundTransaction struct {
	Reference   string  `json:"reference"`
	Status      string  `json:"status"`
	ProcessedAt *string `json:"processedAt"`
}

type RefundResponse struct {
	Type              string            `json:"type"`
	Reference         string            `json:"reference"`
	OriginalReference string            `json:"originalReference"`
	Amount            int               `json:"amount"`
	Fee               int               `json:"fee"`
	Currency          string            `json:"currency"`
	Status            string            `json:"status"`
	Reason            string            `json:"reason"`
	RefundType        string            `json:"refundType"`
	CustomerMSISDN    string            `json:"customerMsisdn"`
	CustomerName      string            `json:"customerName"`
	CreatedAt         string            `json:"createdAt"`
	Transaction       RefundTransaction `json:"transaction"`
}

// ── Wallet API ───────────────────────────────────────────────────────────────

func (c *Client) CreateWalletPayment(ctx context.Context, req *WalletPaymentRequest) (*PaymentResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("novasend: request is required")
	}
	if err := validateReference(req.Reference); err != nil {
		return nil, err
	}
	if err := validateSandboxScenario(req.SandboxScenario); err != nil {
		return nil, err
	}
	if err := validatePayinAmount(req.Amount); err != nil {
		return nil, err
	}
	if err := validatePhone(req.MSISDN, req.Country); err != nil {
		return nil, err
	}
	var resp PaymentResponse
	if err := c.doRequest(ctx, "POST", "/v1/payin/sessions", req, &resp, req.Reference); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) CreateWalletTransfer(ctx context.Context, req *WalletTransferRequest) (*PaymentResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("novasend: request is required")
	}
	if err := validateReference(req.Reference); err != nil {
		return nil, err
	}
	if err := validateSandboxScenario(req.SandboxScenario); err != nil {
		return nil, err
	}
	if err := validatePayoutAmount(req.Amount); err != nil {
		return nil, err
	}
	if err := validatePhone(req.MSISDN, req.Country); err != nil {
		return nil, err
	}
	var resp PaymentResponse
	if err := c.doRequest(ctx, "POST", "/v1/payout", req, &resp, req.Reference); err != nil {
		return nil, err
	}
	return &resp, nil
}

// ── Direct API ───────────────────────────────────────────────────────────────

func (c *Client) CreateDirectPayment(ctx context.Context, req *DirectPaymentRequest) (*PaymentResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("novasend: request is required")
	}
	if err := validateReference(req.Reference); err != nil {
		return nil, err
	}
	if err := validateSandboxScenario(req.SandboxScenario); err != nil {
		return nil, err
	}
	if err := validatePayinAmount(req.Payin.Amount); err != nil {
		return nil, err
	}
	if err := validatePhone(req.Payin.MSISDN, req.Payin.Country); err != nil {
		return nil, err
	}
	if err := validateOTP(req.Payin.Provider, req.Payin.OTP); err != nil {
		return nil, err
	}
	if err := validateProvider(req.Payin.Provider); err != nil {
		return nil, err
	}
	var resp PaymentResponse
	if err := c.doRequest(ctx, "POST", "/v1/direct/payin", req, &resp, req.Reference); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) CreateDirectTransfer(ctx context.Context, req *DirectTransferRequest) (*PaymentResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("novasend: request is required")
	}
	if err := validateReference(req.Reference); err != nil {
		return nil, err
	}
	if err := validateSandboxScenario(req.SandboxScenario); err != nil {
		return nil, err
	}
	if err := validatePayoutAmount(req.Payout.Amount); err != nil {
		return nil, err
	}
	if err := validatePhone(req.Payout.MSISDN, req.Payout.Country); err != nil {
		return nil, err
	}
	if err := validateProvider(req.Payout.Provider); err != nil {
		return nil, err
	}
	var resp PaymentResponse
	if err := c.doRequest(ctx, "POST", "/v1/direct/payout", req, &resp, req.Reference); err != nil {
		return nil, err
	}
	return &resp, nil
}

// ── Status ───────────────────────────────────────────────────────────────────

func (c *Client) GetPaymentStatus(ctx context.Context, reference string) (*PaymentResponse, error) {
	if err := validateReference(reference); err != nil {
		return nil, err
	}
	var resp PaymentResponse
	if err := c.doRequest(ctx, "GET", "/v1/payin/"+pathEscape(reference), nil, &resp, ""); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) GetWalletTransferStatus(ctx context.Context, reference string) (*PaymentResponse, error) {
	if err := validateReference(reference); err != nil {
		return nil, err
	}
	var resp PaymentResponse
	if err := c.doRequest(ctx, "GET", "/v1/payout/"+pathEscape(reference), nil, &resp, ""); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) GetDirectTransferStatus(ctx context.Context, reference string) (*PaymentResponse, error) {
	if err := validateReference(reference); err != nil {
		return nil, err
	}
	var resp PaymentResponse
	if err := c.doRequest(ctx, "GET", "/v1/direct/payout/"+pathEscape(reference), nil, &resp, ""); err != nil {
		return nil, err
	}
	return &resp, nil
}

// ── Refund ───────────────────────────────────────────────────────────────────

func (c *Client) RefundPayment(ctx context.Context, reference string, req *RefundRequest) (*RefundResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("novasend: request is required")
	}
	if err := validateReference(reference); err != nil {
		return nil, err
	}
	if err := validateReference(req.ExternalReference); err != nil {
		return nil, err
	}
	if err := validateSandboxScenario(req.SandboxScenario); err != nil {
		return nil, err
	}
	var resp RefundResponse
	if err := c.doRequest(ctx, "POST", "/v1/payin/"+pathEscape(reference)+"/refund", req, &resp, req.ExternalReference); err != nil {
		return nil, err
	}
	return &resp, nil
}
