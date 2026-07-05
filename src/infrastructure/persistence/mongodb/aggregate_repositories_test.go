package mongodb

import (
	"bytes"
	"context"
	"errors"
	"os"
	"strconv"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson"

	auditmodel "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/auditandcompliance/model"
	authzmodel "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/authorization/model"
	identitymodel "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/identityandaccess/model"
	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/infrastructure/crypto"
)

// newTestCodec builds a Codec backed by a deterministic master key so the
// encrypted-repository tests are self-contained.
func newTestCodec(t *testing.T) *crypto.Codec {
	t.Helper()
	master := bytes.Repeat([]byte{0x2a}, crypto.KeySize)
	env, err := crypto.NewAESKeyEnvelope("test-master-v1", master)
	if err != nil {
		t.Fatalf("NewAESKeyEnvelope: %v", err)
	}
	return crypto.NewCodec(crypto.NewFieldCipher(env))
}

// registeredAccount builds a persistable UserAccountAggregate by running the
// RegisterUserCmd, mirroring how the account reaches the repository in practice.
func registeredAccount(t *testing.T, id, email string) *identitymodel.UserAccountAggregate {
	t.Helper()
	a := &identitymodel.UserAccountAggregate{ID: id}
	if _, err := a.Execute(identitymodel.RegisterUserCmd{
		Email:    email,
		Password: "s3cret-pass",
		Role:     "clinician",
		TenantId: "tenant-1",
	}); err != nil {
		t.Fatalf("RegisterUserCmd: %v", err)
	}
	return a
}

func TestUserAccountRepository_RoundTrip_EncryptsEmailAtRest(t *testing.T) {
	ctx := context.Background()
	store := NewMemStore()
	repo := NewUserAccountRepository(store, newTestCodec(t))

	acct := registeredAccount(t, "user-1", "clinician@ancora.health")
	if err := repo.Save(ctx, acct); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// At rest the email must be an encrypted sub-document, never the plaintext.
	var raw bson.M
	if err := store.FindOne(ctx, "user-1", &raw); err != nil {
		t.Fatalf("raw FindOne: %v", err)
	}
	if s, ok := raw["email"].(string); ok {
		t.Fatalf("email persisted as plaintext string: %q", s)
	}
	stored, err := bson.Marshal(raw)
	if err != nil {
		t.Fatalf("marshal raw: %v", err)
	}
	if bytes.Contains(stored, []byte("clinician@ancora.health")) {
		t.Fatal("plaintext PII present in stored BSON")
	}

	got, err := repo.FindByID(ctx, "user-1")
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if got.Email != "clinician@ancora.health" {
		t.Fatalf("email round trip mismatch: %q", got.Email)
	}
	if !got.EmailRegistered || got.Role != "clinician" || got.TenantID != "tenant-1" {
		t.Fatalf("non-PII fields not restored: %+v", got)
	}
}

func TestUserAccountRepository_Save_IsUpsert(t *testing.T) {
	ctx := context.Background()
	repo := NewUserAccountRepository(NewMemStore(), newTestCodec(t))

	acct := registeredAccount(t, "user-2", "a@ancora.health")
	if err := repo.Save(ctx, acct); err != nil {
		t.Fatalf("first Save: %v", err)
	}
	// A second Save of the same identity must update, not fail on a duplicate id.
	if _, err := acct.Execute(identitymodel.LockAccountCmd{AccountId: "user-2", Reason: "manual"}); err == nil {
		// LockAccount is rejected once EmailRegistered; ignore the domain error and
		// simply re-save the current state to exercise the update path.
	}
	if err := repo.Save(ctx, acct); err != nil {
		t.Fatalf("second Save (update): %v", err)
	}
}

func TestUserAccountRepository_FindByID_NotFound(t *testing.T) {
	repo := NewUserAccountRepository(NewMemStore(), newTestCodec(t))
	if _, err := repo.FindByID(context.Background(), "missing"); !errors.Is(err, ErrDocumentNotFound) {
		t.Fatalf("expected ErrDocumentNotFound, got %v", err)
	}
}

func TestSessionRepository_RoundTrip(t *testing.T) {
	ctx := context.Background()
	repo := NewSessionRepository(NewMemStore())

	sess := &identitymodel.SessionAggregate{ID: "sess-1", Authenticated: true}
	if _, err := sess.Execute(identitymodel.IssueSessionCmd{
		AccountId:         "user-1",
		Role:              "clinician",
		DeviceFingerprint: "device-abc",
		RequestedLifetime: time.Hour,
	}); err != nil {
		t.Fatalf("IssueSessionCmd: %v", err)
	}
	if err := repo.Save(ctx, sess); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := repo.FindByID(ctx, "sess-1")
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if !got.Issued || got.AccountID != "user-1" || got.Role != "clinician" || got.DeviceFingerprint != "device-abc" {
		t.Fatalf("session round trip mismatch: %+v", got)
	}
	if got.ExpiresAt.IsZero() {
		t.Fatal("expected a non-zero session expiry (TTL boundary)")
	}
}

func TestCryptoKeyEnvelopeRepository_RoundTrip_NoPlaintextKeyMaterial(t *testing.T) {
	ctx := context.Background()
	store := NewMemStore()
	repo := NewCryptoKeyEnvelopeRepository(store)

	env := &auditmodel.CryptoKeyEnvelopeAggregate{
		ID:              "env-1",
		MasterKeyActive: true,
		ExpiresAt:       time.Now().Add(24 * time.Hour),
	}
	if err := repo.Save(ctx, env); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// The persisted document carries only lifecycle metadata; no field name
	// hints at raw key bytes ever being written.
	var raw bson.M
	if err := store.FindOne(ctx, "env-1", &raw); err != nil {
		t.Fatalf("raw FindOne: %v", err)
	}
	for _, forbidden := range []string{"dek", "master_key", "plaintext", "key_material"} {
		if _, ok := raw[forbidden]; ok {
			t.Fatalf("unexpected key-material field %q persisted", forbidden)
		}
	}

	got, err := repo.FindByID(ctx, "env-1")
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if !got.MasterKeyActive || got.Revoked {
		t.Fatalf("envelope metadata not restored: %+v", got)
	}
}

func TestAuthorizationPolicyRepository_RoundTrip_EncryptsAuthor(t *testing.T) {
	ctx := context.Background()
	store := NewMemStore()
	repo := NewAuthorizationPolicyRepository(store, newTestCodec(t))

	policy := &authzmodel.AuthorizationPolicyAggregate{ID: "pol-1"}
	if _, err := policy.Execute(authzmodel.PublishPolicyVersionCmd{
		RegoBundle: "package authz\n default allow = false",
		Author:     "dr.house@ancora.health",
	}); err != nil {
		t.Fatalf("PublishPolicyVersionCmd: %v", err)
	}
	if err := repo.Save(ctx, policy); err != nil {
		t.Fatalf("Save: %v", err)
	}

	var raw bson.M
	if err := store.FindOne(ctx, "pol-1", &raw); err != nil {
		t.Fatalf("raw FindOne: %v", err)
	}
	if s, ok := raw["author"].(string); ok {
		t.Fatalf("author persisted as plaintext: %q", s)
	}

	got, err := repo.FindByID(ctx, "pol-1")
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if got.Author != "dr.house@ancora.health" || got.Status != authzmodel.PolicyStatusPublished {
		t.Fatalf("policy round trip mismatch: %+v", got)
	}
}

func TestCareRelationshipRepository_RoundTrip_EncryptsPHI(t *testing.T) {
	ctx := context.Background()
	store := NewMemStore()
	repo := NewCareRelationshipRepository(store, newTestCodec(t))

	rel := &authzmodel.CareRelationshipAggregate{ID: "rel-1"}
	if _, err := rel.Execute(authzmodel.EstablishCareRelationshipCmd{
		ProviderID: "provider-42",
		PatientID:  "patient-99",
		ClinicID:   "clinic-7",
	}); err != nil {
		t.Fatalf("EstablishCareRelationshipCmd: %v", err)
	}
	if err := repo.Save(ctx, rel); err != nil {
		t.Fatalf("Save: %v", err)
	}

	var raw bson.M
	if err := store.FindOne(ctx, "rel-1", &raw); err != nil {
		t.Fatalf("raw FindOne: %v", err)
	}
	stored, _ := bson.Marshal(raw)
	for _, phi := range []string{"provider-42", "patient-99"} {
		if bytes.Contains(stored, []byte(phi)) {
			t.Fatalf("plaintext PHI %q present in stored BSON", phi)
		}
	}
	// Clinic scoping is not PHI and remains queryable in the clear.
	if raw["clinic_id"] != "clinic-7" {
		t.Fatalf("clinic id should persist in the clear, got %v", raw["clinic_id"])
	}

	got, err := repo.FindByID(ctx, "rel-1")
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if got.ProviderID != "provider-42" || got.PatientID != "patient-99" || got.Status != authzmodel.RelationshipStatusActive {
		t.Fatalf("relationship round trip mismatch: %+v", got)
	}
}

// appendableTrail builds a trail of n chained entries via AppendAuditEntryCmd.
func appendableTrail(t *testing.T, id string, n int) *auditmodel.AuditTrailAggregate {
	t.Helper()
	trail := &auditmodel.AuditTrailAggregate{ID: id}
	base := time.Now().Truncate(time.Second)
	for i := 0; i < n; i++ {
		if _, err := trail.Execute(auditmodel.AppendAuditEntryCmd{
			ActorContext: "actor-" + strconv.Itoa(i),
			ResourceRef:  "record-" + strconv.Itoa(i),
			Action:       "record.read",
			OccurredAt:   base.Add(time.Duration(i) * time.Second),
			PrevHash:     trail.HeadHash(),
		}); err != nil {
			t.Fatalf("AppendAuditEntryCmd[%d]: %v", i, err)
		}
	}
	return trail
}

func TestAuditTrailRepository_RoundTrip_And_ChainIntegrity(t *testing.T) {
	ctx := context.Background()
	repo := NewAuditTrailRepository(NewMemAuditEntryCollection())

	trail := appendableTrail(t, "trail-1", 3)
	if err := repo.Save(ctx, trail); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := repo.FindByID(ctx, "trail-1")
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if len(loaded.Entries()) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(loaded.Entries()))
	}

	events, err := loaded.Execute(auditmodel.VerifyChainIntegrityCmd{FromSequence: 1, ToSequence: 3})
	if err != nil {
		t.Fatalf("VerifyChainIntegrityCmd: %v", err)
	}
	if _, ok := events[0].(auditmodel.ChainIntegrityVerifiedEvent); !ok {
		t.Fatalf("expected ChainIntegrityVerifiedEvent, got %T", events[0])
	}
}

func TestAuditTrailRepository_DetectsTampering(t *testing.T) {
	ctx := context.Background()
	repo := NewAuditTrailRepository(NewMemAuditEntryCollection())
	if err := repo.Save(ctx, appendableTrail(t, "trail-2", 3)); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := repo.FindByID(ctx, "trail-2")
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}

	// Alter an entry's sealed content while leaving its hash and chain links
	// intact — the verification routine must recompute the hash and flag it.
	entries := loaded.Entries()
	entries[1].ActorContext = "smuggled-actor"
	tampered := auditmodel.RehydrateAuditTrail("trail-2", entries)

	events, err := tampered.Execute(auditmodel.VerifyChainIntegrityCmd{FromSequence: 1, ToSequence: 3})
	if err != nil {
		t.Fatalf("VerifyChainIntegrityCmd: %v", err)
	}
	detected, ok := events[0].(auditmodel.ChainTamperingDetectedEvent)
	if !ok {
		t.Fatalf("expected ChainTamperingDetectedEvent, got %T", events[0])
	}
	if detected.TamperedAt != 2 {
		t.Fatalf("expected tampering flagged at sequence 2, got %d", detected.TamperedAt)
	}
}

func TestAuditTrailRepository_AppendOnly_RejectsRewrite(t *testing.T) {
	ctx := context.Background()
	repo := NewAuditTrailRepository(NewMemAuditEntryCollection())
	if err := repo.Save(ctx, appendableTrail(t, "trail-3", 2)); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := repo.FindByID(ctx, "trail-3")
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	// Attempt to rewrite an already-sealed entry (different content ⇒ different
	// hash) and re-save. The append-only repository must reject it.
	entries := loaded.Entries()
	entries[0].Hash = "0000000000000000000000000000000000000000000000000000000000000000"
	rewritten := auditmodel.RehydrateAuditTrail("trail-3", entries)

	if err := repo.Save(ctx, rewritten); !errors.Is(err, auditmodel.ErrAuditEntryImmutable) {
		t.Fatalf("expected ErrAuditEntryImmutable on rewrite, got %v", err)
	}
}

func TestAuditTrailRepository_Save_AppendsNewEntriesIdempotently(t *testing.T) {
	ctx := context.Background()
	repo := NewAuditTrailRepository(NewMemAuditEntryCollection())

	trail := appendableTrail(t, "trail-4", 2)
	if err := repo.Save(ctx, trail); err != nil {
		t.Fatalf("first Save: %v", err)
	}
	// Extend the same trail and re-save: existing entries are unchanged (no-op),
	// the new one is appended.
	if _, err := trail.Execute(auditmodel.AppendAuditEntryCmd{
		ActorContext: "actor-2",
		ResourceRef:  "record-2",
		Action:       "record.read",
		OccurredAt:   time.Now(),
		PrevHash:     trail.HeadHash(),
	}); err != nil {
		t.Fatalf("append third: %v", err)
	}
	if err := repo.Save(ctx, trail); err != nil {
		t.Fatalf("second Save: %v", err)
	}

	loaded, err := repo.FindByID(ctx, "trail-4")
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if len(loaded.Entries()) != 3 {
		t.Fatalf("expected 3 entries after idempotent re-save, got %d", len(loaded.Entries()))
	}
}

// TestIntegration_AggregateRepositories exercises the encrypted account
// repository and the append-only audit repository against a real MongoDB
// instance. It is skipped unless MONGODB_URI is set, keeping the default run
// hermetic (mirroring TestIntegration_MongoRoundTrip).
func TestIntegration_AggregateRepositories(t *testing.T) {
	uri := os.Getenv("MONGODB_URI")
	if uri == "" {
		t.Skip("MONGODB_URI not set; skipping live MongoDB integration test")
	}

	ctx := context.Background()
	cfg, err := ConfigFromEnv()
	if err != nil {
		t.Fatalf("ConfigFromEnv: %v", err)
	}
	client, err := NewClient(ctx, cfg)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	t.Cleanup(func() { _ = client.Disconnect(context.Background()) })

	suffix := strconv.FormatInt(time.Now().UnixNano(), 10)
	userColl := client.Collection("s69_users_" + suffix)
	auditColl := client.Collection("s69_audit_" + suffix)
	t.Cleanup(func() {
		_ = userColl.Drop(context.Background())
		_ = auditColl.Drop(context.Background())
	})

	// Encrypted user account: CRUD + at-rest ciphertext.
	userRepo := NewUserAccountRepository(NewMongoStore(userColl), newTestCodec(t))
	acct := registeredAccount(t, "user-it", "it@ancora.health")
	if err := userRepo.Save(ctx, acct); err != nil {
		t.Fatalf("user Save: %v", err)
	}
	var rawUser bson.M
	if err := userColl.FindOne(ctx, bson.M{"_id": "user-it"}).Decode(&rawUser); err != nil {
		t.Fatalf("raw user FindOne: %v", err)
	}
	if s, ok := rawUser["email"].(string); ok {
		t.Fatalf("email stored in plaintext at rest: %q", s)
	}
	gotUser, err := userRepo.FindByID(ctx, "user-it")
	if err != nil {
		t.Fatalf("user FindByID: %v", err)
	}
	if gotUser.Email != "it@ancora.health" {
		t.Fatalf("user email round trip mismatch: %q", gotUser.Email)
	}

	// Append-only audit trail: CRUD, hash-chain verify, rewrite rejection.
	auditRepo := NewMongoAuditTrailRepository(auditColl)
	if err := auditRepo.Save(ctx, appendableTrail(t, "trail-it", 3)); err != nil {
		t.Fatalf("audit Save: %v", err)
	}
	loaded, err := auditRepo.FindByID(ctx, "trail-it")
	if err != nil {
		t.Fatalf("audit FindByID: %v", err)
	}
	events, err := loaded.Execute(auditmodel.VerifyChainIntegrityCmd{FromSequence: 1, ToSequence: 3})
	if err != nil {
		t.Fatalf("verify chain: %v", err)
	}
	if _, ok := events[0].(auditmodel.ChainIntegrityVerifiedEvent); !ok {
		t.Fatalf("expected ChainIntegrityVerifiedEvent, got %T", events[0])
	}

	entries := loaded.Entries()
	entries[0].Hash = "deadbeef"
	if err := auditRepo.Save(ctx, auditmodel.RehydrateAuditTrail("trail-it", entries)); !errors.Is(err, auditmodel.ErrAuditEntryImmutable) {
		t.Fatalf("expected append-only rewrite rejection, got %v", err)
	}
}
