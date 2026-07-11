```markdown
# Ancora Health — Navigation Specification

---

## 1. Site Map

### Public (No Auth Required)

#### Primary Nav
- Provider Directory (`/provider-directory`)

#### No Nav
- Home / Marketing Landing (`/`)
- Provider Public Profile (`/providers/:id`)
- Forgot Password (`/forgot-password`)
- Reset Password (`/reset-password`)
- Email Verification (`/verify-email`)

#### Utility Nav
- Login (`/login`)
- Patient Sign-Up (`/signup`)

---

### Authenticated — Patient

#### Primary Nav
- Patient Dashboard (`/patient/dashboard`)
- Find a Provider (`/patient/providers`)
- My Appointments (`/patient/appointments`)
- Messages (`/patient/messages`)

#### Secondary Nav
- Health Records (`/patient/records`)
- My Prescriptions (`/patient/prescriptions`)
- Billing & Invoices (`/patient/billing`)
- Insurance Information (`/patient/insurance`)

#### Utility Nav
- Patient Profile & Settings (`/patient/settings`)

#### No Nav (Workflow / Deep Pages)
- Book Appointment (`/patient/book`)
- Appointment Detail (`/patient/appointments/:id`)
- Patient Waiting Room (`/patient/appointments/:id/waiting-room`)
- Video Visit (`/patient/appointments/:id/visit`)
- Message Thread (`/patient/messages/:threadId`)
- Make a Payment (`/patient/billing/pay`)

---

### Authenticated — Provider

#### Primary Nav
- Provider Dashboard (`/provider/dashboard`)
- Schedule / Calendar (`/provider/schedule`)
- Today's Appointment Queue (`/provider/queue`)
- Messages (`/provider/messages`)

#### Secondary Nav
- Billing & Claims Review (`/provider/billing`)
- Availability Settings (`/provider/availability`)

#### Utility Nav
- Edit Provider Profile (`/provider/profile`)

#### No Nav (Workflow / Deep Pages)
- Appointment Detail (`/provider/appointments/:id`)
- Provider Waiting Room (`/provider/appointments/:id/waiting-room`)
- Video Visit (`/provider/appointments/:id/visit`)
- Patient Chart (`/provider/patients/:patientId/chart`)
- Clinical Notes / Documentation (`/provider/appointments/:id/notes`)
- E-Prescribing (`/provider/appointments/:id/prescribe`)
- Message Thread (`/provider/messages/:threadId`)

---

### Authenticated — Scheduler (Front Desk)

#### Primary Nav
- Scheduler Dashboard (`/scheduler/dashboard`)
- Appointment Management (`/scheduler/appointments`)
- Patient Search (`/scheduler/patients`)

#### Secondary Nav
- Waitlist Management (`/scheduler/waitlist`)

#### No Nav (Workflow / Deep Pages)
- Schedule Appointment (`/scheduler/appointments/new`)
- Patient Profile — Front Desk View (`/scheduler/patients/:id`)
- Patient Check-In (`/scheduler/appointments/:id/check-in`)

---

### Authenticated — Clinic Admin

#### Primary Nav
- Admin Dashboard (`/admin/dashboard`)
- Provider Management (`/admin/providers`)
- Patient Management (`/admin/patients`)
- Billing Operations (`/admin/billing`)

#### Secondary Nav
- Utilization Report (`/admin/reports/utilization`)
- Revenue Report (`/admin/reports/revenue`)
- No-Show Report (`/admin/reports/no-shows`)
- HIPAA Audit Log (`/admin/audit-log`)

#### Utility Nav
- Clinic Settings (`/admin/settings`)
- User Management (`/admin/users`)
- Role & Permissions (`/admin/roles`)

#### No Nav (Shared / Deep Pages)
- Patient Chart (`/admin/patients/:patientId/chart`) *(shared with provider role)*

---

## 2. Public Routes

Pages accessible without authentication.

| Page | Path | Nav Section | Description |
|------|------|-------------|-------------|
| Home / Marketing Landing | `/` | — | Platform introduction; drives patient and provider sign-ups |
| Login | `/login` | Utility | Unified sign-in for all roles; routes to role-specific dashboard on success |
| Patient Sign-Up | `/signup` | Utility | New patient registration form |
| Forgot Password | `/forgot-password` | — | Initiates email-based password-reset flow |
| Reset Password | `/reset-password` | — | Token-authenticated form to set a new password |
| Email Verification | `/verify-email` | — | Post-registration email confirmation gate |
| Provider Directory | `/provider-directory` | Primary | Searchable, filterable directory of all clinicians |
| Provider Public Profile | `/providers/:id` | — | Clinician specialty, credentials, availability, and booking CTA |

---

## 3. Authenticated Routes

### Patient Role

| Page | Path | Nav Section |
|------|------|-------------|
| Patient Dashboard | `/patient/dashboard` | Primary |
| Find a Provider | `/patient/providers` | Primary |
| My Appointments | `/patient/appointments` | Primary |
| Messages | `/patient/messages` | Primary |
| Health Records | `/patient/records` | Secondary |
| My Prescriptions | `/patient/prescriptions` | Secondary |
| Billing & Invoices | `/patient/billing` | Secondary |
| Insurance Information | `/patient/insurance` | Secondary |
| Patient Profile & Settings | `/patient/settings` | Utility |
| Book Appointment | `/patient/book` | — |
| Appointment Detail | `/patient/appointments/:id` | — |
| Patient Waiting Room | `/patient/appointments/:id/waiting-room` | — |
| Video Visit | `/patient/appointments/:id/visit` | — |
| Message Thread | `/patient/messages/:threadId` | — |
| Make a Payment | `/patient/billing/pay` | — |

### Provider Role

| Page | Path | Nav Section |
|------|------|-------------|
| Provider Dashboard | `/provider/dashboard` | Primary |
| Schedule / Calendar | `/provider/schedule` | Primary |
| Today's Appointment Queue | `/provider/queue` | Primary |
| Messages | `/provider/messages` | Primary |
| Billing & Claims Review | `/provider/billing` | Secondary |
| Availability Settings | `/provider/availability` | Secondary |
| Edit Provider Profile | `/provider/profile` | Utility |
| Appointment Detail | `/provider/appointments/:id` | — |
| Provider Waiting Room | `/provider/appointments/:id/waiting-room` | — |
| Video Visit | `/provider/appointments/:id/visit` | — |
| Patient Chart | `/provider/patients/:patientId/chart` | — |
| Clinical Notes / Documentation | `/provider/appointments/:id/notes` | — |
| E-Prescribing | `/provider/appointments/:id/prescribe` | — |
| Message Thread | `/provider/messages/:threadId` | — |

### Scheduler (Front Desk) Role

| Page | Path | Nav Section |
|------|------|-------------|
| Scheduler Dashboard | `/scheduler/dashboard` | Primary |
| Appointment Management | `/scheduler/appointments` | Primary |
| Patient Search | `/scheduler/patients` | Primary |
| Waitlist Management | `/scheduler/waitlist` | Secondary |
| Schedule Appointment | `/scheduler/appointments/new` | — |
| Patient Profile (Front Desk) | `/scheduler/patients/:id` | — |
| Patient Check-In | `/scheduler/appointments/:id/check-in` | — |

### Clinic Admin Role

| Page | Path | Nav Section |
|------|------|-------------|
| Admin Dashboard | `/admin/dashboard` | Primary |
| Provider Management | `/admin/providers` | Primary |
| Patient Management | `/admin/patients` | Primary |
| Billing Operations | `/admin/billing` | Primary |
| Utilization Report | `/admin/reports/utilization` | Secondary |
| Revenue Report | `/admin/reports/revenue` | Secondary |
| No-Show Report | `/admin/reports/no-shows` | Secondary |
| HIPAA Audit Log | `/admin/audit-log` | Secondary |
| Clinic Settings | `/admin/settings` | Utility |
| User Management | `/admin/users` | Utility |
| Role & Permissions | `/admin/roles` | Utility |
| Patient Chart | `/admin/patients/:patientId/chart` | — |

### Shared Access

| Page | Roles |
|------|-------|
| Patient Chart | `provider`, `clinic-admin` |

---

## 4. User Journeys

### Patient Onboarding
New patients register, verify their identity, and configure their account before reaching their dashboard.

```
Home (/)
  → Patient Sign-Up (/signup)
  → Email Verification (/verify-email)
  → Insurance Information (/patient/insurance)
  → Patient Profile & Settings (/patient/settings)
  → Patient Dashboard (/patient/dashboard)
```

---

### Patient Books a Visit
A patient discovers a provider and books an appointment.

```
Patient Dashboard (/patient/dashboard)
  → Find a Provider (/patient/providers)
  → Provider Public Profile (/providers/:id)
  → Book Appointment (/patient/book)
  → My Appointments (/patient/appointments)
```

---

### Patient Attends a Video Visit
A patient enters the visit workflow from their appointment list.

```
My Appointments (/patient/appointments)
  → Appointment Detail (/patient/appointments/:id)
  → Patient Waiting Room (/patient/appointments/:id/waiting-room)
  → Video Visit (/patient/appointments/:id/visit)
```

---

### Patient Reviews Visit & Pays
After a visit, the patient reviews clinical output and settles their balance.

```
Appointment Detail (/patient/appointments/:id)
  → Health Records (/patient/records)
  → My Prescriptions (/patient/prescriptions)
  → Billing & Invoices (/patient/billing)
  → Make a Payment (/patient/billing/pay)
```

---

### Patient Sends a Secure Message
A patient initiates a HIPAA-compliant message to their care team.

```
Patient Dashboard (/patient/dashboard)
  → Messages (/patient/messages)
  → Message Thread (/patient/messages/:threadId)
```

---

### Provider Conducts a Full Visit
A provider moves from their queue through the complete clinical visit workflow.

```
Provider Dashboard (/provider/dashboard)
  → Today's Appointment Queue (/provider/queue)
  → Appointment Detail (/provider/appointments/:id)
  → Patient Chart (/provider/patients/:patientId/chart)
  → Provider Waiting Room (/provider/appointments/:id/waiting-room)
  → Video Visit (/provider/appointments/:id/visit)
  → Clinical Notes (/provider/appointments/:id/notes)
  → E-Prescribing (/provider/appointments/:id/prescribe)
```

---

### Provider Manages Availability
A provider updates their recurring schedule from the dashboard.

```
Provider Dashboard (/provider/dashboard)
  → Availability Settings (/provider/availability)
```

---

### Front Desk Schedules a Patient
A scheduler looks up a patient and books a visit on their behalf.

```
Scheduler Dashboard (/scheduler/dashboard)
  → Patient Search (/scheduler/patients)
  → Patient Profile — Front Desk (/scheduler/patients/:id)
  → Schedule Appointment (/scheduler/appointments/new)
  → Appointment Management (/scheduler/appointments)
```

---

### Front Desk Checks In a Patient
A scheduler confirms a patient's arrival and notifies the provider.

```
Scheduler Dashboard (/scheduler/dashboard)
  → Patient Search (/scheduler/patients)
  → Patient Check-In (/scheduler/appointments/:id/check-in)
```

---

### Admin Reviews Clinic Performance
An admin surveys operational KPIs and detailed reports.

```
Admin Dashboard (/admin/dashboard)
  → Utilization Report (/admin/reports/utilization)
  → Revenue Report (/admin/reports/revenue)
  → No-Show Report (/admin/reports/no-shows)
```

---

### Admin Onboards a New Provider
An admin creates a user account and sets up the provider's clinical profile.

```
User Management (/admin/users)
  → Provider Management (/admin/providers)
```

---

### Admin Conducts HIPAA Compliance Review
An admin audits PHI access events from the dashboard.

```
Admin Dashboard (/admin/dashboard)
  → HIPAA Audit Log (/admin/audit-log)
```

---

### Admin Configures Role Permissions
An admin adjusts role-based access control for platform users.

```
User Management (/admin/users)
  → Role & Permissions (/admin/roles)
```
```