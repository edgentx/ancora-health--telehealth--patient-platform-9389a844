# Ancora Health — Navigation Specification

---

## 1. Site Map

### Public Area (no authentication required)

- **Marketing Home** (`/`) — `none`
- **Login** (`/login`) — `none`
- **Patient Sign-Up** (`/signup`) — `none`
- **Provider Registration** (`/provider-signup`) — `none`
- **Forgot Password** (`/forgot-password`) — `none`
- **Reset Password** (`/reset-password`) — `none`
- **Email Verification** (`/verify-email`) — `none`
- **Provider Discovery** (`/providers`) — `primary`
- **Provider Public Profile** (`/providers/:id`) — `none`

### Patient Area (role: `patient`)

#### Primary Navigation
- **Patient Dashboard** (`/dashboard`)
- **Patient Appointments** (`/appointments`)
- **Provider Discovery** (`/providers`) *(shared with public)*
- **Secure Messaging Inbox** (`/messages`) *(shared across roles)*

#### Secondary Navigation
- **Health Profile** (`/health-profile`)
- **Prescriptions** (`/prescriptions`)
- **Documents & Lab Results** (`/documents`)
- **Billing & Invoices** (`/billing`)

#### Utility Navigation *(shared across all authenticated roles)*
- **Notifications** (`/notifications`)
- **Account Settings** (`/settings/account`)
- **Security Settings** (`/settings/security`)

#### Unlisted / Flow Pages
- **Patient Onboarding** (`/onboarding`)
- **Appointment Detail** (`/appointments/:id`)
- **Book Appointment** (`/appointments/book`)
- **Virtual Waiting Room** (`/visit/:id/waiting-room`)
- **Video Visit** (`/visit/:id`)
- **After-Visit Summary** (`/visit/:id/summary`)
- **Make a Payment** (`/billing/pay`)
- **Secure Message Thread** (`/messages/:id`)

### Provider Area (role: `provider`)

#### Primary Navigation
- **Provider Dashboard** (`/provider/dashboard`)
- **Provider Schedule** (`/provider/schedule`)
- **Secure Messaging Inbox** (`/messages`) *(shared across roles)*

#### Secondary Navigation
- **Clinical Notes Editor** (`/provider/notes`)
- **Patient Chart (EHR)** (`/provider/chart/:patientId`) *(also accessible to `scheduler`)*
- **E-Prescribing** (`/provider/prescribe`)
- **Provider Billing & Claims** (`/provider/billing`)
- **Patient Lookup** (`/provider/patients`) *(also accessible to `scheduler`)*

#### Utility Navigation *(shared)*
- **Notifications** (`/notifications`)
- **Account Settings** (`/settings/account`)
- **Security Settings** (`/settings/security`)

#### Unlisted / Flow Pages
- **Provider Onboarding** (`/provider/onboarding`)
- **Appointment Detail (Provider)** (`/provider/appointments/:id`)
- **Video Visit (Provider)** (`/provider/visit/:id`)
- **Visit Sign-Off** (`/provider/visit/:id/sign-off`)
- **Secure Message Thread** (`/messages/:id`)

### Scheduler Area (role: `scheduler`)

#### Primary Navigation
- **Scheduler Dashboard** (`/scheduler/dashboard`)
- **Schedule Management** (`/scheduler/schedule`)
- **Secure Messaging Inbox** (`/messages`) *(shared across roles)*

#### Secondary Navigation
- **Patient Chart (EHR)** (`/provider/chart/:patientId`) *(shared with `provider`)*
- **Patient Lookup** (`/scheduler/patients`) *(shared with `provider`)*
- **Provider Availability Management** (`/scheduler/availability`) *(also accessible to `admin`)*
- **Insurance Eligibility Verification** (`/scheduler/eligibility`)

#### Utility Navigation *(shared)*
- **Notifications** (`/notifications`)
- **Account Settings** (`/settings/account`)
- **Security Settings** (`/settings/security`)

#### Unlisted / Flow Pages
- **New Appointment** (`/scheduler/appointments/new`)
- **Patient Check-In** (`/scheduler/check-in/:appointmentId`)
- **Secure Message Thread** (`/messages/:id`)

### Admin Area (role: `admin`)

#### Primary Navigation
- **Admin Dashboard** (`/admin/dashboard`)
- **Analytics & Reporting** (`/admin/analytics`)
- **Provider Management** (`/admin/providers`)
- **User & Role Management** (`/admin/users`)
- **Secure Messaging Inbox** (`/messages`) *(shared across roles)*

#### Secondary Navigation
- **Billing Operations** (`/admin/billing`)
- **PHI Access Audit Log** (`/admin/audit-log`)
- **Clinic Settings** (`/admin/settings`)
- **Provider Availability Management** (`/admin/availability`) *(shared with `scheduler`)*

#### Utility Navigation *(shared)*
- **Notifications** (`/notifications`)
- **Account Settings** (`/settings/account`)
- **Security Settings** (`/settings/security`)

#### Unlisted / Flow Pages
- **Secure Message Thread** (`/messages/:id`)

---

## 2. Public Routes

Pages accessible without authentication.

| Page | ID | Notes |
|---|---|---|
| Marketing Home | `home` | Primary marketing entry point; CTAs for patient and provider sign-up |
| Login | `login` | Supports MFA; redirects to role-appropriate dashboard post-login |
| Patient Sign-Up | `signup` | Collects identity, contact, and insurance |
| Provider Registration | `provider-signup` | Collects credentials, NPI, specialties, and license |
| Forgot Password | `forgot-password` | Dispatches reset link to verified email |
| Reset Password | `reset-password` | Token-gated; valid only via recovery link |
| Email Verification | `verify-email` | Reached via registration confirmation link |
| Provider Discovery | `provider-search` | Searchable directory; publicly browsable |
| Provider Public Profile | `provider-profile` | Read-only bio and booking CTA; auth required to complete booking |

---

## 3. Authenticated Routes

### All Authenticated Users (any role)

| Page | ID | Notes |
|---|---|---|
| Secure Messaging Inbox | `messaging-inbox` | HIPAA-compliant; primary nav for all roles |
| Secure Message Thread | `messaging-thread` | Unlisted; opened from inbox |
| Notifications | `notifications` | Utility nav; all roles |
| Account Settings | `settings-account` | Utility nav; all roles |
| Security Settings | `settings-security` | Utility nav; all roles |

### Role: `patient`

| Page | ID | Nav |
|---|---|---|
| Patient Onboarding | `onboarding-patient` | Unlisted (first-run wizard) |
| Patient Dashboard | `patient-dashboard` | Primary |
| Patient Appointments | `appointments-patient` | Primary |
| Appointment Detail (Patient) | `appointment-detail-patient` | Unlisted |
| Book Appointment | `appointment-book` | Unlisted (flow entry from provider profile or dashboard) |
| Virtual Waiting Room | `waiting-room` | Unlisted |
| Video Visit (Patient) | `video-visit-patient` | Unlisted |
| After-Visit Summary (Patient) | `visit-summary-patient` | Unlisted |
| Health Profile | `health-profile` | Secondary |
| Prescriptions | `prescriptions-patient` | Secondary |
| Documents & Lab Results | `documents-patient` | Secondary |
| Billing & Invoices (Patient) | `billing-patient` | Secondary |
| Make a Payment | `payment` | Unlisted |

### Role: `provider`

| Page | ID | Nav |
|---|---|---|
| Provider Onboarding | `onboarding-provider` | Unlisted (first-run wizard) |
| Provider Dashboard | `provider-dashboard` | Primary |
| Provider Schedule | `provider-schedule` | Primary |
| Appointment Detail (Provider) | `appointment-detail-provider` | Unlisted |
| Video Visit (Provider) | `video-visit-provider` | Unlisted |
| Clinical Notes Editor | `clinical-notes` | Secondary |
| Patient Chart (EHR) | `patient-chart` | Secondary *(shared with `scheduler`)* |
| E-Prescribing | `eprescribing` | Secondary |
| Visit Sign-Off (Provider) | `visit-summary-provider` | Unlisted |
| Provider Billing & Claims | `provider-billing` | Secondary |
| Patient Lookup | `patient-lookup` | Secondary *(shared with `scheduler`)* |

### Role: `scheduler`

| Page | ID | Nav |
|---|---|---|
| Scheduler Dashboard | `scheduler-dashboard` | Primary |
| Schedule Management | `schedule-management` | Primary |
| Patient Lookup | `patient-lookup` | Secondary *(shared with `provider`)* |
| Patient Chart (EHR) | `patient-chart` | Secondary *(shared with `provider`)* |
| New Appointment (Scheduler) | `new-appointment` | Unlisted |
| Provider Availability Management | `provider-availability` | Secondary *(shared with `admin`)* |
| Patient Check-In | `patient-checkin` | Unlisted |
| Insurance Eligibility Verification | `insurance-verification` | Secondary |

### Role: `admin`

| Page | ID | Nav |
|---|---|---|
| Admin Dashboard | `admin-dashboard` | Primary |
| Analytics & Reporting | `analytics` | Primary |
| Provider Management | `provider-management` | Primary |
| User & Role Management | `user-management` | Primary |
| Billing Operations | `billing-operations` | Secondary |
| PHI Access Audit Log | `audit-log` | Secondary |
| Clinic Settings | `clinic-settings` | Secondary |
| Provider Availability Management | `provider-availability` | Secondary *(shared with `scheduler`)* |

---

## 4. User Journeys

### Patient First-Time Registration

New patient creates an account, verifies their email, completes health onboarding, and lands in the provider directory.

1. **Patient Sign-Up** (`signup`) — submit registration form
2. **Email Verification** (`verify-email`) — click confirmation link
3. **Patient Onboarding** (`onboarding-patient`) — complete health history, insurance, and consents wizard
4. **Provider Discovery** (`provider-search`) — browse and filter available providers

---

### Patient Books an Appointment

Authenticated patient selects a provider and schedules a visit.

1. **Provider Discovery** (`provider-search`) — search by specialty, availability, or location
2. **Provider Public Profile** (`provider-profile`) — review bio and trigger booking
3. **Book Appointment** (`appointment-book`) — choose time slot, specify reason, confirm
4. **Patient Appointments** (`appointments-patient`) — confirm appointment appears in list
5. **Appointment Detail (Patient)** (`appointment-detail-patient`) — review pre-visit instructions

---

### Patient Attends a Video Visit

Patient joins and completes a virtual care encounter.

1. **Appointment Detail (Patient)** (`appointment-detail-patient`) — review instructions and join
2. **Virtual Waiting Room** (`waiting-room`) — device check and queue status
3. **Video Visit (Patient)** (`video-visit-patient`) — live consultation with provider
4. **After-Visit Summary (Patient)** (`visit-summary-patient`) — review diagnosis notes, prescriptions, and follow-up actions

---

### Patient Pays a Balance

Patient reviews charges from a completed visit and submits payment.

1. **After-Visit Summary (Patient)** (`visit-summary-patient`) — see charges generated
2. **Billing & Invoices (Patient)** (`billing-patient`) — review outstanding balance and EOBs
3. **Make a Payment** (`payment`) — submit payment by card or HSA/FSA

---

### Provider First-Time Onboarding

New clinician registers, verifies their account, and completes their professional profile.

1. **Provider Registration** (`provider-signup`) — submit credentials, NPI, and specialties
2. **Email Verification** (`verify-email`) — click confirmation link
3. **Provider Onboarding** (`onboarding-provider`) — upload credentials, set availability, configure billing
4. **Provider Dashboard** (`provider-dashboard`) — arrive at home screen with today's schedule

---

### Provider Conducts a Video Visit

Provider prepares for, conducts, documents, and closes out a video encounter.

1. **Provider Schedule** (`provider-schedule`) — locate appointment and launch prep
2. **Appointment Detail (Provider)** (`appointment-detail-provider`) — review patient intake and prior notes
3. **Video Visit (Provider)** (`video-visit-provider`) — live consultation with embedded chart sidebar
4. **Clinical Notes Editor** (`clinical-notes`) — author SOAP/DAP note with voice-to-text
5. **E-Prescribing** (`eprescribing`) — issue prescriptions with drug-interaction check
6. **Visit Sign-Off (Provider)** (`visit-summary-provider`) — attest note, finalize diagnosis codes, submit charges

---

### Provider Reviews Patient Chart Between Visits

Provider looks up a patient outside of a scheduled appointment to review history or update notes.

1. **Provider Dashboard** (`provider-dashboard`) — start from home screen
2. **Patient Lookup** (`patient-lookup`) — search by name or MRN
3. **Patient Chart (EHR)** (`patient-chart`) — review longitudinal record
4. **Clinical Notes Editor** (`clinical-notes`) — add or amend a note

---

### Scheduler Creates an Appointment

Front-desk staff books a visit on behalf of a patient.

1. **Scheduler Dashboard** (`scheduler-dashboard`) — start from front-desk home screen
2. **Patient Lookup** (`patient-lookup`) — locate existing patient record
3. **New Appointment (Scheduler)** (`new-appointment`) — select provider, time, and visit type
4. **Schedule Management** (`schedule-management`) — confirm appointment appears on multi-provider calendar

---

### Front-Desk Patient Check-In

Scheduler confirms a patient's arrival and validates their insurance before the visit.

1. **Schedule Management** (`schedule-management`) — identify arriving patient on today's schedule
2. **Patient Check-In** (`patient-checkin`) — confirm arrival, verify identity, update demographics
3. **Insurance Eligibility Verification** (`insurance-verification`) — run real-time eligibility check and review coverage

---

### Admin Reviews Clinic Performance

Admin monitors KPIs, drills into detailed reports, and audits PHI access activity.

1. **Admin Dashboard** (`admin-dashboard`) — review real-time KPI tiles (utilization, no-shows, revenue)
2. **Analytics & Reporting** (`analytics`) — drill down by date range, provider, or payer
3. **PHI Access Audit Log** (`audit-log`) — review access events for HIPAA compliance
