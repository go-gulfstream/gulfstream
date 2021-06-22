package command

import (
	"bytes"
	"encoding"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"unsafe"

	"github.com/go-gulfstream/gulfstream/pkg/util"

	"github.com/go-gulfstream/gulfstream/pkg/codec"
)

const (
	magicNumber   = uint16(121)
	containerSize = int(unsafe.Sizeof(Command{})) - 24
)

var (
	ErrInvalidInputData = errors.New("command: decode error: invalid data input")
	ErrCodecNotFound    = errors.New("command: codec not found")
)

var defaultCodec = NewCodec()

type Encoding interface {
	Decode([]byte) (*Command, error)
	Encode(*Command) ([]byte, error)
}

type Codec struct {
	codec  map[string]codec.Codec
	types  map[string]reflect.Type
	global codec.Codec
}

func NewCodec() *Codec {
	return &Codec{
		codec: make(map[string]codec.Codec),
		types: make(map[string]reflect.Type),
	}
}

func (c *Codec) Decode(data []byte) (*Command, error) {
	if len(data) < containerSize {
		return nil, ErrInvalidInputData
	}
	command, rawPayload, err := c.decodeContainer(data)
	if err != nil {
		return nil, err
	}
	payload, err := c.decodePayload(command.name, rawPayload)
	if err != nil {
		return nil, err
	}
	command.payload = payload
	return command, nil
}

func (c *Codec) Encode(command *Command) ([]byte, error) {
	payload, err := c.encodePayload(command)
	if err != nil {
		return nil, err
	}
	return c.encodeContainer(command, payload)
}

func (c *Codec) encodePayload(command *Command) ([]byte, error) {
	if command.payload == nil {
		return nil, nil
	}
	if c.global != nil {
		return c.global.Encode(command.payload)
	}
	cc, found := c.codec[command.name]
	if found {
		return cc.Encode(command.payload)
	}
	_, found = c.types[command.name]
	if found {
		if enc, ok := command.payload.(encoding.BinaryMarshaler); ok {
			return enc.MarshalBinary()
		}
		if enc, ok := command.payload.(json.Marshaler); ok {
			return enc.MarshalJSON()
		}
	}
	return nil, fmt.Errorf("%w for %s command",
		ErrCodecNotFound, command.name)
}

func (c *Codec) encodeContainer(command *Command, payload []byte) ([]byte, error) {
	w := newWriter(command, payload)
	if err := util.ErrOneOf(
		w.writeMagicNumber,
		w.writePayloadSize,
		w.writeNameSize,
		w.writeStreamSize,
		w.writeID,
		w.writeStreamID,
		w.writeOwnerID,
		w.writeName,
		w.writeStreamName,
		w.writeCreatedAt,
		w.writePayload,
	); err != nil {
		return nil, err
	}
	return w.buf.Bytes(), nil
}

func (c *Codec) decodeContainer(data []byte) (*Command, []byte, error) {
	reader := newReader(data)
	reader.container = new(Command)
	if err := util.ErrOneOf(
		reader.checkMagicNumber,
		reader.readPayloadSize,
		reader.readNameSize,
		reader.readStreamSize,
		reader.readID,
		reader.readStreamID,
		reader.readOwnerID,
		reader.readName,
		reader.readStreamName,
		reader.readCreatedAt,
	); err != nil {
		return nil, nil, err
	}
	payload, err := reader.readPayload()
	if err != nil {
		return nil, nil, err
	}
	return reader.container, payload, nil
}

func (c *Codec) decodePayload(name string, data []byte) (interface{}, error) {
	if len(data) == 0 {
		return nil, nil
	}
	if c.global != nil {
		return c.global.Decode(data)
	}
	cc, found := c.codec[name]
	if found {
		return cc.Decode(data)
	}
	t, found := c.types[name]
	if found && t.Kind() == reflect.Ptr {
		val := reflect.New(t.Elem())
		if dec, ok := val.Interface().(encoding.BinaryUnmarshaler); ok {
			if err := dec.UnmarshalBinary(data); err != nil {
				return nil, err
			}
		}
		if dec, ok := val.Interface().(json.Unmarshaler); ok {
			if err := dec.UnmarshalJSON(data); err != nil {
				return nil, err
			}
		}
		return val.Interface(), nil
	}
	return nil, fmt.Errorf("%w for %s command",
		ErrCodecNotFound, name)
}

func (c *Codec) Register(name string, cc codec.Codec) {
	if name == "*" {
		c.global = cc
	} else {
		c.codec[name] = cc
	}
}

func (c *Codec) AddKnownType(types ...interface{}) error {
	for _, typ := range types {
		_, binUn := typ.(encoding.BinaryUnmarshaler)
		_, jsonUn := typ.(json.Unmarshaler)
		if !binUn && !jsonUn {
			return fmt.Errorf("%s does not support encoding.BinaryUnmarshaler or json.Unmarshaler",
				reflect.TypeOf(typ).String(),
			)
		}
		typ := reflect.TypeOf(typ)
		if typ.Kind() != reflect.Ptr {
			return fmt.Errorf("non-pointer %s",
				reflect.TypeOf(typ).String())
		}
		path := strings.Split(typ.String(), ".")
		name := path[len(path)-1]
		c.types[name] = typ
	}
	return nil
}

func AddKnownType(types ...interface{}) error {
	return defaultCodec.AddKnownType(types...)
}

func Register(name string, cc codec.Codec) {
	defaultCodec.Register(name, cc)
}

func Encode(command *Command) ([]byte, error) {
	return defaultCodec.Encode(command)
}

func Decode(data []byte) (*Command, error) {
	return defaultCodec.Decode(data)
}

type writer struct {
	buf       *bytes.Buffer
	prev      uintptr
	container *Command
	payload   []byte
}

func newWriter(c *Command, payload []byte) *writer {
	return &writer{
		buf:       bytes.NewBuffer(nil),
		container: c,
		payload:   payload,
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

func (w *writer) writeStreamSize() error {
	return binary.Write(w.buf, binary.LittleEndian, uint32(len(w.container.streamName)))
}

func (w *writer) writeID() error {
	return binary.Write(w.buf, binary.LittleEndian, w.container.id)
}

func (w *writer) writeStreamID() error {
	return binary.Write(w.buf, binary.LittleEndian, w.container.streamID)
}

func (w *writer) writeOwnerID() error {
	return binary.Write(w.buf, binary.LittleEndian, w.container.owner)
}

func (w *writer) writeName() error {
	return binary.Write(w.buf, binary.LittleEndian, []byte(w.container.name))
}

func (w *writer) writeStreamName() error {
	return binary.Write(w.buf, binary.LittleEndian, []byte(w.container.streamName))
}

func (w *writer) writeCreatedAt() error {
	return binary.Write(w.buf, binary.LittleEndian, w.container.createdAt)
}

func (w *writer) writePayload() error {
	return binary.Write(w.buf, binary.LittleEndian, w.payload)
}

type reader struct {
	reader      *bytes.Reader
	data        []byte
	prev        uintptr
	nameSize    uint32
	streamSize  uint32
	payloadSize uint32
	container   *Command
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

func (r *reader) readNameSize() error {
	r.next(unsafe.Sizeof(r.nameSize))
	return binary.Read(r.reader, binary.LittleEndian, &r.nameSize)
}

func (r *reader) readStreamSize() error {
	r.next(unsafe.Sizeof(r.streamSize))
	return binary.Read(r.reader, binary.LittleEndian, &r.streamSize)
}

func (r *reader) readPayloadSize() error {
	r.next(unsafe.Sizeof(r.payloadSize))
	return binary.Read(r.reader, binary.LittleEndian, &r.payloadSize)
}

func (r *reader) readID() error {
	r.next(unsafe.Sizeof(r.container.id))
	return binary.Read(r.reader, binary.LittleEndian, &r.container.id)
}

func (r *reader) readStreamID() error {
	r.next(unsafe.Sizeof(r.container.streamID))
	return binary.Read(r.reader, binary.LittleEndian, &r.container.streamID)
}

func (r *reader) readOwnerID() error {
	r.next(unsafe.Sizeof(r.container.owner))
	return binary.Read(r.reader, binary.LittleEndian, &r.container.owner)
}

func (r *reader) readCreatedAt() error {
	r.next(unsafe.Sizeof(r.container.createdAt))
	return binary.Read(r.reader, binary.LittleEndian, &r.container.createdAt)
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

func (r *reader) readStreamName() error {
	r.next(uintptr(r.streamSize))
	v := make([]byte, r.streamSize)
	if err := binary.Read(r.reader, binary.LittleEndian, &v); err != nil {
		return err
	}
	r.container.streamName = string(v)
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
