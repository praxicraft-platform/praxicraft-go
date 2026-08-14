package praxicraft

import (
	"math"
	"math/rand"
	"strconv"
	"strings"
	"time"
)

const (
	defaultMaxRetries = 2
	defaultRetryBase  = 500 * time.Millisecond
	defaultRetryCap   = 8 * time.Second
)

func shouldRetryStatus(statusCode int) bool {
	switch statusCode {
	case 429, 500, 502, 503, 504:
		return true
	default:
		return false
	}
}

// parseRetryAfterSeconds parses Retry-After as delay-seconds or HTTP-date.
// Returns seconds until retry (>= 0), or nil if missing/unparseable.
func parseRetryAfterSeconds(value string) *float64 {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	if n, err := strconv.ParseFloat(value, 64); err == nil {
		if n < 0 {
			n = 0
		}
		return &n
	}
	if t, err := httpDate(value); err == nil {
		sec := time.Until(t).Seconds()
		if sec < 0 {
			sec = 0
		}
		return &sec
	}
	return nil
}

func httpDate(value string) (time.Time, error) {
	layouts := []string{
		time.RFC1123,
		time.RFC1123Z,
		time.RFC850,
		time.ANSIC,
	}
	var last error
	for _, layout := range layouts {
		t, err := time.Parse(layout, value)
		if err == nil {
			return t, nil
		}
		last = err
	}
	return time.Time{}, last
}

func retryDelay(attempt int, retryAfterHeader string) time.Duration {
	if secs := parseRetryAfterSeconds(retryAfterHeader); secs != nil {
		d := time.Duration(*secs * float64(time.Second))
		if d > defaultRetryCap {
			return defaultRetryCap
		}
		return d
	}
	// Exponential backoff with full jitter: random in [0, min(cap, base*2^attempt)]
	exp := math.Pow(2, float64(attempt))
	ceiling := time.Duration(float64(defaultRetryBase) * exp)
	if ceiling > defaultRetryCap {
		ceiling = defaultRetryCap
	}
	if ceiling <= 0 {
		return defaultRetryBase
	}
	return time.Duration(rand.Int63n(int64(ceiling) + 1))
}

// sleepFn is overridable in tests.
var sleepFn = time.Sleep
