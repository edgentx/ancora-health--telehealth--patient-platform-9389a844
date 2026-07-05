import { LoginView } from '@/components/login-view';
import { resolveIdentity } from '@/lib/identity';

/**
 * Authentication entry route (`/login`).
 *
 * Server component: it reads the identity the edge resolved from the trusted
 * headers (dynamic per request) and hands it to the client {@link LoginView},
 * which either redirects an authenticated user to their surface or presents the
 * unauthenticated entry state. No token parsing or RBAC happens here.
 */
export default async function LoginPage() {
  const identity = await resolveIdentity();
  return <LoginView identity={identity} />;
}
