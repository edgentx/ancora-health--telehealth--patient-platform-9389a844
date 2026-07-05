import { AppShell } from '@/components/app-shell';
import { resolveIdentity } from '@/lib/identity';

/** Provider surface shell — gates to the `provider` role. */
export default async function ProviderLayout({ children }: { children: React.ReactNode }) {
  const identity = await resolveIdentity();
  return (
    <AppShell surface="provider" identity={identity}>
      {children}
    </AppShell>
  );
}
