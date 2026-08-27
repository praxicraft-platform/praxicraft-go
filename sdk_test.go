package praxicraft_test

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	praxicraft "github.com/praxicraft-platform/praxicraft-go"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func jsonResp(status int, body any, headers map[string]string) *http.Response {
	var raw []byte
	if body != nil {
		raw, _ = json.Marshal(body)
	}
	h := http.Header{}
	h.Set("Content-Type", "application/json")
	for k, v := range headers {
		h.Set(k, v)
	}
	return &http.Response{
		StatusCode: status,
		Header:     h,
		Body:       io.NopCloser(bytes.NewReader(raw)),
	}
}

func testClient(t *testing.T, rt http.RoundTripper, opts ...praxicraft.ClientOption) *praxicraft.Client {
	t.Helper()
	base := []praxicraft.ClientOption{
		praxicraft.WithAPIKey("ct_live_test"),
		praxicraft.WithBaseURL("https://assess.example.com"),
		praxicraft.WithHTTPClient(&http.Client{Transport: rt}),
	}
	base = append(base, opts...)
	c, err := praxicraft.New(base...)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestNewRejectsMissingKey(t *testing.T) {
	t.Setenv("PRAXICRAFT_API_KEY", "")
	_, err := praxicraft.New()
	var apiErr *praxicraft.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected APIError, got %v", err)
	}
}

func TestNewRejectsBlankKey(t *testing.T) {
	_, err := praxicraft.New(praxicraft.WithAPIKey("   "))
	var apiErr *praxicraft.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected APIError, got %v", err)
	}
}

func TestOrgRetrieveTyped(t *testing.T) {
	c := testClient(t, roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.Header.Get("Authorization") != "Bearer ct_live_test" {
			t.Fatalf("auth = %s", r.Header.Get("Authorization"))
		}
		return jsonResp(200, map[string]any{"name": "Acme", "plan": "starter", "invites_remaining": 10}, nil), nil
	}))
	org, err := c.Org.Retrieve()
	if err != nil {
		t.Fatal(err)
	}
	if org.Name != "Acme" || org.Plan != "starter" || org.InvitesRemaining == nil || *org.InvitesRemaining != 10 {
		t.Fatalf("%#v", org)
	}
}

func TestAuthForcedOverCustomHeaders(t *testing.T) {
	c := testClient(t, roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.Header.Get("Authorization") != "Bearer ct_live_test" {
			t.Fatalf("auth overridden: %s", r.Header.Get("Authorization"))
		}
		if !strings.HasPrefix(r.Header.Get("User-Agent"), "praxicraft-go/") {
			t.Fatalf("ua overridden: %s", r.Header.Get("User-Agent"))
		}
		return jsonResp(200, map[string]any{"ok": true}, nil), nil
	}))
	_, err := c.Do("GET", "/org/", &praxicraft.RequestOptions{
		Headers: map[string]string{
			"Authorization": "Bearer stolen",
			"User-Agent":    "evil",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestMap401(t *testing.T) {
	c := testClient(t, roundTripFunc(func(*http.Request) (*http.Response, error) {
		return jsonResp(401, map[string]any{
			"error": map[string]any{"code": "INVALID_API_KEY", "message": "bad key"},
		}, nil), nil
	}), praxicraft.WithMaxRetries(0))
	_, err := c.Org.Retrieve()
	var auth *praxicraft.AuthenticationError
	if !errors.As(err, &auth) {
		t.Fatalf("expected AuthenticationError, got %T %v", err, err)
	}
}

func TestMap403RequiredPlan(t *testing.T) {
	c := testClient(t, roundTripFunc(func(*http.Request) (*http.Response, error) {
		return jsonResp(403, map[string]any{
			"error": map[string]any{
				"code":          "PLAN_REQUIRED",
				"message":       "Starter required",
				"required_plan": "starter",
			},
		}, nil), nil
	}), praxicraft.WithMaxRetries(0))
	_, err := c.Org.Stats(nil)
	var scope *praxicraft.InsufficientScopeError
	if !errors.As(err, &scope) {
		t.Fatalf("expected InsufficientScopeError, got %T", err)
	}
	if scope.RequiredPlan != "starter" {
		t.Fatalf("required_plan = %q", scope.RequiredPlan)
	}
}

func TestMap429RetryAfterHTTPDate(t *testing.T) {
	when := time.Now().UTC().Add(3 * time.Second).Format(time.RFC1123)
	c := testClient(t, roundTripFunc(func(*http.Request) (*http.Response, error) {
		return jsonResp(429, map[string]any{
			"error": map[string]any{"code": "RATE_LIMITED", "message": "slow down"},
		}, map[string]string{"Retry-After": when}), nil
	}), praxicraft.WithMaxRetries(0))
	_, err := c.Assessments.List(nil)
	var rl *praxicraft.RateLimitError
	if !errors.As(err, &rl) {
		t.Fatalf("expected RateLimitError, got %T", err)
	}
	if rl.RetryAfter == nil || *rl.RetryAfter < 0 || *rl.RetryAfter > 5 {
		t.Fatalf("retry_after = %#v", rl.RetryAfter)
	}
}

func TestRetriesOn429ThenSucceeds(t *testing.T) {
	var calls atomic.Int32
	c := testClient(t, roundTripFunc(func(*http.Request) (*http.Response, error) {
		n := calls.Add(1)
		if n == 1 {
			return jsonResp(429, map[string]any{
				"error": map[string]any{"code": "RATE_LIMITED", "message": "slow"},
			}, map[string]string{"Retry-After": "0"}), nil
		}
		return jsonResp(200, map[string]any{"name": "Acme"}, nil), nil
	}), praxicraft.WithMaxRetries(2))
	org, err := c.Org.Retrieve()
	if err != nil {
		t.Fatal(err)
	}
	if org.Name != "Acme" || calls.Load() != 2 {
		t.Fatalf("org=%#v calls=%d", org, calls.Load())
	}
}

func TestRetriesOn503ThenFails(t *testing.T) {
	var calls atomic.Int32
	c := testClient(t, roundTripFunc(func(*http.Request) (*http.Response, error) {
		calls.Add(1)
		return jsonResp(503, map[string]any{
			"error": map[string]any{"code": "UNAVAILABLE", "message": "down"},
		}, map[string]string{"Retry-After": "0"}), nil
	}), praxicraft.WithMaxRetries(2))
	_, err := c.Org.Retrieve()
	var st *praxicraft.APIStatusError
	if !errors.As(err, &st) {
		t.Fatalf("expected APIStatusError, got %T", err)
	}
	if calls.Load() != 3 {
		t.Fatalf("calls=%d", calls.Load())
	}
}

func TestNoRetryOn400(t *testing.T) {
	var calls atomic.Int32
	c := testClient(t, roundTripFunc(func(*http.Request) (*http.Response, error) {
		calls.Add(1)
		return jsonResp(400, map[string]any{
			"error": map[string]any{"code": "VALIDATION_ERROR", "message": "bad"},
		}, nil), nil
	}), praxicraft.WithMaxRetries(3))
	_, err := c.Invites.Create("demo", praxicraft.InviteCreateParams{Email: "x@example.com"})
	var ve *praxicraft.ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected ValidationError, got %T", err)
	}
	if calls.Load() != 1 {
		t.Fatalf("calls=%d", calls.Load())
	}
}

func TestMapHTML502(t *testing.T) {
	c := testClient(t, roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: 502,
			Header:     http.Header{},
			Body:       io.NopCloser(strings.NewReader("<html>Bad Gateway</html>")),
		}, nil
	}), praxicraft.WithMaxRetries(0))
	_, err := c.Org.Retrieve()
	var st *praxicraft.APIStatusError
	if !errors.As(err, &st) {
		t.Fatalf("expected APIStatusError, got %T", err)
	}
}

func TestInviteCreateInviteToken(t *testing.T) {
	c := testClient(t, roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return jsonResp(201, map[string]any{
			"invite_token": "11111111-1111-1111-1111-111111111111",
			"status":       "pending",
		}, nil), nil
	}))
	send := true
	invite, err := c.Invites.Create("demo", praxicraft.InviteCreateParams{
		Email:     "jane@example.com",
		SendEmail: &send,
	})
	if err != nil {
		t.Fatal(err)
	}
	if invite.InviteToken == "" {
		t.Fatalf("%#v", invite)
	}
}

func TestPathEncoding(t *testing.T) {
	c := testClient(t, roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if !strings.Contains(r.URL.EscapedPath(), "/assessments/a%2Fb/") {
			t.Fatalf("path %s", r.URL.EscapedPath())
		}
		return jsonResp(200, map[string]any{"slug": "a/b"}, nil), nil
	}))
	got, err := c.Assessments.Retrieve("a/b")
	if err != nil || got.Slug != "a/b" {
		t.Fatalf("%#v %v", got, err)
	}
}

func TestClientSideValidation(t *testing.T) {
	c := testClient(t, roundTripFunc(func(*http.Request) (*http.Response, error) {
		t.Fatal("should not hit network")
		return nil, nil
	}))
	if _, err := c.Assessments.Retrieve("  "); err == nil {
		t.Fatal("expected error")
	}
	if _, err := c.Invites.Create("demo", praxicraft.InviteCreateParams{Email: "  "}); err == nil {
		t.Fatal("expected error")
	}
	if _, err := c.Webhooks.Update("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee", map[string]any{}); err == nil {
		t.Fatal("expected error")
	}
}

func TestActivateAndEnroll(t *testing.T) {
	c := testClient(t, roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if strings.Contains(r.URL.Path, "/update/") {
			return jsonResp(200, map[string]any{"status": "active", "slug": "demo"}, nil), nil
		}
		return jsonResp(201, map[string]any{
			"enrollment_id": "11111111-1111-1111-1111-111111111111",
			"status":        "in_progress",
		}, nil), nil
	}))
	a, err := c.Assessments.Activate("demo")
	if err != nil || a.Status != "active" {
		t.Fatalf("%#v %v", a, err)
	}
	en, err := c.Pipelines.Enroll("grad-2025", praxicraft.PipelineEnrollParams{Email: "alex@example.com"})
	if err != nil || en.EnrollmentID == "" {
		t.Fatalf("%#v %v", en, err)
	}
}

func TestIterAllStopsOnRepeatCursor(t *testing.T) {
	calls := 0
	c := testClient(t, roundTripFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.URL.Query().Get("cursor") == "" {
			return jsonResp(200, map[string]any{
				"next":    "https://assess.example.com/api/v1/public/assessments/demo/results/?cursor=stuck",
				"results": []any{map[string]any{"email": "a@example.com"}},
			}, nil), nil
		}
		return jsonResp(200, map[string]any{
			"next":    "https://assess.example.com/api/v1/public/assessments/demo/results/?cursor=stuck",
			"results": []any{map[string]any{"email": "b@example.com"}},
		}, nil), nil
	}))
	var rows []praxicraft.ResultRow
	err := c.Results.IterAll("demo", nil, nil, func(row praxicraft.ResultRow) error {
		rows = append(rows, row)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 || calls != 2 {
		t.Fatalf("rows=%d calls=%d", len(rows), calls)
	}
}

func TestRemoveTaskKeepsBodyIDRaw(t *testing.T) {
	c := testClient(t, roundTripFunc(func(r *http.Request) (*http.Response, error) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["assessment_task_id"] != "task/with/slash" {
			t.Fatalf("%#v", body)
		}
		return &http.Response{StatusCode: 204, Body: io.NopCloser(bytes.NewReader(nil)), Header: http.Header{}}, nil
	}))
	if err := c.Assessments.RemoveTask("demo", "task/with/slash"); err != nil {
		t.Fatal(err)
	}
}

func TestVerifySignatureNilBodySameAsEmpty(t *testing.T) {
	secret := "whsec_test"
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte{})
	digest := hex.EncodeToString(mac.Sum(nil))
	sig := "sha256=" + digest
	if !praxicraft.VerifySignature(secret, nil, sig) {
		t.Fatal("nil body should verify as empty payload")
	}
	if !praxicraft.VerifySignature(secret, []byte{}, sig) {
		t.Fatal("empty body should verify")
	}
}

func TestVerifySignature(t *testing.T) {
	secret := "whsec_test"
	body := []byte(`{"event":"webhook.test"}`)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	digest := hex.EncodeToString(mac.Sum(nil))

	if !praxicraft.VerifySignature(secret, body, "sha256="+digest) {
		t.Fatal("expected valid sha256")
	}
	if !praxicraft.VerifySignature(secret, body, digest) {
		t.Fatal("expected legacy hex")
	}
	if praxicraft.VerifySignature(secret, body, "sha256=ab") {
		t.Fatal("expected reject short sig")
	}
}

func TestConnectionError(t *testing.T) {
	c := testClient(t, roundTripFunc(func(*http.Request) (*http.Response, error) {
		return nil, errors.New("dial failed")
	}), praxicraft.WithMaxRetries(0))
	_, err := c.Org.Retrieve()
	var ce *praxicraft.APIConnectionError
	if !errors.As(err, &ce) {
		t.Fatalf("expected APIConnectionError, got %T", err)
	}
}

func TestNoContentCancel(t *testing.T) {
	c := testClient(t, roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 204, Body: io.NopCloser(bytes.NewReader(nil)), Header: http.Header{}}, nil
	}))
	if err := c.Invites.Cancel("11111111-1111-1111-1111-111111111111"); err != nil {
		t.Fatal(err)
	}
}
