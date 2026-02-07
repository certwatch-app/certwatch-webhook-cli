package internal

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// StreamCallbacks defines the callback functions invoked for each SSE event type.
type StreamCallbacks struct {
	OnMeta     func(meta StreamMeta)
	OnPayload  func(payload WebhookPayload)
	OnComplete func(message string)
	OnError    func(message string)
}

// ConnectStream connects to the SSE stream at streamURL and processes events
// via the provided callbacks. It blocks until the stream ends, the context is
// cancelled, or an error occurs.
func ConnectStream(ctx context.Context, streamURL, secret string, callbacks StreamCallbacks) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, streamURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create stream request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+secret)
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Connection", "keep-alive")

	// No timeout on the SSE client -- the stream is long-lived.
	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to connect to stream: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck // response body close error is non-actionable

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("stream returned status %d", resp.StatusCode)
	}

	scanner := bufio.NewScanner(resp.Body)
	// Allow up to 1 MB per SSE line to handle large payloads.
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	var currentEvent string

	for scanner.Scan() {
		// Check for context cancellation between lines.
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		line := scanner.Text()

		// SSE spec: lines starting with ":" are comments, skip them.
		if strings.HasPrefix(line, ":") {
			continue
		}

		// Empty line = end of event block.
		if line == "" {
			currentEvent = ""
			continue
		}

		if strings.HasPrefix(line, "event:") {
			currentEvent = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
			continue
		}

		if strings.HasPrefix(line, "data:") {
			data := strings.TrimPrefix(line, "data:")
			data = strings.TrimSpace(data)

			dispatchEvent(currentEvent, data, callbacks)
			continue
		}
	}

	if err := scanner.Err(); err != nil {
		// If the context was cancelled, treat it as a clean shutdown.
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		return fmt.Errorf("stream read error: %w", err)
	}

	return nil
}

// dispatchEvent routes a parsed SSE data payload to the appropriate callback
// based on the event type.
func dispatchEvent(eventType, data string, callbacks StreamCallbacks) {
	switch eventType {
	case "meta":
		if callbacks.OnMeta != nil {
			var meta StreamMeta
			if err := json.Unmarshal([]byte(data), &meta); err == nil {
				callbacks.OnMeta(meta)
			}
		}

	case "complete":
		if callbacks.OnComplete != nil {
			callbacks.OnComplete(data)
		}

	case "error":
		if callbacks.OnError != nil {
			callbacks.OnError(data)
		}

	default:
		// Default: treat as webhook payload.
		if callbacks.OnPayload != nil {
			var payload WebhookPayload
			if err := json.Unmarshal([]byte(data), &payload); err == nil {
				callbacks.OnPayload(payload)
			}
		}
	}
}
