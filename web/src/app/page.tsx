import Link from 'next/link';

import { ROLE_LABELS, ROLE_LANDING, ROLES } from '@/lib/roles';

/**
 * Public marketing home (`/`). Video-First Hero pattern is fleshed out by later
 * marketing stories; this scaffold ships the Swiss/minimal baseline plus quick
 * links into each role surface's landing route for development and smoke tests.
 */
export default function MarketingHome() {
  return (
    <div className="marketing">
      <p className="shell__brand" style={{ color: 'var(--color-primary)' }}>
        Ancora Health
      </p>
      <h1 className="marketing__title">Telehealth care, coordinated end to end.</h1>
      <p className="marketing__subtitle">
        Patients, providers, front-desk schedulers, and clinic admins — one platform, each with a
        workspace built for the job.
      </p>
      <div className="marketing__cta-row">
        <Link href="/signup" className="btn btn--cta">
          Create your account
        </Link>
        <Link href="/login" className="btn btn--primary">
          Sign in
        </Link>
      </div>
      <div className="role-links">
        {ROLES.map((role) => (
          <Link key={role} href={ROLE_LANDING[role]}>
            {ROLE_LABELS[role]} workspace →
          </Link>
        ))}
      </div>
    </div>
  );
}
