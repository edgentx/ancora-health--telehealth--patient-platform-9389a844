package crypto

import (
	"bytes"
	"context"
	"errors"
	"reflect"
	"testing"

	"go.mongodb.org/mongo-driver/bson"
)

var errEnvelope = errors.New("envelope boom")

// stubEnvelope is a KeyEnvelope double for injecting wrap/unwrap failures and
// returning a controllable data key.
type stubEnvelope struct {
	keyRef    string
	dek       []byte
	wrapErr   error
	unwrapErr error
}

func (s stubEnvelope) WrapDataKey(_ context.Context, _ []byte) ([]byte, string, error) {
	if s.wrapErr != nil {
		return nil, "", s.wrapErr
	}
	return []byte("wrapped"), s.keyRef, nil
}

func (s stubEnvelope) UnwrapDataKey(_ context.Context, _ []byte, _ string) ([]byte, error) {
	if s.unwrapErr != nil {
		return nil, s.unwrapErr
	}
	return s.dek, nil
}

// --- AESKeyEnvelope ---------------------------------------------------------

func TestNewAESKeyEnvelope_Errors(t *testing.T) {
	if _, err := NewAESKeyEnvelope("", make([]byte, KeySize)); !errors.Is(err, ErrEmptyKeyRef) {
		t.Fatalf("empty keyRef: want ErrEmptyKeyRef, got %v", err)
	}
	if _, err := NewAESKeyEnvelope("k", make([]byte, 8)); !errors.Is(err, ErrInvalidKeySize) {
		t.Fatalf("short key: want ErrInvalidKeySize, got %v", err)
	}
}

func TestAESKeyEnvelope_KeyRef(t *testing.T) {
	e, err := NewAESKeyEnvelope("master-1", make([]byte, KeySize))
	if err != nil {
		t.Fatalf("NewAESKeyEnvelope: %v", err)
	}
	if e.KeyRef() != "master-1" {
		t.Fatalf("KeyRef = %q", e.KeyRef())
	}
}

func TestAESKeyEnvelope_WrapUnwrap(t *testing.T) {
	e, err := NewAESKeyEnvelope("master-1", bytes.Repeat([]byte{7}, KeySize))
	if err != nil {
		t.Fatalf("NewAESKeyEnvelope: %v", err)
	}
	ctx := context.Background()
	dek := bytes.Repeat([]byte{9}, KeySize)

	wrapped, ref, err := e.WrapDataKey(ctx, dek)
	if err != nil {
		t.Fatalf("WrapDataKey: %v", err)
	}
	if ref != "master-1" {
		t.Fatalf("ref = %q", ref)
	}

	got, err := e.UnwrapDataKey(ctx, wrapped, ref)
	if err != nil {
		t.Fatalf("UnwrapDataKey: %v", err)
	}
	if !bytes.Equal(got, dek) {
		t.Fatal("unwrapped DEK mismatch")
	}

	// Wrong dek size is rejected.
	if _, _, err := e.WrapDataKey(ctx, make([]byte, 5)); !errors.Is(err, ErrInvalidKeySize) {
		t.Fatalf("wrap short dek: want ErrInvalidKeySize, got %v", err)
	}
	// Unknown key reference is rejected.
	if _, err := e.UnwrapDataKey(ctx, wrapped, "other"); err == nil {
		t.Fatal("expected unknown key reference error")
	}
	// A too-short wrapped blob is rejected by open().
	if _, err := e.UnwrapDataKey(ctx, []byte{1, 2}, ref); err == nil {
		t.Fatal("expected ciphertext-too-short error")
	}
}

func TestNewGCM_BadKey(t *testing.T) {
	if _, err := newGCM(make([]byte, 3)); !errors.Is(err, ErrInvalidKeySize) {
		t.Fatalf("want ErrInvalidKeySize, got %v", err)
	}
}

// --- FieldCipher error paths ------------------------------------------------

func TestFieldCipher_EncryptWrapError(t *testing.T) {
	c := NewFieldCipher(stubEnvelope{wrapErr: errEnvelope})
	if _, err := c.Encrypt(context.Background(), []byte("x")); !errors.Is(err, errEnvelope) {
		t.Fatalf("want errEnvelope, got %v", err)
	}
}

func TestFieldCipher_DecryptUnwrapError(t *testing.T) {
	c := NewFieldCipher(stubEnvelope{unwrapErr: errEnvelope})
	if _, err := c.Decrypt(context.Background(), CipherText{}); !errors.Is(err, errEnvelope) {
		t.Fatalf("want errEnvelope, got %v", err)
	}
}

func TestFieldCipher_DecryptBadDEK(t *testing.T) {
	// A recovered data key of the wrong size makes newGCM fail inside Decrypt.
	c := NewFieldCipher(stubEnvelope{dek: make([]byte, 7)})
	if _, err := c.Decrypt(context.Background(), CipherText{}); !errors.Is(err, ErrInvalidKeySize) {
		t.Fatalf("want ErrInvalidKeySize, got %v", err)
	}
}

// --- Codec: EncryptDocument -------------------------------------------------

func TestEncryptDocument_NonStruct(t *testing.T) {
	codec := NewCodec(newTestCipher(t))
	if _, err := codec.EncryptDocument(context.Background(), 42); err == nil {
		t.Fatal("expected error for non-struct input")
	}
}

func TestEncryptDocument_SkipsAndTags(t *testing.T) {
	codec := NewCodec(newTestCipher(t))
	type doc struct {
		Ignored   string `bson:"-"`
		Untagged  string // no bson tag => lower-cased field name
		OnlyOpts  string `bson:",omitempty"` // empty name => lower-cased field name
		unexp     string // unexported => skipped
		PublicVal int    `bson:"public_val"`
	}
	d := doc{Ignored: "no", Untagged: "u", OnlyOpts: "o", unexp: "x", PublicVal: 5}
	out, err := codec.EncryptDocument(context.Background(), d)
	if err != nil {
		t.Fatalf("EncryptDocument: %v", err)
	}
	if _, ok := out["-"]; ok {
		t.Fatal("bson:\"-\" field must be skipped")
	}
	if _, ok := out["unexp"]; ok {
		t.Fatal("unexported field must be skipped")
	}
	if out["untagged"] != "u" {
		t.Fatalf("untagged => %v", out["untagged"])
	}
	if out["onlyopts"] != "o" {
		t.Fatalf("onlyopts => %v", out["onlyopts"])
	}
	if out["public_val"] != 5 {
		t.Fatalf("public_val => %v", out["public_val"])
	}
}

func TestEncryptDocument_CipherError(t *testing.T) {
	codec := NewCodec(NewFieldCipher(stubEnvelope{wrapErr: errEnvelope}))
	type doc struct {
		Name string `bson:"name" phi:"true"`
	}
	if _, err := codec.EncryptDocument(context.Background(), doc{Name: "Alice"}); !errors.Is(err, errEnvelope) {
		t.Fatalf("want errEnvelope, got %v", err)
	}
}

// --- Codec: DecryptDocument -------------------------------------------------

func TestDecryptDocument_NotPointer(t *testing.T) {
	codec := NewCodec(newTestCipher(t))
	var notPtr phiRecord
	if err := codec.DecryptDocument(context.Background(), bson.M{}, notPtr); err == nil {
		t.Fatal("expected error for non-pointer dst")
	}
}

func TestDecryptDocument_SkipsMissingAndDashAndUnexported(t *testing.T) {
	codec := NewCodec(newTestCipher(t))
	type doc struct {
		Ignored string `bson:"-"`
		unexp   string
		Present string `bson:"present"`
		Absent  string `bson:"absent"`
	}
	var out doc
	// Only "present" is in the document; "absent" is missing and skipped.
	if err := codec.DecryptDocument(context.Background(), bson.M{"present": "hi"}, &out); err != nil {
		t.Fatalf("DecryptDocument: %v", err)
	}
	if out.Present != "hi" {
		t.Fatalf("Present = %q", out.Present)
	}
	if out.Absent != "" {
		t.Fatalf("Absent should stay empty, got %q", out.Absent)
	}
}

func TestDecryptDocument_AssignError(t *testing.T) {
	codec := NewCodec(newTestCipher(t))
	type doc struct {
		N int `bson:"n"`
	}
	var out doc
	// A string cannot be assigned or converted to an int field.
	if err := codec.DecryptDocument(context.Background(), bson.M{"n": "not-an-int"}, &out); err == nil {
		t.Fatal("expected assign error")
	}
}

func TestDecryptDocument_ConvertibleField(t *testing.T) {
	codec := NewCodec(newTestCipher(t))
	type doc struct {
		N int64 `bson:"n"`
	}
	var out doc
	// int32 is convertible to int64 (mirrors BSON integer decoding).
	if err := codec.DecryptDocument(context.Background(), bson.M{"n": int32(7)}, &out); err != nil {
		t.Fatalf("DecryptDocument: %v", err)
	}
	if out.N != 7 {
		t.Fatalf("N = %d", out.N)
	}
}

func TestDecryptDocument_DecryptError(t *testing.T) {
	// Encrypt with one master key, then attempt to decrypt with a codec whose
	// envelope uses a different key reference — UnwrapDataKey rejects it.
	enc := NewCodec(newTestCipher(t))
	in := phiRecord{ID: "p-1", FirstName: "Alice"}
	doc, err := enc.EncryptDocument(context.Background(), in)
	if err != nil {
		t.Fatalf("EncryptDocument: %v", err)
	}

	other, err := NewAESKeyEnvelope("different-master", bytes.Repeat([]byte{1}, KeySize))
	if err != nil {
		t.Fatalf("NewAESKeyEnvelope: %v", err)
	}
	dec := NewCodec(NewFieldCipher(other))
	var out phiRecord
	if err := dec.DecryptDocument(context.Background(), doc, &out); err == nil {
		t.Fatal("expected decrypt error with mismatched key reference")
	}
}

func TestDecryptDocument_BSONCipherText(t *testing.T) {
	// A CipherText that arrived as a decoded BSON document (primitive.M / bson.M),
	// as it would from MongoDB, must still decrypt via toCipherText's slow path.
	codec := NewCodec(newTestCipher(t))
	ct, err := codec.cipher.Encrypt(context.Background(), []byte("Bob"))
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	raw, err := bson.Marshal(ct)
	if err != nil {
		t.Fatalf("marshal ct: %v", err)
	}
	var asBSON bson.M
	if err := bson.Unmarshal(raw, &asBSON); err != nil {
		t.Fatalf("unmarshal ct: %v", err)
	}

	var out phiRecord
	if err := codec.DecryptDocument(context.Background(), bson.M{"first_name": asBSON}, &out); err != nil {
		t.Fatalf("DecryptDocument: %v", err)
	}
	if out.FirstName != "Bob" {
		t.Fatalf("FirstName = %q", out.FirstName)
	}
}

// --- toCipherText & assign (direct) -----------------------------------------

func TestToCipherText(t *testing.T) {
	orig := CipherText{Version: cipherVersion, KeyRef: "k", Nonce: []byte{1}, Ciphertext: []byte{2}}

	// Fast path: already a CipherText.
	if got, err := toCipherText(orig); err != nil || got.KeyRef != "k" {
		t.Fatalf("fast path: got %+v err %v", got, err)
	}

	// Marshal error: a channel cannot be BSON-encoded.
	if _, err := toCipherText(make(chan int)); err == nil {
		t.Fatal("expected marshal error for un-encodable value")
	}
}

func TestAssign(t *testing.T) {
	// Invalid (nil) source is a no-op.
	var s string
	fv := reflect.ValueOf(&s).Elem()
	if err := assign(fv, nil); err != nil {
		t.Fatalf("nil source: %v", err)
	}

	// Assignable path.
	if err := assign(fv, "hello"); err != nil || s != "hello" {
		t.Fatalf("assignable: s=%q err=%v", s, err)
	}

	// Convertible path.
	var n int64
	nv := reflect.ValueOf(&n).Elem()
	if err := assign(nv, int32(3)); err != nil || n != 3 {
		t.Fatalf("convertible: n=%d err=%v", n, err)
	}

	// Incompatible path.
	if err := assign(nv, "nope"); err == nil {
		t.Fatal("expected incompatible-type error")
	}
}

func TestBSONName_Direct(t *testing.T) {
	type doc struct {
		A string `bson:"a_name"`
		B string
		C string `bson:",omitempty"`
	}
	rt := reflect.TypeOf(doc{})
	if got := bsonName(rt.Field(0)); got != "a_name" {
		t.Fatalf("A => %q", got)
	}
	if got := bsonName(rt.Field(1)); got != "b" {
		t.Fatalf("B => %q", got)
	}
	if got := bsonName(rt.Field(2)); got != "c" {
		t.Fatalf("C => %q", got)
	}
}
