import { headers } from 'next/headers';

import { serverEnv } from './env';
import { isRole, type ResolvedRole } from './roles';

/**
 * The identity resolved from trusted edge headers.
 *
 * The edge/proxy performs authentication (JWT validation, session lookup, MFA)
 * and injects the result as trusted headers. This app does NOT parse tokens or
 * run its own RBAC — it reads what the edge already vouched for and gates UI on
 * the resolved role. `guest` means no valid role header was present.
 */
export interface Identity {
  role: ResolvedRole;
  /** Display name or subject id, if the edge supplied one. */
  user: string | null;
}

/**
 * Read the resolved identity from the request's trusted headers. Server-only:
 * `headers()` is a dynamic server API, so any component calling this is rendered
 * per-request rather than statically.
 */
export async function resolveIdentity(): Promise<Identity> {
  const h = await headers();
  const rawRole = h.get(serverEnv.roleHeader);
  const user = h.get(serverEnv.userHeader);

  return {
    role: isRole(rawRole) ? rawRole : 'guest',
    user: user && user.length > 0 ? user : null,
  };
}
