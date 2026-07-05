import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { RealtimeClient, type RealtimeEnvelope } from '../websocket';

/**
 * A minimal in-memory WebSocket double. Tests drive `open`/`message` manually so
 * the client's buffering, dispatch and signaling logic can be asserted without a
 * real socket.
 */
class MockWebSocket {
  static last: MockWebSocket | null = null;
  readonly sent: string[] = [];
  private listeners: Record<string, Array<(event: unknown) => void>> = {};

  constructor(readonly url: string) {
    MockWebSocket.last = this;
  }

  addEventListener(type: string, listener: (event: unknown) => void): void {
    (this.listeners[type] ??= []).push(listener);
  }

  send(data: string): void {
    this.sent.push(data);
  }

  close(): void {
    this.emit('close', {});
  }

  emit(type: string, event: unknown): void {
    for (const listener of this.listeners[type] ?? []) listener(event);
  }

  open(): void {
    this.emit('open', {});
  }

  receive(envelope: RealtimeEnvelope): void {
    this.emit('message', { data: JSON.stringify(envelope) });
  }
}

const config = { url: 'wss://ws.test', maxReconnectAttempts: 0, reconnectBackoffMs: 1 };

beforeEach(() => {
  MockWebSocket.last = null;
  vi.stubGlobal('WebSocket', MockWebSocket as unknown as typeof WebSocket);
});

afterEach(() => {
  vi.unstubAllGlobals();
});

describe('RealtimeClient', () => {
  it('dispatches inbound frames to channel subscribers', () => {
    const client = new RealtimeClient(config);
    const received: RealtimeEnvelope[] = [];
    client.subscribe('notifications', (envelope) => received.push(envelope));

    MockWebSocket.last!.open();
    MockWebSocket.last!.receive({
      type: 'notification.created',
      channel: 'notifications',
      payload: { id: 'n1' },
    });

    expect(received).toHaveLength(1);
    expect(received[0]?.payload).toEqual({ id: 'n1' });
  });

  it('buffers sends made before the socket is open, then flushes', () => {
    const client = new RealtimeClient(config);
    client.send({ type: 'ping', channel: 'messaging', payload: 1 });

    const socket = MockWebSocket.last!;
    expect(socket.sent).toHaveLength(0); // buffered while connecting

    socket.open();
    expect(socket.sent).toHaveLength(1);
    expect(JSON.parse(socket.sent[0]!)).toMatchObject({ type: 'ping' });
  });

  it('routes WebRTC signaling frames per session', () => {
    const client = new RealtimeClient(config);
    const channel = client.openSignalingChannel('sess-1');
    const signals: unknown[] = [];
    channel.onMessage((signal) => signals.push(signal));

    MockWebSocket.last!.open();
    channel.send({ sdp: 'offer' });
    expect(JSON.parse(MockWebSocket.last!.sent[0]!)).toMatchObject({
      channel: 'signaling',
      payload: { sessionId: 'sess-1', signal: { sdp: 'offer' } },
    });

    // A frame for this session is delivered; one for another session is ignored.
    MockWebSocket.last!.receive({
      type: 'webrtc.signal',
      channel: 'signaling',
      payload: { sessionId: 'sess-1', signal: { candidate: 'a' } },
    });
    MockWebSocket.last!.receive({
      type: 'webrtc.signal',
      channel: 'signaling',
      payload: { sessionId: 'other', signal: { candidate: 'b' } },
    });

    expect(signals).toEqual([{ candidate: 'a' }]);
    channel.close();
  });
});
