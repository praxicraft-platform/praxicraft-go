# Praxicraft Assess Go SDK

Official Go client for the **[Praxicraft Assess](https://assess.praxicraft.com)** Public API.

Use it to invite candidates, check invite quota, manage webhooks, enroll hiring pipelines, and fetch results from your ATS, backend, or automation scripts.

```bash
go get github.com/praxicraft-platform/praxicraft-go
```

**Requires Go 1.22+.** Full API reference: [https://docs.praxicraft.com](https://docs.praxicraft.com/sdks/go) 

## Table of Contents

- [Authentication](#authentication)
- [Quickstart](#quickstart)
- [What you can do](#what-you-can-do)
  - [Check invite quota before bulk sends](#check-invite-quota-before-bulk-sends)
  - [Register and test a webhook](#register-and-test-a-webhook)
  - [Verify webhook signatures](#verify-webhook-signatures)
  - [Paginate cohort results](#paginate-cohort-results)
- [Errors](#errors)
- [Requirements & support](#requirements--support)
- [License](#license)

---

## Authentication

Create an organisation API key in Assess:

**Assess → Developer → API Keys** → create key → copy `ct_live_…` (shown once).

```bash
export PRAXICRAFT_API_KEY="ct_live_xxxxxxxxxxxxxxxx"
```

Or pass the key when constructing the client:

```go
import praxicraft "github.com/praxicraft-platform/praxicraft-go"

client, err := praxicraft.New(praxicraft.WithAPIKey("ct_live_xxxxxxxxxxxxxxxx"))
if err != nil {
    log.Fatal(err)
}
```

Optional: override the API host with `PRAXICRAFT_API_BASE_URL` or `WithBaseURL(...)`.
Default host: `https://assess.praxicraft.com`.

Never commit API keys. Prefer environment variables or a secrets manager.

Scopes and rotation: [Authentication](https://docs.praxicraft.com/authentication)

---

## Quickstart

```go
package main

import (
    "fmt"
    "log"

    praxicraft "github.com/praxicraft-platform/praxicraft-go"
)

func main() {
    client, err := praxicraft.New() // reads PRAXICRAFT_API_KEY
    if err != nil {
        log.Fatal(err)
    }

    org, err := client.Org.Retrieve()
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println(org.Name, org.Plan, org.InvitesRemaining)

    send := true
    invite, err := client.Invites.Create("senior-backend-screen", praxicraft.InviteCreateParams{
        Email:     "candidate@example.com",
        Name:      "Jane Doe",
        SendEmail: &send,
    })
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println(invite.InviteToken, invite.InviteURL)

    result, err := client.Results.Retrieve(invite.InviteToken)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println(result)
}
```

Responses decode into typed structs on resource methods. Use `client.Do` / `DoAs` for escape-hatch / forward-compatible access. Success bodies are flat JSON (no `{ data: … }` wrapper).

Transient `429` / `5xx` / transport failures retry automatically (default 2 retries). Override with `WithMaxRetries(0)` to disable.

---

## What you can do

| Resource | Common methods |
|----------|----------------|
| `client.Org` | `Retrieve()`, `Stats()` |
| `client.Assessments` | `List()`, `Retrieve()`, `Create()`, `Update()`, `Activate()`, `ListTasks()`, `AttachTasks()`, `ReplaceTasks()`, `RemoveTask()` |
| `client.Invites` | `Create()`, `BulkCreate()`, `List()`, `Retrieve()`, `Remind()`, `Cancel()` |
| `client.Results` | `List()`, `Retrieve()`, `IterAll()` |
| `client.Webhooks` | `List()`, `Create()`, `Retrieve()`, `Update()`, `Delete()`, `Test()`, `Deliveries()` |
| `client.Pipelines` | `List()`, `Retrieve()`, `Enroll()`, `BulkEnroll()`, `ListEnrollments()`, `GetEnrollment()` |
| `VerifySignature` | Verify `X-Praxicraft-Signature` on webhook payloads |

All paths target `/api/v1/public/…` on the Assess host.

### Check invite quota before bulk sends

```go
org, err := client.Org.Retrieve()
if err != nil {
    log.Fatal(err)
}
if org.InvitesRemaining != nil && *org.InvitesRemaining < len(candidates) {
    log.Fatal("Not enough invites remaining this month")
}
```

### Register and test a webhook

```go
hook, err := client.Webhooks.Create(
    "https://example.com/hooks/praxicraft",
    []string{"assessment.completed", "candidate.passed"},
    nil,
)
if err != nil {
    log.Fatal(err)
}
// Store hook.SecretKey (whsec_…) — shown once
_, _ = client.Webhooks.Test(hook.ID)
_, _ = client.Webhooks.Update(hook.ID, map[string]any{"is_active": true})
```

### Verify webhook signatures

```go
ok := praxicraft.VerifySignature(secret, rawBody, signatureHeader)
```

Header format: `X-Praxicraft-Signature: sha256=<hex>`

Event catalog: [Webhooks](https://docs.praxicraft.com/webhooks)

### Paginate cohort results

```go
pageSize := 50
err = client.Results.IterAll("senior-backend-screen", &pageSize, nil, func(row praxicraft.ResultRow) error {
    fmt.Println(row.Email, row.Score, row.Passed)
    return nil
})
```

---

## Errors

Branch on typed errors / `ErrCode` (stable), not the message text:

```go
_, err := client.Invites.Create("demo", praxicraft.InviteCreateParams{Email: "candidate@example.com"})
var (
    ve   *praxicraft.ValidationError
    scope *praxicraft.InsufficientScopeError
    auth *praxicraft.AuthenticationError
    rl   *praxicraft.RateLimitError
)
switch {
case errors.As(err, &ve):
    fmt.Println(ve.ErrCode, ve.Details)
case errors.As(err, &scope):
    fmt.Println(scope.ErrCode, scope.RequiredPlan)
case errors.As(err, &auth):
    fmt.Println(auth.ErrCode)
case errors.As(err, &rl):
    fmt.Println(rl.RetryAfter)
default:
    log.Fatal(err)
}
```

Error codes: [Errors](https://docs.praxicraft.com/errors)

---

## Requirements & support

- Go **1.22+**
- Product docs: [docs.praxicraft.com](https://docs.praxicraft.com)
- Issues: [GitHub Issues](https://github.com/praxicraft-platform/praxicraft-go/issues)

---

## License

[MIT](LICENSE)
