package mongodb

import (
	"context"

	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/patientengagement/model"
	engagerepo "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/patientengagement/repository"
)

// messageThreadsCollection is the collection message-thread documents live in.
const messageThreadsCollection = "message_threads"

// messageThreadDoc is the at-rest projection of a MessageThreadAggregate. The
// aggregate tracks thread metadata and a posted-message counter — the message
// bodies themselves live outside the aggregate — so no PHI content is persisted here.
type messageThreadDoc struct {
	DocID   string `bson:"_id"`
	Ver     int    `bson:"version"`
	StatusV string `bson:"status"`

	ScopedPatientID         string   `bson:"scoped_patient_id"`
	ScopedCareTeamMemberIDs []string `bson:"scoped_care_team_member_ids"`
	Subject                 string   `bson:"subject"`

	AccessNotRestricted      bool `bson:"access_not_restricted"`
	ContentNotEncrypted      bool `bson:"content_not_encrypted"`
	NoActiveCareRelationship bool `bson:"no_active_care_relationship"`
	PostedMessageCount       int  `bson:"posted_message_count"`
}

func (d *messageThreadDoc) ID() string       { return d.DocID }
func (d *messageThreadDoc) Version() int     { return d.Ver }
func (d *messageThreadDoc) SetVersion(v int) { d.Ver = v }

// MessageThreadRepository is the MongoDB-backed MessageThreadRepository.
type MessageThreadRepository struct {
	base *BaseRepository
}

var _ engagerepo.MessageThreadRepository = (*MessageThreadRepository)(nil)

// NewMessageThreadRepository builds a message-thread repository over a store.
func NewMessageThreadRepository(store DocumentStore) *MessageThreadRepository {
	return &MessageThreadRepository{base: NewBaseRepository(store, messageThreadsCollection)}
}

// Save persists the message-thread aggregate with optimistic concurrency.
func (r *MessageThreadRepository) Save(ctx context.Context, a *model.MessageThreadAggregate) error {
	doc := &messageThreadDoc{
		DocID:                    a.ID,
		Ver:                      a.GetVersion(),
		StatusV:                  string(a.Status),
		ScopedPatientID:          a.ScopedPatientID,
		ScopedCareTeamMemberIDs:  a.ScopedCareTeamMemberIDs,
		Subject:                  a.Subject,
		AccessNotRestricted:      a.AccessNotRestricted,
		ContentNotEncrypted:      a.ContentNotEncrypted,
		NoActiveCareRelationship: a.NoActiveCareRelationship,
		PostedMessageCount:       a.PostedMessageCount,
	}
	return saveAggregate(ctx, r.base, doc, a)
}

// FindByID loads a message-thread aggregate by identity.
func (r *MessageThreadRepository) FindByID(ctx context.Context, id string) (*model.MessageThreadAggregate, error) {
	var doc messageThreadDoc
	if err := r.base.FindByID(ctx, id, &doc); err != nil {
		return nil, err
	}
	a := &model.MessageThreadAggregate{
		ID:                       doc.DocID,
		Status:                   model.MessageThreadStatus(doc.StatusV),
		ScopedPatientID:          doc.ScopedPatientID,
		ScopedCareTeamMemberIDs:  doc.ScopedCareTeamMemberIDs,
		Subject:                  doc.Subject,
		AccessNotRestricted:      doc.AccessNotRestricted,
		ContentNotEncrypted:      doc.ContentNotEncrypted,
		NoActiveCareRelationship: doc.NoActiveCareRelationship,
		PostedMessageCount:       doc.PostedMessageCount,
	}
	a.Version = doc.Ver
	return a, nil
}
