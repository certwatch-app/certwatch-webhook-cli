/** Preview mode: generate a realistic sample payload locally */

import { randomBytes, randomUUID } from 'node:crypto'
import { signPayload } from './sender.js'
import type { WebhookPayload } from './types.js'

/** Generate a realistic-looking sample webhook payload */
export function generateSamplePayload(): WebhookPayload {
  const now = new Date()
  const notAfter = new Date(now.getTime() + 90 * 24 * 60 * 60 * 1000) // +90 days

  return {
    event: 'ct.certificate.new',
    event_id: `evt_${randomUUID()}`,
    timestamp: now.toISOString(),
    api_version: '2024-01-01',
    data: {
      fingerprint: `sha256:${randomBytes(32).toString('hex')}`,
      serial_number: Array.from(randomBytes(16))
        .map((b) => b.toString(16).padStart(2, '0').toUpperCase())
        .join(':'),
      common_name: '*.example.com',
      domains: ['*.example.com', 'example.com'],
      issuer_org: "Let's Encrypt",
      issuer_cn: 'R11',
      not_before: now.toISOString(),
      not_after: notAfter.toISOString(),
      ct_log_sources: ['Google Argon 2026'],
      seen_at: now.toISOString(),
    },
  }
}

/** Format and print a full preview of what a webhook delivery looks like */
export function printPreview(secret: string, version: string): void {
  const payload = generateSamplePayload()
  const body = JSON.stringify(payload, null, 2)
  const bodyCompact = JSON.stringify(payload)
  const signature = signPayload(bodyCompact, secret)

  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    'User-Agent': 'CertWatch-Webhook/1.0',
    'X-CertWatch-Event-Id': payload.event_id,
    'X-CertWatch-Timestamp': payload.timestamp,
    'X-CertWatch-Signature': `sha256=${signature}`,
  }

  console.log()
  console.log(`  CertWatch Webhook CLI v${version} — Preview`)
  console.log()
  console.log('  This is what your endpoint will receive:')
  console.log()
  console.log('  ┌─ POST Request ──────────────────────────────────────')
  console.log('  │')
  console.log('  │  Headers:')
  for (const [key, value] of Object.entries(headers)) {
    console.log(`  │    ${key}: ${value}`)
  }
  console.log('  │')
  console.log('  │  Body:')
  for (const line of body.split('\n')) {
    console.log(`  │    ${line}`)
  }
  console.log('  │')
  console.log('  └─────────────────────────────────────────────────────')
  console.log()
  console.log(`  Signing secret: ${secret}`)
  console.log()
  console.log('  Verify the signature in your endpoint:')
  console.log('    HMAC-SHA256(JSON.stringify(body), secret) === signature')
  console.log()
}
