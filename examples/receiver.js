#!/usr/bin/env node

/**
 * CertWatch Webhook Receiver — Example Server
 *
 * A minimal HTTP server that receives webhook payloads from the CertWatch CLI,
 * verifies HMAC-SHA256 signatures, and pretty-prints the results.
 *
 * Usage:
 *   node receiver.js --secret <secret> [--port <port>]
 *
 * Example:
 *   # Terminal 1 — start the receiver
 *   node examples/receiver.js --secret abc123 --port 3000
 *
 *   # Terminal 2 — send payloads via CLI
 *   npx certwatch-webhook-cli --secret abc123 --url http://localhost:3000/webhook
 *
 * Zero dependencies — uses only Node.js built-ins.
 */

const http = require('node:http')
const crypto = require('node:crypto')
const { parseArgs } = require('node:util')

// ---------------------------------------------------------------------------
// Config
// ---------------------------------------------------------------------------

let secret, port

try {
  const { values } = parseArgs({
    options: {
      secret: { type: 'string' },
      port: { type: 'string', default: '3000' },
      help: { type: 'boolean', default: false },
    },
    strict: true,
  })

  if (values.help || !values.secret) {
    console.error('Usage: node receiver.js --secret <secret> [--port <port>]')
    console.error('')
    console.error('  --secret <secret>  The same secret passed to the CLI (--secret)')
    console.error('  --port <port>      Port to listen on (default: 3000)')
    process.exit(values.help ? 0 : 1)
  }

  secret = values.secret
  port = parseInt(values.port, 10)
} catch (err) {
  console.error(err.message)
  console.error('Usage: node receiver.js --secret <secret> [--port <port>]')
  process.exit(1)
}

// ---------------------------------------------------------------------------
// HMAC verification
// ---------------------------------------------------------------------------

function verifySignature(body, signatureHeader, secret) {
  if (!signatureHeader || !signatureHeader.startsWith('sha256=')) return false

  const provided = signatureHeader.slice(7) // strip "sha256="
  const expected = crypto.createHmac('sha256', secret).update(body).digest('hex')

  // Constant-time comparison to prevent timing attacks
  try {
    return crypto.timingSafeEqual(
      Buffer.from(expected, 'hex'),
      Buffer.from(provided, 'hex')
    )
  } catch {
    return false
  }
}

// ---------------------------------------------------------------------------
// Pretty-print helpers
// ---------------------------------------------------------------------------

const green = (s) => `\x1b[32m${s}\x1b[0m`
const red = (s) => `\x1b[31m${s}\x1b[0m`
const dim = (s) => `\x1b[2m${s}\x1b[0m`
const bold = (s) => `\x1b[1m${s}\x1b[0m`
const cyan = (s) => `\x1b[36m${s}\x1b[0m`

let count = 0

function printPayload(body, headers, verified) {
  count++
  const payload = JSON.parse(body)
  const cn = payload.data?.common_name || 'unknown'
  const domains = payload.data?.domains?.length || 0
  const issuer = payload.data?.issuer_cn || 'unknown'
  const status = verified ? green('VERIFIED') : red('FAILED')

  console.log('')
  console.log(`  ${bold(`#${count}`)}  ${cyan(cn)}`)
  console.log(`      Domains: ${domains}  Issuer: ${issuer}`)
  console.log(`      Event:   ${payload.event_id}`)
  console.log(`      HMAC:    ${status}`)
  console.log(`      ${dim(new Date().toISOString())}`)
}

// ---------------------------------------------------------------------------
// HTTP Server
// ---------------------------------------------------------------------------

const server = http.createServer((req, res) => {
  // Health check
  if (req.method === 'GET' && req.url === '/health') {
    res.writeHead(200, { 'Content-Type': 'application/json' })
    res.end(JSON.stringify({ status: 'ok', received: count }))
    return
  }

  // Only accept POST to /webhook
  if (req.method !== 'POST' || req.url !== '/webhook') {
    res.writeHead(404, { 'Content-Type': 'application/json' })
    res.end(JSON.stringify({ error: 'POST /webhook only' }))
    return
  }

  const chunks = []
  req.on('data', (chunk) => chunks.push(chunk))
  req.on('end', () => {
    const body = Buffer.concat(chunks).toString()
    const signature = req.headers['x-certwatch-signature']
    const verified = verifySignature(body, signature, secret)

    printPayload(body, req.headers, verified)

    res.writeHead(200, { 'Content-Type': 'application/json' })
    res.end(JSON.stringify({ received: true, verified }))
  })
})

server.listen(port, () => {
  console.log('')
  console.log(`  ${bold('CertWatch Webhook Receiver')}`)
  console.log('')
  console.log(`  Listening: ${cyan(`http://localhost:${port}/webhook`)}`)
  console.log(`  Health:    http://localhost:${port}/health`)
  console.log(`  Secret:    ${secret.slice(0, 8)}...`)
  console.log('')
  console.log(dim('  Waiting for payloads...'))
})
