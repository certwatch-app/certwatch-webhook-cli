#!/usr/bin/env node

/**
 * CertWatch Webhook CLI
 *
 * Connects to a CertWatch SSE stream and delivers real CT certificate
 * webhook payloads to your local endpoint for testing.
 */

import { randomBytes } from 'node:crypto'
import { createWriteStream } from 'node:fs'
import type { WriteStream } from 'node:fs'
import { parseArgs } from 'node:util'
import { createSession } from './session.js'
import { connectStream } from './stream.js'
import { deliverPayload } from './sender.js'
import { printPreview } from './preview.js'
import {
  setColor,
  printBanner,
  printConnecting,
  printConnected,
  printDelivery,
  printFileSaved,
  printVerbosePayload,
  printSummary,
  printError,
  printInfo,
} from './output.js'
import type { CliOptions, DeliveryResult } from './types.js'
import { VERSION } from './version.js'

function printUsage(): void {
  console.log(`
  CertWatch Webhook CLI v${VERSION}

  Deliver real CT certificate webhook payloads to your local endpoint.

  Usage:
    certwatch-webhook-cli --url <url> --secret <secret>
    certwatch-webhook-cli --url <url> --api-key <key>
    certwatch-webhook-cli --raw --secret <secret>
    certwatch-webhook-cli --file payloads.jsonl --secret <secret>
    certwatch-webhook-cli --preview [--secret <secret>]

  Output modes (at least one required, combinable):
    --url <url>           Deliver payloads via HTTP POST to this URL
    --file <path>         Save payloads to a JSONL file (one JSON per line)
    --raw                 Print raw NDJSON to stdout (pipe-friendly)
    --preview             Show a sample payload and exit (no session needed)

  Authentication (required for --url, --file, --raw):
    --secret <secret>     Test secret from web page (anonymous mode)
    --api-key <key>       CertWatch API key (auto-creates session)

  Options:
    --verbose             Print full JSON payload per delivery
    --no-color            Disable ANSI colors (also: NO_COLOR env)
    --api-endpoint <url>  Override API base (default: https://api.certwatch.app)
    --version             Print version
    --help                Print this help

  Examples:
    # Preview the webhook format (no secret needed)
    npx certwatch-webhook-cli --preview

    # Preview with your secret to test HMAC verification
    npx certwatch-webhook-cli --preview --secret abc123...

    # Deliver to your endpoint
    npx certwatch-webhook-cli --secret abc123... --url http://localhost:3000/webhook

    # Save payloads to a file for test fixtures
    npx certwatch-webhook-cli --secret abc123... --file payloads.jsonl

    # Pipe raw JSON to another tool
    npx certwatch-webhook-cli --secret abc123... --raw | jq '.data.common_name'

    # Combine: deliver + save
    npx certwatch-webhook-cli --secret abc123... --url http://localhost:3000/webhook --file payloads.jsonl

  Documentation: https://github.com/certwatch-app/certwatch-webhook-cli
`)
}

function parseCliArgs(): CliOptions | null {
  try {
    const { values } = parseArgs({
      options: {
        url: { type: 'string' },
        secret: { type: 'string' },
        'api-key': { type: 'string' },
        file: { type: 'string' },
        raw: { type: 'boolean', default: false },
        preview: { type: 'boolean', default: false },
        verbose: { type: 'boolean', default: false },
        'no-color': { type: 'boolean', default: false },
        'api-endpoint': { type: 'string', default: 'https://api.certwatch.app' },
        version: { type: 'boolean', default: false },
        help: { type: 'boolean', default: false },
      },
      strict: true,
    })

    if (values.version) {
      console.log(VERSION)
      process.exit(0)
    }

    if (values.help) {
      printUsage()
      process.exit(0)
    }

    return {
      url: values.url,
      secret: values.secret,
      apiKey: values['api-key'],
      file: values.file,
      raw: values.raw ?? false,
      preview: values.preview ?? false,
      verbose: values.verbose ?? false,
      noColor: values['no-color'] ?? false,
      apiEndpoint: values['api-endpoint'] || 'https://api.certwatch.app',
    }
  } catch (err) {
    printError((err as Error).message)
    printUsage()
    return null
  }
}

async function run(): Promise<void> {
  const opts = parseCliArgs()
  if (!opts) process.exit(1)

  // Respect NO_COLOR env
  if (opts.noColor || process.env.NO_COLOR !== undefined) {
    setColor(false)
  }

  // ── Preview mode ────────────────────────────────────────────────────
  if (opts.preview) {
    const secret = opts.secret || randomBytes(32).toString('hex')
    printPreview(secret, VERSION)
    if (!opts.secret) {
      printInfo('Tip: pass --secret <your-secret> to preview with your real HMAC key')
    }
    process.exit(0)
  }

  // ── Stream mode — validate args ─────────────────────────────────────
  const hasOutput = opts.url || opts.file || opts.raw
  if (!hasOutput) {
    printError('At least one output mode is required: --url, --file, --raw, or --preview')
    process.exit(1)
  }

  if (!opts.secret && !opts.apiKey) {
    printError('Either --secret or --api-key is required (or use --preview)')
    process.exit(1)
  }

  // Validate URL if provided
  if (opts.url) {
    try {
      const parsed = new URL(opts.url)
      if (!['http:', 'https:'].includes(parsed.protocol)) {
        throw new Error('URL must use http or https')
      }
    } catch {
      printError(`Invalid URL: ${opts.url}`)
      process.exit(1)
    }
  }

  let secret: string
  let streamUrl: string
  let streamDurationSeconds = 60
  let mode: string

  // Resolve session
  if (opts.apiKey) {
    mode = 'API key'
    printInfo('Creating session via API key...')

    const result = await createSession(opts.apiEndpoint, opts.apiKey, opts.secret)
    if (!result.success || !result.data) {
      printError(result.error?.message || 'Failed to create session')
      process.exit(1)
    }

    secret = result.data.secret
    streamUrl = result.data.streamUrl
    streamDurationSeconds = result.data.streamDurationSeconds || 60
    printInfo(`Session created: ${result.data.testId}`)
    printInfo(`Signing secret: ${secret}`)
  } else {
    mode = 'Secret (anonymous)'
    secret = opts.secret!
    streamUrl = `${opts.apiEndpoint}/api/v1/tools/webhook-test/stream`
  }

  // Build output description
  const outputs: string[] = []
  if (opts.url) outputs.push(opts.url)
  if (opts.file) outputs.push(`file: ${opts.file}`)
  if (opts.raw) outputs.push('stdout')
  const targetDisplay = outputs.join(' + ')

  printBanner(VERSION, targetDisplay, mode, streamDurationSeconds)

  // Open file stream if needed
  let fileStream: WriteStream | null = null
  if (opts.file) {
    fileStream = createWriteStream(opts.file, { flags: 'a' })
    printInfo(`Writing payloads to ${opts.file}`)
  }

  // Connect and relay
  const results: DeliveryResult[] = []
  let deliveryIndex = 0
  const startTime = performance.now()
  const abortController = new AbortController()

  // Handle Ctrl+C gracefully
  const onSignal = () => {
    abortController.abort()
  }
  process.on('SIGINT', onSignal)
  process.on('SIGTERM', onSignal)

  if (!opts.raw) printConnecting()

  try {
    await connectStream(
      streamUrl,
      secret,
      {
        onMeta: (meta) => {
          streamDurationSeconds = meta.streamDurationSeconds
          if (!opts.raw) printConnected()
        },
        onPayload: async (payload) => {
          deliveryIndex++

          // Raw NDJSON to stdout
          if (opts.raw) {
            process.stdout.write(JSON.stringify(payload) + '\n')
          }

          // Append to file
          if (fileStream) {
            fileStream.write(JSON.stringify(payload) + '\n')
          }

          // Deliver via HTTP
          if (opts.url) {
            const result = await deliverPayload(payload, opts.url, secret, deliveryIndex)
            results.push(result)
            if (!opts.raw) printDelivery(result)
          } else if (fileStream && !opts.raw) {
            // File-only mode — show progress per payload
            printFileSaved(deliveryIndex, payload.data?.common_name || 'unknown')
          }

          if (opts.verbose && !opts.raw) {
            printVerbosePayload(payload)
          }
        },
        onComplete: () => {
          // Stream ended normally
        },
        onError: (message) => {
          if (!opts.raw) printError(`Stream error: ${message}`)
        },
      },
      abortController.signal
    )
  } catch (err) {
    if ((err as Error).name === 'AbortError') {
      // Ctrl+C or timeout — expected
    } else {
      printError((err as Error).message)
      process.exit(1)
    }
  }

  // Cleanup
  process.off('SIGINT', onSignal)
  process.off('SIGTERM', onSignal)
  if (fileStream) {
    fileStream.end()
    if (!opts.raw) printInfo(`Saved ${deliveryIndex} payloads to ${opts.file}`)
  }

  const elapsedMs = performance.now() - startTime

  if (!opts.raw) {
    if (opts.url) {
      printSummary(results, elapsedMs)
    } else {
      // No URL delivery — just print count
      console.log()
      console.log(`  Received ${deliveryIndex} payloads in ${(elapsedMs / 1000).toFixed(1)}s`)
      console.log()
    }
  }

  // Exit code: 0 if no URL mode or all succeeded, 1 if any delivery failures
  const hasFailures = results.some((r) => !r.success)
  process.exit(hasFailures ? 1 : 0)
}

run().catch((err) => {
  printError(err.message || 'Unexpected error')
  process.exit(1)
})
