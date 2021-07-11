package command

import (
	"bytes"
	"encoding"
	"encoding/binary"
	"errors"
	"fmt"
	"reflect"
	"unsafe"

	"github.com/go-gulfstream/gulfstream/pkg/util"

	"github.com/go-gulfstream/gulfstream/pkg/codec"
)

const (
	commandMagicNumber = uint16(121)
	replyMagicNumber   = uint16(122)
	containerSize      = int(unsafe.Sizeof(Command{})) - 24
)

var (
	ErrInvalidInputData = errors.New("command: invalid data input for codec")
	ErrCodecNotFound    = errors.New("command: codec not found")
)

var defaultCodec = NewCodec()

type Encoding interface {
	Decode([]byte) (*Command, error)
	Encode(*Command) ([]byte, error)
}

type Codec struct {
	codec map[string]codec.Codec
}

func NewCodec() *Codec {
	return &Codec{
		codec: make(map[string]codec.Codec),
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

func (c *Codec) encodePayload(cmd *Command) ([]byte, error) {
	if cmd.payload == nil {
		return nil, nil
	}
	_, found := c.codec[cmd.name]
	if !found {
		return nil, fmt.Errorf("%w %s", ErrCodecNotFound, cmd)
	}
	return cmd.Payload().MarshalBinary()
}

func (c *Codec) encodeContainer(command *Command, payload []byte) ([]byte, error) {
	w := newCommandWriter(command, payload)
	if err := util.ErrOneOf(
		w.writeMagicNumber,
		w.writePayloadSize,
		w.writeNameSize,
		w.writeStreamSize,
		w.writeID,
		w.writeStreamID,
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
	reader := newCommandReader(data)
	reader.container = new(Command)
	if err := util.ErrOneOf(
		reader.checkMagicNumber,
		reader.readPayloadSize,
		reader.readNameSize,
		reader.readStreamSize,
		reader.readID,
		reader.readStreamID,
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

func (c *Codec) decodePayload(command string, data []byte) (codec.Codec, error) {
	if len(data) == 0 {
		return nil, nil
	}
	cc, found := c.codec[command]
	if !found {
		return nil, fmt.Errorf("command: decoder for %s payload not found", command)
	}
	val := reflect.New(reflect.TypeOf(cc).Elem())
	if dec, ok := val.Interface().(encoding.BinaryUnmarshaler); ok {
		if err := dec.UnmarshalBinary(data); err != nil {
			return nil, err
		}
		return val.Interface().(codec.Codec), nil
	}
	return nil, fmt.Errorf("command: decode payload for %s", command)
}

func (c *Codec) RegisterMap(commands map[string]codec.Codec) {
	for command, cc := range commands {
		c.Register(command, cc)
	}
}

func (c *Codec) Register(command string, cc codec.Codec) {
	if cc == nil {
		return
	}
	val := reflect.ValueOf(cc)
	if val.Kind() != reflect.Ptr {
		panic("command: Codec.Register(non-pointer " + command + ")")
	}
	c.codec[command] = cc
}

func RegisterCodec(command string, cc codec.Codec) {
	defaultCodec.Register(command, cc)
}

func RegisterCodecs(commands map[string]codec.Codec) {
	defaultCodec.RegisterMap(commands)
}

func Encode(command *Command) ([]byte, error) {
	return defaultCodec.Encode(command)
}

func Decode(data []byte) (*Command, error) {
	return defaultCodec.Decode(data)
}

type commandWriter struct {
	buf       *bytes.Buffer
	prev      uintptr
	container *Command
	payload   []byte
}

func newCommandWriter(c *Command, payload []byte) *commandWriter {
	return &commandWriter{
		buf:       bytes.NewBuffer(nil),
		container: c,
		payload:   payload,
	}
}

func (w *commandWriter) writeMagicNumber() error {
	return binary.Write(w.buf, binary.LittleEndian, commandMagicNumber)
}

func (w *commandWriter) writePayloadSize() error {
	return binary.Write(w.buf, binary.LittleEndian, uint32(len(w.payload)))
}

func (w *commandWriter) writeNameSize() error {
	return binary.Write(w.buf, binary.LittleEndian, uint32(len(w.container.name)))
}

func (w *commandWriter) writeStreamSize() error {
	return binary.Write(w.buf, binary.LittleEndian, uint32(len(w.container.streamName)))
}

func (w *commandWriter) writeID() error {
	return binary.Write(w.buf, binary.LittleEndian, w.container.id)
}

func (w *commandWriter) writeStreamID() error {
	return binary.Write(w.buf, binary.LittleEndian, w.container.streamID)
}

func (w *commandWriter) writeName() error {
	return binary.Write(w.buf, binary.LittleEndian, []byte(w.container.name))
}

func (w *commandWriter) writeStreamName() error {
	return binary.Write(w.buf, binary.LittleEndian, []byte(w.container.streamName))
}

func (w *commandWriter) writeCreatedAt() error {
	return binary.Write(w.buf, binary.LittleEndian, w.container.createdAt)
}

func (w *commandWriter) writePayload() error {
	return binary.Write(w.buf, binary.LittleEndian, w.payload)
}

type commandReader struct {
	reader      *bytes.Reader
	data        []byte
	prev        uintptr
	nameSize    uint32
	streamSize  uint32
	payloadSize uint32
	container   *Command
}

func newCommandReader(data []byte) *commandReader {
	return &commandReader{
		reader: bytes.NewReader(data),
		data:   data,
	}
}

func (r *commandReader) next(offset uintptr) {
	r.reader.Reset(r.data[r.prev : r.prev+offset])
	r.prev += offset
}

func (r *commandReader) checkMagicNumber() error {
	r.next(unsafe.Sizeof(commandMagicNumber))
	var val uint16
	if err := binary.Read(r.reader, binary.LittleEndian, &val); err != nil {
		return err
	}
	if val != commandMagicNumber {
		return ErrInvalidInputData
	}
	return nil
}

func (r *commandReader) readNameSize() error {
	r.next(unsafe.Sizeof(r.nameSize))
	return binary.Read(r.reader, binary.LittleEndian, &r.nameSize)
}

func (r *commandReader) readStreamSize() error {
	r.next(unsafe.Sizeof(r.streamSize))
	return binary.Read(r.reader, binary.LittleEndian, &r.streamSize)
}

func (r *commandReader) readPayloadSize() error {
	r.next(unsafe.Sizeof(r.payloadSize))
	return binary.Read(r.reader, binary.LittleEndian, &r.payloadSize)
}

func (r *commandReader) readID() error {
	r.next(unsafe.Sizeof(r.container.id))
	return binary.Read(r.reader, binary.LittleEndian, &r.container.id)
}

func (r *commandReader) readStreamID() error {
	r.next(unsafe.Sizeof(r.container.streamID))
	return binary.Read(r.reader, binary.LittleEndian, &r.container.streamID)
}

func (r *commandReader) readCreatedAt() error {
	r.next(unsafe.Sizeof(r.container.createdAt))
	return binary.Read(r.reader, binary.LittleEndian, &r.container.createdAt)
}

func (r *commandReader) readName() error {
	r.next(uintptr(r.nameSize))
	v := make([]byte, r.nameSize)
	if err := binary.Read(r.reader, binary.LittleEndian, &v); err != nil {
		return err
	}
	r.container.name = string(v)
	return nil
}

func (r *commandReader) readStreamName() error {
	r.next(uintptr(r.streamSize))
	v := make([]byte, r.streamSize)
	if err := binary.Read(r.reader, binary.LittleEndian, &v); err != nil {
		return err
	}
	r.container.streamName = string(v)
	return nil
}

func (r *commandReader) readPayload() ([]byte, error) {
	r.next(uintptr(r.payloadSize))
	b := make([]byte, r.payloadSize)
	if err := binary.Read(r.reader, binary.LittleEndian, &b); err != nil {
		return nil, err
	}
	return b, nil
}
