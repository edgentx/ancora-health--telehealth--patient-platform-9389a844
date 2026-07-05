/**
 * Realtime React hooks bridging the WebSocket client into components and the
 * query cache.
 *
 * - `useConnectionStatus` exposes the socket state for reconnect banners.
 * - `useRealtimeNotifications` folds pushed notifications straight into the
 *   TanStack Query cache, so a live push and a REST refetch converge on one
 *   source of truth.
 * - `useSignalingChannel` hands a component a per-session WebRTC signaling
 *   channel, cleaned up on unmount.
 */
'use client';

import { useEffect, useRef, useState } from 'react';
import { useQueryClient } from '@tanstack/react-query';

import { apiClient } from '../client';
import type { Notification } from '../models/engagement';
import type { ConnectionStatus, SignalingChannel } from '../websocket';
import { queryKeys } from './keys';

export function useConnectionStatus(): ConnectionStatus {
  const [status, setStatus] = useState<ConnectionStatus>(
    apiClient.realtime.connectionStatus,
  );
  useEffect(() => apiClient.realtime.onStatus(setStatus), []);
  return status;
}

/**
 * Subscribe to notification pushes and merge them into the notifications query
 * cache, deduplicating by id. Components read the cache via `useNotifications`.
 */
export function useRealtimeNotifications(): void {
  const qc = useQueryClient();
  useEffect(() => {
    return apiClient.realtime.subscribe<Notification>('notifications', (envelope) => {
      qc.setQueryData<Notification[]>(
        queryKeys.engagement.notifications(),
        (current = []) =>
          current.some((n) => n.id === envelope.payload.id)
            ? current
            : [envelope.payload, ...current],
      );
    });
  }, [qc]);
}

/**
 * Open a WebRTC signaling channel for `sessionId`, closed automatically when the
 * component unmounts or the session changes. Returns `null` on the server.
 */
export function useSignalingChannel(sessionId: string): SignalingChannel | null {
  const channelRef = useRef<SignalingChannel | null>(null);
  const [channel, setChannel] = useState<SignalingChannel | null>(null);

  useEffect(() => {
    if (!sessionId) return;
    const opened = apiClient.realtime.openSignalingChannel(sessionId);
    channelRef.current = opened;
    setChannel(opened);
    return () => {
      opened.close();
      channelRef.current = null;
      setChannel(null);
    };
  }, [sessionId]);

  return channel;
}
