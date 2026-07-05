package crypto

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
)

// phiTag marks a struct field as PHI. A field tagged `phi:"true"` is transparently
// encrypted on the way to storage and decrypted on the way back.
const phiTag = "phi"

// Codec applies field-level PHI encryption to whole documents. It sits at the
// persistence boundary: repositories hand it a domain persistence struct, and it
// returns a bson.M in which every PHI-tagged field has been replaced by a
// CipherText. On read it reverses the transformation. Only string PHI fields are
// supported by the struct codec — arbitrary-typed secrets can use FieldCipher
// directly.
type Codec struct {
	cipher *FieldCipher
}

// NewCodec builds a Codec over the given field cipher.
func NewCodec(cipher *FieldCipher) *Codec {
	return &Codec{cipher: cipher}
}

// EncryptDocument reflects over src (a struct or pointer to struct) and returns a
// bson.M keyed by each field's BSON name. PHI-tagged string fields are replaced
// by their CipherText; all other fields are copied through unchanged.
func (c *Codec) EncryptDocument(ctx context.Context, src any) (bson.M, error) {
	rv := reflect.Indirect(reflect.ValueOf(src))
	if rv.Kind() != reflect.Struct {
		return nil, fmt.Errorf("crypto: EncryptDocument requires a struct, got %s", rv.Kind())
	}
	rt := rv.Type()

	out := bson.M{}
	for i := 0; i < rt.NumField(); i++ {
		field := rt.Field(i)
		if field.PkgPath != "" { // unexported
			continue
		}
		name := bsonName(field)
		if name == "-" {
			continue
		}
		if !isPHI(field) {
			out[name] = rv.Field(i).Interface()
			continue
		}
		fv := rv.Field(i)
		if fv.Kind() != reflect.String {
			return nil, fmt.Errorf("crypto: PHI field %q must be a string, got %s", field.Name, fv.Kind())
		}
		ct, err := c.cipher.Encrypt(ctx, []byte(fv.String()))
		if err != nil {
			return nil, err
		}
		out[name] = ct
	}
	return out, nil
}

// DecryptDocument reverses EncryptDocument, decoding doc into dst (a pointer to
// struct) and decrypting every PHI-tagged field on the way in.
func (c *Codec) DecryptDocument(ctx context.Context, doc bson.M, dst any) error {
	pv := reflect.ValueOf(dst)
	if pv.Kind() != reflect.Ptr || pv.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("crypto: DecryptDocument requires a pointer to struct")
	}
	rv := pv.Elem()
	rt := rv.Type()

	for i := 0; i < rt.NumField(); i++ {
		field := rt.Field(i)
		if field.PkgPath != "" {
			continue
		}
		name := bsonName(field)
		if name == "-" {
			continue
		}
		raw, ok := doc[name]
		if !ok {
			continue
		}
		if !isPHI(field) {
			if err := assign(rv.Field(i), raw); err != nil {
				return fmt.Errorf("crypto: field %q: %w", field.Name, err)
			}
			continue
		}
		ct, err := toCipherText(raw)
		if err != nil {
			return fmt.Errorf("crypto: field %q: %w", field.Name, err)
		}
		plain, err := c.cipher.Decrypt(ctx, ct)
		if err != nil {
			return err
		}
		rv.Field(i).SetString(string(plain))
	}
	return nil
}

// isPHI reports whether a struct field is tagged for PHI encryption.
func isPHI(field reflect.StructField) bool {
	return strings.EqualFold(field.Tag.Get(phiTag), "true")
}

// bsonName derives the storage key for a field from its `bson` tag, falling back
// to the lower-cased field name (matching the mongo driver's default).
func bsonName(field reflect.StructField) string {
	tag := field.Tag.Get("bson")
	if tag == "" {
		return strings.ToLower(field.Name)
	}
	name := strings.Split(tag, ",")[0]
	if name == "" {
		return strings.ToLower(field.Name)
	}
	return name
}

// toCipherText coerces a stored value back into a CipherText, accepting both an
// in-process CipherText (same-process round trip) and a BSON document decoded
// from MongoDB (primitive.M / bson.M).
func toCipherText(raw any) (CipherText, error) {
	if ct, ok := raw.(CipherText); ok {
		return ct, nil
	}
	data, err := bson.Marshal(raw)
	if err != nil {
		return CipherText{}, err
	}
	var ct CipherText
	if err := bson.Unmarshal(data, &ct); err != nil {
		return CipherText{}, err
	}
	return ct, nil
}

// assign sets a non-PHI field from a decoded value, converting between
// compatible types where necessary (BSON decodes integers as int32/int64).
func assign(field reflect.Value, raw any) error {
	rv := reflect.ValueOf(raw)
	if !rv.IsValid() {
		return nil
	}
	switch {
	case rv.Type().AssignableTo(field.Type()):
		field.Set(rv)
	case rv.Type().ConvertibleTo(field.Type()):
		field.Set(rv.Convert(field.Type()))
	default:
		return fmt.Errorf("cannot assign %s to %s", rv.Type(), field.Type())
	}
	return nil
}
