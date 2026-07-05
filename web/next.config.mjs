/** @type {import('next').NextConfig} */
const nextConfig = {
  reactStrictMode: true,
  // Emit a self-contained server bundle (.next/standalone) so the container
  // image (web/Dockerfile) ships only the traced runtime deps on a distroless
  // node base rather than the whole node_modules tree.
  output: 'standalone',
  // Type and lint errors fail the production build — the pipeline stays honest.
  typescript: { ignoreBuildErrors: false },
  eslint: { ignoreDuringBuilds: false },
};

export default nextConfig;
