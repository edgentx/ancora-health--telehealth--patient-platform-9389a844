import { VisitRoom } from '@/components/provider/visit-room';

/**
 * WebRTC visit room (`/provider/visit/[sessionId]`). The session id is the
 * appointment being seen; it keys the signaling channel so concurrent visits
 * stay isolated.
 */
export default async function ProviderVisitPage({
  params,
}: {
  params: Promise<{ sessionId: string }>;
}) {
  const { sessionId } = await params;
  return <VisitRoom sessionId={sessionId} />;
}
