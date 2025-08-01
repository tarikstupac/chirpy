package auth

import (
	"fmt"
	"net/http"
	"strings"
)

func GetAPIKey(headers http.Header) (string, error) {
	apiKeyHeader := headers.Get("Authorization")
	if apiKeyHeader == "" {
		return "", fmt.Errorf("missing API key in headers")
	}
	apiKey := strings.TrimPrefix(apiKeyHeader, "ApiKey ")
	if apiKey == "" {
		return "", fmt.Errorf("API key is empty")
	}
	return apiKey, nil
}
