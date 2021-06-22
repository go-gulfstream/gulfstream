package stream

import (
	"bytes"
	"fmt"
)

const magic = uint16(791)

func (s *Stream) MarshalBinary() (data []byte, err error) {
	data, err = s.state.MarshalBinary()
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("state data size is zero")
	}
	buf := new(bytes.Buffer)
	// magic, len(streamName), len(state), byte(id), byte(owner), int(version), byte(state)
	//err = errOneOf(
	//	binary.Write(buf, binary.LittleEndian, magic),              // 2
	//	binary.Write(buf, binary.LittleEndian, uint32(len(s.typ))), // 4
	//	binary.Write(buf, binary.LittleEndian, uint32(len(data))),  // 4
	//	binary.Write(buf, binary.LittleEndian, s.id),               // 16
	//	binary.Write(buf, binary.LittleEndian, s.owner),            // 16
	//	binary.Write(buf, binary.LittleEndian, int64(s.version)),   // 8
	//	binary.Write(buf, binary.LittleEndian, []byte(s.typ)),
	//	binary.Write(buf, binary.LittleEndian, data),
	//)
	return buf.Bytes(), err
}

func (s *Stream) UnmarshalBinary(data []byte) error {
	//if len(data) < 50 {
	//	return fmt.Errorf("invalid data size")
	//}
	//mc := binary.LittleEndian.Uint16(data[0:2])
	//if mc != magic {
	//	return fmt.Errorf("unmarshal stream error. invalid stream rawdata")
	//}
	//streamTypeLen := binary.LittleEndian.Uint32(data[2:6])
	//stateLen := binary.LittleEndian.Uint32(data[6:10])
	//state := make([]byte, stateLen)
	//stype := make([]byte, streamTypeLen)
	//var v int64
	//if err := errOneOf(
	//	binary.Read(bytes.NewReader(data[10:26]), binary.LittleEndian, &s.id),
	//	binary.Read(bytes.NewReader(data[26:42]), binary.LittleEndian, &s.owner),
	//	binary.Read(bytes.NewReader(data[42:50]), binary.LittleEndian, &v),
	//	binary.Read(bytes.NewReader(data[50:50+streamTypeLen]), binary.LittleEndian, &stype),
	//	binary.Read(bytes.NewReader(data[50+streamTypeLen:50+streamTypeLen+stateLen]), binary.LittleEndian, &state),
	//); err != nil {
	//	return err
	//}
	//s.version = int(v)
	//if s.typ != string(stype) {
	//	return fmt.Errorf("mismatch stream type. got %s, expected %s",
	//		string(stype), s.typ)
	//}
	//s.typ = string(stype)
	//return s.state.UnmarshalBinary(state)
	return nil
}
