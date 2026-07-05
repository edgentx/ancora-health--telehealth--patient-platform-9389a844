package realtime

import (
	"context"
	"encoding/json"
	"io"
	"sync"

	authzmodel "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/authorization/model"
	engagemodel "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/patientengagement/model"
	schedmodel "github.com/edgentx/ancora-health--telehealth--patient-platform-9389a844/src/domain/scheduling/model"
)

// fakeConn is an in-memory Conn: frames pushed to it are decoded by ReadJSON,
// and frames written by WriteJSON are captured for assertion. It marshals
// through JSON exactly as a real socket would, so Pion SDP/ICE payloads
// round-trip verbatim.
type fakeConn struct {
	in     chan []byte
	out    chan []byte
	closed chan struct{}
	once   sync.Once
}

func newFakeConn() *fakeConn {
	return &fakeConn{
		in:     make(chan []byte, 32),
		out:    make(chan []byte, 32),
		closed: make(chan struct{}),
	}
}

func (c *fakeConn) ReadJSON(v any) error {
	select {
	case b, ok := <-c.in:
		if !ok {
			return io.EOF
		}
		return json.Unmarshal(b, v)
	case <-c.closed:
		return io.EOF
	}
}

func (c *fakeConn) WriteJSON(v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	select {
	case c.out <- b:
		return nil
	case <-c.closed:
		return io.ErrClosedPipe
	}
}

func (c *fakeConn) Close() error {
	c.once.Do(func() { close(c.closed) })
	return nil
}

// push enqueues a frame for the gateway's read loop to consume.
func (c *fakeConn) push(v any) {
	b, _ := json.Marshal(v)
	c.in <- b
}

// next decodes the next frame the gateway wrote into v. A buffered frame is
// preferred over the close signal so a frame written just before the connection
// is torn down is still observed.
func (c *fakeConn) next(v any) error {
	select {
	case b := <-c.out:
		return json.Unmarshal(b, v)
	default:
	}
	select {
	case b := <-c.out:
		return json.Unmarshal(b, v)
	case <-c.closed:
		return io.EOF
	}
}

// fakeAuthorizer is a stub SessionAuthorizer returning a fixed scope or error.
type fakeAuthorizer struct {
	err error
}

func (f fakeAuthorizer) Authorize(_ context.Context, h Handshake) (SessionContext, error) {
	if f.err != nil {
		return SessionContext{}, f.err
	}
	return SessionContext{
		AppointmentID: h.AppointmentID,
		PatientID:     "patient-1",
		ProviderID:    "provider-1",
		CallerRole:    h.Role,
	}, nil
}

// fakeAppointmentRepo is an in-memory AppointmentRepository for authorizer tests.
type fakeAppointmentRepo struct {
	m map[string]*schedmodel.AppointmentAggregate
}

func (r *fakeAppointmentRepo) Save(_ context.Context, a *schedmodel.AppointmentAggregate) error {
	r.m[a.ID] = a
	return nil
}

func (r *fakeAppointmentRepo) FindByID(_ context.Context, id string) (*schedmodel.AppointmentAggregate, error) {
	a, ok := r.m[id]
	if !ok {
		return nil, io.EOF // stand-in for a not-found error; the authorizer treats any error as unscoped
	}
	return a, nil
}

// fakeCareRelRepo is an in-memory CareRelationshipRepository for authorizer tests.
type fakeCareRelRepo struct {
	m map[string]*authzmodel.CareRelationshipAggregate
}

func (r *fakeCareRelRepo) Save(_ context.Context, a *authzmodel.CareRelationshipAggregate) error {
	r.m[a.ID] = a
	return nil
}

func (r *fakeCareRelRepo) FindByID(_ context.Context, id string) (*authzmodel.CareRelationshipAggregate, error) {
	a, ok := r.m[id]
	if !ok {
		return nil, io.EOF
	}
	return a, nil
}

// fakeThreadRepo is an in-memory MessageThreadRepository. Save mutates the stored
// aggregate in place, so a test can observe the persisted posted-message count.
type fakeThreadRepo struct {
	mu sync.Mutex
	m  map[string]*engagemodel.MessageThreadAggregate
}

func (r *fakeThreadRepo) Save(_ context.Context, a *engagemodel.MessageThreadAggregate) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.m[a.ID] = a
	return nil
}

func (r *fakeThreadRepo) FindByID(_ context.Context, id string) (*engagemodel.MessageThreadAggregate, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	a, ok := r.m[id]
	if !ok {
		return nil, io.EOF
	}
	return a, nil
}
