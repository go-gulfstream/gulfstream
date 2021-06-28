package command

import (
	"bytes"
	"encoding/binary"
	"errors"
	"unsafe"

	"github.com/go-gulfstream/gulfstream/pkg/util"
)

func (r *Reply) MarshalBinary() ([]byte, error) {
	w := newReplyWriter()
	w.container = r
	if err := util.ErrOneOf(
		w.writeMagicNumber,
		w.writeErrorSize,
		w.writeCommand,
		w.writeCreatedAt,
		w.writeVersion,
		w.writErr,
	); err != nil {
		return nil, err
	}
	return w.buf.Bytes(), nil
}

func (r *Reply) UnmarshalBinary(data []byte) error {
	if len(data) < 38 {
		return ErrInvalidInputData
	}
	reader := newReplyReader(data)
	reader.container = r
	return util.ErrOneOf(
		reader.checkMagicNumber,
		reader.readErrorSize,
		reader.readCommand,
		reader.readCreatedAt,
		reader.readVersion,
		reader.readErr,
	)
}

type replyWriter struct {
	buf       *bytes.Buffer
	prev      uintptr
	container *Reply
}

func newReplyWriter() *replyWriter {
	return &replyWriter{
		buf: bytes.NewBuffer(nil),
	}
}

func (w *replyWriter) writeMagicNumber() error {
	return binary.Write(w.buf, binary.LittleEndian, replyMagicNumber)
}

func (w *replyWriter) writErr() error {
	var err []byte
	if w.container.err != nil {
		err = []byte(w.container.err.Error())
	}
	return binary.Write(w.buf, binary.LittleEndian, err)
}

func (w *replyWriter) writeErrorSize() error {
	var size uint32
	if w.container.err != nil {
		size = uint32(len(w.container.err.Error()))
	}
	return binary.Write(w.buf, binary.LittleEndian, size)
}

func (w *replyWriter) writeCommand() error {
	return binary.Write(w.buf, binary.LittleEndian, w.container.command)
}

func (w *replyWriter) writeVersion() error {
	return binary.Write(w.buf, binary.LittleEndian, int64(w.container.version))
}

func (w *replyWriter) writeCreatedAt() error {
	return binary.Write(w.buf, binary.LittleEndian, w.container.createdAt)
}

type replyReader struct {
	reader    *bytes.Reader
	data      []byte
	prev      uintptr
	errSize   uint32
	container *Reply
}

func newReplyReader(data []byte) *replyReader {
	return &replyReader{
		data:   data,
		reader: bytes.NewReader(data),
	}
}

func (r *replyReader) next(offset uintptr) {
	r.reader.Reset(r.data[r.prev : r.prev+offset])
	r.prev += offset
}

func (r *replyReader) readErr() error {
	if r.errSize == 0 {
		return nil
	}
	r.next(uintptr(r.errSize))
	v := make([]byte, r.errSize)
	if err := binary.Read(r.reader, binary.LittleEndian, &v); err != nil {
		return err
	}
	r.container.err = errors.New(string(v))
	return nil
}

func (r *replyReader) readErrorSize() error {
	r.next(unsafe.Sizeof(r.errSize))
	return binary.Read(r.reader, binary.LittleEndian, &r.errSize)
}

func (r *replyReader) readCommand() error {
	r.next(unsafe.Sizeof(r.container.command))
	return binary.Read(r.reader, binary.LittleEndian, &r.container.command)
}

func (r *replyReader) readVersion() error {
	r.next(unsafe.Sizeof(r.container.version))
	var v int64
	if err := binary.Read(r.reader, binary.LittleEndian, &v); err != nil {
		return err
	}
	r.container.version = int(v)
	return nil
}

func (r *replyReader) readCreatedAt() error {
	r.next(unsafe.Sizeof(r.container.createdAt))
	return binary.Read(r.reader, binary.LittleEndian, &r.container.createdAt)
}

func (r *replyReader) checkMagicNumber() error {
	r.next(unsafe.Sizeof(replyMagicNumber))
	var val uint16
	if err := binary.Read(r.reader, binary.LittleEndian, &val); err != nil {
		return err
	}
	if val != replyMagicNumber {
		return ErrInvalidInputData
	}
	return nil
}
