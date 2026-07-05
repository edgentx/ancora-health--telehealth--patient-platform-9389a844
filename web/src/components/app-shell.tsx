import Link from 'next/link';

import type { Identity } from '@/lib/identity';
import { ROLE_LABELS, ROLE_LANDING, ROLE_NAV, UTILITY_NAV, type Role } from '@/lib/roles';

import { LogoutButton } from './logout-button';
import { RoleBadge } from './role-badge';

/**
 * Shared application shell for the four role surfaces: masthead + role-scoped
 * sidebar navigation + main content region.
 *
 * `surface` is the role this shell serves. If the resolved identity's role does
 * not match, we render a {@link RoleGate} instead of the surface — UI gating by
 * the trusted role, with zero client-side RBAC.
 */
export function AppShell({
  surface,
  identity,
  children,
}: {
  surface: Role;
  identity: Identity;
  children: React.ReactNode;
}) {
  if (identity.role !== surface) {
    return <RoleGate surface={surface} identity={identity} />;
  }

  const groups = [...ROLE_NAV[surface], UTILITY_NAV];

  return (
    <div className="shell">
      <aside className="shell__sidebar" aria-label="Primary navigation">
        <Link href={ROLE_LANDING[surface]} className="shell__brand">
          Ancora <span className="shell__brand-surface">{ROLE_LABELS[surface]}</span>
        </Link>
        <nav>
          {groups.map((group) => (
            <div key={group.title} className="nav-group">
              <p className="nav-group__title">{group.title}</p>
              <ul className="nav-group__list">
                {group.items.map((item) => (
                  <li key={item.href}>
                    <Link href={item.href} className="nav-link">
                      {item.label}
                    </Link>
                  </li>
                ))}
              </ul>
            </div>
          ))}
        </nav>
      </aside>
      <div className="shell__main">
        <header className="shell__topbar">
          <span className="shell__surface-label">{ROLE_LABELS[surface]} workspace</span>
          <div className="shell__topbar-actions">
            <RoleBadge />
            <LogoutButton />
          </div>
        </header>
        <main className="shell__content">{children}</main>
      </div>
    </div>
  );
}

/**
 * Shown when the resolved role does not match the requested surface (including
 * unauthenticated `guest`). Presentation-only: real enforcement is the edge's
 * job — this simply avoids rendering another role's workspace.
 */
function RoleGate({ surface, identity }: { surface: Role; identity: Identity }) {
  const landing = identity.role !== 'guest' ? ROLE_LANDING[identity.role] : '/login';
  return (
    <main className="gate">
      <div className="gate__card">
        <h1 className="gate__title">Not available for your role</h1>
        <p className="gate__body">
          The <strong>{ROLE_LABELS[surface]}</strong> workspace is gated to the{' '}
          <code>{surface}</code> role. Your resolved role is <code>{identity.role}</code>.
        </p>
        <Link href={landing} className="gate__link">
          {identity.role === 'guest' ? 'Go to sign in' : 'Go to your workspace'}
        </Link>
      </div>
    </main>
  );
}
