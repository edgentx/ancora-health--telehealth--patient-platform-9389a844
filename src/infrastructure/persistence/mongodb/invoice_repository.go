package mongodb

import (
	"context"

	"github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/billingandinsurance/model"
	billrepo "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/billingandinsurance/repository"
)

// invoicesCollection is the collection invoice documents live in.
const invoicesCollection = "invoices"

// invoiceDoc is the at-rest projection of an InvoiceAggregate. Its charge amounts
// are financial data, not PHI/PCI cardholder data, so they are stored in the
// clear where the revenue rollups can aggregate over them.
type invoiceDoc struct {
	DocID   string `bson:"_id"`
	Ver     int    `bson:"version"`
	StatusV string `bson:"status"`

	EncounterID string           `bson:"encounter_id"`
	LineItems   []invoiceLineDoc `bson:"line_items"`
	PolicyID    string           `bson:"policy_id"`

	EncounterNotCompleted         bool `bson:"encounter_not_completed"`
	PatientResponsibilityMismatch bool `bson:"patient_responsibility_mismatch"`
	PaymentExceedsOutstanding     bool `bson:"payment_exceeds_outstanding"`
	Voided                        bool `bson:"voided"`

	CoverageCents int64 `bson:"coverage_cents"`
	CopayCents    int64 `bson:"copay_cents"`
}

// invoiceLineDoc mirrors an InvoiceLineItem.
type invoiceLineDoc struct {
	Description string `bson:"description"`
	AmountCents int64  `bson:"amount_cents"`
}

func (d *invoiceDoc) ID() string       { return d.DocID }
func (d *invoiceDoc) Version() int     { return d.Ver }
func (d *invoiceDoc) SetVersion(v int) { d.Ver = v }

// InvoiceRepository is the MongoDB-backed InvoiceRepository.
type InvoiceRepository struct {
	base *BaseRepository
}

var _ billrepo.InvoiceRepository = (*InvoiceRepository)(nil)

// NewInvoiceRepository builds an invoice repository over a store.
func NewInvoiceRepository(store DocumentStore) *InvoiceRepository {
	return &InvoiceRepository{base: NewBaseRepository(store, invoicesCollection)}
}

// Save persists the invoice aggregate with optimistic concurrency.
func (r *InvoiceRepository) Save(ctx context.Context, a *model.InvoiceAggregate) error {
	doc := &invoiceDoc{
		DocID:                         a.ID,
		Ver:                           a.GetVersion(),
		StatusV:                       string(a.Status),
		EncounterID:                   a.EncounterID,
		PolicyID:                      a.PolicyID,
		EncounterNotCompleted:         a.EncounterNotCompleted,
		PatientResponsibilityMismatch: a.PatientResponsibilityMismatch,
		PaymentExceedsOutstanding:     a.PaymentExceedsOutstanding,
		Voided:                        a.Voided,
		CoverageCents:                 a.CoverageCents,
		CopayCents:                    a.CopayCents,
	}
	for _, li := range a.LineItems {
		doc.LineItems = append(doc.LineItems, invoiceLineDoc{Description: li.Description, AmountCents: li.AmountCents})
	}
	return saveAggregate(ctx, r.base, doc, a)
}

// FindByID loads an invoice aggregate by identity.
func (r *InvoiceRepository) FindByID(ctx context.Context, id string) (*model.InvoiceAggregate, error) {
	var doc invoiceDoc
	if err := r.base.FindByID(ctx, id, &doc); err != nil {
		return nil, err
	}
	a := &model.InvoiceAggregate{
		ID:                            doc.DocID,
		Status:                        model.InvoiceStatus(doc.StatusV),
		EncounterID:                   doc.EncounterID,
		PolicyID:                      doc.PolicyID,
		EncounterNotCompleted:         doc.EncounterNotCompleted,
		PatientResponsibilityMismatch: doc.PatientResponsibilityMismatch,
		PaymentExceedsOutstanding:     doc.PaymentExceedsOutstanding,
		Voided:                        doc.Voided,
		CoverageCents:                 doc.CoverageCents,
		CopayCents:                    doc.CopayCents,
	}
	for _, li := range doc.LineItems {
		a.LineItems = append(a.LineItems, model.InvoiceLineItem{Description: li.Description, AmountCents: li.AmountCents})
	}
	a.Version = doc.Ver
	return a, nil
}
