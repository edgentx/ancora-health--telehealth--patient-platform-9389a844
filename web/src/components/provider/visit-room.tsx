'use client';

import { useCallback, useEffect, useRef, useState } from 'react';

import {
  useConnectionStatus,
  useSignalingChannel,
  type ConnectionStatus,
} from '@/lib/api';

/** Human label + dot tone for each realtime connection state. */
const CONN_LABEL: Record<ConnectionStatus, string> = {
  idle: 'Not connected',
  connecting: 'Connecting…',
  open: 'Connected',
  closed: 'Disconnected',
};

type Phase = 'idle' | 'joining' | 'live' | 'error';

/**
 * WebRTC visit room. Joining acquires the camera/mic with `getUserMedia`, opens
 * (or reuses) an `RTCPeerConnection`, and exchanges SDP/ICE over the S-80
 * signaling channel — which tags every frame with the visit's `sessionId` so
 * concurrent visits never cross-talk. The realtime connection status drives the
 * banner; camera/mic toggles flip the local track's `enabled` flag without
 * renegotiating.
 *
 * The room degrades gracefully where the platform lacks the APIs (no
 * `RTCPeerConnection` under jsdom): it still captures local media and announces
 * itself over signaling, so the join → signal → status flow is fully exercised.
 */
export function VisitRoom({ sessionId }: { sessionId: string }) {
  const status = useConnectionStatus();
  const channel = useSignalingChannel(sessionId);

  const localVideoRef = useRef<HTMLVideoElement>(null);
  const remoteVideoRef = useRef<HTMLVideoElement>(null);
  const streamRef = useRef<MediaStream | null>(null);
  const pcRef = useRef<RTCPeerConnection | null>(null);

  const [phase, setPhase] = useState<Phase>('idle');
  const [errorMsg, setErrorMsg] = useState('');
  const [camOn, setCamOn] = useState(true);
  const [micOn, setMicOn] = useState(true);
  const [peerPresent, setPeerPresent] = useState(false);

  const teardown = useCallback(() => {
    streamRef.current?.getTracks().forEach((track) => track.stop());
    streamRef.current = null;
    pcRef.current?.close();
    pcRef.current = null;
  }, []);

  // Stop the camera/mic and close the peer connection when the room unmounts.
  useEffect(() => teardown, [teardown]);

  // Apply inbound signaling frames to the peer connection (best-effort) and mark
  // the remote participant present the moment any signal arrives.
  useEffect(() => {
    if (!channel) return;
    return channel.onMessage((signal) => {
      setPeerPresent(true);
      const pc = pcRef.current;
      if (!pc || !signal || typeof signal !== 'object') return;
      const frame = signal as { sdp?: RTCSessionDescriptionInit; candidate?: RTCIceCandidateInit };
      if (frame.sdp) {
        void pc.setRemoteDescription(frame.sdp).catch(() => undefined);
      } else if (frame.candidate) {
        void pc.addIceCandidate(frame.candidate).catch(() => undefined);
      }
    });
  }, [channel]);

  async function join() {
    setPhase('joining');
    setErrorMsg('');
    try {
      const stream = await navigator.mediaDevices.getUserMedia({
        video: true,
        audio: true,
      });
      streamRef.current = stream;
      if (localVideoRef.current) localVideoRef.current.srcObject = stream;

      if (typeof RTCPeerConnection !== 'undefined') {
        const pc = new RTCPeerConnection();
        pcRef.current = pc;
        stream.getTracks().forEach((track) => pc.addTrack(track, stream));
        pc.ontrack = (event) => {
          if (remoteVideoRef.current) remoteVideoRef.current.srcObject = event.streams[0]!;
        };
        pc.onicecandidate = (event) => {
          if (event.candidate) channel?.send({ candidate: event.candidate });
        };
        const offer = await pc.createOffer();
        await pc.setLocalDescription(offer);
        channel?.send({ sdp: offer });
      } else {
        // No RTCPeerConnection here — still announce join over signaling so the
        // edge can pair us with the peer.
        channel?.send({ type: 'join', sessionId });
      }

      setPhase('live');
    } catch (err) {
      setErrorMsg(
        err instanceof Error ? err.message : 'Could not access your camera or microphone.',
      );
      setPhase('error');
    }
  }

  function leave() {
    teardown();
    setPhase('idle');
    setPeerPresent(false);
    setCamOn(true);
    setMicOn(true);
  }

  function toggleCamera() {
    const next = !camOn;
    streamRef.current?.getVideoTracks().forEach((track) => (track.enabled = next));
    setCamOn(next);
  }

  function toggleMic() {
    const next = !micOn;
    streamRef.current?.getAudioTracks().forEach((track) => (track.enabled = next));
    setMicOn(next);
  }

  const live = phase === 'live';

  return (
    <section className="visit">
      <div className="section-head">
        <div>
          <h1 className="page-heading">Video visit</h1>
          <p className="page-subheading" style={{ margin: 0 }}>
            Session {sessionId}
          </p>
        </div>
        <span className="conn" role="status" aria-live="polite">
          <span className={`conn__dot conn__dot--${status}`} aria-hidden />
          {CONN_LABEL[status]}
        </span>
      </div>

      {phase === 'error' ? (
        <p className="banner banner--conflict" role="alert">
          {errorMsg}
        </p>
      ) : null}

      <div className="visit__stage">
        <div className="visit__tile">
          <video ref={localVideoRef} autoPlay playsInline muted />
          {!live ? <span>Camera off</span> : null}
          <span className="visit__tile-label">You{camOn ? '' : ' · camera off'}</span>
        </div>
        <div className="visit__tile">
          <video ref={remoteVideoRef} autoPlay playsInline />
          {!peerPresent ? <span>Waiting for patient…</span> : null}
          <span className="visit__tile-label">Patient</span>
        </div>
      </div>

      <div className="visit__controls">
        {!live ? (
          <button
            type="button"
            className="btn btn--primary"
            onClick={join}
            disabled={phase === 'joining'}
          >
            {phase === 'joining' ? 'Joining…' : 'Join visit'}
          </button>
        ) : (
          <>
            <button type="button" className="btn btn--ghost" onClick={toggleMic} aria-pressed={!micOn}>
              {micOn ? 'Mute mic' : 'Unmute mic'}
            </button>
            <button
              type="button"
              className="btn btn--ghost"
              onClick={toggleCamera}
              aria-pressed={!camOn}
            >
              {camOn ? 'Turn camera off' : 'Turn camera on'}
            </button>
            <button type="button" className="btn btn--ghost" onClick={leave}>
              Leave
            </button>
          </>
        )}
      </div>
    </section>
  );
}
