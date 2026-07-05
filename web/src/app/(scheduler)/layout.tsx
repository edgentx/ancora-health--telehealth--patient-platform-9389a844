import { AppShell } from '@/components/app-shell';
import { resolveIdentity } from '@/lib/identity';

/** Front-desk / scheduler surface shell — gates to the `scheduler` role. */
export default async function SchedulerLayout({ children }: { children: React.ReactNode }) {
  const identity = await resolveIdentity();
  return (
    <AppShell surface="scheduler" identity={identity}>
      {children}
    </AppShell>
  );
}
