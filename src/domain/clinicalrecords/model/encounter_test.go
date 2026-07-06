package model

import (
	"errors"
	"testing"

	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/shared"
)

func validOpenEncounterCmd() OpenEncounterCmd {
	return OpenEncounterCmd{
		AppointmentId: "appointment-1",
		ProviderId:    "provider-1",
		PatientId:     "patient-1",
	}
}

func codedDiagnoses() []Diagnosis {
	return []Diagnosis{{Code: "J06.9", Description: "Acute upper respiratory infection"}}
}

func validSignSoapNoteCmd() SignSoapNoteCmd {
	return SignSoapNoteCmd{
		EncounterId: "encounter-1",
		ProviderId:  "provider-1",
		SoapNote:    "S: cough O: clear A: URI P: rest",
		Diagnoses:   codedDiagnoses(),
	}
}

// assertEncounterRejected checks that a command execution produced the expected
// sentinel error, emitted no events, buffered nothing and left the version at
// the supplied baseline.
func assertEncounterRejected(t *testing.T, agg *EncounterAggregate, events []shared.DomainEvent, err, wantErr error, wantVersion int) {
	t.Helper()
	if !errors.Is(err, wantErr) {
		t.Fatalf("error = %v, want %v", err, wantErr)
	}
	if len(events) != 0 {
		t.Fatalf("expected no events on rejection, got %d", len(events))
	}
	if got := agg.Events(); len(got) != 0 {
		t.Fatalf("expected no buffered events on rejection, got %d", len(got))
	}
	if agg.Version != wantVersion {
		t.Fatalf("expected version %d on rejection, got %d", wantVersion, agg.Version)
	}
}

// openedEncounter returns a freshly opened encounter (version 1) scoped to
// provider-1 / patient-1, ready for the next lifecycle step.
func openedEncounter(t *testing.T) *EncounterAggregate {
	t.Helper()
	agg := &EncounterAggregate{ID: "encounter-1"}
	if _, err := agg.Execute(validOpenEncounterCmd()); err != nil {
		t.Fatalf("open step failed: %v", err)
	}
	return agg
}

// ---------------------------------------------------------------------------
// OpenEncounterCmd
// ---------------------------------------------------------------------------

func TestEncounterExecuteOpenEmitsOpenedEvent(t *testing.T) {
	agg := &EncounterAggregate{ID: "encounter-1"}
	cmd := validOpenEncounterCmd()

	events, err := agg.Execute(cmd)
	if err != nil {
		t.Fatalf("Execute(OpenEncounterCmd) returned error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	evt, ok := events[0].(EncounterOpenedEvent)
	if !ok {
		t.Fatalf("event type = %T, want EncounterOpenedEvent", events[0])
	}
	if evt.Type() != EncounterOpenedEventType || evt.Type() != "encounter.opened" {
		t.Fatalf("unexpected event type %q", evt.Type())
	}
	if evt.AggregateID() != "encounter-1" || evt.EncounterID != "encounter-1" {
		t.Fatalf("event aggregate id = %q / %q, want encounter-1", evt.AggregateID(), evt.EncounterID)
	}
	if evt.AppointmentID != cmd.AppointmentId || evt.ProviderID != cmd.ProviderId || evt.PatientID != cmd.PatientId {
		t.Fatalf("event fields not copied from command: %+v", evt)
	}
	if evt.VideoRoomID != "vr-encounter-1" {
		t.Fatalf("event video room id = %q, want vr-encounter-1", evt.VideoRoomID)
	}

	if agg.Status != EncounterStatusOpen {
		t.Fatalf("aggregate status = %q, want %q", agg.Status, EncounterStatusOpen)
	}
	if agg.ScopedProviderID != cmd.ProviderId || agg.ScopedPatientID != cmd.PatientId {
		t.Fatalf("aggregate not scoped to participants: %+v", agg)
	}
	if agg.VideoRoomID != "vr-encounter-1" {
		t.Fatalf("aggregate video room id = %q, want vr-encounter-1", agg.VideoRoomID)
	}
	if agg.Version != 1 {
		t.Fatalf("aggregate version = %d, want 1", agg.Version)
	}
	if buffered := agg.Events(); len(buffered) != 1 {
		t.Fatalf("aggregate buffered %d events, want 1", len(buffered))
	}
}

func TestEncounterExecuteOpenRejectsMissingFields(t *testing.T) {
	tests := []struct {
		name    string
		cmd     OpenEncounterCmd
		wantErr error
	}{
		{
			name:    "missing appointment",
			cmd:     OpenEncounterCmd{ProviderId: "provider-1", PatientId: "patient-1"},
			wantErr: ErrMissingAppointment,
		},
		{
			name:    "missing provider",
			cmd:     OpenEncounterCmd{AppointmentId: "appointment-1", PatientId: "patient-1"},
			wantErr: ErrMissingProvider,
		},
		{
			name:    "missing patient",
			cmd:     OpenEncounterCmd{AppointmentId: "appointment-1", ProviderId: "provider-1"},
			wantErr: ErrMissingPatient,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agg := &EncounterAggregate{ID: "encounter-1"}
			events, err := agg.Execute(tt.cmd)
			assertEncounterRejected(t, agg, events, err, tt.wantErr, 0)
		})
	}
}

func TestEncounterExecuteOpenRejectsInvariantViolations(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*EncounterAggregate)
		wantErr error
	}{
		{
			name: "provider not scoped",
			mutate: func(a *EncounterAggregate) {
				a.ScopedProviderID = "other-provider"
			},
			wantErr: ErrParticipantNotScoped,
		},
		{
			name: "patient not scoped",
			mutate: func(a *EncounterAggregate) {
				a.ScopedPatientID = "other-patient"
			},
			wantErr: ErrParticipantNotScoped,
		},
		{
			name: "signed note immutable",
			mutate: func(a *EncounterAggregate) {
				a.Note = &ClinicalNote{Content: "sealed", Signed: true}
			},
			wantErr: ErrSignedNoteImmutable,
		},
		{
			name: "diagnosis uncoded",
			mutate: func(a *EncounterAggregate) {
				a.Diagnoses = []Diagnosis{{Description: "no code"}}
			},
			wantErr: ErrDiagnosisUncoded,
		},
		{
			name: "completed without signed note",
			mutate: func(a *EncounterAggregate) {
				a.Status = EncounterStatusCompleted
			},
			wantErr: ErrIncompleteWithoutSignedNote,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agg := &EncounterAggregate{ID: "encounter-1"}
			tt.mutate(agg)
			events, err := agg.Execute(validOpenEncounterCmd())
			assertEncounterRejected(t, agg, events, err, tt.wantErr, 0)
		})
	}
}

// ---------------------------------------------------------------------------
// SignSoapNoteCmd
// ---------------------------------------------------------------------------

func TestEncounterExecuteSignEmitsSignedEvent(t *testing.T) {
	agg := openedEncounter(t)
	cmd := validSignSoapNoteCmd()

	events, err := agg.Execute(cmd)
	if err != nil {
		t.Fatalf("Execute(SignSoapNoteCmd) returned error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	evt, ok := events[0].(SoapNoteSignedEvent)
	if !ok {
		t.Fatalf("event type = %T, want SoapNoteSignedEvent", events[0])
	}
	if evt.Type() != SoapNoteSignedEventType || evt.Type() != "encounter.note.signed" {
		t.Fatalf("unexpected event type %q", evt.Type())
	}
	if evt.AggregateID() != "encounter-1" || evt.EncounterID != "encounter-1" {
		t.Fatalf("event aggregate id = %q / %q, want encounter-1", evt.AggregateID(), evt.EncounterID)
	}
	if evt.ProviderID != cmd.ProviderId || evt.SoapNote != cmd.SoapNote {
		t.Fatalf("event fields not copied from command: %+v", evt)
	}
	if len(evt.Diagnoses) != 1 || evt.Diagnoses[0].Code != "J06.9" {
		t.Fatalf("event diagnoses = %+v, want coded J06.9", evt.Diagnoses)
	}

	if agg.Note == nil || !agg.Note.Signed || agg.Note.Content != cmd.SoapNote {
		t.Fatalf("aggregate note not signed/sealed: %+v", agg.Note)
	}
	if len(agg.Diagnoses) != 1 || agg.Diagnoses[0].Code != "J06.9" {
		t.Fatalf("aggregate diagnoses = %+v, want coded J06.9", agg.Diagnoses)
	}
	if agg.Status != EncounterStatusOpen {
		t.Fatalf("aggregate status = %q, want still open", agg.Status)
	}
	if agg.Version != 2 {
		t.Fatalf("aggregate version = %d, want 2", agg.Version)
	}
}

func TestEncounterExecuteSignRejectsMissingFields(t *testing.T) {
	tests := []struct {
		name    string
		cmd     SignSoapNoteCmd
		wantErr error
	}{
		{
			name:    "missing encounter",
			cmd:     SignSoapNoteCmd{ProviderId: "provider-1", SoapNote: "note", Diagnoses: codedDiagnoses()},
			wantErr: ErrMissingEncounter,
		},
		{
			name:    "missing soap note",
			cmd:     SignSoapNoteCmd{EncounterId: "encounter-1", ProviderId: "provider-1", Diagnoses: codedDiagnoses()},
			wantErr: ErrMissingSoapNote,
		},
		{
			name:    "missing diagnoses",
			cmd:     SignSoapNoteCmd{EncounterId: "encounter-1", ProviderId: "provider-1", SoapNote: "note"},
			wantErr: ErrMissingDiagnoses,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agg := &EncounterAggregate{ID: "encounter-1"}
			events, err := agg.Execute(tt.cmd)
			assertEncounterRejected(t, agg, events, err, tt.wantErr, 0)
		})
	}
}

func TestEncounterExecuteSignRejectsInvariantViolations(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*EncounterAggregate)
		cmd     SignSoapNoteCmd
		wantErr error
	}{
		{
			name: "provider not scoped",
			mutate: func(a *EncounterAggregate) {
				a.ScopedProviderID = "other-provider"
			},
			cmd:     validSignSoapNoteCmd(),
			wantErr: ErrParticipantNotScoped,
		},
		{
			name: "signed note immutable",
			mutate: func(a *EncounterAggregate) {
				a.Note = &ClinicalNote{Content: "sealed", Signed: true}
			},
			cmd:     validSignSoapNoteCmd(),
			wantErr: ErrSignedNoteImmutable,
		},
		{
			name:   "diagnosis uncoded",
			mutate: func(a *EncounterAggregate) {},
			cmd: SignSoapNoteCmd{
				EncounterId: "encounter-1",
				ProviderId:  "provider-1",
				SoapNote:    "note",
				Diagnoses:   []Diagnosis{{Description: "no code"}},
			},
			wantErr: ErrDiagnosisUncoded,
		},
		{
			name: "completed without signed note",
			mutate: func(a *EncounterAggregate) {
				a.Status = EncounterStatusCompleted
			},
			cmd:     validSignSoapNoteCmd(),
			wantErr: ErrIncompleteWithoutSignedNote,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agg := &EncounterAggregate{ID: "encounter-1"}
			tt.mutate(agg)
			events, err := agg.Execute(tt.cmd)
			assertEncounterRejected(t, agg, events, err, tt.wantErr, 0)
		})
	}
}

// ---------------------------------------------------------------------------
// CompleteEncounterCmd
// ---------------------------------------------------------------------------

func TestEncounterExecuteCompleteEmitsCompletedEvent(t *testing.T) {
	// Full lifecycle: open then document then complete.
	agg := openedEncounter(t)
	if _, err := agg.Execute(validSignSoapNoteCmd()); err != nil {
		t.Fatalf("sign step failed: %v", err)
	}

	events, err := agg.Execute(CompleteEncounterCmd{EncounterId: "encounter-1", ProviderId: "provider-1"})
	if err != nil {
		t.Fatalf("Execute(CompleteEncounterCmd) returned error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	evt, ok := events[0].(EncounterCompletedEvent)
	if !ok {
		t.Fatalf("event type = %T, want EncounterCompletedEvent", events[0])
	}
	if evt.Type() != EncounterCompletedEventType || evt.Type() != "encounter.completed" {
		t.Fatalf("unexpected event type %q", evt.Type())
	}
	if evt.AggregateID() != "encounter-1" || evt.EncounterID != "encounter-1" {
		t.Fatalf("event aggregate id = %q / %q, want encounter-1", evt.AggregateID(), evt.EncounterID)
	}
	if evt.ProviderID != "provider-1" {
		t.Fatalf("event provider id = %q, want provider-1", evt.ProviderID)
	}

	if agg.Status != EncounterStatusCompleted {
		t.Fatalf("aggregate status = %q, want %q", agg.Status, EncounterStatusCompleted)
	}
	if agg.Version != 3 {
		t.Fatalf("aggregate version = %d, want 3", agg.Version)
	}
}

func TestEncounterExecuteCompleteRejectsMissingEncounter(t *testing.T) {
	agg := &EncounterAggregate{ID: "encounter-1"}
	events, err := agg.Execute(CompleteEncounterCmd{ProviderId: "provider-1"})
	assertEncounterRejected(t, agg, events, err, ErrMissingEncounter, 0)
}

func TestEncounterExecuteCompleteRejectsInvariantViolations(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*EncounterAggregate)
		wantErr error
	}{
		{
			name: "provider not scoped",
			mutate: func(a *EncounterAggregate) {
				a.ScopedProviderID = "other-provider"
				a.Note = &ClinicalNote{Content: "sealed", Signed: true}
			},
			wantErr: ErrParticipantNotScoped,
		},
		{
			name: "already completed is sealed",
			mutate: func(a *EncounterAggregate) {
				a.ScopedProviderID = "provider-1"
				a.Status = EncounterStatusCompleted
				a.Note = &ClinicalNote{Content: "sealed", Signed: true}
			},
			wantErr: ErrSignedNoteImmutable,
		},
		{
			name: "diagnosis uncoded",
			mutate: func(a *EncounterAggregate) {
				a.ScopedProviderID = "provider-1"
				a.Note = &ClinicalNote{Content: "sealed", Signed: true}
				a.Diagnoses = []Diagnosis{{Description: "no code"}}
			},
			wantErr: ErrDiagnosisUncoded,
		},
		{
			name: "incomplete without signed note",
			mutate: func(a *EncounterAggregate) {
				a.ScopedProviderID = "provider-1"
				a.Status = EncounterStatusOpen
			},
			wantErr: ErrIncompleteWithoutSignedNote,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agg := &EncounterAggregate{ID: "encounter-1"}
			tt.mutate(agg)
			events, err := agg.Execute(CompleteEncounterCmd{EncounterId: "encounter-1", ProviderId: "provider-1"})
			assertEncounterRejected(t, agg, events, err, tt.wantErr, 0)
		})
	}
}

// ---------------------------------------------------------------------------
// AppendAddendumCmd
// ---------------------------------------------------------------------------

func TestEncounterExecuteAppendAddendumEmitsAppendedEvent(t *testing.T) {
	// Lifecycle: open then document then amend.
	agg := openedEncounter(t)
	if _, err := agg.Execute(validSignSoapNoteCmd()); err != nil {
		t.Fatalf("sign step failed: %v", err)
	}

	events, err := agg.Execute(AppendAddendumCmd{
		EncounterId:  "encounter-1",
		AddendumText: "correction: BP was 120/80",
		AuthorId:     "provider-1",
	})
	if err != nil {
		t.Fatalf("Execute(AppendAddendumCmd) returned error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	evt, ok := events[0].(AddendumAppendedEvent)
	if !ok {
		t.Fatalf("event type = %T, want AddendumAppendedEvent", events[0])
	}
	if evt.Type() != AddendumAppendedEventType || evt.Type() != "encounter.addendum.appended" {
		t.Fatalf("unexpected event type %q", evt.Type())
	}
	if evt.AggregateID() != "encounter-1" || evt.EncounterID != "encounter-1" {
		t.Fatalf("event aggregate id = %q / %q, want encounter-1", evt.AggregateID(), evt.EncounterID)
	}
	if evt.AuthorID != "provider-1" || evt.AddendumText != "correction: BP was 120/80" {
		t.Fatalf("event fields not copied from command: %+v", evt)
	}

	if len(agg.Addenda) != 1 {
		t.Fatalf("expected 1 addendum, got %d", len(agg.Addenda))
	}
	if agg.Addenda[0].Text != "correction: BP was 120/80" || agg.Addenda[0].AuthorID != "provider-1" {
		t.Fatalf("addendum not recorded correctly: %+v", agg.Addenda[0])
	}
	// Signed note body is left untouched.
	if agg.Note == nil || agg.Note.Content != validSignSoapNoteCmd().SoapNote {
		t.Fatalf("signed note body was mutated: %+v", agg.Note)
	}
	if agg.Version != 3 {
		t.Fatalf("aggregate version = %d, want 3", agg.Version)
	}
}

func TestEncounterExecuteAppendAddendumRejectsMissingFields(t *testing.T) {
	tests := []struct {
		name    string
		cmd     AppendAddendumCmd
		wantErr error
	}{
		{
			name:    "missing encounter",
			cmd:     AppendAddendumCmd{AddendumText: "text", AuthorId: "provider-1"},
			wantErr: ErrMissingEncounter,
		},
		{
			name:    "missing addendum text",
			cmd:     AppendAddendumCmd{EncounterId: "encounter-1", AuthorId: "provider-1"},
			wantErr: ErrMissingAddendumText,
		},
		{
			name:    "missing author",
			cmd:     AppendAddendumCmd{EncounterId: "encounter-1", AddendumText: "text"},
			wantErr: ErrMissingAuthor,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agg := &EncounterAggregate{ID: "encounter-1"}
			events, err := agg.Execute(tt.cmd)
			assertEncounterRejected(t, agg, events, err, tt.wantErr, 0)
		})
	}
}

func TestEncounterExecuteAppendAddendumRejectsInvariantViolations(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*EncounterAggregate)
		wantErr error
	}{
		{
			name: "author not scoped",
			mutate: func(a *EncounterAggregate) {
				a.ScopedProviderID = "other-provider"
				a.Note = &ClinicalNote{Content: "sealed", Signed: true}
			},
			wantErr: ErrParticipantNotScoped,
		},
		{
			name: "note missing cannot amend",
			mutate: func(a *EncounterAggregate) {
				a.ScopedProviderID = "provider-1"
				a.Note = nil
			},
			wantErr: ErrSignedNoteImmutable,
		},
		{
			name: "unsigned note cannot amend",
			mutate: func(a *EncounterAggregate) {
				a.ScopedProviderID = "provider-1"
				a.Note = &ClinicalNote{Content: "draft", Signed: false}
			},
			wantErr: ErrSignedNoteImmutable,
		},
		{
			name: "diagnosis uncoded",
			mutate: func(a *EncounterAggregate) {
				a.ScopedProviderID = "provider-1"
				a.Note = &ClinicalNote{Content: "sealed", Signed: true}
				a.Diagnoses = []Diagnosis{{Description: "no code"}}
			},
			wantErr: ErrDiagnosisUncoded,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agg := &EncounterAggregate{ID: "encounter-1"}
			tt.mutate(agg)
			events, err := agg.Execute(AppendAddendumCmd{
				EncounterId:  "encounter-1",
				AddendumText: "correction",
				AuthorId:     "provider-1",
			})
			assertEncounterRejected(t, agg, events, err, tt.wantErr, 0)
		})
	}
}

// ---------------------------------------------------------------------------
// Execute dispatch + aggregate-root helpers
// ---------------------------------------------------------------------------

func TestEncounterExecuteRejectsUnknownCommand(t *testing.T) {
	agg := &EncounterAggregate{ID: "encounter-1"}

	events, err := agg.Execute(struct{ Unrecognized string }{Unrecognized: "x"})
	if !errors.Is(err, shared.ErrUnknownCommand) {
		t.Fatalf("error = %v, want %v", err, shared.ErrUnknownCommand)
	}
	if events != nil {
		t.Fatalf("expected nil events, got %v", events)
	}
	if agg.Version != 0 {
		t.Fatalf("expected version 0, got %d", agg.Version)
	}
	if got := agg.Events(); len(got) != 0 {
		t.Fatalf("expected no buffered events, got %d", len(got))
	}
}

func TestEncounterAggregateRootHelpers(t *testing.T) {
	agg := openedEncounter(t)

	if agg.GetVersion() != 1 {
		t.Fatalf("expected GetVersion 1, got %d", agg.GetVersion())
	}
	if len(agg.Events()) != 1 {
		t.Fatalf("expected 1 buffered event, got %d", len(agg.Events()))
	}

	agg.ClearEvents()
	if len(agg.Events()) != 0 {
		t.Fatalf("expected events cleared, got %d", len(agg.Events()))
	}
	if agg.GetVersion() != 1 {
		t.Fatalf("expected version unchanged after ClearEvents, got %d", agg.GetVersion())
	}
}
