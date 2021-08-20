package borsh

type BorshMarshaler interface {
	MarshalBorsh(encoder *Encoder) error
}

type BorshUnmarshaler interface {
	UnmarshalBorsh(decoder *Decoder) error
}
