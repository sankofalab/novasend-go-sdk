package novasend

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// ── Constants ────────────────────────────────────────────────────────────────

const (
	PayinAmountMin  = 300
	PayinAmountMax  = 2_000_000
	PayoutAmountMin = 500
	PayoutAmountMax = 1_500_000

	DefaultTimeout  = 30 * time.Second
	maxResponseSize = 10 << 20
)

var phoneRegex = map[string]*regexp.Regexp{
	"CI": regexp.MustCompile(`^\+225\d{10}$`),
	"CM": regexp.MustCompile(`^\+237\d{9}$`),
}

// ── Client ───────────────────────────────────────────────────────────────────

type Config struct {
	APIKey    string
	SecretKey string
	BaseURL   string
	Language  string
	Timeout   time.Duration
}

type Client struct {
	apiKey     string
	secretKey  string
	baseURL    string
	language   string
	httpClient *http.Client
}

func NewClient(cfg Config) (*Client, error) {
	if strings.TrimSpace(cfg.APIKey) == "" || strings.TrimSpace(cfg.SecretKey) == "" {
		return nil, fmt.Errorf("novasend: apiKey and secretKey are required")
	}
	baseURL, err := validateBaseURL(cfg.BaseURL)
	if err != nil {
		return nil, err
	}
	lang := cfg.Language
	if lang == "" {
		lang = "fr"
	}
	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = DefaultTimeout
	}
	if timeout < 0 {
		return nil, fmt.Errorf("novasend: timeout must be positive")
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.DialContext = (&net.Dialer{
		Timeout:   10 * time.Second,
		KeepAlive: 30 * time.Second,
	}).DialContext
	return &Client{
		apiKey:    strings.TrimSpace(cfg.APIKey),
		secretKey: strings.TrimSpace(cfg.SecretKey),
		baseURL:   baseURL,
		language:  lang,
		httpClient: &http.Client{
			Timeout:   timeout,
			Transport: transport,
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}, nil
}

// ── Error ────────────────────────────────────────────────────────────────────

type APIError struct {
	StatusCode int
	Message    string
	Body       map[string]interface{}
}

func (e *APIError) Error() string {
	return fmt.Sprintf("novasend: API error %d: %s", e.StatusCode, e.Message)
}

// ── Validation ───────────────────────────────────────────────────────────────

func validatePayinAmount(amount int) error {
	if amount < PayinAmountMin || amount > PayinAmountMax {
		return fmt.Errorf("novasend: payin amount must be between %d and %d FCFA, got %d", PayinAmountMin, PayinAmountMax, amount)
	}
	return nil
}

func validatePayoutAmount(amount int) error {
	if amount < PayoutAmountMin || amount > PayoutAmountMax {
		return fmt.Errorf("novasend: payout amount must be between %d and %d FCFA, got %d", PayoutAmountMin, PayoutAmountMax, amount)
	}
	return nil
}

func validatePhone(msisdn, country string) error {
	re, ok := phoneRegex[country]
	if !ok {
		return fmt.Errorf("novasend: unsupported country %q", country)
	}
	if !re.MatchString(msisdn) {
		return fmt.Errorf("novasend: invalid phone number %q for country %s", msisdn, country)
	}
	return nil
}

func validateOTP(provider, otp string) error {
	if strings.EqualFold(provider, "ORANGE") && strings.TrimSpace(otp) == "" {
		return fmt.Errorf("novasend: OTP is required for Orange Money payments (obtained via #144*82#)")
	}
	return nil
}

func validateProvider(provider string) error {
	switch strings.ToUpper(provider) {
	case "WAVE", "ORANGE", "MOMO", "MOOV":
		return nil
	default:
		return fmt.Errorf("novasend: unsupported provider %q", provider)
	}
}

func validateSandboxScenario(scenario string) error {
	switch scenario {
	case "", "completed", "failed", "pending":
		return nil
	default:
		return fmt.Errorf("novasend: sandboxScenario must be completed, failed or pending")
	}
}

func validateReference(reference string) error {
	if strings.TrimSpace(reference) == "" || len(reference) > 128 || strings.IndexFunc(reference, func(r rune) bool { return r < 32 || r == 127 }) >= 0 {
		return fmt.Errorf("novasend: reference must be between 1 and 128 characters and contain no control characters")
	}
	return nil
}

func validateBaseURL(value string) (string, error) {
	trimmed := strings.TrimRight(strings.TrimSpace(value), "/")
	u, err := url.Parse(trimmed)
	if err != nil || u.Host == "" {
		return "", fmt.Errorf("novasend: baseURL must be a valid absolute URL")
	}
	ip := net.ParseIP(u.Hostname())
	isLoopback := u.Hostname() == "localhost" || (ip != nil && ip.IsLoopback())
	if u.User != nil || (u.Scheme != "https" && !(u.Scheme == "http" && isLoopback)) {
		return "", fmt.Errorf("novasend: baseURL must use HTTPS (HTTP is allowed only for localhost) and must not contain credentials")
	}
	if u.RawQuery != "" || u.Fragment != "" {
		return "", fmt.Errorf("novasend: baseURL must not contain a query or fragment")
	}
	return trimmed, nil
}

// ── HTTP ─────────────────────────────────────────────────────────────────────

func (c *Client) doRequest(ctx context.Context, method, endpoint string, body interface{}, result interface{}, idempotencyKey string) error {
	reqURL := c.baseURL + endpoint
	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("novasend: failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequestWithContext(ctx, method, reqURL, bodyReader)
	if err != nil {
		return fmt.Errorf("novasend: failed to create request: %w", err)
	}

	auth := base64.StdEncoding.EncodeToString([]byte(c.apiKey + ":" + c.secretKey))
	req.Header.Set("Authorization", "Basic "+auth)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Accept-Language", c.language)
	if idempotencyKey != "" {
		req.Header.Set("X-Idempotency-Key", idempotencyKey)
	}
	req.Header.Set("User-Agent", "Novasend-Go-SDK/1.0")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("novasend: request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseSize+1))
	if err != nil {
		return fmt.Errorf("novasend: failed to read response body: %w", err)
	}
	if len(respBody) > maxResponseSize {
		return fmt.Errorf("novasend: response body exceeds %d bytes", maxResponseSize)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var errData map[string]interface{}
		_ = json.Unmarshal(respBody, &errData)
		msg := "API Error"
		if m, ok := errData["message"].(string); ok {
			msg = m
		}
		return &APIError{
			StatusCode: resp.StatusCode,
			Message:    msg,
			Body:       errData,
		}
	}

	if result != nil {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("novasend: failed to parse response: %w", err)
		}
	}
	return nil
}

func pathEscape(s string) string {
	return url.PathEscape(s)
}
