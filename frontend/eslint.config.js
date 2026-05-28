import { defineConfig, globalIgnores } from 'eslint/config'
import nextVitals from 'eslint-config-next/core-web-vitals'
import nextTs from 'eslint-config-next/typescript'

export default defineConfig([
  ...nextVitals,
  ...nextTs,
  globalIgnores([
    '.next/**',
    'out/**',
    'dist/**',
    'e2e/**',
    'src/App.tsx',
    'src/chat/**',
    'src/components/**',
    'src/hooks/**',
    'src/lib/**',
    'src/main.tsx',
    'src/legacy-pages/**',
    'src/types/**',
    'vite.config.ts',
    'playwright.config.ts',
  ]),
])
