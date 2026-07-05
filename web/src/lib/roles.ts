/**
 * Role model + navigation map for the four role surfaces.
 *
 * This mirrors `design-system/navigation.md`. It is the single source of truth
 * for: which roles exist, each role's landing route, and the navigation shown in
 * that role's app shell. The UI is gated purely by the resolved role — there is
 * no client-side RBAC or token parsing here (see {@link ./identity}).
 */

/** The four authenticated role surfaces plus the unauthenticated `guest`. */
export const ROLES = ['patient', 'provider', 'scheduler', 'admin'] as const;
export type Role = (typeof ROLES)[number];

/** Anyone whose request carries no (or an unrecognised) role header. */
export type ResolvedRole = Role | 'guest';

export function isRole(value: string | null | undefined): value is Role {
  return value != null && (ROLES as readonly string[]).includes(value);
}

/** Human-facing surface labels (Clinic Admin, Front-desk/Scheduler, ...). */
export const ROLE_LABELS: Record<Role, string> = {
  patient: 'Patient',
  provider: 'Provider',
  scheduler: 'Front Desk',
  admin: 'Clinic Admin',
};

/**
 * Where each role lands after the edge resolves their identity. These are the
 * routes the production smoke test renders, one per role.
 */
export const ROLE_LANDING: Record<Role, string> = {
  patient: '/dashboard',
  provider: '/provider/dashboard',
  scheduler: '/scheduler/dashboard',
  admin: '/admin/dashboard',
};

export interface NavItem {
  label: string;
  href: string;
}

export interface NavGroup {
  /** primary | secondary | utility — matches the navigation spec's grouping. */
  title: string;
  items: NavItem[];
}

/**
 * Navigation shown in each surface's app shell. Kept deliberately close to
 * `navigation.md`; later stories flesh out the individual pages behind each link.
 */
export const ROLE_NAV: Record<Role, NavGroup[]> = {
  patient: [
    {
      title: 'primary',
      items: [
        { label: 'Dashboard', href: '/dashboard' },
        { label: 'Appointments', href: '/appointments' },
        { label: 'Find a Provider', href: '/providers' },
        { label: 'Messages', href: '/messages' },
      ],
    },
    {
      title: 'secondary',
      items: [
        { label: 'Health Profile', href: '/health-profile' },
        { label: 'Prescriptions', href: '/prescriptions' },
        { label: 'Documents & Labs', href: '/documents' },
        { label: 'Billing & Invoices', href: '/billing' },
      ],
    },
  ],
  provider: [
    {
      title: 'primary',
      items: [
        { label: 'Dashboard', href: '/provider/dashboard' },
        { label: 'Schedule', href: '/provider/schedule' },
        { label: 'Messages', href: '/messages' },
      ],
    },
    {
      title: 'secondary',
      items: [
        { label: 'Clinical Notes', href: '/provider/notes' },
        { label: 'E-Prescribing', href: '/provider/prescribe' },
        { label: 'Patient Lookup', href: '/provider/patients' },
        { label: 'Billing & Claims', href: '/provider/billing' },
      ],
    },
  ],
  scheduler: [
    {
      title: 'primary',
      items: [
        { label: 'Dashboard', href: '/scheduler/dashboard' },
        { label: 'Schedule Management', href: '/scheduler/schedule' },
        { label: 'Messages', href: '/messages' },
      ],
    },
    {
      title: 'secondary',
      items: [
        { label: 'Patient Lookup', href: '/scheduler/patients' },
        { label: 'Availability', href: '/scheduler/availability' },
        { label: 'Eligibility', href: '/scheduler/eligibility' },
      ],
    },
  ],
  admin: [
    {
      title: 'primary',
      items: [
        { label: 'Dashboard', href: '/admin/dashboard' },
        { label: 'Analytics', href: '/admin/analytics' },
        { label: 'Providers', href: '/admin/providers' },
        { label: 'Users & Roles', href: '/admin/users' },
        { label: 'Messages', href: '/messages' },
      ],
    },
    {
      title: 'secondary',
      items: [
        { label: 'Billing Operations', href: '/admin/billing' },
        { label: 'Audit Log', href: '/admin/audit-log' },
        { label: 'Clinic Settings', href: '/admin/settings' },
      ],
    },
  ],
};

/** Utility nav shared by every authenticated role. */
export const UTILITY_NAV: NavGroup = {
  title: 'utility',
  items: [
    { label: 'Notifications', href: '/notifications' },
    { label: 'Account Settings', href: '/settings/account' },
    { label: 'Security', href: '/settings/security' },
  ],
};
