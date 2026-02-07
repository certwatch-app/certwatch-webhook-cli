package internal

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const deliveryTimeout = 10 * time.Second

// SignPayload computes the HMAC-SHA256 signature of body using the provided
// secret and returns the hex-encoded digest.
func SignPayload(body, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(body))
	return hex.EncodeToString(mac.Sum(nil))
}

// DeliverPayload sends the webhook payload as a JSON POST to targetURL with
// the appropriate CertWatch webhook headers and HMAC signature. It returns a
// DeliveryResult describing the outcome.
func DeliverPayload(payload WebhookPayload, targetURL, secret string, index int) DeliveryResult {
	result := DeliveryResult{
		Index:      index,
		CommonName: payload.Data.CommonName,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		result.Error = fmt.Sprintf("failed to marshal payload: %v", err)
		return result
	}

	signature := SignPayload(string(body), secret)

	req, err := http.NewRequest(http.MethodPost, targetURL, bytes.NewReader(body))
	if err != nil {
		result.Error = fmt.Sprintf("failed to create request: %v", err)
		return result
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "CertWatch-Webhook/1.0")
	req.Header.Set("X-CertWatch-Event-Id", payload.EventID)
	req.Header.Set("X-CertWatch-Timestamp", payload.Timestamp)
	req.Header.Set("X-CertWatch-Signature", "sha256="+signature)

	client := &http.Client{Timeout: deliveryTimeout}

	start := time.Now()
	resp, err := client.Do(req)
	elapsed := time.Since(start)

	result.LatencyMs = elapsed.Milliseconds()

	if err != nil {
		result.Error = fmt.Sprintf("delivery failed: %v", err)
		return result
	}
	defer resp.Body.Close() //nolint:errcheck // response body close error is non-actionable

	result.Status = resp.StatusCode
	result.StatusText = http.StatusText(resp.StatusCode)
	result.Success = resp.StatusCode >= 200 && resp.StatusCode < 300

	if !result.Success {
		result.Error = fmt.Sprintf("received status %d %s", resp.StatusCode, result.StatusText)
	}

	return result
}
