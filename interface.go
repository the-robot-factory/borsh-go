package borsh

type MarshalerBorsh interface {
	MarshalBorsh(encoder *Encoder) error
}

type UnmarshalerBorsh interface {
	UnmarshalBorsh(decoder *Decoder) error
}
