package mongodb

import (
	"reflect"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsoncodec"
	"go.mongodb.org/mongo-driver/bson/bsonrw"
)

// timeType is the reflect.Type of time.Time, used to register the UTC codec.
var timeType = reflect.TypeOf(time.Time{})

// utcTimeEncoder normalizes every time.Time to UTC before it is written. Domain
// value objects that carry timestamps therefore serialize consistently
// regardless of the location attached to the in-memory value, so equality and
// range queries behave predictably across services.
type utcTimeEncoder struct{}

// EncodeValue writes the time as a UTC BSON datetime (milliseconds since epoch).
func (utcTimeEncoder) EncodeValue(_ bsoncodec.EncodeContext, vw bsonrw.ValueWriter, val reflect.Value) error {
	if !val.IsValid() || val.Type() != timeType {
		return bsoncodec.ValueEncoderError{
			Name:     "utcTimeEncoder",
			Types:    []reflect.Type{timeType},
			Received: val,
		}
	}
	t := val.Interface().(time.Time).UTC()
	return vw.WriteDateTime(t.UnixMilli())
}

// NewRegistry builds a BSON codec registry with the driver defaults plus the
// project's shared value-object codecs. Pass it to the client (SetRegistry) so
// every collection marshals domain value objects the same way.
func NewRegistry() *bsoncodec.Registry {
	rb := bson.NewRegistryBuilder()
	rb.RegisterTypeEncoder(timeType, utcTimeEncoder{})
	return rb.Build()
}
