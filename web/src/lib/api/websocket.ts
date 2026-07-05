/**
 * Realtime WebSocket client.
 *
 * A single socket to `ws.{project}.vforce360.ai` (via {@link publicEnv.wsBaseUrl})
 * multiplexes every realtime concern: secure-messaging deliveries, notification
 * pushes, and WebRTC signaling for video visits. Messages are framed as a small
 * typed envelope `{ type, channel, payload }`; consumers subscribe per channel
 * rather than sharing one firehose handler.
 *
 * The client owns connection lifecycle — lazy connect, heartbeat-free reconnect
 * with exponential backoff, and buffered sends while offline — so hooks and
 * components treat realtime as "always available" and never touch the raw
 * `WebSocket`. It is browser-only; on the server `connect()` is a no-op.
 */

import { publicEnv } from '@/lib/env';

/** Logical streams multiplexed over the one socket. */
export type RealtimeChannel = 'messaging' | 'notifications' | 'signaling';

/** The wire envelope for every realtime frame. */
export interface RealtimeEnvelope<T = unknown> {
  type: string;
  channel: RealtimeChannel;
  payload: T;
}

/** Connection lifecycle states surfaced to `onStatus` subscribers. */
export type ConnectionStatus = 'idle' | 'connecting' | 'open' | 'closed';

type Listener<T> = (value: T) => void;

/** Tuning knobs; defaults suit the deployed edge. */
export interface RealtimeConfig {
  url: string;
  /** Max reconnect attempts before giving up (0 disables reconnect). */
  maxReconnectAttempts: number;
  /** Base reconnect backoff in ms; grows exponentially with jitter. */
  reconnectBackoffMs: number;
}

export function defaultRealtimeConfig(): RealtimeConfig {
  return {
    url: publicEnv.wsBaseUrl,
    maxReconnectAttempts: 6,
    reconnectBackoffMs: 500,
  };
}

/**
 * A scoped view over the shared socket for one WebRTC session. Each video visit
 * opens its own signaling channel keyed by `sessionId`; frames are tagged so
 * concurrent visits never cross-talk.
 */
export interface SignalingChannel {
  sessionId: string;
  /** Send an SDP offer/answer or ICE candidate to the peer via the edge. */
  send(payload: unknown): void;
  /** Subscribe to inbound signaling frames for this session only. */
  onMessage(listener: Listener<unknown>): () => void;
  /** Stop listening and release the session subscription. */
  close(): void;
}

export class RealtimeClient {
  private socket: WebSocket | null = null;
  private status: ConnectionStatus = 'idle';
  private reconnectAttempts = 0;
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null;
  private closedByCaller = false;

  /** Frames queued while the socket is not yet open. */
  private outbound: string[] = [];

  private readonly channelListeners = new Map<
    RealtimeChannel,
    Set<Listener<RealtimeEnvelope>>
  >();
  private readonly statusListeners = new Set<Listener<ConnectionStatus>>();

  constructor(private readonly config: RealtimeConfig = defaultRealtimeConfig()) {}

  /** Open the socket. Idempotent; a no-op on the server (no `WebSocket`). */
  connect(): void {
    if (typeof WebSocket === 'undefined') return; // SSR / non-browser runtime.
    if (this.socket && this.status !== 'closed') return;

    this.closedByCaller = false;
    this.setStatus('connecting');
    const socket = new WebSocket(this.config.url);
    this.socket = socket;

    socket.addEventListener('open', () => {
      this.reconnectAttempts = 0;
      this.setStatus('open');
      this.flushOutbound();
    });
    socket.addEventListener('message', (event) => this.dispatch(event.data));
    socket.addEventListener('close', () => this.handleClose());
    socket.addEventListener('error', () => {
      // Errors are followed by a close event; reconnect is handled there.
      socket.close();
    });
  }

  /** Close the socket for good; cancels any pending reconnect. */
  disconnect(): void {
    this.closedByCaller = true;
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer);
      this.reconnectTimer = null;
    }
    this.socket?.close();
    this.socket = null;
    this.setStatus('closed');
  }

  /** Publish a frame; buffered and sent on (re)connect if not yet open. */
  send<T>(envelope: RealtimeEnvelope<T>): void {
    const frame = JSON.stringify(envelope);
    if (this.socket && this.status === 'open') {
      this.socket.send(frame);
    } else {
      this.outbound.push(frame);
      this.connect();
    }
  }

  /** Subscribe to all frames on a channel. Returns an unsubscribe function. */
  subscribe<T>(
    channel: RealtimeChannel,
    listener: Listener<RealtimeEnvelope<T>>,
  ): () => void {
    let set = this.channelListeners.get(channel);
    if (!set) {
      set = new Set();
      this.channelListeners.set(channel, set);
    }
    set.add(listener as Listener<RealtimeEnvelope>);
    this.connect();
    return () => {
      set?.delete(listener as Listener<RealtimeEnvelope>);
    };
  }

  /** Observe connection status transitions (for reconnect banners, etc.). */
  onStatus(listener: Listener<ConnectionStatus>): () => void {
    this.statusListeners.add(listener);
    listener(this.status);
    return () => {
      this.statusListeners.delete(listener);
    };
  }

  get connectionStatus(): ConnectionStatus {
    return this.status;
  }

  /**
   * Open a WebRTC signaling channel for one video session. SDP and ICE frames
   * are tagged with `sessionId` so the edge routes them to the right peer and
   * concurrent visits stay isolated.
   */
  openSignalingChannel(sessionId: string): SignalingChannel {
    const listeners = new Set<Listener<unknown>>();

    const unsubscribe = this.subscribe<{ sessionId: string; signal: unknown }>(
      'signaling',
      (envelope) => {
        if (envelope.payload?.sessionId === sessionId) {
          for (const listener of listeners) listener(envelope.payload.signal);
        }
      },
    );

    return {
      sessionId,
      send: (payload) =>
        this.send({
          type: 'webrtc.signal',
          channel: 'signaling',
          payload: { sessionId, signal: payload },
        }),
      onMessage: (listener) => {
        listeners.add(listener);
        return () => listeners.delete(listener);
      },
      close: () => {
        listeners.clear();
        unsubscribe();
      },
    };
  }

  private dispatch(raw: unknown): void {
    if (typeof raw !== 'string') return;
    let envelope: RealtimeEnvelope;
    try {
      envelope = JSON.parse(raw) as RealtimeEnvelope;
    } catch {
      return; // Ignore frames we cannot parse rather than tear down the socket.
    }
    const set = this.channelListeners.get(envelope.channel);
    if (!set) return;
    for (const listener of set) listener(envelope);
  }

  private handleClose(): void {
    this.socket = null;
    if (this.closedByCaller) {
      this.setStatus('closed');
      return;
    }
    this.setStatus('closed');
    this.scheduleReconnect();
  }

  private scheduleReconnect(): void {
    if (this.reconnectAttempts >= this.config.maxReconnectAttempts) return;
    const attempt = this.reconnectAttempts;
    this.reconnectAttempts += 1;
    const ceiling = this.config.reconnectBackoffMs * 2 ** attempt;
    const delay = Math.round(Math.random() * ceiling);
    this.reconnectTimer = setTimeout(() => this.connect(), delay);
  }

  private flushOutbound(): void {
    if (!this.socket) return;
    const pending = this.outbound;
    this.outbound = [];
    for (const frame of pending) this.socket.send(frame);
  }

  private setStatus(status: ConnectionStatus): void {
    this.status = status;
    for (const listener of this.statusListeners) listener(status);
  }
}
