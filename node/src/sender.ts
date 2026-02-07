/** HMAC-SHA256 signing and HTTP delivery of webhook payloads */

import { createHmac } from 'node:crypto'
import type { WebhookPayload, DeliveryResult } from './types.js'

export function signPayload(body: string, secret: string): string {
  return createHmac('sha256', secret).update(body).digest('hex')
}

export async function deliverPayload(
  payload: WebhookPayload,
  targetUrl: string,
  secret: string,
  index: number
): Promise<DeliveryResult> {
  const body = JSON.stringify(payload)
  const signature = signPayload(body, secret)
  const start = performance.now()

  try {
    const res = await fetch(targetUrl, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'User-Agent': 'CertWatch-Webhook/1.0',
        'X-CertWatch-Event-Id': payload.event_id,
        'X-CertWatch-Timestamp': payload.timestamp,
        'X-CertWatch-Signature': `sha256=${signature}`,
      },
      body,
      signal: AbortSignal.timeout(10_000),
    })

    const latencyMs = Math.round(performance.now() - start)

    return {
      index,
      commonName: payload.data.common_name || '(empty)',
      status: res.status,
      statusText: res.statusText,
      latencyMs,
      success: res.status >= 200 && res.status < 300,
    }
  } catch (err) {
    const latencyMs = Math.round(performance.now() - start)
    const message = err instanceof Error ? err.message : 'Unknown error'

    return {
      index,
      commonName: payload.data.common_name || '(empty)',
      status: 0,
      statusText: 'Error',
      latencyMs,
      success: false,
      error: message,
    }
  }
}
