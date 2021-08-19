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
	rt := reflect.TypeOf(dst)
	if rt.Kind() != reflect.Ptr {
		return errors.New("argument must be pointer")
	}
	val, err := dec.deserialize(reflect.TypeOf(dst).Elem(), false)
	if err != nil {
		return err
	}
	rv := reflect.ValueOf(dst)
	rv.Elem().Set(reflect.ValueOf(val))
	return nil
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

func (dec *Decoder) deserialize(rt reflect.Type, keepNil bool) (interface{}, error) {
	if rt.Kind() == reflect.Uint8 {
		tmp, err := dec.ReadUint8()
		if err != nil {
			return nil, err
		}
		e := reflect.New(rt)
		e.Elem().Set(reflect.ValueOf(tmp).Convert(rt))
		return e.Elem().Interface(), nil
	}

	switch rt.Kind() {
	case reflect.Bool:
		return dec.ReadBool()
	case reflect.Int8:
		return dec.ReadInt8()
	case reflect.Int16:
		return dec.ReadInt16()
	case reflect.Int32:
		return dec.ReadInt32()
	case reflect.Int64:
		return dec.ReadInt64()
	case reflect.Int:
		return dec.ReadInt()
	case reflect.Uint8:
		return dec.ReadUint8()
	case reflect.Uint16:
		return dec.ReadUint16()
	case reflect.Uint32:
		return dec.ReadUint32()
	case reflect.Uint64:
		return dec.ReadUint64()
	case reflect.Uint:
		return dec.ReadUint()
	case reflect.Float32:
		return dec.ReadFloat32()
	case reflect.Float64:
		return dec.ReadFloat64()
	case reflect.String:
		return dec.ReadString()
	case reflect.Array:
		l := rt.Len()
		a := reflect.New(rt).Elem()
		for i := 0; i < l; i++ {
			av, err := dec.deserialize(rt.Elem(), false)
			if err != nil {
				return nil, err
			}
			a.Index(i).Set(reflect.ValueOf(av))
		}
		return a.Interface(), nil
	case reflect.Slice:
		l, err := dec.ReadUint32()
		if err != nil {
			return nil, err
		}
		a := reflect.New(rt).Elem()
		if l == 0 {
			return a.Interface(), nil
		}
		for i := 0; i < int(l); i++ {
			av, err := dec.deserialize(rt.Elem(), false)
			if err != nil {
				return nil, err
			}
			a = reflect.Append(a, reflect.ValueOf(av))
		}
		return a.Interface(), nil
	case reflect.Map:
		l, err := dec.ReadUint32()
		if err != nil {
			return nil, err
		}
		m := reflect.MakeMap(rt)
		if l == 0 {
			return m.Interface(), nil
		}
		for i := 0; i < int(l); i++ {
			k, err := dec.deserialize(rt.Key(), false)
			if err != nil {
				return nil, err
			}
			v, err := dec.deserialize(rt.Elem(), false)
			if err != nil {
				return nil, err
			}
			m.SetMapIndex(reflect.ValueOf(k), reflect.ValueOf(v))
		}
		return m.Interface(), nil
	case reflect.Ptr:
		valid, err := dec.ReadUint8()
		if err != nil {
			return nil, err
		}
		if valid == 0 {
			if keepNil {
				return nil, nil
			}
			p := reflect.New(rt.Elem())
			return p.Interface(), nil
		} else {
			p := reflect.New(rt.Elem())
			de, err := dec.deserialize(rt.Elem(), false)
			if err != nil {
				return nil, err
			}
			p.Elem().Set(reflect.ValueOf(de))
			return p.Interface(), nil
		}
	case reflect.Struct:
		if rt == reflect.TypeOf(*big.NewInt(0)) {
			s, err := dec.deserializeUint128(rt)
			if err != nil {
				return nil, err
			}
			return s, nil
		} else {
			s, err := dec.deserializeStruct(rt)
			if err != nil {
				return nil, err
			}
			return s, nil
		}
	default:
		return nil, fmt.Errorf("decoding not supported for %q", rt)
	}
}

func (dec *Decoder) deserializeComplexEnum(rt reflect.Type) (interface{}, error) {
	rv := reflect.New(rt).Elem()
	// read enum identifier
	tmp, err := dec.ReadUint8()
	if err != nil {
		return nil, err
	}
	enum := Enum(tmp)
	rv.Field(0).Set(reflect.ValueOf(enum))
	// read enum field, if necessary
	if int(enum)+1 >= rt.NumField() {
		return nil, errors.New("complex enum too large")
	}
	fv, err := dec.deserialize(rt.Field(int(enum)+1).Type, true)
	if err != nil {
		return nil, err
	}
	rv.Field(int(enum) + 1).Set(reflect.ValueOf(fv))

	return rv.Interface(), nil
}

func (dec *Decoder) deserializeStruct(rt reflect.Type) (interface{}, error) {
	// handle complex enum, if necessary
	if rt.NumField() > 0 {
		// if the first field has type borsh.Enum and is flagged with "borsh_enum"
		// we have a complex enum
		firstField := rt.Field(0)
		if firstField.Type.Kind() == reflect.Uint8 &&
			firstField.Tag.Get("borsh_enum") == "true" {
			return dec.deserializeComplexEnum(rt)
		}
	}

	v := reflect.New(rt).Elem()

	for i := 0; i < rt.NumField(); i++ {
		field := rt.Field(i)
		tag := field.Tag
		if tag.Get("borsh_skip") == "true" {
			continue
		}

		fv, err := dec.deserialize(rt.Field(i).Type, true)
		if err != nil {
			return nil, err
		}
		if fv != nil {
			v.Field(i).Set(reflect.ValueOf(fv).Convert(field.Type))
		}
	}

	return v.Interface(), nil
}

func (dec *Decoder) deserializeUint128(rt reflect.Type) (interface{}, error) {
	d, err := dec.ReadNBytes(16)
	if err != nil {
		return nil, err
	}
	// make it big-endian
	for i, j := 0, 15; i < j; i, j = i+1, j-1 {
		d[i], d[j] = d[j], d[i]
	}
	var u big.Int
	u.SetBytes(d[:])
	return u, nil
}
