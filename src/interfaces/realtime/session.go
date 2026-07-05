package realtime

import (
	"context"
	"errors"

	authzmodel "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/authorization/model"
	authzrepo "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/authorization/repository"
	schedmodel "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/scheduling/model"
	schedrepo "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/scheduling/repository"
)

// ErrSessionNotScoped is returned when a signaling handshake cannot be tied to a
// valid appointment/care-relationship context: no appointment, an appointment
// that is not a booked video visit, a caller who is not a participant, or a
// referenced care relationship that is not active. The gateway refuses such a
// session.
var ErrSessionNotScoped = errors.New("realtime: signaling session is not scoped to a valid appointment/care relationship")

// SessionContext is the validated scope of a signaling session. The gateway
// carries it for the life of the connection: it identifies the appointment the
// video visit belongs to and the two participants permitted to exchange
// signaling, and is the key peers are grouped under for relay.
type SessionContext struct {
	// AppointmentID is the booked video visit the session belongs to. It doubles
	// as the relay key: only peers sharing an AppointmentID exchange signaling.
	AppointmentID string
	// PatientID and ProviderID are the two participants scoped to the visit.
	PatientID  string
	ProviderID string
	// CareRelationshipID is the active care relationship the session is bound to,
	// when one was supplied and verified at handshake.
	CareRelationshipID string
	// CallerRole records whether the connecting principal joined as the patient or
	// the provider, for the audit actor context.
	CallerRole string
}

// SessionAuthorizer validates a signaling handshake and returns the scope the
// session runs under, or ErrSessionNotScoped when the handshake cannot be tied
// to a valid appointment/care relationship.
type SessionAuthorizer interface {
	Authorize(ctx context.Context, h Handshake) (SessionContext, error)
}

// AppointmentSessionAuthorizer is the production SessionAuthorizer. It scopes a
// video-visit signaling session to a booked Appointment whose participant set
// includes the connecting principal, optionally further constrained to an active
// CareRelationship. It is the enforcement point for "sessions without valid
// context are refused".
type AppointmentSessionAuthorizer struct {
	appointments  schedrepo.AppointmentRepository
	relationships authzrepo.CareRelationshipRepository
}

// NewAppointmentSessionAuthorizer builds an authorizer over the appointment and
// care-relationship repositories. The care-relationship repository may be nil,
// in which case handshakes that reference a relationship are refused because the
// reference cannot be verified.
func NewAppointmentSessionAuthorizer(
	appointments schedrepo.AppointmentRepository,
	relationships authzrepo.CareRelationshipRepository,
) *AppointmentSessionAuthorizer {
	return &AppointmentSessionAuthorizer{appointments: appointments, relationships: relationships}
}

// Authorize refuses any handshake that lacks an authenticated principal or a
// booked appointment the principal participates in, and — when a care
// relationship is referenced — that is not active and matching the appointment's
// participants.
func (a *AppointmentSessionAuthorizer) Authorize(ctx context.Context, h Handshake) (SessionContext, error) {
	if h.UserID == "" || h.AppointmentID == "" {
		return SessionContext{}, ErrSessionNotScoped
	}

	appt, err := a.appointments.FindByID(ctx, h.AppointmentID)
	if err != nil || appt == nil {
		return SessionContext{}, ErrSessionNotScoped
	}

	// A video visit only exists once the appointment is a committed booking.
	if appt.Status != schedmodel.AppointmentStatusBooked {
		return SessionContext{}, ErrSessionNotScoped
	}

	// The connecting principal must be one of the two scoped participants.
	role, ok := participantRole(appt, h.UserID)
	if !ok {
		return SessionContext{}, ErrSessionNotScoped
	}

	scope := SessionContext{
		AppointmentID: appt.ID,
		PatientID:     appt.ScopedPatientID,
		ProviderID:    appt.ScopedProviderID,
		CallerRole:    role,
	}

	// When the handshake names a care relationship, it must resolve to an active
	// grant between this appointment's provider and patient.
	if h.CareRelationshipID != "" {
		if a.relationships == nil {
			return SessionContext{}, ErrSessionNotScoped
		}
		rel, err := a.relationships.FindByID(ctx, h.CareRelationshipID)
		if err != nil || rel == nil {
			return SessionContext{}, ErrSessionNotScoped
		}
		if rel.Status != authzmodel.RelationshipStatusActive ||
			rel.ProviderID != appt.ScopedProviderID ||
			rel.PatientID != appt.ScopedPatientID {
			return SessionContext{}, ErrSessionNotScoped
		}
		scope.CareRelationshipID = rel.ID
	}

	return scope, nil
}

// participantRole reports the role the principal holds on the appointment, or
// false when the principal is neither the scoped patient nor provider.
func participantRole(appt *schedmodel.AppointmentAggregate, userID string) (string, bool) {
	switch userID {
	case appt.ScopedPatientID:
		return "patient", true
	case appt.ScopedProviderID:
		return "provider", true
	default:
		return "", false
	}
}

// Compile-time assertion.
var _ SessionAuthorizer = (*AppointmentSessionAuthorizer)(nil)
