package borsh

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type Example struct {
	Prefix byte
	Value  uint32
}

func (e *Example) UnmarshalBorsh(decoder *Decoder) (err error) {
	if e.Prefix, err = decoder.ReadByte(); err != nil {
		return err
	}
	if e.Value, err = decoder.ReadUint32(); err != nil {
		return err
	}
	return nil
}

func (e *Example) MarshalBorsh(encoder *Encoder) error {
	if err := encoder.WriteByte(e.Prefix); err != nil {
		return err
	}
	return encoder.WriteUint32(e.Value)
}

func TestBorshEncodeDecode(t *testing.T) {
	buf := new(bytes.Buffer)
	x := &Example{Prefix: byte('b'), Value: 72}
	enc := NewEncoder(buf)
	err := enc.Encode(x)
	require.NoError(t, err)

	assert.Equal(t, []byte{
		byte('b'), 72, 0x00, 0x00, 0,
	}, buf.Bytes())

	{
		y := &Example{}
		d := NewDecoder(buf)
		err := d.Decode(y)
		assert.NoError(t, err)
		assert.Equal(t, x, y)
	}
}

func TestBorshMarshal(t *testing.T) {
	buf := new(bytes.Buffer)
	e := &Example{Prefix: byte('b'), Value: 84}
	enc := NewEncoder(buf)
	err := enc.Encode(e)
	require.NoError(t, err)

	assert.Equal(t, []byte{
		byte('b'), 84, 0x00, 0x00, 0x00,
	}, buf.Bytes())
}

func TestBorshUnmarshal(t *testing.T) {
	buf := []byte{
		byte('b'), 72, 0x00, 0x00, 0x00,
	}

	e := &Example{}
	d := NewDecoder(bytes.NewReader(buf))
	err := d.Decode(e)
	assert.NoError(t, err)
	assert.Equal(t, &Example{Value: 72, Prefix: byte('b')}, e)
}
