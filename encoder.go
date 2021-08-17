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
	if _, err := enc.writer.Write([]byte{byte(enum)}); err != nil {
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
	_, err := enc.writer.Write(d[:])
	return err
}

func (enc *Encoder) serialize(rv reflect.Value) error {
	var err error
	switch rv.Kind() {
	case reflect.Int8:
		_, err = enc.writer.Write([]byte{byte((rv.Int()))})
	case reflect.Int16:
		tmp := make([]byte, 2)
		binary.LittleEndian.PutUint16(tmp, uint16(rv.Int()))
		_, err = enc.writer.Write(tmp)
	case reflect.Int32:
		tmp := make([]byte, 4)
		binary.LittleEndian.PutUint32(tmp, uint32(rv.Int()))
		_, err = enc.writer.Write(tmp)
	case reflect.Int64:
		tmp := make([]byte, 8)
		binary.LittleEndian.PutUint64(tmp, uint64(rv.Int()))
		_, err = enc.writer.Write(tmp)
	case reflect.Int:
		tmp := make([]byte, 8)
		binary.LittleEndian.PutUint64(tmp, uint64(rv.Interface().(int)))
		_, err = enc.writer.Write(tmp)
	case reflect.Uint8:
		// user-defined Enum type is also uint8, so can't directly assert type here
		_, err = enc.writer.Write([]byte{byte(rv.Uint())})
	case reflect.Uint16:
		tmp := make([]byte, 2)
		binary.LittleEndian.PutUint16(tmp, uint16(rv.Uint()))
		_, err = enc.writer.Write(tmp)
	case reflect.Uint32:
		tmp := make([]byte, 4)
		binary.LittleEndian.PutUint32(tmp, uint32(rv.Uint()))
		_, err = enc.writer.Write(tmp)
	case reflect.Uint64, reflect.Uint:
		tmp := make([]byte, 8)
		binary.LittleEndian.PutUint64(tmp, rv.Uint())
		_, err = enc.writer.Write(tmp)
	case reflect.Float32:
		tmp := make([]byte, 4)
		f := rv.Float()
		if f == math.NaN() {
			return errors.New("NaN float value")
		}
		binary.LittleEndian.PutUint32(tmp, math.Float32bits(float32(f)))
		_, err = enc.writer.Write(tmp)
	case reflect.Float64:
		tmp := make([]byte, 8)
		f := rv.Float()
		if f == math.NaN() {
			return errors.New("NaN float value")
		}
		binary.LittleEndian.PutUint64(tmp, math.Float64bits(f))
		_, err = enc.writer.Write(tmp)
	case reflect.String:
		tmp := make([]byte, 4)
		binary.LittleEndian.PutUint32(tmp, uint32(len(rv.String())))
		_, err = enc.writer.Write(tmp)
		if err != nil {
			break
		}
		_, err = enc.writer.Write([]byte(rv.String()))
	case reflect.Array:
		for i := 0; i < rv.Len(); i++ {
			err = enc.serialize(rv.Index(i))
			if err != nil {
				break
			}
		}
	case reflect.Slice:
		tmp := make([]byte, 4)
		binary.LittleEndian.PutUint32(tmp, uint32(rv.Len()))
		_, err = enc.writer.Write(tmp)
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
		tmp := make([]byte, 4)
		binary.LittleEndian.PutUint32(tmp, uint32(rv.Len()))
		_, err = enc.writer.Write(tmp)
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
			_, err = enc.writer.Write([]byte{0})
		} else {
			_, err = enc.writer.Write([]byte{1})
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
	case reflect.Bool:
		return enc.WriteBool(rv.Bool())
	default:
		panic(fmt.Sprintf("encoding not implemented for %v kind", rv.Kind()))
	}
	return err
}

func (enc *Encoder) WriteByte(b byte) (err error) {
	return enc.writeBytes([]byte{b})
}

func (enc *Encoder) WriteBool(b bool) (err error) {
	var out byte
	if b {
		out = 1
	}
	return enc.WriteByte(out)
}

func (enc *Encoder) WriteUint8(i uint8) (err error) {
	return enc.WriteByte(i)
}

func (enc *Encoder) writeBytes(bytes []byte) (err error) {
	_, err = enc.writer.Write(bytes)
	return
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
