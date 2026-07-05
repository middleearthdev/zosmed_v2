import type { NextConfig } from 'next';
import { API_BASE } from './lib/env';

const nextConfig: NextConfig = {
  // Workspace packages ship TS/TSX source; let Next transpile them.
  transpilePackages: ['@zosmed/ui', '@zosmed/types', '@zosmed/kits'],

  // Proxy `/api/*` and `/connect/*` through the Next server so the browser
  // talks same-origin to `apps/api`. This keeps the `zsid` session cookie
  // first-party (`SameSite=Lax` is enough) instead of relying on cross-site
  // CORS + `SameSite=None` (ADR-003 §0 keputusan #5, AC-12).
  async rewrites() {
    return [
      { source: '/api/:path*', destination: `${API_BASE}/api/:path*` },
      { source: '/connect/:path*', destination: `${API_BASE}/connect/:path*` },
    ];
  },
};

export default nextConfig;
