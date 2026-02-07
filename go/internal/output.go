package internal

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// ANSI color codes for terminal output.
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorCyan   = "\033[36m"
	colorDim    = "\033[2m"
	colorBold   = "\033[1m"
)

var useColor = true

// SetColor enables or disables ANSI color output. When disabled, all color
// functions return plain text. The NO_COLOR environment variable is also
// respected: if set (to any value), color is disabled.
func SetColor(enabled bool) {
	useColor = enabled
	if _, ok := os.LookupEnv("NO_COLOR"); ok {
		useColor = false
	}
}

func color(code, text string) string {
	if !useColor {
		return text
	}
	return code + text + colorReset
}

// PrintBanner prints the CLI startup banner with version, target URL, mode,
// and stream duration.
func PrintBanner(version, target, mode string, duration int) {
	fmt.Println()
	fmt.Printf("  %s\n", color(colorBold, "CertWatch Webhook CLI v"+version))
	fmt.Printf("  %s %s\n", color(colorDim, "Target:"), target)
	if duration > 0 {
		fmt.Printf("  %s %s%s Stream: %ds\n", color(colorDim, "Mode:  "), mode, color(colorDim, " ·"), duration)
	} else {
		fmt.Printf("  %s %s\n", color(colorDim, "Mode:  "), mode)
	}
	fmt.Println()
}

// PrintConnecting prints the "Connecting..." message without a newline.
func PrintConnecting() {
	fmt.Printf("  %s", color(colorDim, "Connecting... "))
}

// PrintConnected prints the connection success indicator.
func PrintConnected() {
	fmt.Printf("%s %s\n\n", color(colorGreen, "✓"), color(colorGreen, "Connected"))
}

// PrintDelivery prints a single delivery result line showing the index,
// common name, HTTP status, and latency.
func PrintDelivery(result DeliveryResult) {
	index := fmt.Sprintf("#%-3d", result.Index)
	cn := truncate(result.CommonName, 28)
	cn = fmt.Sprintf("%-28s", cn)

	if result.Success {
		status := fmt.Sprintf("%d %s", result.Status, result.StatusText)
		latency := fmt.Sprintf("(%dms)", result.LatencyMs)
		fmt.Printf("  %s %s %s %s  %s\n",
			color(colorDim, index),
			cn,
			color(colorDim, "->"),
			color(colorGreen, status),
			color(colorDim, latency),
		)
	} else if result.Error != "" && result.Status == 0 {
		// Network error -- no status code.
		fmt.Printf("  %s %s %s %s\n",
			color(colorDim, index),
			cn,
			color(colorDim, "->"),
			color(colorRed, "ERR "+result.Error),
		)
	} else {
		status := fmt.Sprintf("%d %s", result.Status, result.StatusText)
		latency := fmt.Sprintf("(%dms)", result.LatencyMs)
		fmt.Printf("  %s %s %s %s  %s\n",
			color(colorDim, index),
			cn,
			color(colorDim, "->"),
			color(colorRed, status),
			color(colorDim, latency),
		)
	}
}

// PrintFileSaved prints a per-payload progress line for file-only mode.
func PrintFileSaved(index int, commonName string) {
	idx := fmt.Sprintf("#%-3d", index)
	cn := truncate(commonName, 28)
	cn = fmt.Sprintf("%-28s", cn)
	fmt.Printf("  %s %s %s %s\n",
		color(colorDim, idx),
		cn,
		color(colorDim, "->"),
		color(colorGreen, "saved"),
	)
}

// PrintVerbosePayload pretty-prints a JSON payload when verbose mode is enabled.
func PrintVerbosePayload(payload interface{}) {
	data, err := json.MarshalIndent(payload, "    ", "  ")
	if err != nil {
		return
	}
	fmt.Printf("    %s\n", color(colorDim, string(data)))
}

// PrintSummary prints the final delivery summary showing success rate,
// failures, elapsed time, and average latency.
func PrintSummary(results []DeliveryResult, elapsedMs int64) {
	total := len(results)
	succeeded := 0
	var totalLatency int64

	for _, r := range results {
		if r.Success {
			succeeded++
		}
		totalLatency += r.LatencyMs
	}

	failed := total - succeeded
	elapsedSec := float64(elapsedMs) / 1000.0

	var avgMs int64
	if total > 0 {
		avgMs = totalLatency / int64(total)
	}

	var pct float64
	if total > 0 {
		pct = float64(succeeded) / float64(total) * 100.0
	}

	separator := strings.Repeat("\u2500", 36)

	fmt.Println()
	fmt.Printf("  %s\n", color(colorDim, separator))
	fmt.Printf("  %s\n", color(colorBold, "Summary"))
	fmt.Printf("  %s\n", color(colorDim, separator))

	deliveredColor := colorGreen
	if succeeded < total {
		deliveredColor = colorYellow
	}

	fmt.Printf("  %s %s\n",
		color(colorDim, "Delivered:"),
		color(deliveredColor, fmt.Sprintf("%d/%d (%.1f%%)", succeeded, total, pct)),
	)

	if failed > 0 {
		fmt.Printf("  %s %s\n",
			color(colorDim, "Failed:   "),
			color(colorRed, fmt.Sprintf("%d", failed)),
		)
	}

	fmt.Printf("  %s %s\n",
		color(colorDim, "Elapsed:  "),
		fmt.Sprintf("%.1fs", elapsedSec),
	)

	if total > 0 {
		fmt.Printf("  %s %s\n",
			color(colorDim, "Avg:      "),
			fmt.Sprintf("%dms", avgMs),
		)
	}

	fmt.Println()
}

// PrintError prints a red error message to stderr.
func PrintError(msg string) {
	fmt.Fprintf(os.Stderr, "  %s %s\n", color(colorRed, "Error:"), msg)
}

// PrintInfo prints a cyan informational message.
func PrintInfo(msg string) {
	fmt.Printf("  %s %s\n", color(colorCyan, "Info:"), msg)
}

// truncate shortens s to maxLen characters, appending "..." if truncated.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}
