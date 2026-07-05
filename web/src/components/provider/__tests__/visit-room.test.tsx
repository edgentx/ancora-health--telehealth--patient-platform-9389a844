import { cleanup, fireEvent, screen, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { apiClient } from '@/lib/api';
import { renderWithClient } from '@/test/render';

import { VisitRoom } from '../visit-room';

/**
 * In-memory WebSocket double (mirrors the S-80 websocket unit test) so the
 * signaling channel can be driven without a real socket. Tests open it manually
 * to flush buffered signaling frames.
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
}

function fakeTrack(kind: string) {
  return { kind, enabled: true, stop: vi.fn() };
}

function stubMedia(getUserMedia: () => Promise<unknown>) {
  Object.defineProperty(navigator, 'mediaDevices', {
    configurable: true,
    value: { getUserMedia: vi.fn(getUserMedia) },
  });
}

beforeEach(() => {
  MockWebSocket.last = null;
  vi.stubGlobal('WebSocket', MockWebSocket as unknown as typeof WebSocket);
});

afterEach(() => {
  cleanup();
  // Reset the shared realtime singleton so the next test gets a fresh socket.
  apiClient.realtime.disconnect();
  vi.unstubAllGlobals();
});

describe('VisitRoom', () => {
  it('joins the visit: acquires camera/mic and announces over signaling', async () => {
    const video = fakeTrack('video');
    const audio = fakeTrack('audio');
    stubMedia(() =>
      Promise.resolve({
        getTracks: () => [video, audio],
        getVideoTracks: () => [video],
        getAudioTracks: () => [audio],
      }),
    );

    renderWithClient(<VisitRoom sessionId="appt-1" />);

    fireEvent.click(screen.getByRole('button', { name: /join visit/i }));

    // Media acquired → live controls appear.
    expect(await screen.findByRole('button', { name: /leave/i })).toBeTruthy();
    expect(navigator.mediaDevices.getUserMedia).toHaveBeenCalledWith({
      video: true,
      audio: true,
    });

    // Flush the buffered signaling frame and assert it is scoped to this session.
    const socket = MockWebSocket.last!;
    socket.open();
    await waitFor(() => expect(socket.sent.length).toBeGreaterThan(0));
    const frame = JSON.parse(socket.sent[0]!);
    expect(frame).toMatchObject({
      channel: 'signaling',
      payload: { sessionId: 'appt-1' },
    });

    // The realtime status banner reflects the open connection.
    expect(await screen.findByText(/connected/i)).toBeTruthy();
  });

  it('mic and camera toggles flip the local track state', async () => {
    const video = fakeTrack('video');
    const audio = fakeTrack('audio');
    stubMedia(() =>
      Promise.resolve({
        getTracks: () => [video, audio],
        getVideoTracks: () => [video],
        getAudioTracks: () => [audio],
      }),
    );

    renderWithClient(<VisitRoom sessionId="appt-1" />);
    fireEvent.click(screen.getByRole('button', { name: /join visit/i }));
    await screen.findByRole('button', { name: /mute mic/i });

    fireEvent.click(screen.getByRole('button', { name: /mute mic/i }));
    expect(audio.enabled).toBe(false);
    fireEvent.click(screen.getByRole('button', { name: /turn camera off/i }));
    expect(video.enabled).toBe(false);
  });

  it('surfaces an error when camera/mic access is denied', async () => {
    stubMedia(() => Promise.reject(new Error('Permission denied')));

    renderWithClient(<VisitRoom sessionId="appt-1" />);
    fireEvent.click(screen.getByRole('button', { name: /join visit/i }));

    const alert = await screen.findByRole('alert');
    expect(alert.textContent).toMatch(/permission denied/i);
  });
});
