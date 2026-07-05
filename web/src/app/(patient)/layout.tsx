import { AppShell } from '@/components/app-shell';
import { resolveIdentity } from '@/lib/identity';

/** Patient surface shell — gates to the `patient` role. */
export default async function PatientLayout({ children }: { children: React.ReactNode }) {
  const identity = await resolveIdentity();
  return (
    <AppShell surface="patient" identity={identity}>
      {children}
    </AppShell>
  );
}
