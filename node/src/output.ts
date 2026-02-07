/** ANSI terminal formatting for CLI output */

import type { DeliveryResult } from './types.js'

let useColor = true

export function setColor(enabled: boolean): void {
  useColor = enabled
}

// ANSI codes
const RESET = '\x1b[0m'
const BOLD = '\x1b[1m'
const DIM = '\x1b[2m'
const GREEN = '\x1b[32m'
const RED = '\x1b[31m'
const YELLOW = '\x1b[33m'
const CYAN = '\x1b[36m'
const MAGENTA = '\x1b[35m'

function c(code: string, text: string): string {
  return useColor ? `${code}${text}${RESET}` : text
}

export function printBanner(version: string, targetUrl: string, mode: string, durationSeconds: number): void {
  console.log()
  console.log(`  ${c(BOLD + CYAN, `CertWatch Webhook CLI v${version}`)}`)
  console.log(`  ${c(DIM, 'Target:')} ${targetUrl}`)
  console.log(`  ${c(DIM, 'Mode:  ')} ${mode} ${c(DIM, `· Stream: ${durationSeconds}s`)}`)
  console.log()
}

export function printConnecting(): void {
  process.stdout.write(`  ${c(DIM, 'Connecting...')} `)
}

export function printConnected(): void {
  console.log(c(GREEN, '✓ Connected'))
  console.log()
}

export function printDelivery(result: DeliveryResult): void {
  const num = `#${result.index}`.padStart(4)
  const cn = result.commonName.length > 30
    ? result.commonName.slice(0, 27) + '...'
    : result.commonName
  const cnPad = cn.padEnd(30)
  const latency = `(${result.latencyMs}ms)`

  if (result.success) {
    console.log(`  ${c(DIM, num)}  ${cnPad} ${c(DIM, '→')} ${c(GREEN, `${result.status} ${result.statusText}`)}  ${c(DIM, latency)}`)
  } else if (result.status > 0) {
    console.log(`  ${c(DIM, num)}  ${cnPad} ${c(DIM, '→')} ${c(RED, `${result.status} ${result.statusText}`)}  ${c(DIM, latency)}`)
  } else {
    console.log(`  ${c(DIM, num)}  ${cnPad} ${c(DIM, '→')} ${c(RED, 'Error')}  ${c(DIM, result.error || 'Unknown')}`)
  }
}

export function printFileSaved(index: number, commonName: string): void {
  const num = `#${index}`.padStart(4)
  const cn = commonName.length > 30
    ? commonName.slice(0, 27) + '...'
    : commonName
  const cnPad = cn.padEnd(30)
  console.log(`  ${c(DIM, num)}  ${cnPad} ${c(DIM, '→')} ${c(GREEN, 'saved')}`)
}

export function printVerbosePayload(payload: unknown): void {
  console.log(c(DIM, '  ┌─ Payload:'))
  const lines = JSON.stringify(payload, null, 2).split('\n')
  for (const line of lines) {
    console.log(c(DIM, `  │ ${line}`))
  }
  console.log(c(DIM, '  └─'))
}

export function printSummary(results: DeliveryResult[], elapsedMs: number): void {
  const total = results.length
  const succeeded = results.filter((r) => r.success).length
  const failed = total - succeeded
  const avgLatency = total > 0
    ? Math.round(results.reduce((sum, r) => sum + r.latencyMs, 0) / total)
    : 0
  const pct = total > 0 ? ((succeeded / total) * 100).toFixed(1) : '0.0'
  const elapsed = (elapsedMs / 1000).toFixed(1)

  console.log()
  console.log(`  ${c(DIM, '─── Summary ─────────────────────────')}`)
  console.log(`  ${c(DIM, 'Delivered:')} ${c(succeeded === total ? GREEN : YELLOW, `${succeeded}/${total}`)} ${c(DIM, `(${pct}%)`)}`)
  if (failed > 0) {
    console.log(`  ${c(DIM, 'Failed:   ')} ${c(RED, String(failed))}`)
  }
  console.log(`  ${c(DIM, 'Elapsed:  ')} ${elapsed}s`)
  console.log(`  ${c(DIM, 'Avg:      ')} ${avgLatency}ms`)
  console.log()
}

export function printError(message: string): void {
  console.error(`  ${c(RED, '✗')} ${message}`)
}

export function printInfo(message: string): void {
  console.log(`  ${c(MAGENTA, 'ℹ')} ${message}`)
}
