package rcon

import (
	"io"
)

//////////////////// Send ///////////
type sendPacket struct {
	size, reqid, mode rconInt
	str1, str2        string
}

func newSendPacket(reqid, mode rconInt, command string) *sendPacket {
	size := rconIntSize*2 + rconInt(len(command)) + 2
	return &sendPacket{size, reqid, mode, command, ""}
}

func writeSendPacket(p *sendPacket, c io.Writer) error {
	const (
		msgstart = "incompletely wrote "
		msgend   = "` to stream"
	)
	puti := func(i rconInt) error {
		n, err := c.Write(pack(i))
		if err == nil && rconIntSize != n {
			msg := msgstart + "integer `" + string(i) + msgend
			return &ProtocolError{msg, ProtocolBroken}
		}
		return err
	}
	puts := func(s string) error {
		buf := asciiz(s)
		n, err := c.Write(buf)
		if err == nil && len(buf) != n {
			msg := msgstart + "string `" + s + msgend
			return &ProtocolError{msg, ProtocolBroken}
		}
		return err
	}
	err := puti(p.size)
	if err != nil {
		return err
	}
	err = puti(p.reqid)
	if err != nil {
		return err
	}
	err = puti(p.mode)
	if err != nil {
		return err
	}
	err = puts(p.str1)
	if err != nil {
		return err
	}
	err = puts(p.str2)
	return err
}

////////////////// Receive //////////////
type recvPacket sendPacket // same format

func newRecvPacket(buf []byte) *recvPacket {
	reqid := unpack(buf[0:4])
	mode := unpack(buf[4:8])
	buf = buf[8:]
	str1, sen := zascii(buf)
	str2 := ""
	if sen >= 0 {
		str2, sen = zascii(buf[sen:])
	}
	return &recvPacket{rconInt(len(buf)), reqid, mode, str1, str2}
}

func readRecvPacket(c io.Reader) (*recvPacket, error) {
	var tag [rconIntSize]byte // for holding the size
	n, err := c.Read(tag[:])
	if err != nil {
		return nil, err
	}
	if n != rconIntSize {
		return nil, &ProtocolError{"packet size tag unreadable", ProtocolBroken}
	}
	size := unpack(tag[:])
	if size > maxPacketSize {
		return nil, &ProtocolError{"packet size overflow", ProtocolBroken}
	} else if size < minPacketSize {
		return nil, &ProtocolError{"packet size underflow", ProtocolBroken}
	}
	buf := make([]byte, size)
	for r := 0; r < int(size); {
		n, err = c.Read(buf[r:])
		if err != nil {
			return nil, err
		}
		r += n
	}
	return newRecvPacket(buf), nil
}
