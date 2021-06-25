package stream

import (
	"bytes"
	"encoding/binary"
	"errors"
	"unsafe"

	"github.com/go-gulfstream/gulfstream/pkg/util"
)

const (
	magicNumber   = uint16(791)
	containerSize = 60
)

var ErrInvalidInputData = errors.New("stream: unmarshal error: invalid data input")

func (s *Stream) MarshalBinary() (data []byte, err error) {
	rawState, err := s.state.MarshalBinary()
	if err != nil {
		return nil, err
	}
	if len(rawState) == 0 {
		return nil, ErrInvalidInputData
	}
	w := newWriter(rawState)
	w.container = s
	if err := util.ErrOneOf(
		w.writeMagicNumber,
		w.writePayloadSize,
		w.writeNameSize,
		w.writeID,
		w.writeOwnerID,
		w.writeName,
		w.writeVersion,
		w.writeUpdatedAt,
		w.writePayload(rawState),
	); err != nil {
		return nil, err
	}
	return w.buf.Bytes(), nil
}

func (s *Stream) UnmarshalBinary(data []byte) error {
	if len(data) < containerSize {
		return ErrInvalidInputData
	}
	reader := newReader(data)
	reader.container = s
	if err := util.ErrOneOf(
		reader.checkMagicNumber,
		reader.readPayloadSize,
		reader.readNameSize,
		reader.readID,
		reader.readOwnerID,
		reader.readName,
		reader.readVersion,
		reader.readUpdatedAt,
	); err != nil {
		return err
	}
	rawPayload, err := reader.readPayload()
	if err != nil {
		return err
	}
	if s.state == nil {
		return nil
	}
	return s.state.UnmarshalBinary(rawPayload)
}

type writer struct {
	buf       *bytes.Buffer
	prev      uintptr
	container *Stream
	payload   []byte
}

func newWriter(payload []byte) *writer {
	return &writer{
		buf:     bytes.NewBuffer(nil),
		payload: payload,
	}
}

func (w *writer) writeMagicNumber() error {
	return binary.Write(w.buf, binary.LittleEndian, magicNumber)
}

func (w *writer) writePayloadSize() error {
	return binary.Write(w.buf, binary.LittleEndian, uint32(len(w.payload)))
}

func (w *writer) writeNameSize() error {
	return binary.Write(w.buf, binary.LittleEndian, uint32(len(w.container.name)))
}

func (w *writer) writeID() error {
	return binary.Write(w.buf, binary.LittleEndian, w.container.id)
}

func (w *writer) writeOwnerID() error {
	return binary.Write(w.buf, binary.LittleEndian, w.container.owner)
}

func (w *writer) writeName() error {
	return binary.Write(w.buf, binary.LittleEndian, []byte(w.container.name))
}

func (w *writer) writeVersion() error {
	return binary.Write(w.buf, binary.LittleEndian, int64(w.container.Version()))
}

func (w *writer) writeUpdatedAt() error {
	return binary.Write(w.buf, binary.LittleEndian, w.container.updatedAt)
}

func (w *writer) writePayload(data []byte) func() error {
	return func() error {
		return binary.Write(w.buf, binary.LittleEndian, data)
	}
}

type reader struct {
	reader      *bytes.Reader
	data        []byte
	prev        uintptr
	payloadSize uint32
	nameSize    uint32
	container   *Stream
}

func newReader(data []byte) *reader {
	return &reader{
		reader: bytes.NewReader(data),
		data:   data,
	}
}

func (r *reader) next(offset uintptr) {
	r.reader.Reset(r.data[r.prev : r.prev+offset])
	r.prev += offset
}

func (r *reader) checkMagicNumber() error {
	r.next(unsafe.Sizeof(magicNumber))
	var val uint16
	if err := binary.Read(r.reader, binary.LittleEndian, &val); err != nil {
		return err
	}
	if val != magicNumber {
		return ErrInvalidInputData
	}
	return nil
}

func (r *reader) readPayloadSize() error {
	r.next(unsafe.Sizeof(r.payloadSize))
	return binary.Read(r.reader, binary.LittleEndian, &r.payloadSize)
}

func (r *reader) readNameSize() error {
	r.next(unsafe.Sizeof(r.nameSize))
	return binary.Read(r.reader, binary.LittleEndian, &r.nameSize)
}

func (r *reader) readID() error {
	r.next(unsafe.Sizeof(r.container.id))
	return binary.Read(r.reader, binary.LittleEndian, &r.container.id)
}

func (r *reader) readOwnerID() error {
	r.next(unsafe.Sizeof(r.container.owner))
	return binary.Read(r.reader, binary.LittleEndian, &r.container.owner)
}

func (r *reader) readVersion() error {
	r.next(unsafe.Sizeof(r.container.version))
	var v int64
	if err := binary.Read(r.reader, binary.LittleEndian, &v); err != nil {
		return err
	}
	r.container.version = int(v)
	return nil
}

func (r *reader) readUpdatedAt() error {
	r.next(unsafe.Sizeof(r.container.updatedAt))
	return binary.Read(r.reader, binary.LittleEndian, &r.container.updatedAt)
}

func (r *reader) readName() error {
	r.next(uintptr(r.nameSize))
	v := make([]byte, r.nameSize)
	if err := binary.Read(r.reader, binary.LittleEndian, &v); err != nil {
		return err
	}
	r.container.name = string(v)
	return nil
}

func (r *reader) readPayload() ([]byte, error) {
	r.next(uintptr(r.payloadSize))
	b := make([]byte, r.payloadSize)
	if err := binary.Read(r.reader, binary.LittleEndian, &b); err != nil {
		return nil, err
	}
	return b, nil
}
