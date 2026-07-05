import { AppShell } from '@/components/app-shell';
import { resolveIdentity } from '@/lib/identity';

/** Clinic admin surface shell — gates to the `admin` role. */
export default async function AdminLayout({ children }: { children: React.ReactNode }) {
  const identity = await resolveIdentity();
  return (
    <AppShell surface="admin" identity={identity}>
      {children}
    </AppShell>
  );
}
