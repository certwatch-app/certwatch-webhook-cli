package internal

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const sessionPath = "/api/v1/tools/webhook-test/session"

// CreateSession creates a new webhook test session by calling the CertWatch API.
// It returns the session response containing the stream URL, secret, and duration,
// or an error if the request fails. If userSecret is non-empty, it is sent to the
// backend so the session uses the caller's signing secret instead of a random one.
func CreateSession(ctx context.Context, apiEndpoint, apiKey, userSecret string) (*SessionResponse, error) {
	url := apiEndpoint + sessionPath

	var bodyReader *bytes.Reader
	if userSecret != "" {
		payload, _ := json.Marshal(map[string]string{"secret": userSecret})
		bodyReader = bytes.NewReader(payload)
	} else {
		bodyReader = bytes.NewReader(nil)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create session request: %w", err)
	}

	req.Header.Set("X-API-Key", apiKey)
	req.Header.Set("Accept", "application/json")
	if userSecret != "" {
		req.Header.Set("Content-Type", "application/json")
	}

	client := &http.Client{Timeout: 15 * time.Second}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck // response body close error is non-actionable

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		var sessResp SessionResponse
		if decErr := json.NewDecoder(resp.Body).Decode(&sessResp); decErr == nil && sessResp.Error != nil {
			return nil, fmt.Errorf("session creation failed (%d): %s - %s",
				resp.StatusCode, sessResp.Error.Code, sessResp.Error.Message)
		}
		return nil, fmt.Errorf("session creation failed with status %d", resp.StatusCode)
	}

	var sessResp SessionResponse
	if err := json.NewDecoder(resp.Body).Decode(&sessResp); err != nil {
		return nil, fmt.Errorf("failed to decode session response: %w", err)
	}

	if !sessResp.Success || sessResp.Data == nil {
		if sessResp.Error != nil {
			return nil, fmt.Errorf("session creation failed: %s - %s",
				sessResp.Error.Code, sessResp.Error.Message)
		}
		return nil, fmt.Errorf("session creation returned unsuccessful response")
	}

	return &sessResp, nil
}
