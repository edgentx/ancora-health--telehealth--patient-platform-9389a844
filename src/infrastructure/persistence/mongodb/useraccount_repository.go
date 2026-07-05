package mongodb

import (
	"context"

	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/identityandaccess/model"
	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/identityandaccess/repository"
	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/infrastructure/crypto"
)

// UserAccountCollection is the MongoDB collection user accounts persist to.
const UserAccountCollection = "user_accounts"

// userAccountDoc is the BSON persistence shape of a UserAccountAggregate. Email
// is PII and is tagged `phi:"true"` so the codec encrypts it at rest; the
// lockout and reset-token timestamps are stored as epoch millis so they survive
// the codec's bson.M round trip. version backs the store's optimistic-
// concurrency guard, while aggregate_version preserves the domain aggregate's
// own version across a reload.
type userAccountDoc struct {
	DocID string `bson:"_id"`
	Ver   int    `bson:"version"`

	AggregateVersion int    `bson:"aggregate_version"`
	TenantID         string `bson:"tenant_id"`
	Email            string `bson:"email" phi:"true"`
	Role             string `bson:"role"`

	EmailRegistered         bool  `bson:"email_registered"`
	LockedUntilMillis       int64 `bson:"locked_until_ms"`
	ResetTokenConsumed      bool  `bson:"reset_token_consumed"`
	ResetTokenExpiresMillis int64 `bson:"reset_token_expires_ms"`
	MFAEnrolled             bool  `bson:"mfa_enrolled"`
	SecondFactorVerified    bool  `bson:"second_factor_verified"`
	CredentialVerified      bool  `bson:"credential_verified"`
	FailedLoginAttempts     int   `bson:"failed_login_attempts"`
}

func (d *userAccountDoc) ID() string       { return d.DocID }
func (d *userAccountDoc) Version() int     { return d.Ver }
func (d *userAccountDoc) SetVersion(v int) { d.Ver = v }

// UserAccountRepository is the MongoDB adapter for the UserAccountRepository
// port. PII on the account (the login email) is encrypted through the S-68
// crypto codec, so the stored document never carries it in plaintext.
type UserAccountRepository struct {
	store *encryptedStore
}

// NewUserAccountRepository builds a repository over a document store and codec.
func NewUserAccountRepository(store DocumentStore, codec *crypto.Codec) *UserAccountRepository {
	return &UserAccountRepository{store: newEncryptedStore(store, codec, UserAccountCollection)}
}

// Save persists the account, encrypting its PII before it reaches the store.
func (r *UserAccountRepository) Save(ctx context.Context, a *model.UserAccountAggregate) error {
	return r.store.save(ctx, userAccountToDoc(a))
}

// FindByID loads and decrypts the account with the given id.
func (r *UserAccountRepository) FindByID(ctx context.Context, id string) (*model.UserAccountAggregate, error) {
	doc := &userAccountDoc{}
	if err := r.store.load(ctx, id, doc); err != nil {
		return nil, err
	}
	return userAccountFromDoc(doc), nil
}

// userAccountToDoc maps the aggregate onto its persistence document.
func userAccountToDoc(a *model.UserAccountAggregate) *userAccountDoc {
	return &userAccountDoc{
		DocID:                   a.ID,
		AggregateVersion:        a.GetVersion(),
		TenantID:                a.TenantID,
		Email:                   a.Email,
		Role:                    a.Role,
		EmailRegistered:         a.EmailRegistered,
		LockedUntilMillis:       epochMillis(a.LockedUntil),
		ResetTokenConsumed:      a.ResetTokenConsumed,
		ResetTokenExpiresMillis: epochMillis(a.ResetTokenExpiresAt),
		MFAEnrolled:             a.MFAEnrolled,
		SecondFactorVerified:    a.SecondFactorVerified,
		CredentialVerified:      a.CredentialVerified,
		FailedLoginAttempts:     a.FailedLoginAttempts,
	}
}

// userAccountFromDoc reconstructs the aggregate from its persistence document.
func userAccountFromDoc(d *userAccountDoc) *model.UserAccountAggregate {
	a := &model.UserAccountAggregate{
		ID:                   d.DocID,
		TenantID:             d.TenantID,
		Email:                d.Email,
		Role:                 d.Role,
		EmailRegistered:      d.EmailRegistered,
		LockedUntil:          fromEpochMillis(d.LockedUntilMillis),
		ResetTokenConsumed:   d.ResetTokenConsumed,
		ResetTokenExpiresAt:  fromEpochMillis(d.ResetTokenExpiresMillis),
		MFAEnrolled:          d.MFAEnrolled,
		SecondFactorVerified: d.SecondFactorVerified,
		CredentialVerified:   d.CredentialVerified,
		FailedLoginAttempts:  d.FailedLoginAttempts,
	}
	a.Version = d.AggregateVersion
	return a
}

// Compile-time assertion that the adapter satisfies its domain port.
var _ repository.UserAccountRepository = (*UserAccountRepository)(nil)
