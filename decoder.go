package borsh

import (
	"bytes"
	"encoding/binary"
	"errors"
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
	val, err := dec.deserialize(reflect.TypeOf(dst).Elem())
	if err != nil {
		return err
	}
	reflect.ValueOf(dst).Elem().Set(reflect.ValueOf(val))
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

func read(reader io.ByteReader, n int) ([]byte, error) {
	return readNBytes(n, reader)
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

func (dec *Decoder) deserialize(rt reflect.Type) (interface{}, error) {
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
	case reflect.Int8:
		tmp, err := dec.ReadUint8()
		if err != nil {
			return nil, err
		}
		return int8(tmp), nil
	case reflect.Int16:
		tmp, err := dec.ReadNBytes(2)
		if err != nil {
			return nil, err
		}
		return int16(binary.LittleEndian.Uint16(tmp)), nil
	case reflect.Int32:
		tmp, err := dec.ReadNBytes(4)
		if err != nil {
			return nil, err
		}
		return int32(binary.LittleEndian.Uint32(tmp)), nil
	case reflect.Int64:
		tmp, err := dec.ReadNBytes(8)
		if err != nil {
			return nil, err
		}
		return int64(binary.LittleEndian.Uint64(tmp)), nil
	case reflect.Int:
		tmp, err := dec.ReadNBytes(8)
		if err != nil {
			return nil, err
		}
		return int(binary.LittleEndian.Uint64(tmp)), nil
	case reflect.Uint8:
		tmp, err := dec.ReadUint8()
		if err != nil {
			return nil, err
		}
		return tmp, nil
	case reflect.Uint16:
		tmp, err := dec.ReadNBytes(2)
		if err != nil {
			return nil, err
		}
		return uint16(binary.LittleEndian.Uint16(tmp)), nil
	case reflect.Uint32:
		tmp, err := dec.ReadNBytes(4)
		if err != nil {
			return nil, err
		}
		return uint32(binary.LittleEndian.Uint32(tmp)), nil
	case reflect.Uint64:
		tmp, err := dec.ReadNBytes(8)
		if err != nil {
			return nil, err
		}
		return uint64(binary.LittleEndian.Uint64(tmp)), nil
	case reflect.Uint:
		tmp, err := dec.ReadNBytes(8)
		if err != nil {
			return nil, err
		}
		return uint(binary.LittleEndian.Uint64(tmp)), nil
	case reflect.Float32:
		tmp, err := dec.ReadNBytes(4)
		if err != nil {
			return nil, err
		}
		bits := binary.LittleEndian.Uint32(tmp)
		f := math.Float32frombits(bits)
		if math.IsNaN(float64(f)) {
			return nil, errors.New("NaN for float not allowed")
		}
		return f, nil
	case reflect.Float64:
		tmp, err := dec.ReadNBytes(8)
		if err != nil {
			return nil, err
		}
		bits := binary.LittleEndian.Uint64(tmp)
		f := math.Float64frombits(bits)
		if math.IsNaN(f) {
			return nil, errors.New("NaN for float not allowed")
		}
		return f, nil
	case reflect.String:
		tmp, err := dec.ReadNBytes(4)
		if err != nil {
			return nil, err
		}
		l := int(binary.LittleEndian.Uint32(tmp))
		if l == 0 {
			return "", nil
		}
		tmp2, err := dec.ReadNBytes(l)
		if err != nil {
			return nil, err
		}
		s := string(tmp2)
		return s, nil
	case reflect.Array:
		l := rt.Len()
		a := reflect.New(rt).Elem()
		for i := 0; i < l; i++ {
			av, err := dec.deserialize(rt.Elem())
			if err != nil {
				return nil, err
			}
			a.Index(i).Set(reflect.ValueOf(av))
		}
		return a.Interface(), nil
	case reflect.Slice:
		tmp, err := dec.ReadNBytes(4)
		if err != nil {
			return nil, err
		}
		l := int(binary.LittleEndian.Uint32(tmp))
		a := reflect.New(rt).Elem()
		if l == 0 {
			return a.Interface(), nil
		}
		for i := 0; i < l; i++ {
			av, err := dec.deserialize(rt.Elem())
			if err != nil {
				return nil, err
			}
			a = reflect.Append(a, reflect.ValueOf(av))
		}
		return a.Interface(), nil
	case reflect.Map:
		tmp, err := dec.ReadNBytes(4)
		if err != nil {
			return nil, err
		}
		l := int(binary.LittleEndian.Uint32(tmp))
		m := reflect.MakeMap(rt)
		if l == 0 {
			return m.Interface(), nil
		}
		for i := 0; i < l; i++ {
			k, err := dec.deserialize(rt.Key())
			if err != nil {
				return nil, err
			}
			v, err := dec.deserialize(rt.Elem())
			if err != nil {
				return nil, err
			}
			m.SetMapIndex(reflect.ValueOf(k), reflect.ValueOf(v))
		}
		return m.Interface(), nil
	case reflect.Ptr:
		tmp, err := dec.ReadNBytes(1)
		if err != nil {
			return nil, err
		}
		valid := uint8(tmp[0])
		if valid == 0 {
			p := reflect.New(rt.Elem())
			return p.Interface(), nil
		} else {
			p := reflect.New(rt.Elem())
			de, err := dec.deserialize(rt.Elem())
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
	}

	return nil, nil
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
	fv, err := dec.deserialize(rt.Field(int(enum) + 1).Type)
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

		fv, err := dec.deserialize(rt.Field(i).Type)
		if err != nil {
			return nil, err
		}
		v.Field(i).Set(reflect.ValueOf(fv).Convert(field.Type))
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
