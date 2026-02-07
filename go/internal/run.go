package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"
)

// Run is the main orchestration function for the webhook CLI. It creates a
// session (if using an API key), connects to the SSE stream, delivers each
// payload to the target URL, and prints a summary at the end.
//
// It also supports --preview (show sample and exit), --file (append JSONL),
// and --raw (NDJSON to stdout). These modes are combinable with --url.
func Run(opts CliOptions, version string) error {
	SetColor(!opts.NoColor)

	// --preview mode: generate a sample payload, print it, and exit.
	if opts.Preview {
		secret := opts.Secret
		userProvidedSecret := secret != ""
		if secret == "" {
			secret = randomHex(32)
		}
		PrintPreview(secret, version)
		if !userProvidedSecret {
			fmt.Printf("  %s\n\n", color(colorDim, "Tip: pass -secret <your-secret> to preview with your real HMAC key"))
		}
		return nil
	}

	secret := opts.Secret
	streamURL := ""
	streamDuration := 0
	mode := ""

	// Set up cancellable context for graceful shutdown on SIGINT/SIGTERM.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if opts.APIKey != "" {
		// API key mode: create a session to get stream URL and secret.
		mode = "API key"

		if !opts.Raw {
			PrintConnecting()
		}

		sess, err := CreateSession(ctx, opts.APIEndpoint, opts.APIKey, opts.Secret)
		if err != nil {
			if !opts.Raw {
				fmt.Println() // newline after "Connecting..."
			}
			return fmt.Errorf("failed to create session: %w", err)
		}

		secret = sess.Data.Secret
		streamURL = sess.Data.StreamURL
		streamDuration = sess.Data.StreamDurationSeconds

		if !opts.Raw {
			PrintInfo("Signing secret: " + secret)
			printStreamBanner(version, opts, mode, streamDuration)
			PrintConnecting()
			PrintConnected()
		}
	} else {
		// Direct secret mode: construct stream URL from API endpoint.
		mode = "Secret"
		streamURL = opts.APIEndpoint + "/api/v1/tools/webhook-test/stream?secret=" + opts.Secret

		if !opts.Raw {
			printStreamBanner(version, opts, mode, streamDuration)
			PrintConnecting()
		}
	}

	// Open JSONL file for appending if --file is set.
	var outFile *os.File
	if opts.File != "" {
		var err error
		outFile, err = os.OpenFile(opts.File, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return fmt.Errorf("failed to open output file %s: %w", opts.File, err)
		}
		defer outFile.Close() //nolint:errcheck // file close on exit is non-actionable
	}

	var (
		mu           sync.Mutex
		results      []DeliveryResult
		index        int
		filePayloads int
	)
	startTime := time.Now()

	callbacks := StreamCallbacks{
		OnMeta: func(meta StreamMeta) {
			streamDuration = meta.StreamDurationSeconds
		},

		OnPayload: func(payload WebhookPayload) {
			mu.Lock()
			index++
			currentIndex := index
			mu.Unlock()

			// --raw: write NDJSON to stdout.
			if opts.Raw {
				line, err := json.Marshal(payload)
				if err == nil {
					fmt.Fprintln(os.Stdout, string(line))
				}
			}

			// --file: append JSONL to file.
			if outFile != nil {
				line, err := json.Marshal(payload)
				if err == nil {
					mu.Lock()
					_, _ = fmt.Fprintln(outFile, string(line))
					filePayloads++
					mu.Unlock()
				}
			}

			// --url: deliver via HTTP.
			if opts.URL != "" {
				result := DeliverPayload(payload, opts.URL, secret, currentIndex)

				if !opts.Raw {
					PrintDelivery(result)
				}

				if opts.Verbose && !opts.Raw {
					PrintVerbosePayload(payload)
				}

				mu.Lock()
				results = append(results, result)
				mu.Unlock()
			} else if outFile != nil && !opts.Raw {
				// File-only mode — show progress per payload.
				PrintFileSaved(currentIndex, payload.Data.CommonName)
			}
		},

		OnComplete: func(message string) {
			if !opts.Raw {
				PrintInfo("Stream complete: " + message)
			}
		},

		OnError: func(message string) {
			if !opts.Raw {
				PrintError("Stream error: " + message)
			}
		},
	}

	// If we're in API key mode, the "Connected" was already printed.
	// In secret mode we need to print it once the stream connects.
	if opts.APIKey == "" && !opts.Raw {
		// Wrap OnMeta to print Connected on the first meta event.
		originalOnMeta := callbacks.OnMeta
		connectedOnce := sync.Once{}
		callbacks.OnMeta = func(meta StreamMeta) {
			connectedOnce.Do(func() {
				PrintConnected()
			})
			if originalOnMeta != nil {
				originalOnMeta(meta)
			}
		}

		// Also detect connection through first payload if no meta arrives.
		originalOnPayload := callbacks.OnPayload
		callbacks.OnPayload = func(payload WebhookPayload) {
			connectedOnce.Do(func() {
				PrintConnected()
			})
			if originalOnPayload != nil {
				originalOnPayload(payload)
			}
		}
	}

	err := ConnectStream(ctx, streamURL, secret, callbacks)

	elapsedMs := time.Since(startTime).Milliseconds()

	mu.Lock()
	finalResults := make([]DeliveryResult, len(results))
	copy(finalResults, results)
	finalFilePayloads := filePayloads
	mu.Unlock()

	// Print file save summary.
	if opts.File != "" && !opts.Raw {
		PrintInfo(fmt.Sprintf("Saved %d payloads to %s", finalFilePayloads, opts.File))
	}

	// Print delivery summary (only if we have URL deliveries and not in raw mode).
	if !opts.Raw && opts.URL != "" {
		PrintSummary(finalResults, elapsedMs)
	}

	// If the context was cancelled (SIGINT/SIGTERM), don't treat it as an error
	// if we already have results or file output.
	hasOutput := len(finalResults) > 0 || finalFilePayloads > 0
	if err != nil && ctx.Err() != nil && hasOutput {
		if !opts.Raw {
			PrintInfo("Interrupted by signal")
		}
		return checkForFailures(finalResults)
	}

	if err != nil && ctx.Err() == nil {
		return fmt.Errorf("stream error: %w", err)
	}

	return checkForFailures(finalResults)
}

// printStreamBanner prints the CLI banner with combined output targets.
func printStreamBanner(version string, opts CliOptions, mode string, duration int) {
	var targets []string
	if opts.URL != "" {
		targets = append(targets, opts.URL)
	}
	if opts.File != "" {
		targets = append(targets, "file: "+opts.File)
	}
	if opts.Raw {
		targets = append(targets, "stdout")
	}

	target := strings.Join(targets, " + ")
	PrintBanner(version, target, mode, duration)
}

// checkForFailures returns an error if any deliveries failed, suitable for
// setting a non-zero exit code.
func checkForFailures(results []DeliveryResult) error {
	for _, r := range results {
		if !r.Success {
			return fmt.Errorf("some deliveries failed")
		}
	}
	return nil
}
