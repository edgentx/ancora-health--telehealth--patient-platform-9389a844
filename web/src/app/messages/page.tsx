import { redirect } from 'next/navigation';

import { AppShell } from '@/components/app-shell';
import { SecureMessaging } from '@/components/secure-messaging';
import { resolveIdentity } from '@/lib/identity';

/**
 * Secure messaging (`/messages`) is shared by every authenticated role — the
 * patient portal and each staff console link here. Because it is cross-role it
 * lives outside the per-surface route groups; it renders the caller's own role
 * shell (resolved from the edge headers) around the shared messaging view. A
 * guest is bounced to the entry view rather than shown an empty shell.
 */
export default async function MessagesPage() {
  const identity = await resolveIdentity();
  if (identity.role === 'guest') {
    redirect('/login');
  }

  return (
    <AppShell surface={identity.role} identity={identity}>
      <SecureMessaging />
    </AppShell>
  );
}
