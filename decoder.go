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
	r io.Reader
}

func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{r: r}
}

func (d *Decoder) Decode(s interface{}) error {
	t := reflect.TypeOf(s)
	if t.Kind() != reflect.Ptr {
		return errors.New("argument must be pointer")
	}
	val, err := deserialize(t, d.r)
	if err != nil {
		return nil
	}
	reflect.ValueOf(s).Elem().Set(reflect.ValueOf(val))
	return nil
}

func (d *Decoder) Close() error {
	return nil
}

// Deserialize `data` according to the schema of `s`, and store the value into it. `s` must be a pointer type variable
// that points to the original schema of `data`.
func Deserialize(s interface{}, data []byte) error {
	reader := bytes.NewReader(data)
	v := reflect.ValueOf(s)
	if v.Kind() != reflect.Ptr {
		return errors.New("passed struct must be pointer")
	}
	result, err := deserialize(reflect.TypeOf(s).Elem(), reader)
	if err != nil {
		return err
	}
	v.Elem().Set(reflect.ValueOf(result))
	return nil
}

func read(r io.Reader, n int) ([]byte, error) {
	b := make([]byte, n)
	l, err := r.Read(b)
	if l != n {
		return nil, errors.New("failed to read required bytes")
	}
	if err != nil {
		return nil, err
	}
	return b, nil
}

func deserialize(t reflect.Type, r io.Reader) (interface{}, error) {
	if t.Kind() == reflect.Uint8 {
		tmp, err := read(r, 1)
		if err != nil {
			return nil, err
		}
		e := reflect.New(t)
		e.Elem().Set(reflect.ValueOf(uint8(tmp[0])).Convert(t))
		return e.Elem().Interface(), nil
	}

	switch t.Kind() {
	case reflect.Int8:
		tmp, err := read(r, 1)
		if err != nil {
			return nil, err
		}
		return int8(tmp[0]), nil
	case reflect.Int16:
		tmp, err := read(r, 2)
		if err != nil {
			return nil, err
		}
		return int16(binary.LittleEndian.Uint16(tmp)), nil
	case reflect.Int32:
		tmp, err := read(r, 4)
		if err != nil {
			return nil, err
		}
		return int32(binary.LittleEndian.Uint32(tmp)), nil
	case reflect.Int64:
		tmp, err := read(r, 8)
		if err != nil {
			return nil, err
		}
		return int64(binary.LittleEndian.Uint64(tmp)), nil
	case reflect.Int:
		tmp, err := read(r, 8)
		if err != nil {
			return nil, err
		}
		return int(binary.LittleEndian.Uint64(tmp)), nil
	case reflect.Uint8:
		tmp, err := read(r, 1)
		if err != nil {
			return nil, err
		}
		return uint8(tmp[0]), nil
	case reflect.Uint16:
		tmp, err := read(r, 2)
		if err != nil {
			return nil, err
		}
		return uint16(binary.LittleEndian.Uint16(tmp)), nil
	case reflect.Uint32:
		tmp, err := read(r, 4)
		if err != nil {
			return nil, err
		}
		return uint32(binary.LittleEndian.Uint32(tmp)), nil
	case reflect.Uint64:
		tmp, err := read(r, 8)
		if err != nil {
			return nil, err
		}
		return uint64(binary.LittleEndian.Uint64(tmp)), nil
	case reflect.Uint:
		tmp, err := read(r, 8)
		if err != nil {
			return nil, err
		}
		return uint(binary.LittleEndian.Uint64(tmp)), nil
	case reflect.Float32:
		tmp, err := read(r, 4)
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
		tmp, err := read(r, 8)
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
		tmp, err := read(r, 4)
		if err != nil {
			return nil, err
		}
		l := int(binary.LittleEndian.Uint32(tmp))
		if l == 0 {
			return "", nil
		}
		tmp2, err := read(r, l)
		if err != nil {
			return nil, err
		}
		s := string(tmp2)
		return s, nil
	case reflect.Array:
		l := t.Len()
		a := reflect.New(t).Elem()
		for i := 0; i < l; i++ {
			av, err := deserialize(t.Elem(), r)
			if err != nil {
				return nil, err
			}
			a.Index(i).Set(reflect.ValueOf(av))
		}
		return a.Interface(), nil
	case reflect.Slice:
		tmp, err := read(r, 4)
		if err != nil {
			return nil, err
		}
		l := int(binary.LittleEndian.Uint32(tmp))
		a := reflect.New(t).Elem()
		if l == 0 {
			return a.Interface(), nil
		}
		for i := 0; i < l; i++ {
			av, err := deserialize(t.Elem(), r)
			if err != nil {
				return nil, err
			}
			a = reflect.Append(a, reflect.ValueOf(av))
		}
		return a.Interface(), nil
	case reflect.Map:
		tmp, err := read(r, 4)
		if err != nil {
			return nil, err
		}
		l := int(binary.LittleEndian.Uint32(tmp))
		m := reflect.MakeMap(t)
		if l == 0 {
			return m.Interface(), nil
		}
		for i := 0; i < l; i++ {
			k, err := deserialize(t.Key(), r)
			if err != nil {
				return nil, err
			}
			v, err := deserialize(t.Elem(), r)
			if err != nil {
				return nil, err
			}
			m.SetMapIndex(reflect.ValueOf(k), reflect.ValueOf(v))
		}
		return m.Interface(), nil
	case reflect.Ptr:
		tmp, err := read(r, 1)
		if err != nil {
			return nil, err
		}
		valid := uint8(tmp[0])
		if valid == 0 {
			p := reflect.New(t.Elem())
			return p.Interface(), nil
		} else {
			p := reflect.New(t.Elem())
			de, err := deserialize(t.Elem(), r)
			if err != nil {
				return nil, err
			}
			p.Elem().Set(reflect.ValueOf(de))
			return p.Interface(), nil
		}
	case reflect.Struct:
		if t == reflect.TypeOf(*big.NewInt(0)) {
			s, err := deserializeUint128(t, r)
			if err != nil {
				return nil, err
			}
			return s, nil
		} else {
			s, err := deserializeStruct(t, r)
			if err != nil {
				return nil, err
			}
			return s, nil
		}
	}

	return nil, nil
}

func deserializeComplexEnum(t reflect.Type, r io.Reader) (interface{}, error) {
	v := reflect.New(t).Elem()
	// read enum identifier
	tmp, err := read(r, 1)
	if err != nil {
		return nil, err
	}
	enum := Enum(tmp[0])
	v.Field(0).Set(reflect.ValueOf(enum))
	// read enum field, if necessary
	if int(enum)+1 >= t.NumField() {
		return nil, errors.New("complex enum too large")
	}
	fv, err := deserialize(t.Field(int(enum)+1).Type, r)
	if err != nil {
		return nil, err
	}
	v.Field(int(enum) + 1).Set(reflect.ValueOf(fv))

	return v.Interface(), nil
}

func deserializeStruct(t reflect.Type, r io.Reader) (interface{}, error) {
	// handle complex enum, if necessary
	if t.NumField() > 0 {
		// if the first field has type borsh.Enum and is flagged with "borsh_enum"
		// we have a complex enum
		firstField := t.Field(0)
		if firstField.Type.Kind() == reflect.Uint8 &&
			firstField.Tag.Get("borsh_enum") == "true" {
			return deserializeComplexEnum(t, r)
		}
	}

	v := reflect.New(t).Elem()

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		tag := field.Tag
		if tag.Get("borsh_skip") == "true" {
			continue
		}

		fv, err := deserialize(t.Field(i).Type, r)
		if err != nil {
			return nil, err
		}
		v.Field(i).Set(reflect.ValueOf(fv).Convert(field.Type))
	}

	return v.Interface(), nil
}

func deserializeUint128(t reflect.Type, r io.Reader) (interface{}, error) {
	d, err := read(r, 16)
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
