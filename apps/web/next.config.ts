import type { NextConfig } from 'next';

const nextConfig: NextConfig = {
  // Workspace packages ship TS/TSX source; let Next transpile them.
  transpilePackages: ['@zosmed/ui', '@zosmed/types', '@zosmed/kits'],
};

export default nextConfig;
