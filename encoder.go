package borsh

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"math/big"
	"reflect"
	"sort"
)

type Encoder struct {
	writer io.Writer
}

func NewEncoder(writer io.Writer) *Encoder {
	return &Encoder{writer: writer}
}

func (enc *Encoder) Encode(src interface{}) error {
	return enc.serialize(reflect.ValueOf(src))
}

func (enc *Encoder) Close() error {
	return nil
}

// Serialize `src` into bytes according to Borsh's specification (https://borsh.io/).
//
// The type mapping can be found at https://github.com/near/borsh-go.
func Serialize(src interface{}) ([]byte, error) {
	result := new(bytes.Buffer)
	enc := NewEncoder(result)
	err := enc.Encode(src)
	return result.Bytes(), err
}

func (enc *Encoder) serializeComplexEnum(rv reflect.Value) error {
	t := rv.Type()
	enum := Enum(rv.Field(0).Uint())
	// write enum identifier
	if err := enc.WriteByte(byte(enum)); err != nil {
		return err
	}
	// write enum field, if necessary
	if int(enum)+1 >= t.NumField() {
		return errors.New("complex enum too large")
	}
	field := rv.Field(int(enum) + 1)
	if field.Kind() == reflect.Struct {
		return enc.serializeStruct(field)
	}
	return nil
}

func (enc *Encoder) serializeStruct(rv reflect.Value) error {
	rt := rv.Type()

	// handle complex enum, if necessary
	if rt.NumField() > 0 {
		// if the first field has type borsh.Enum and is flagged with "borsh_enum"
		// we have a complex enum
		firstField := rt.Field(0)
		if firstField.Type.Kind() == reflect.Uint8 &&
			firstField.Tag.Get("borsh_enum") == "true" {
			return enc.serializeComplexEnum(rv)
		}
	}

	for i := 0; i < rt.NumField(); i++ {
		field := rt.Field(i)
		if field.Tag.Get("borsh_skip") == "true" {
			continue
		}
		// Skip unexported fields:
		fn := rv.FieldByName(field.Name)
		if !fn.CanInterface() {
			continue
		}

		err := enc.serialize(rv.Field(i))
		if err != nil {
			return err
		}
	}
	return nil
}

func (enc *Encoder) serializeUint128(rv reflect.Value) error {
	u := rv.Interface().(big.Int)
	buf := u.Bytes()
	if len(buf) > 16 {
		return errors.New("big.Int too large for u128")
	}
	// fill big-endian buffer
	var d [16]byte
	copy(d[16-len(buf):], buf)
	// make it little-endian
	for i, j := 0, 15; i < j; i, j = i+1, j-1 {
		d[i], d[j] = d[j], d[i]
	}
	return enc.WriteBytes(d[:])
}

func (enc *Encoder) WriteBytes(bytes []byte) (err error) {
	_, err = enc.writer.Write(bytes)
	return
}

func (enc *Encoder) WriteBytesWithLength(bytes []byte) (err error) {
	err = enc.WriteUint32(uint32(len(bytes)))
	if err != nil {
		return err
	}
	return enc.WriteBytes(bytes)
}

func (enc *Encoder) WriteByte(b byte) (err error) {
	return enc.WriteBytes([]byte{b})
}

func (enc *Encoder) WriteString(s string) (err error) {
	return enc.WriteBytesWithLength([]byte(s))
}

func (enc *Encoder) WriteUint8(i uint8) (err error) {
	return enc.WriteByte(i)
}

func (enc *Encoder) WriteBool(b bool) (err error) {
	var out byte
	if b {
		out = 1
	}
	return enc.WriteByte(out)
}

func (enc *Encoder) WriteUint16(i uint16) (err error) {
	tmp := make([]byte, 2)
	binary.LittleEndian.PutUint16(tmp, i)
	return enc.WriteBytes(tmp)
}

func (enc *Encoder) WriteInt16(i int16) (err error) {
	return enc.WriteUint16(uint16(i))
}

func (enc *Encoder) WriteUint32(i uint32) (err error) {
	tmp := make([]byte, 4)
	binary.LittleEndian.PutUint32(tmp, i)
	return enc.WriteBytes(tmp)
}

func (enc *Encoder) WriteInt32(i int32) (err error) {
	return enc.WriteUint32(uint32(i))
}

func (enc *Encoder) WriteUint64(i uint64) (err error) {
	tmp := make([]byte, 8)
	binary.LittleEndian.PutUint64(tmp, i)
	return enc.WriteBytes(tmp)
}

func (enc *Encoder) WriteInt64(i int64) (err error) {
	return enc.WriteUint64(uint64(i))
}

func (enc *Encoder) WriteFloat32(f float64) (err error) {
	tmp := make([]byte, 4)
	if f == math.NaN() {
		return errors.New("NaN float value")
	}
	binary.LittleEndian.PutUint32(tmp, math.Float32bits(float32(f)))
	return enc.WriteBytes(tmp)
}

func (enc *Encoder) WriteFloat64(f float64) (err error) {
	tmp := make([]byte, 8)
	if f == math.NaN() {
		return errors.New("NaN float value")
	}
	binary.LittleEndian.PutUint64(tmp, math.Float64bits(f))
	return enc.WriteBytes(tmp)
}

func (enc *Encoder) serialize(rv reflect.Value) error {
	var err error
	switch rv.Kind() {
	case reflect.Bool:
		return enc.WriteBool(rv.Bool())
	case reflect.Int8:
		return enc.WriteByte(byte(rv.Int()))
	case reflect.Int16:
		return enc.WriteInt16(int16(rv.Int()))
	case reflect.Int32:
		return enc.WriteInt32(int32(rv.Int()))
	case reflect.Int64:
		return enc.WriteInt64(int64(rv.Int()))
	case reflect.Int:
		return enc.WriteInt64(int64(rv.Interface().(int)))
	case reflect.Uint8:
		// user-defined Enum type is also uint8, so can't directly assert type here
		return enc.WriteByte(byte(rv.Uint()))
	case reflect.Uint16:
		return enc.WriteUint16(uint16(rv.Uint()))
	case reflect.Uint32:
		return enc.WriteUint32(uint32(rv.Uint()))
	case reflect.Uint64, reflect.Uint:
		return enc.WriteUint64(uint64(rv.Uint()))
	case reflect.Float32:
		return enc.WriteFloat32(rv.Float())
	case reflect.Float64:
		return enc.WriteFloat64(rv.Float())
	case reflect.String:
		return enc.WriteString(rv.String())
	case reflect.Array:
		for i := 0; i < rv.Len(); i++ {
			err = enc.serialize(rv.Index(i))
			if err != nil {
				break
			}
		}
	case reflect.Slice:
		err = enc.WriteUint32(uint32(rv.Len()))
		if err != nil {
			break
		}
		for i := 0; i < rv.Len(); i++ {
			err = enc.serialize(rv.Index(i))
			if err != nil {
				break
			}
		}
	case reflect.Map:
		err = enc.WriteUint32(uint32(rv.Len()))
		if err != nil {
			break
		}
		keys := rv.MapKeys()
		sort.Slice(keys, vComp(keys))
		for _, k := range keys {
			err = enc.serialize(k)
			if err != nil {
				break
			}
			err = enc.serialize(rv.MapIndex(k))
		}
	case reflect.Ptr:
		if rv.IsNil() {
			err = enc.WriteByte(0)
		} else {
			err = enc.WriteByte(1)
			if err != nil {
				break
			}
			err = enc.serialize(rv.Elem())
		}
	case reflect.Struct:
		if rv.Type() == reflect.TypeOf(*big.NewInt(0)) {
			err = enc.serializeUint128(rv)
		} else {
			err = enc.serializeStruct(rv)
		}
	case reflect.Invalid:
		// skip
		return nil
	default:
		return fmt.Errorf("encoding not supported for %q", rv)
	}
	return err
}

func vComp(keys []reflect.Value) func(int, int) bool {
	return func(i int, j int) bool {
		a, b := keys[i], keys[j]
		if a.Kind() == reflect.Interface {
			a = a.Elem()
			b = b.Elem()
		}
		switch a.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32:
			return a.Int() < b.Int()
		case reflect.Int64:
			return a.Interface().(int64) < b.Interface().(int64)
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32:
			return a.Uint() < b.Uint()
		case reflect.Uint64:
			return a.Interface().(uint64) < b.Interface().(uint64)
		case reflect.Float32, reflect.Float64:
			return a.Float() < b.Float()
		case reflect.String:
			return a.String() < b.String()
		}
		panic("unsupported key compare")
	}
}
