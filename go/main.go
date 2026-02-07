package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/certwatch-app/certwatch-webhook-cli/internal"
)

// version is set at build time via ldflags.
var version = "dev"

func main() {
	url := flag.String("url", "", "Target URL to deliver webhook payloads to")
	secret := flag.String("secret", "", "Webhook signing secret (for direct secret mode)")
	apiKey := flag.String("api-key", "", "CertWatch API key (creates a test session automatically)")
	file := flag.String("file", "", "Save payloads to a JSONL file (one JSON per line)")
	raw := flag.Bool("raw", false, "Print raw NDJSON to stdout (pipe-friendly)")
	preview := flag.Bool("preview", false, "Show a sample payload and exit (no session needed)")
	verbose := flag.Bool("verbose", false, "Print full JSON payload for each delivery")
	noColor := flag.Bool("no-color", false, "Disable colored output")
	apiEndpoint := flag.String("api-endpoint", "https://api.certwatch.app", "CertWatch API endpoint")
	showVersion := flag.Bool("version", false, "Print version and exit")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "CertWatch Webhook CLI v%s\n\n", version)
		fmt.Fprintf(os.Stderr, "Connects to a CertWatch SSE stream and delivers real CT certificate\n")
		fmt.Fprintf(os.Stderr, "webhook payloads to your local endpoint for testing.\n\n")
		fmt.Fprintf(os.Stderr, "Usage:\n")
		fmt.Fprintf(os.Stderr, "  certwatch-webhook-cli -url <target> -api-key <key>\n")
		fmt.Fprintf(os.Stderr, "  certwatch-webhook-cli -url <target> -secret <secret>\n")
		fmt.Fprintf(os.Stderr, "  certwatch-webhook-cli -file payloads.jsonl -secret <secret>\n")
		fmt.Fprintf(os.Stderr, "  certwatch-webhook-cli -raw -secret <secret> | jq .\n")
		fmt.Fprintf(os.Stderr, "  certwatch-webhook-cli -preview\n")
		fmt.Fprintf(os.Stderr, "  certwatch-webhook-cli -url <target> -file out.jsonl -secret <secret>\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	if *showVersion {
		fmt.Printf("certwatch-webhook-cli v%s\n", version)
		os.Exit(0)
	}

	// --preview mode: skip all validation, just show sample and exit.
	if *preview {
		opts := internal.CliOptions{
			Secret:  *secret,
			Preview: true,
			NoColor: *noColor,
		}
		if err := internal.Run(opts, version); err != nil {
			internal.PrintError(err.Error())
			os.Exit(1)
		}
		os.Exit(0)
	}

	// For stream modes, require at least one output target.
	if *url == "" && *file == "" && !*raw {
		fmt.Fprintln(os.Stderr, "Error: at least one of -url, -file, or -raw is required")
		fmt.Fprintln(os.Stderr)
		flag.Usage()
		os.Exit(1)
	}

	// Require authentication for stream modes.
	if *apiKey == "" && *secret == "" {
		fmt.Fprintln(os.Stderr, "Error: either -api-key or -secret is required")
		fmt.Fprintln(os.Stderr)
		flag.Usage()
		os.Exit(1)
	}

	opts := internal.CliOptions{
		URL:         *url,
		Secret:      *secret,
		APIKey:      *apiKey,
		File:        *file,
		Raw:         *raw,
		Preview:     *preview,
		Verbose:     *verbose,
		NoColor:     *noColor,
		APIEndpoint: *apiEndpoint,
	}

	if err := internal.Run(opts, version); err != nil {
		internal.PrintError(err.Error())
		os.Exit(1)
	}
}
