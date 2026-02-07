package internal

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// GenerateSamplePayload returns a realistic-looking sample WebhookPayload
// using crypto/rand for all random values.
func GenerateSamplePayload() WebhookPayload {
	now := time.Now().UTC()
	notAfter := now.Add(90 * 24 * time.Hour)

	return WebhookPayload{
		Event:      "ct.certificate.new",
		EventID:    "evt_" + generateUUIDv4(),
		Timestamp:  now.Format(time.RFC3339),
		APIVersion: "2024-01-01",
		Data: PayloadData{
			Fingerprint:  "sha256:" + randomHex(32),
			SerialNumber: randomSerialNumber(16),
			CommonName:   "*.example.com",
			Domains:      []string{"*.example.com", "example.com"},
			IssuerOrg:    "Let's Encrypt",
			IssuerCN:     "R11",
			NotBefore:    now.Format(time.RFC3339),
			NotAfter:     notAfter.Format(time.RFC3339),
			CTLogSources: []string{"Google Argon 2026"},
			SeenAt:       now.Format(time.RFC3339),
		},
	}
}

// PrintPreview renders a boxed preview of a sample POST request including
// headers, JSON body, and HMAC-SHA256 signature computed from the secret.
func PrintPreview(secret, version string) {
	payload := GenerateSamplePayload()

	body, err := json.MarshalIndent(payload, "  ", "  ")
	if err != nil {
		PrintError(fmt.Sprintf("failed to marshal sample payload: %v", err))
		return
	}

	signature := SignPayload(string(body), secret)

	fmt.Println()
	fmt.Printf("  %s\n", color(colorBold, "CertWatch Webhook CLI v"+version)+" "+color(colorDim, "-- Preview"))
	fmt.Println()
	fmt.Printf("  %s\n", "This is what your endpoint will receive:")
	fmt.Println()

	// Box top.
	boxWidth := 55
	fmt.Printf("  %s\n", color(colorDim, "\u250c\u2500 POST Request "+strings.Repeat("\u2500", boxWidth-15)))
	fmt.Printf("  %s\n", color(colorDim, "\u2502"))

	// Headers.
	fmt.Printf("  %s  %s\n", color(colorDim, "\u2502"), color(colorBold, "Headers:"))
	printBoxLine("Content-Type: application/json")
	printBoxLine("User-Agent: CertWatch-Webhook/1.0")
	printBoxLine("X-CertWatch-Event-Id: " + payload.EventID)
	printBoxLine("X-CertWatch-Timestamp: " + payload.Timestamp)
	printBoxLine("X-CertWatch-Signature: sha256=" + signature)

	fmt.Printf("  %s\n", color(colorDim, "\u2502"))

	// Body.
	fmt.Printf("  %s  %s\n", color(colorDim, "\u2502"), color(colorBold, "Body:"))
	for _, line := range strings.Split(string(body), "\n") {
		fmt.Printf("  %s    %s\n", color(colorDim, "\u2502"), line)
	}

	fmt.Printf("  %s\n", color(colorDim, "\u2502"))

	// Box bottom.
	fmt.Printf("  %s\n", color(colorDim, "\u2514"+strings.Repeat("\u2500", boxWidth)))

	fmt.Println()
	fmt.Printf("  %s %s\n", color(colorDim, "Signing secret:"), secret)
	fmt.Println()
	fmt.Printf("  %s\n", "Verify the signature in your endpoint:")
	fmt.Printf("    %s\n", color(colorCyan, "HMAC-SHA256(JSON.stringify(body), secret) === signature"))
	fmt.Println()
}

// printBoxLine prints a single indented line inside the box border.
func printBoxLine(text string) {
	fmt.Printf("  %s    %s\n", color(colorDim, "\u2502"), text)
}

// generateUUIDv4 generates a random UUID v4 string using crypto/rand.
func generateUUIDv4() string {
	var uuid [16]byte
	if _, err := rand.Read(uuid[:]); err != nil {
		// Fallback to zeros on read failure; should never happen.
		return "00000000-0000-4000-8000-000000000000"
	}
	// Set version 4 bits.
	uuid[6] = (uuid[6] & 0x0f) | 0x40
	// Set variant bits (RFC 4122).
	uuid[8] = (uuid[8] & 0x3f) | 0x80

	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		uuid[0:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:16])
}

// randomHex generates n random bytes and returns the hex-encoded string
// using crypto/rand.
func randomHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return strings.Repeat("00", n)
	}
	return hex.EncodeToString(b)
}

// randomSerialNumber generates n random bytes and returns them as an
// uppercase colon-separated hex string (e.g., "AB:CD:EF:...").
func randomSerialNumber(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return strings.Repeat("00:", n-1) + "00"
	}

	parts := make([]string, n)
	for i, v := range b {
		parts[i] = fmt.Sprintf("%02X", v)
	}
	return strings.Join(parts, ":")
}
