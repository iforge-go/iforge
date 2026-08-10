import type { NextConfig } from 'next'
import { readFileSync } from 'fs'

const isDev = process.env.NODE_ENV === 'development'

// Read version from package.json for build-time injection
const packageJson = JSON.parse(readFileSync('./package.json', 'utf-8'))
const APP_VERSION = packageJson.version

const nextConfig: NextConfig = {
  reactStrictMode: true,
  output: 'standalone',
  env: {
    NEXT_PUBLIC_APP_VERSION: APP_VERSION,
  },
  
  ...(isDev && {
    async rewrites() {
      return [
        {
          source: '/api/:path*',
          destination: 'http://localhost:8081/api/:path*',
        },
        {
          source: '/:owner/:repo/raw/:ref/:path*',
          destination: 'http://localhost:8081/:owner/:repo/raw/:ref/:path*',
        },
        {
          source: '/:owner/:repo/archive/:path*',
          destination: 'http://localhost:8081/:owner/:repo/archive/:path*',
        },
        {
          source: '/feeds/:path*',
          destination: 'http://localhost:8081/feeds/:path*',
        },
        {
          source: '/keys/:username.keys',
          destination: 'http://localhost:8081/keys/:username.keys',
        },
      ]
    },
  }),
}

export default nextConfig
