/** Shared interfaces for the CertWatch Webhook CLI */

export interface CliOptions {
  url?: string
  secret?: string
  apiKey?: string
  file?: string
  raw: boolean
  preview: boolean
  verbose: boolean
  noColor: boolean
  apiEndpoint: string
}

export interface SessionResponse {
  success: boolean
  data?: {
    testId: string
    secret: string
    streamUrl: string
    expiresInSeconds: number
    streamDurationSeconds: number
  }
  error?: {
    code: string
    message: string
  }
}

export interface WebhookPayload {
  event: string
  event_id: string
  timestamp: string
  api_version: string
  data: {
    fingerprint: string
    serial_number: string
    common_name: string
    domains: string[]
    issuer_org: string
    issuer_cn: string
    not_before: string
    not_after: string
    ct_log_sources: string[]
    seen_at: string
  }
}

export interface StreamMeta {
  testId: string
  streamDurationSeconds: number
}

export interface DeliveryResult {
  index: number
  commonName: string
  status: number
  statusText: string
  latencyMs: number
  success: boolean
  error?: string
}
