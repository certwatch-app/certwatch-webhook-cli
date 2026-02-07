// CertWatch Webhook Receiver — Example Server
//
// A minimal HTTP server that receives webhook payloads from the CertWatch CLI,
// verifies HMAC-SHA256 signatures, and pretty-prints the results.
//
// Usage:
//
//	go run receiver.go -secret <secret> [-port <port>]
//
// Example:
//
//	# Terminal 1 — start the receiver
//	go run examples/receiver.go -secret abc123 -port 3000
//
//	# Terminal 2 — send payloads via CLI
//	certwatch-webhook-cli -secret abc123 -url http://localhost:3000/webhook
//
// Zero dependencies — uses only Go standard library.
package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"time"
)

// ---------------------------------------------------------------------------
// HMAC verification
// ---------------------------------------------------------------------------

func verifySignature(body []byte, signatureHeader, secret string) bool {
	if !strings.HasPrefix(signatureHeader, "sha256=") {
		return false
	}
	provided := signatureHeader[7:] // strip "sha256="

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(expected), []byte(provided))
}

// ---------------------------------------------------------------------------
// Pretty-print helpers
// ---------------------------------------------------------------------------

func green(s string) string { return "\033[32m" + s + "\033[0m" }
func red(s string) string   { return "\033[31m" + s + "\033[0m" }
func dim(s string) string   { return "\033[2m" + s + "\033[0m" }
func bold(s string) string  { return "\033[1m" + s + "\033[0m" }
func cyan(s string) string  { return "\033[36m" + s + "\033[0m" }

// ---------------------------------------------------------------------------
// Payload types (minimal subset for display)
// ---------------------------------------------------------------------------

type webhookPayload struct {
	Event   string `json:"event"`
	EventID string `json:"event_id"`
	Data    struct {
		CommonName string   `json:"common_name"`
		Domains    []string `json:"domains"`
		IssuerCN   string   `json:"issuer_cn"`
	} `json:"data"`
}

// ---------------------------------------------------------------------------
// Server
// ---------------------------------------------------------------------------

func main() {
	secretFlag := flag.String("secret", "", "The same secret passed to the CLI (-secret)")
	portFlag := flag.String("port", "3000", "Port to listen on (default: 3000)")
	flag.Parse()

	if *secretFlag == "" {
		fmt.Fprintln(os.Stderr, "Usage: go run receiver.go -secret <secret> [-port <port>]")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "  -secret <secret>  The same secret passed to the CLI (-secret)")
		fmt.Fprintln(os.Stderr, "  -port <port>      Port to listen on (default: 3000)")
		os.Exit(1)
	}

	secret := *secretFlag
	port := *portFlag

	var count atomic.Int64

	// Health check
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"status":"ok","received":%d}`, count.Load())
	})

	// Webhook endpoint
	http.HandleFunc("/webhook", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			fmt.Fprintf(w, `{"error":"POST only"}`)
			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, `{"error":"failed to read body"}`)
			return
		}
		defer r.Body.Close()

		sig := r.Header.Get("X-CertWatch-Signature")
		verified := verifySignature(body, sig, secret)

		var payload webhookPayload
		json.Unmarshal(body, &payload)

		n := count.Add(1)

		status := green("VERIFIED")
		if !verified {
			status = red("FAILED")
		}

		cn := payload.Data.CommonName
		if cn == "" {
			cn = "unknown"
		}
		issuer := payload.Data.IssuerCN
		if issuer == "" {
			issuer = "unknown"
		}

		fmt.Println()
		fmt.Printf("  %s  %s\n", bold(fmt.Sprintf("#%d", n)), cyan(cn))
		fmt.Printf("      Domains: %d  Issuer: %s\n", len(payload.Data.Domains), issuer)
		fmt.Printf("      Event:   %s\n", payload.EventID)
		fmt.Printf("      HMAC:    %s\n", status)
		fmt.Printf("      %s\n", dim(time.Now().Format(time.RFC3339)))

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"received":true,"verified":%t}`, verified)
	})

	fmt.Println()
	fmt.Printf("  %s\n", bold("CertWatch Webhook Receiver"))
	fmt.Println()
	fmt.Printf("  Listening: %s\n", cyan(fmt.Sprintf("http://localhost:%s/webhook", port)))
	fmt.Printf("  Health:    http://localhost:%s/health\n", port)
	fmt.Printf("  Secret:    %s...\n", secret[:min(8, len(secret))])
	fmt.Println()
	fmt.Println(dim("  Waiting for payloads..."))

	if err := http.ListenAndServe(":"+port, nil); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
