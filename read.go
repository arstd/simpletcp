package simpletcp

//
// import (
// 	"bufio"
// 	"encoding/binary"
// 	"fmt"
// 	"io"
// )
//
// func Read(br *bufio.Reader, fixed [2]byte, maxLength uint32) (f *Frame, err error) {
// 	f = &Frame{}
//
// 	// read fixed header
// 	if err = expectByte(br, "fixed[0]", fixed[0]); err != nil {
// 		return
// 	}
// 	if err = expectByte(br, "fixed[1]", fixed[1]); err != nil {
// 		return
// 	}
//
// 	// read version
// 	if f.Version, err = br.ReadByte(); err != nil {
// 		return
// 	}
//
// 	// read data type
// 	if f.DataType, err = br.ReadByte(); err != nil {
// 		return
// 	}
//
// 	// read message id
// 	if f.MessageId, err = readUint32(br); err != nil {
// 		return
// 	}
//
// 	// read data length
// 	if f.DataLength, err = readUint32(br); err != nil {
// 		return
// 	} else if f.DataLength > maxLength {
// 		err = fmt.Errorf("data length exceed, max %d, but got %d", maxLength, f.DataLength)
// 		return
// 	}
//
// 	// read reserved
// 	if _, err = io.ReadFull(br, f.Reserved[:]); err != nil {
// 		return
// 	}
//
// 	// read data
// 	f.Data = make([]byte, f.DataLength)
// 	if _, err = io.ReadFull(br, f.Data); err != nil {
// 		return
// 	}
//
// 	return
// }
//
// func expectByte(br *bufio.Reader, name string, expect byte) error {
// 	if b, err := br.ReadByte(); err != nil {
// 		return err
// 	} else if b != expect {
// 		err := fmt.Errorf("%s got %x, expect %x", name, b, expect)
// 		return err
// 	}
// 	return nil
// }
//
// func readUint32(br *bufio.Reader) (u uint32, err error) {
// 	if err := binary.Read(br, binary.BigEndian, &u); err != nil {
// 		return 0, err
// 	}
// 	return
// }
