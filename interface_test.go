package borsh

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
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

func TestMarshalBorsh(t *testing.T) {
	buf := new(bytes.Buffer)
	e := &Example{Prefix: 0xaa, Value: 72}
	enc := NewEncoder(buf)
	enc.Encode(e)

	assert.Equal(t, []byte{
		0xaa, 0x00, 0x00, 0x00, 0x48,
	}, buf.Bytes())
}

func TestUnmarshalBorsh(t *testing.T) {
	buf := []byte{
		0xaa, 0x00, 0x00, 0x00, 0x48,
	}

	e := &Example{}
	d := NewDecoder(bytes.NewReader(buf))
	err := d.Decode(e)
	assert.NoError(t, err)
	assert.Equal(t, e, &Example{Value: 72, Prefix: 0xaa})
}
