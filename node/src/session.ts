/** Create a webhook test session via the CertWatch API */

import type { SessionResponse } from './types.js'

export async function createSession(
  apiEndpoint: string,
  apiKey: string,
  secret?: string
): Promise<SessionResponse> {
  const url = `${apiEndpoint}/api/v1/tools/webhook-test/session`

  const headers: Record<string, string> = {
    'X-API-Key': apiKey,
    Accept: 'application/json',
  }

  let body: string | undefined
  if (secret) {
    headers['Content-Type'] = 'application/json'
    body = JSON.stringify({ secret })
  }

  const res = await fetch(url, {
    method: 'POST',
    headers,
    body,
  })

  const json = await res.json()

  if (!res.ok) {
    return {
      success: false,
      error: json.error || { code: 'HTTP_ERROR', message: `HTTP ${res.status}: ${res.statusText}` },
    }
  }

  return json as SessionResponse
}
