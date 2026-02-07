package internal

// CliOptions holds the parsed command-line flags for the webhook CLI.
type CliOptions struct {
	URL         string
	Secret      string
	APIKey      string
	File        string // Path to JSONL output file.
	Raw         bool   // Print NDJSON to stdout (pipe-friendly).
	Preview     bool   // Show a sample payload and exit.
	Verbose     bool
	NoColor     bool
	APIEndpoint string
}

// SessionResponse is the JSON envelope returned by the session creation API.
type SessionResponse struct {
	Success bool          `json:"success"`
	Data    *SessionData  `json:"data,omitempty"`
	Error   *SessionError `json:"error,omitempty"`
}

// SessionData contains the session details for a successful session creation.
type SessionData struct {
	TestID                string `json:"testId"`
	Secret                string `json:"secret"`
	StreamURL             string `json:"streamUrl"`
	ExpiresInSeconds      int    `json:"expiresInSeconds"`
	StreamDurationSeconds int    `json:"streamDurationSeconds"`
}

// SessionError contains error details when session creation fails.
type SessionError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// WebhookPayload represents a single CT certificate event delivered via the stream.
type WebhookPayload struct {
	Event      string      `json:"event"`
	EventID    string      `json:"event_id"`
	Timestamp  string      `json:"timestamp"`
	APIVersion string      `json:"api_version"`
	Data       PayloadData `json:"data"`
}

// PayloadData contains the certificate details within a webhook payload.
type PayloadData struct {
	Fingerprint  string   `json:"fingerprint"`
	SerialNumber string   `json:"serial_number"`
	CommonName   string   `json:"common_name"`
	Domains      []string `json:"domains"`
	IssuerOrg    string   `json:"issuer_org"`
	IssuerCN     string   `json:"issuer_cn"`
	NotBefore    string   `json:"not_before"`
	NotAfter     string   `json:"not_after"`
	CTLogSources []string `json:"ct_log_sources"`
	SeenAt       string   `json:"seen_at"`
}

// StreamMeta contains metadata about the SSE stream session.
type StreamMeta struct {
	TestID                string `json:"testId"`
	StreamDurationSeconds int    `json:"streamDurationSeconds"`
}

// DeliveryResult records the outcome of delivering a single webhook payload
// to the user's local endpoint.
type DeliveryResult struct {
	Index      int
	CommonName string
	Status     int
	StatusText string
	LatencyMs  int64
	Success    bool
	Error      string
}
