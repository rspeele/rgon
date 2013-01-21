package rcon

// Error types

// An RconError includes the operation that was taking place,
// the network address it was working with,  and the underlying error.
// The Err field may be handed down from a failed external function (e.g. Conn.Read())
// or it may be a ProtocolError, which is the result of some miscommunication
// between client and server (e.g. bad password).
type RconError struct {
	Op   string
	Addr string
	Err  error
}

func (e *RconError) Error() string {
	return "rcon " + e.Op + " " + e.Addr + ": " + e.Err.Error()
}

/* Get to the bottom of an error...
 */
func CoreError(err error) error {
	if e, ok := err.(*RconError); ok {
		return CoreError(e.Err)
	}
	return err
}

type ProtocolError struct {
	Problem string
	Level   int
}

const (
	ProtocolOk = iota
	ProtocolWarning
	ProtocolBroken
	ProtocolFatal
)

func (e *ProtocolError) Error() string {
	return e.Problem
}

// Integer type used in network messages
type rconInt int32

const rconIntSize = 4

const (
	serverdataResponseValue = 0
	serverdataAuthResponse  = 2
	serverdataExecCommand   = 2
	serverdataAuth          = 3

	minPacketSize = 10
	maxPacketSize = 8202
	endPacketSize = 2500 // heuristic, used in reading command response

	reqAuthFailed = -1

	rconPort = "27015"
)

// Integer to little endian packed bytes
func pack(i rconInt) []byte {
	return []byte{byte(i), byte(i >> 8), byte(i >> 16), byte(i >> 24)}
}

// Little endian packed bytes to integer
func unpack(s []byte) rconInt {
	return rconInt(s[0]) |
		rconInt(s[1])<<8 |
		rconInt(s[2])<<16 |
		rconInt(s[3])<<24
}

// String to null-terminated slice of bytes
func asciiz(str string) []byte {
	last := len(str)
	buf := make([]byte, last+1)
	copy(buf, str)
	buf[last] = 0
	return buf
}

// Parses a single null-terminated string out of buf
// Returns the index of the terminating byte
func zascii(buf []byte) (string, int) {
	var i int
	for i = 0; i < len(buf); i++ {
		if 0 == buf[i] {
			if i == 0 {
				return "", i
			} else {
				return string(buf[0 : i-1]), i
			}
		}
	}
	return string(buf), -1
}
