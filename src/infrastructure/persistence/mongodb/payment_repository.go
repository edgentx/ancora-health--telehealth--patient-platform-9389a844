package mongodb

import (
	"context"

	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/billingandinsurance/model"
	billrepo "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/billingandinsurance/repository"
	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/infrastructure/crypto"
)

// paymentsCollection is the collection payment documents live in.
const paymentsCollection = "payments"

// paymentDoc is the at-rest projection of a PaymentAggregate. The gateway payment
// token is PCI-sensitive and stored encrypted; the aggregate never persists raw
// card data by construction.
type paymentDoc struct {
	DocID   string `bson:"_id"`
	Ver     int    `bson:"version"`
	StatusV string `bson:"status"`

	InvoiceID          string            `bson:"invoice_id"`
	PaymentTokenCipher crypto.CipherText `bson:"payment_token_cipher"`
	AmountCents        int64             `bson:"amount_cents"`

	RawCardDataPresent   bool `bson:"raw_card_data_present"`
	NoOutstandingBalance bool `bson:"no_outstanding_balance"`
	WebhookNotVerified   bool `bson:"webhook_not_verified"`
}

func (d *paymentDoc) ID() string       { return d.DocID }
func (d *paymentDoc) Version() int     { return d.Ver }
func (d *paymentDoc) SetVersion(v int) { d.Ver = v }

// PaymentRepository is the MongoDB-backed PaymentRepository, encrypting the
// gateway token with the S-68 envelope cipher.
type PaymentRepository struct {
	base   *BaseRepository
	cipher *crypto.FieldCipher
}

var _ billrepo.PaymentRepository = (*PaymentRepository)(nil)

// NewPaymentRepository builds a payment repository over a store and cipher.
func NewPaymentRepository(store DocumentStore, cipher *crypto.FieldCipher) *PaymentRepository {
	return &PaymentRepository{base: NewBaseRepository(store, paymentsCollection), cipher: cipher}
}

// Save encrypts the gateway token and persists the payment with optimistic concurrency.
func (r *PaymentRepository) Save(ctx context.Context, a *model.PaymentAggregate) error {
	token, err := encryptField(ctx, r.cipher, a.PaymentToken)
	if err != nil {
		return err
	}
	doc := &paymentDoc{
		DocID:                a.ID,
		Ver:                  a.GetVersion(),
		StatusV:              string(a.Status),
		InvoiceID:            a.InvoiceID,
		PaymentTokenCipher:   token,
		AmountCents:          a.AmountCents,
		RawCardDataPresent:   a.RawCardDataPresent,
		NoOutstandingBalance: a.NoOutstandingBalance,
		WebhookNotVerified:   a.WebhookNotVerified,
	}
	return saveAggregate(ctx, r.base, doc, a)
}

// FindByID loads and decrypts a payment aggregate by identity.
func (r *PaymentRepository) FindByID(ctx context.Context, id string) (*model.PaymentAggregate, error) {
	var doc paymentDoc
	if err := r.base.FindByID(ctx, id, &doc); err != nil {
		return nil, err
	}
	token, err := decryptField(ctx, r.cipher, doc.PaymentTokenCipher)
	if err != nil {
		return nil, err
	}
	a := &model.PaymentAggregate{
		ID:                   doc.DocID,
		Status:               model.PaymentStatus(doc.StatusV),
		InvoiceID:            doc.InvoiceID,
		PaymentToken:         token,
		AmountCents:          doc.AmountCents,
		RawCardDataPresent:   doc.RawCardDataPresent,
		NoOutstandingBalance: doc.NoOutstandingBalance,
		WebhookNotVerified:   doc.WebhookNotVerified,
	}
	a.Version = doc.Ver
	return a, nil
}
