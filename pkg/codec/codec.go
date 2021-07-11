package codec

import "encoding"

type Codec interface {
	encoding.BinaryUnmarshaler
	encoding.BinaryMarshaler
}
