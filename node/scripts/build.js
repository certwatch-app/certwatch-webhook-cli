/**
 * Pre-build script: generates version.ts from package.json
 */
import { readFileSync, writeFileSync } from 'node:fs'
import { resolve, dirname } from 'node:path'
import { fileURLToPath } from 'node:url'

const __dirname = dirname(fileURLToPath(import.meta.url))
const pkg = JSON.parse(readFileSync(resolve(__dirname, '..', 'package.json'), 'utf8'))
const versionFile = resolve(__dirname, '..', 'src', 'version.ts')
writeFileSync(versionFile, `export const VERSION = '${pkg.version}'\n`, 'utf8')
console.log(`Generated version.ts with v${pkg.version}`)
