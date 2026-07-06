# syntax=docker/dockerfile:1
#
# Multi-stage build for the Ancora Health Next.js frontend (S-78).
#
# Mirrors the backend image's security posture (S-76): a fat build stage that
# never ships, and a minimal, non-root distroless runtime that carries only the
# traced standalone server bundle. Next.js `output: 'standalone'` (next.config
# .mjs) is what makes this small — it copies only the node_modules actually
# reachable from the server into .next/standalone.
#
#   docker build -t ancora/web web
#
# The build is fully offline after `npm ci`; PORT (default 3000) selects the
# listen port at run time.

########################  Dependencies  ########################
# Install with the lockfile only, in its own layer, so a source-only change does
# not re-resolve the dependency graph.
FROM node:22-bookworm-slim AS deps
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci

########################  Build  ########################
# Reuse the resolved node_modules and produce the standalone server bundle.
# NEXT_TELEMETRY_DISABLED keeps the build hermetic (no outbound telemetry).
FROM node:22-bookworm-slim AS build
WORKDIR /app
ENV NEXT_TELEMETRY_DISABLED=1
COPY --from=deps /app/node_modules ./node_modules
COPY . .
RUN npm run build

########################  Runtime  ########################
# distroless nodejs — no shell, no package manager, runs as the unprivileged
# `nonroot` user. The standalone bundle already contains its own minimal
# node_modules; static assets and public files are copied alongside it exactly
# where the traced server.js expects them.
FROM gcr.io/distroless/nodejs22-debian12:nonroot AS web
WORKDIR /app
ENV NODE_ENV=production \
    NEXT_TELEMETRY_DISABLED=1 \
    PORT=3000 \
    HOSTNAME=0.0.0.0
COPY --from=build /app/.next/standalone ./
COPY --from=build /app/.next/static ./.next/static
COPY --from=build /app/public ./public
USER nonroot:nonroot
EXPOSE 3000
# distroless nodejs images set node as the entrypoint, so the CMD is the script.
CMD ["server.js"]
