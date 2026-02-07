/** SSE stream consumer using fetch + ReadableStream */

import type { WebhookPayload, StreamMeta } from './types.js'

export interface StreamCallbacks {
  onMeta: (meta: StreamMeta) => void
  onPayload: (payload: WebhookPayload) => void
  onComplete: (message: string) => void
  onError: (message: string) => void
}

export async function connectStream(
  streamUrl: string,
  secret: string,
  callbacks: StreamCallbacks,
  signal: AbortSignal
): Promise<void> {
  const res = await fetch(streamUrl, {
    headers: {
      Authorization: `Bearer ${secret}`,
      Accept: 'text/event-stream',
    },
    signal,
    cache: 'no-store',
  })

  if (!res.ok) {
    let message = `HTTP ${res.status}: ${res.statusText}`
    try {
      const body = await res.json()
      if (body.error?.message) message = body.error.message
    } catch {
      // Use default message
    }
    throw new Error(message)
  }

  if (!res.body) {
    throw new Error('No response body from stream endpoint')
  }

  const reader = res.body.getReader()
  const decoder = new TextDecoder()
  let buffer = ''
  let currentEvent = ''

  while (true) {
    const { done, value } = await reader.read()
    if (done) break

    buffer += decoder.decode(value, { stream: true })
    const lines = buffer.split('\n')
    buffer = lines.pop() || ''

    for (const line of lines) {
      // SSE event type
      if (line.startsWith('event: ')) {
        currentEvent = line.slice(7).trim()
        continue
      }

      if (!line.startsWith('data: ')) {
        if (line === '') currentEvent = '' // Reset on empty line
        continue
      }

      const jsonStr = line.slice(6)
      if (!jsonStr || jsonStr === '{}') continue

      try {
        const parsed = JSON.parse(jsonStr)

        if (currentEvent === 'meta') {
          callbacks.onMeta(parsed as StreamMeta)
        } else if (currentEvent === 'complete') {
          callbacks.onComplete(parsed.message || 'Stream complete')
        } else if (currentEvent === 'error') {
          callbacks.onError(parsed.message || 'Stream error')
        } else {
          // Default data event = webhook payload
          callbacks.onPayload(parsed as WebhookPayload)
        }
      } catch {
        // Skip malformed JSON
      }

      currentEvent = ''
    }
  }
}
