import { defineConfig } from '@hey-api/openapi-ts'

export default defineConfig({
  input: '../../contracts/openapi/aeon-echoes.v1.yaml',
  output: './lib/generated/api'
})
