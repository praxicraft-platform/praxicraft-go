package praxicraft

import (
	"fmt"
	"net/url"
	"strings"
)

func pathSegment(value, label string) (string, error) {
	text := strings.TrimSpace(value)
	if text == "" {
		if label == "" {
			label = "id"
		}
		return "", &APIError{Message: fmt.Sprintf("%s must be a non-empty string", label), ErrCode: "INVALID_PATH"}
	}
	return url.PathEscape(text), nil
}
