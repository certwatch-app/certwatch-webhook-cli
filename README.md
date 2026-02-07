# CertWatch Webhook CLI

Deliver real Certificate Transparency webhook payloads to your local endpoint for testing.

Works with [CertWatch's free webhook tester](https://certwatch.app/tools/webhook-tester) — generate a secret on the web page, then run the CLI to stream live CT certificate data as formatted webhook payloads to your endpoint.

## Quick Start

### Preview (no setup needed)

```bash
npx certwatch-webhook-cli --preview
```

### Deliver to your endpoint

```bash
npx certwatch-webhook-cli --secret <secret> --url http://localhost:3000/webhook
```

### Go (pre-built binary)

```bash
# macOS / Linux (Homebrew)
brew install certwatch-app/tap/certwatch-webhook-cli

# Or download from GitHub Releases:
# https://github.com/certwatch-app/certwatch-webhook-cli/releases
certwatch-webhook-cli -secret <secret> -url http://localhost:3000/webhook
```

## Usage

### Preview mode (instant, offline)

See exactly what a webhook delivery looks like — no secret or session required:

```bash
# Node.js
npx certwatch-webhook-cli --preview

# Preview with your real secret to test HMAC verification
npx certwatch-webhook-cli --preview --secret abc123...

# Go
certwatch-webhook-cli -preview
certwatch-webhook-cli -preview -secret abc123...
```

### Deliver to your endpoint

1. Visit [certwatch.app/tools/webhook-tester](https://certwatch.app/tools/webhook-tester)
2. Click "Generate Secret" to get a test secret
3. Run the CLI:

```bash
# Node.js
npx certwatch-webhook-cli --secret abc123... --url http://localhost:3000/webhook

# Go
certwatch-webhook-cli -secret abc123... -url http://localhost:3000/webhook
```

### Save payloads to a file

Save received payloads as JSONL for test fixtures or replay:

```bash
npx certwatch-webhook-cli --secret abc123... --file payloads.jsonl
```

### Pipe raw JSON to another tool

Output NDJSON to stdout (suppresses all decorative output):

```bash
npx certwatch-webhook-cli --secret abc123... --raw | jq '.data.common_name'
```

### Combine output modes

Output modes are combinable — deliver, save, and pipe simultaneously:

```bash
npx certwatch-webhook-cli --secret abc123... \
  --url http://localhost:3000/webhook \
  --file payloads.jsonl
```

### API key mode (auto-creates session)

If you have a CertWatch account, use your API key for higher limits:

```bash
# Node.js
npx certwatch-webhook-cli --api-key cw_xxx_yyy --url http://localhost:3000/webhook

# Go
certwatch-webhook-cli -api-key cw_xxx_yyy -url http://localhost:3000/webhook
```

## Options

### Output modes (at least one required, combinable)

| Flag | Description |
|------|-------------|
| `--url` / `-url` | Deliver payloads via HTTP POST to this URL |
| `--file` / `-file` | Save payloads to a JSONL file (one JSON per line) |
| `--raw` / `-raw` | Print raw NDJSON to stdout (pipe-friendly) |
| `--preview` / `-preview` | Show a sample payload and exit (no session needed) |

### Authentication (required for --url, --file, --raw)

| Flag | Description |
|------|-------------|
| `--secret` / `-secret` | Test secret from web page (anonymous mode) |
| `--api-key` / `-api-key` | CertWatch API key (auto-creates session) |

### Other options

| Flag | Description | Default |
|------|-------------|---------|
| `--verbose` / `-verbose` | Print full JSON payload per delivery | `false` |
| `--no-color` / `-no-color` | Disable ANSI colors (also: `NO_COLOR` env) | `false` |
| `--api-endpoint` / `-api-endpoint` | Override API base URL | `https://api.certwatch.app` |
| `--version` / `-version` | Print version | |

## Rate Limits

| Tier | Sessions/hour | Stream duration |
|------|--------------|-----------------|
| Anonymous (no account) | 1 | 60 seconds |
| Signed-up (any account) | 5 | 90 seconds |

## Webhook Payload Format

Each delivery POSTs a JSON payload matching CertWatch's production webhook format:

```json
{
  "event": "ct.certificate.new",
  "event_id": "evt_550e8400-e29b-41d4-a716-446655440000",
  "timestamp": "2026-02-07T12:00:00.000Z",
  "api_version": "2024-01-01",
  "data": {
    "fingerprint": "sha256:abc123...",
    "serial_number": "01:AB:CD:...",
    "common_name": "*.example.com",
    "domains": ["*.example.com", "example.com"],
    "issuer_org": "Let's Encrypt",
    "issuer_cn": "R3",
    "not_before": "2026-02-07T00:00:00.000Z",
    "not_after": "2026-05-08T00:00:00.000Z",
    "ct_log_sources": ["Google Argon 2026"],
    "seen_at": "2026-02-07T12:00:00.000Z"
  }
}
```

### Headers

| Header | Description |
|--------|-------------|
| `Content-Type` | `application/json` |
| `User-Agent` | `CertWatch-Webhook/1.0` |
| `X-CertWatch-Event-Id` | Unique event ID |
| `X-CertWatch-Timestamp` | ISO 8601 timestamp |
| `X-CertWatch-Signature` | `sha256={hmac_hex}` |

### Verifying Signatures

The `X-CertWatch-Signature` header contains an HMAC-SHA256 signature of the raw JSON body, signed with your test secret. Verify it in your endpoint:

```javascript
const crypto = require('crypto');

function verifySignature(body, secret, signatureHeader) {
  const expected = crypto
    .createHmac('sha256', secret)
    .update(body)
    .digest('hex');
  return signatureHeader === `sha256=${expected}`;
}
```

## Example Receiver Server

Don't have a webhook endpoint yet? Use our example receiver to get started. It listens for payloads, verifies HMAC signatures, and pretty-prints the results.

```bash
# Terminal 1 — start the receiver (Node.js)
node examples/receiver.js --secret <secret> --port 3000

# Terminal 1 — start the receiver (Go)
go run examples/receiver.go -secret <secret> -port 3000

# Terminal 2 — send payloads via CLI
npx certwatch-webhook-cli --secret <secret> --url http://localhost:3000/webhook
```

Output:

```
  CertWatch Webhook Receiver

  Listening: http://localhost:3000/webhook
  Health:    http://localhost:3000/health
  Secret:    abc12345...

  Waiting for payloads...

  #1  *.example.com
      Domains: 2  Issuer: R11
      Event:   evt_550e8400-e29b-41d4-a716-446655440000
      HMAC:    VERIFIED
      2026-02-07T14:00:00Z
```

The examples are standalone files with zero dependencies — use them as a starting point for your own webhook handler.

## Development

```bash
# Node.js
cd node && npm install && npm run typecheck && npm run build

# Go
cd go && make build && make test
```

## License

MIT
