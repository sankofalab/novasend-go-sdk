# Novasend Go SDK

SDK officiel Go pour l'API Novasend. Permet d'integrer les paiements Mobile Money,
les transferts et les remboursements dans vos applications Go.

- **Documentation API** : https://docs.novasend.app
- **Portail marchand** : https://portal.novasend.app

---

## Installation

```bash
go get github.com/sankofalab/novasend-go-sdk@v0.1.0
```

Zero dependance externe. Le SDK utilise uniquement la bibliotheque standard Go.

## Configuration

1. Rendez-vous sur https://portal.novasend.app, section **Parametres > Token > Nouveau token**.
2. Copiez `.env.example` vers `.env` et renseignez vos identifiants :

```
NOVASEND_API_KEY=your_api_key
NOVASEND_SECRET_KEY=your_secret_key
NOVASEND_BASE_URL=https://sandbox.novasend.app
```

## Utilisation

### Initialisation

```go
import novasend "github.com/sankofalab/novasend-go-sdk"

client, err := novasend.NewClient(novasend.Config{
    APIKey:    os.Getenv("NOVASEND_API_KEY"),
    SecretKey: os.Getenv("NOVASEND_SECRET_KEY"),
    BaseURL:   os.Getenv("NOVASEND_BASE_URL"),
    Language:  "fr",                    // optionnel, defaut : "fr"
    Timeout:   30 * time.Second,        // optionnel, defaut : 30s
})
```

Utilisez `https://sandbox.novasend.app` pour les tests et `https://business.novasend.app` pour la production. Pour simuler un résultat sur la Sandbox, ajoutez `SandboxScenario: "completed"` à la requête POST. Les valeurs possibles sont `completed`, `failed` et `pending`.

### Paiement Wallet (redirection)

```go
resp, err := client.CreateWalletPayment(ctx, &novasend.WalletPaymentRequest{
    Reference:    "uuid-v4",
    SandboxScenario: "completed", // uniquement sur la Sandbox
    Amount:       1000,
    MSISDN:       "+2250101010101",
    CustomerName: "Kone Hamed",
    Country:      "CI",
    Action: novasend.PaymentAction{
        SuccessURL: "https://example.com/success",
        FailureURL: "https://example.com/failure",
    },
})
fmt.Println(resp.PaymentURL)
```

### Paiement Direct (push USSD)

```go
resp, err := client.CreateDirectPayment(ctx, &novasend.DirectPaymentRequest{
    Reference:    "uuid-v4",
    CustomerName: "Kone Hamed",
    Payin: novasend.DirectPayinData{
        Amount:   1000,
        MSISDN:   "+2250101010101",
        Provider: "WAVE",     // WAVE | ORANGE | MOMO | MOOV
        Country:  "CI",       // CI | CM
    },
    Action: novasend.PaymentAction{
        SuccessURL: "https://example.com/success",
        FailureURL: "https://example.com/failure",
    },
})
```

### Paiement Direct Orange Money (OTP requis)

Pour Orange Money, le client doit fournir un code OTP obtenu en composant `#144*82#`.

```go
resp, err := client.CreateDirectPayment(ctx, &novasend.DirectPaymentRequest{
    Reference:    "uuid-v4",
    CustomerName: "Kone Hamed",
    Payin: novasend.DirectPayinData{
        Amount:   1000,
        MSISDN:   "+2250101010101",
        Provider: "ORANGE",
        Country:  "CI",
        OTP:      "1234",     // requis -- obtenu par le client via #144*82#
    },
    Action: novasend.PaymentAction{
        SuccessURL: "https://example.com/success",
        FailureURL: "https://example.com/failure",
    },
})
```

### Transfert (payout)

```go
resp, err := client.CreateWalletTransfer(ctx, &novasend.WalletTransferRequest{
    Reference:    "uuid-v4",
    Amount:       1000,
    MSISDN:       "+2250101010101",
    Country:      "CI",
    CustomerName: "Kone Hamed",
})
```

### Statut d'un paiement

```go
status, err := client.GetPaymentStatus(ctx, "reference-uuid")
fmt.Println(status.Status) // processing | processed | expired
```

### Remboursement

```go
refund, err := client.RefundPayment(ctx, "reference-uuid", &novasend.RefundRequest{
    Reason:            "Client request",
    ExternalReference: "ext-ref-uuid",
})
```

### Webhook (net/http)

```go
import (
    "fmt"
    "io"
    "log"
    "net/http"
    "os"
)

wh, err := novasend.NewWebhooks(os.Getenv("NOVASEND_WEBHOOK_SECRET"))
if err != nil { log.Fatal(err) }

http.HandleFunc("/webhook", func(w http.ResponseWriter, r *http.Request) {
    body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 1<<20))
    if err != nil { http.Error(w, "Invalid request", http.StatusBadRequest); return }
    sig := r.Header.Get("X-Signature-Value")

    event, err := wh.VerifySignature(body, sig)
    if err != nil {
        http.Error(w, "Unauthorized", http.StatusUnauthorized)
        return
    }
    fmt.Println(event.Reference, event.Status)
    w.WriteHeader(http.StatusOK)
})

log.Fatal(http.ListenAndServe(":3000", nil))
```

### Gestion des erreurs

```go
resp, err := client.CreateDirectPayment(ctx, req)
if err != nil {
    var apiErr *novasend.APIError
    if errors.As(err, &apiErr) {
        fmt.Printf("API error %d: %s\n", apiErr.StatusCode, apiErr.Message)
    } else {
        fmt.Println("Unexpected error:", err)
    }
}
```

## Methodes disponibles

| Methode | Endpoint | Description |
|---------|----------|-------------|
| `CreateWalletPayment` | `POST /v1/payin/sessions` | Paiement wallet (redirection) |
| `CreateDirectPayment` | `POST /v1/direct/payin` | Paiement direct (push USSD) |
| `CreateWalletTransfer` | `POST /v1/payout` | Transfert wallet |
| `CreateDirectTransfer` | `POST /v1/direct/payout` | Transfert direct |
| `GetPaymentStatus` | `GET /v1/payin/{ref}` | Statut d'un paiement |
| `GetWalletTransferStatus` | `GET /v1/payout/{ref}` | Statut d'un transfert wallet |
| `GetDirectTransferStatus` | `GET /v1/direct/payout/{ref}` | Statut d'un transfert direct |
| `RefundPayment` | `POST /v1/payin/{ref}/refund` | Remboursement |

## Validation integree

Le SDK valide automatiquement avant chaque requete :

- **Numeros de telephone** : `+225XXXXXXXXXX` (CI), `+237XXXXXXXXX` (CM)
- **OTP Orange Money** : requis si `provider == "ORANGE"`

## Securite

- Authentification HTTP Basic (credentials encodes en Base64).
- `X-Idempotency-Key` reprend la reference de la mutation pour rester stable lors d'une nouvelle tentative.
- Verification des webhooks via HMAC-SHA256 avec `hmac.Equal` (timing-safe).
- Timeout configurable via `http.Client`.
- Zero dependance externe.
- Aucune donnee sensible n'est codee en dur dans le SDK.

## Tests

```bash
go test ./... -v
```

## Licence

MIT
