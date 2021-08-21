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
)

type Decoder struct {
	reader io.ByteReader
}

func NewDecoder(reader io.ByteReader) *Decoder {
	return &Decoder{reader: reader}
}

func (dec *Decoder) Decode(dst interface{}) error {
	rv := reflect.ValueOf(dst)
	if rv.Kind() != reflect.Ptr {
		return errors.New("argument must be pointer")
	}
	return dec.deserialize(rv.Elem(), false)
}

func (d *Decoder) Close() error {
	return nil
}

// Deserialize `data` according to the schema of `dst`, and store the value into it.
// `dst` must be a pointer type variable
// that points to the original schema of `data`.
func Deserialize(dst interface{}, data []byte) error {
	dec := NewDecoder(bytes.NewReader(data))
	return dec.Decode(dst)
}

func readNBytes(n int, reader io.ByteReader) ([]byte, error) {
	buf := make([]byte, n)
	for i := 0; i < n; i++ {
		b, err := reader.ReadByte()
		if err != nil {
			return nil, err
		}
		buf[i] = b
	}
	return buf, nil
}

func (dec *Decoder) ReadNBytes(n int) (out []byte, err error) {
	return readNBytes(n, dec.reader)
}

func (dec *Decoder) ReadUint8() (out uint8, err error) {
	out, err = dec.ReadByte()
	return
}

func (dec *Decoder) ReadByte() (out byte, err error) {
	return dec.reader.ReadByte()
}

func (dec *Decoder) ReadBool() (out bool, err error) {
	b, err := dec.ReadByte()
	if err != nil {
		err = fmt.Errorf("ReadBool: %w", err)
	}
	out = b != 0
	return
}

func (dec *Decoder) ReadInt8() (int8, error) {
	tmp, err := dec.ReadUint8()
	if err != nil {
		return 0, err
	}
	return int8(tmp), nil
}

func (dec *Decoder) ReadInt16() (int16, error) {
	tmp, err := dec.ReadNBytes(2)
	if err != nil {
		return 0, err
	}
	return int16(binary.LittleEndian.Uint16(tmp)), nil
}

func (dec *Decoder) ReadInt32() (int32, error) {
	tmp, err := dec.ReadNBytes(4)
	if err != nil {
		return 0, err
	}
	return int32(binary.LittleEndian.Uint32(tmp)), nil
}

func (dec *Decoder) ReadInt64() (int64, error) {
	tmp, err := dec.ReadNBytes(8)
	if err != nil {
		return 0, err
	}
	return int64(binary.LittleEndian.Uint64(tmp)), nil
}

func (dec *Decoder) ReadInt() (int, error) {
	tmp, err := dec.ReadNBytes(8)
	if err != nil {
		return 0, err
	}
	return int(binary.LittleEndian.Uint64(tmp)), nil
}

func (dec *Decoder) ReadUint16() (uint16, error) {
	tmp, err := dec.ReadNBytes(2)
	if err != nil {
		return 0, err
	}
	return uint16(binary.LittleEndian.Uint16(tmp)), nil
}

func (dec *Decoder) ReadUint32() (uint32, error) {
	tmp, err := dec.ReadNBytes(4)
	if err != nil {
		return 0, err
	}
	return uint32(binary.LittleEndian.Uint32(tmp)), nil
}

func (dec *Decoder) ReadUint64() (uint64, error) {
	tmp, err := dec.ReadNBytes(8)
	if err != nil {
		return 0, err
	}
	return uint64(binary.LittleEndian.Uint64(tmp)), nil
}

func (dec *Decoder) ReadUint() (uint, error) {
	tmp, err := dec.ReadNBytes(8)
	if err != nil {
		return 0, err
	}
	return uint(binary.LittleEndian.Uint64(tmp)), nil
}

func (dec *Decoder) ReadFloat32() (float32, error) {
	bits, err := dec.ReadUint32()
	if err != nil {
		return 0, err
	}
	out := math.Float32frombits(bits)
	if math.IsNaN(float64(out)) {
		return 0, errors.New("NaN for float not allowed")
	}
	return out, nil
}

func (dec *Decoder) ReadFloat64() (float64, error) {
	bits, err := dec.ReadUint64()
	if err != nil {
		return 0, err
	}
	out := math.Float64frombits(bits)
	if math.IsNaN(out) {
		return 0, errors.New("NaN for float not allowed")
	}
	return out, nil
}

func (dec *Decoder) ReadString() (string, error) {
	l, err := dec.ReadUint32()
	if err != nil {
		return "", err
	}
	if l == 0 {
		return "", nil
	}
	tmp2, err := dec.ReadNBytes(int(l))
	if err != nil {
		return "", err
	}
	out := string(tmp2)
	return out, nil
}

var (
	marshalableType   = reflect.TypeOf((*BorshMarshaler)(nil)).Elem()
	unmarshalableType = reflect.TypeOf((*BorshUnmarshaler)(nil)).Elem()
)

func (dec *Decoder) deserialize(rv reflect.Value, keepNil bool) error {
	rt := rv.Type()
	if reflect.PtrTo(rt).Implements(unmarshalableType) {
		m := reflect.New(rt)
		val := m.Interface()
		err := val.(BorshUnmarshaler).UnmarshalBorsh(dec)
		if err != nil {
			return err
		}
		rv.Set(reflect.ValueOf(val).Elem())
		return err
	}

	if rt.Kind() == reflect.Uint8 {
		tmp, err := dec.ReadUint8()
		if err != nil {
			return err
		}
		rv.Set(reflect.ValueOf(tmp).Convert(rt))
		return nil
	}

	switch rt.Kind() {
	case reflect.Bool:
		val, err := dec.ReadBool()
		if err != nil {
			return err
		}
		rv.SetBool(val)
		return nil
	case reflect.Int8:
		val, err := dec.ReadInt8()
		if err != nil {
			return err
		}
		rv.SetInt(int64(val))
		return nil
	case reflect.Int16:
		val, err := dec.ReadInt16()
		if err != nil {
			return err
		}
		rv.SetInt(int64(val))
		return nil
	case reflect.Int32:
		val, err := dec.ReadInt32()
		if err != nil {
			return err
		}
		rv.SetInt(int64(val))
		return nil
	case reflect.Int64:
		val, err := dec.ReadInt64()
		if err != nil {
			return err
		}
		rv.SetInt(val)
		return nil
	case reflect.Int:
		// TODO: check system x32
		val, err := dec.ReadInt()
		if err != nil {
			return err
		}
		rv.SetInt(int64(val))
		return nil
	case reflect.Uint8:
		val, err := dec.ReadUint8()
		if err != nil {
			return err
		}
		rv.SetUint(uint64(val))
		return nil
	case reflect.Uint16:
		val, err := dec.ReadUint16()
		if err != nil {
			return err
		}
		rv.SetUint(uint64(val))
		return nil
	case reflect.Uint32:
		val, err := dec.ReadUint32()
		if err != nil {
			return err
		}
		rv.SetUint(uint64(val))
		return nil
	case reflect.Uint64:
		val, err := dec.ReadUint64()
		if err != nil {
			return err
		}
		rv.SetUint(val)
		return nil
	case reflect.Uint:
		// TODO: check system x32
		val, err := dec.ReadUint()
		if err != nil {
			return err
		}
		rv.SetUint(uint64(val))
		return nil
	case reflect.Float32:
		val, err := dec.ReadFloat32()
		if err != nil {
			return err
		}
		rv.SetFloat(float64(val))
		return nil
	case reflect.Float64:
		val, err := dec.ReadFloat64()
		if err != nil {
			return err
		}
		rv.SetFloat(val)
		return nil
	case reflect.String:
		val, err := dec.ReadString()
		if err != nil {
			return err
		}
		rv.SetString(val)
		return nil
	case reflect.Array:
		l := rt.Len()
		a := reflect.New(rt).Elem()
		for i := 0; i < l; i++ {
			err := dec.deserialize(a.Index(i), false)
			if err != nil {
				return err
			}
		}
		rv.Set(a)
		return nil
	case reflect.Slice:
		l, err := dec.ReadUint32()
		if err != nil {
			return err
		}
		if l == 0 {
			return nil
		}
		rv.Set(reflect.MakeSlice(rt, int(l), int(l)))
		for i := 0; i < int(l); i++ {
			err := dec.deserialize(rv.Index(i), false)
			if err != nil {
				return err
			}
		}
		return nil
	case reflect.Map:
		l, err := dec.ReadUint32()
		if err != nil {
			return err
		}
		if l == 0 {
			// If the map has no content, keep it nil.
			return nil
		}
		rv.Set(reflect.MakeMap(rt))
		for i := 0; i < int(l); i++ {
			key := reflect.New(rt.Key())
			err := dec.deserialize(key.Elem(), false)
			if err != nil {
				return err
			}
			val := reflect.New(rt.Elem())
			err = dec.deserialize(val.Elem(), false)
			if err != nil {
				return err
			}
			rv.SetMapIndex(key.Elem(), val.Elem())
		}
		return nil
	case reflect.Ptr:
		valid, err := dec.ReadUint8()
		if err != nil {
			return err
		}
		if valid == 0 {
			if keepNil {
				return nil
			}
			// rv.Set(reflect.Zero(rt))
			return nil
		} else {
			p := reflect.New(rt.Elem())
			err := dec.deserialize(p.Elem(), false)
			if err != nil {
				return err
			}
			rv.Set(p)
			return nil
		}
	case reflect.Struct:
		if rt == reflect.TypeOf(*big.NewInt(0)) {
			err := dec.deserializeUint128(rv)
			if err != nil {
				return err
			}
			return nil
		} else {
			err := dec.deserializeStruct(rv)
			if err != nil {
				return err
			}
			return nil
		}
	case reflect.Invalid:
		// skip
		return nil
	case reflect.Interface:
		// skip
		// TODO: check if is UnmarshalerBorsh
		return nil
	default:
		return fmt.Errorf("decoding not supported for %q", rt)
	}
}

func (dec *Decoder) deserializeComplexEnum(rv reflect.Value) error {
	rt := rv.Type()
	// read enum identifier
	tmp, err := dec.ReadUint8()
	if err != nil {
		return err
	}
	enum := Enum(tmp)
	rv.Field(0).Set(reflect.ValueOf(enum).Convert(rv.Field(0).Type()))
	// read enum field, if necessary
	if int(enum)+1 >= rt.NumField() {
		return errors.New("complex enum too large")
	}
	return dec.deserialize(rv.Field(int(enum)+1), true)
}

func (dec *Decoder) deserializeStruct(rv reflect.Value) error {
	rt := rv.Type()

	// handle complex enum, if necessary
	if rt.NumField() > 0 {
		// if the first field has type borsh.Enum and is flagged with "borsh_enum"
		// we have a complex enum
		firstField := rt.Field(0)
		if firstField.Type.Kind() == reflect.Uint8 &&
			firstField.Tag.Get("borsh_enum") == "true" {
			return dec.deserializeComplexEnum(rv)
		}
	}

	for i := 0; i < rt.NumField(); i++ {
		structField := rt.Field(i)
		tag := structField.Tag
		if tag.Get("borsh_skip") == "true" {
			continue
		}

		// Skip unexported fields:
		v := rv.FieldByName(structField.Name)
		if !v.CanInterface() {
			continue
		}
		if !v.CanSet() {
			// This means that the field cannot be set, to fix this
			// we need to create a pointer to said field
			if !v.CanAddr() {
				// we cannot create a point to field skipping
				return fmt.Errorf("unable to decode a none setup struc field %q with type %q", structField.Name, v.Kind())
			}
			v = v.Addr()
		}
		if !v.CanSet() {
			continue
		}

		err := dec.deserialize(v, true)
		if err != nil {
			return err
		}
	}

	return nil
}

func (dec *Decoder) deserializeUint128(rv reflect.Value) error {
	d, err := dec.ReadNBytes(16)
	if err != nil {
		return err
	}
	// make it big-endian
	for i, j := 0, 15; i < j; i, j = i+1, j-1 {
		d[i], d[j] = d[j], d[i]
	}
	var u big.Int
	u.SetBytes(d[:])

	rv.Set(reflect.ValueOf(u))
	return nil
}
