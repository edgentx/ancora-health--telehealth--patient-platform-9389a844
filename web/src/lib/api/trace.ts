/**
 * W3C Trace Context + OpenTelemetry propagation.
 *
 * Every outbound request carries a fresh `traceparent` so the edge (Kong) and
 * the backend services can stitch the client-originated span into the
 * distributed trace. We emit the W3C standard headers that OpenTelemetry's
 * propagators understand out of the box — no vendor SDK is pulled into the
 * bundle; we only need to *originate* correctly-shaped identifiers.
 *
 * @see https://www.w3.org/TR/trace-context/
 */

import { publicEnv } from '@/lib/env';

/** Identifiers for a single client span, plus the ready-to-send headers. */
export interface TraceContext {
  /** 32-hex-char trace id, stable for the whole request (incl. retries). */
  traceId: string;
  /** 16-hex-char span id, unique per attempt. */
  spanId: string;
  /** Rendered W3C/OTel headers to merge into the request. */
  headers: Record<string, string>;
}

/** Fill `bytes` with cryptographically-strong random values where available. */
function randomBytes(length: number): Uint8Array {
  const out = new Uint8Array(length);
  const cryptoObj = globalThis.crypto;
  if (cryptoObj?.getRandomValues) {
    cryptoObj.getRandomValues(out);
    return out;
  }
  // Last-resort fallback for exotic runtimes without WebCrypto. Trace ids are
  // observability metadata, never a security boundary, so a weaker source here
  // only risks a collision in a trace explorer — acceptable and vanishingly rare.
  for (let i = 0; i < length; i += 1) {
    out[i] = Math.floor(Math.random() * 256);
  }
  return out;
}

function toHex(bytes: Uint8Array): string {
  let hex = '';
  for (const byte of bytes) {
    hex += byte.toString(16).padStart(2, '0');
  }
  return hex;
}

/**
 * Mint a new trace context. Pass an existing `traceId` when retrying so all
 * attempts of one logical request share a trace but each gets its own span id.
 */
export function newTraceContext(traceId?: string): TraceContext {
  const resolvedTraceId = traceId ?? toHex(randomBytes(16));
  const spanId = toHex(randomBytes(8));

  // `-01` trace-flags marks the span as sampled; the edge may downgrade it.
  const traceparent = `00-${resolvedTraceId}-${spanId}-01`;

  return {
    traceId: resolvedTraceId,
    spanId,
    headers: {
      traceparent,
      // OTel baggage lets the backend attribute the originating surface without
      // a separate correlation lookup.
      baggage: `service.name=${publicEnv.otelServiceName}`,
      // A human-greppable correlation id mirrored into logs on both sides.
      'x-request-id': resolvedTraceId,
    },
  };
}
