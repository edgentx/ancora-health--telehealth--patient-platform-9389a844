package apiintegration

import (
	"net/http"
	"sync"
	"testing"
)

// TestBookingFlow_ConcurrentSameSlot proves the double-book guarantee end-to-end
// through the API: two concurrent holds on the same provider/time-slot resolve
// to exactly one 201 Created and one 409 Conflict. The exclusion is enforced by
// the slot-hold lock the appointment repository acquires — the real Redis lock
// in CI, the in-process locker locally — not by anything in the test.
func TestBookingFlow_ConcurrentSameSlot(t *testing.T) {
	env := newEnv(t)

	const body = `{"providerId":"prov-77","timeSlot":"2026-09-01T09:00Z","patientId":"pat-a"}`

	const attempts = 2
	codes := make(chan int, attempts)
	var ready sync.WaitGroup
	ready.Add(attempts)
	var release sync.WaitGroup
	release.Add(1)
	var done sync.WaitGroup
	done.Add(attempts)

	for i := 0; i < attempts; i++ {
		go func() {
			defer done.Done()
			ready.Done()
			release.Wait() // line the goroutines up so they contend for the slot
			rec := env.request(http.MethodPost, "/api/v1/appointments", body, nil)
			codes <- rec.Code
		}()
	}

	ready.Wait()
	release.Done()
	done.Wait()
	close(codes)

	var created, conflict int
	for code := range codes {
		switch code {
		case http.StatusCreated:
			created++
		case http.StatusConflict:
			conflict++
		default:
			t.Fatalf("unexpected status %d, want 201 or 409", code)
		}
	}
	if created != 1 || conflict != 1 {
		t.Fatalf("double-book not enforced: got %d created / %d conflict, want 1/1", created, conflict)
	}
}

// TestBookingFlow_DistinctSlotsBothSucceed is the control: two holds on
// different slots contend for nothing and both succeed, proving the lock scopes
// exclusion to the slot rather than serializing all bookings.
func TestBookingFlow_DistinctSlotsBothSucceed(t *testing.T) {
	env := newEnv(t)

	first := env.request(http.MethodPost, "/api/v1/appointments",
		`{"providerId":"prov-9","timeSlot":"2026-09-02T10:00Z","patientId":"pat-1"}`, nil)
	second := env.request(http.MethodPost, "/api/v1/appointments",
		`{"providerId":"prov-9","timeSlot":"2026-09-02T11:00Z","patientId":"pat-2"}`, nil)

	if first.Code != http.StatusCreated || second.Code != http.StatusCreated {
		t.Fatalf("distinct slots should both be created: got %d and %d", first.Code, second.Code)
	}
}
