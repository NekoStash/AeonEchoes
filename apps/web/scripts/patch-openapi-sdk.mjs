import { accessSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import { dirname, resolve } from 'node:path'

const root = resolve(dirname(fileURLToPath(import.meta.url)), '..')
for (const file of [
  'lib/generated/api/client.gen.ts',
  'lib/generated/api/sdk.gen.ts',
  'lib/generated/api/types.gen.ts',
  'lib/generated/api/index.ts'
]) {
  accessSync(resolve(root, file))
}
