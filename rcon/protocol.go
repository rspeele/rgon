package rcon

import (
	"io"
	"net"
	"time"
)

// These timeout variables can be set outside the package.
var (
	// Time allowed for connecting to server.
	OpenTimeout = 5 * time.Second
	// Time allowed for complete authentication.
	AuthTimeout = 5 * time.Second
	// Time allowed for sending a command packet.
	SendTimeout = 3 * time.Second
	// Time allowed for receiving a reply packet.
	ReplyTimeout = 3 * time.Second
	// Tolerance for sequential reply packets.
	SeqTimeout = 200 * time.Millisecond
)

type Session struct {
	Address, password string
	reqid             rconInt
	conn              net.Conn
}

func (rs *Session) send(mode rconInt, cmd string) error {
	op := "send packet"
	rs.reqid++
	s := newSendPacket(rs.reqid, mode, cmd)
	err := writeSendPacket(s, rs.conn)
	if err != nil {
		err = &RconError{op, rs.Address, err}
	}
	return err
}

func (rs *Session) recv() (*recvPacket, error) {
	op := "receive packet"
	p, err := readRecvPacket(rs.conn)
	if err != nil {
		return nil, &RconError{op, rs.Address, err}
	}
	if p.reqid != rs.reqid {
		return p, &RconError{op, rs.Address, &ProtocolError{"mismatched request ID from server", ProtocolWarning}}
	}
	return p, nil
}

func (rs *Session) auth() error {
	rs.conn.SetDeadline(time.Now().Add(AuthTimeout))
	op := "authenticate session"
	err := rs.send(serverdataAuth, rs.password)
	if err != nil {
		return &RconError{op, rs.Address, err}
	}
	rep, err := rs.recv()
	/* There is a case in which Session.recv() returns
	 * both a valid *RecvPacket and a non-nil error.
	 * That case corresponds to a request ID mismatch,
	 * which should be ignored at this stage of authentication.
	 */
	if rep == nil && err != nil {
		return &RconError{op, rs.Address, err}
	}
	/* The server may send an extra packet with mode
	 * SERVERDATA_RESPONSE_VALUE in response to
	 * our auth request. Just keep reading packets
	 * until a SERVERDATA_AUTH_RESPONSE is received.
	 */
	for serverdataAuthResponse != rep.mode {
		rep, err = rs.recv()
		if rep != nil {
			if "" != rep.str1 || "" != rep.str2 { // these should be empty...
				msg := "received unexpected response: `" + rep.str1 + rep.str2 + "`"
				return &RconError{op, rs.Address, &ProtocolError{msg, ProtocolWarning}}
			}
		} else if err != nil {
			return &RconError{op, rs.Address, err}
		}
	}
	if reqAuthFailed == rep.reqid {
		return &RconError{op, rs.Address, &ProtocolError{"bad password", ProtocolFatal}}
	}
	return nil
}

func wasTimeoutError(err error) bool {
	if e, ok := err.(*RconError); ok {
		if n, ok := e.Err.(net.Error); ok && n.Timeout() {
			return true
		}
	}
	return false
}

func changesRconPassword(ln string) string {
	cmds := ParseConsoleCommands(ln)
	for i := 0; i < len(cmds); i++ {
		if cmds[i].Var == "rcon_password" {
			return cmds[i].Arg
		}
	}
	return ""
}

func (rs *Session) Command(cmd string, output io.Writer) error {
	rs.conn.SetDeadline(time.Now().Add(SendTimeout))
	err := rs.send(serverdataExecCommand, cmd)
	if err != nil {
		return err
	}
	/* If the the command directly changes the rcon password,
	 * automatically reconnect with the new password after
	 * sending and do not expect response packets.
	 */
	if pwd := changesRconPassword(cmd); pwd != "" {
		rs.password = pwd
		return rs.Reconnect()
	}
	for timeout, attempt := ReplyTimeout, 0; ; attempt++ {
		var p *recvPacket
		start := time.Now()
		rs.conn.SetDeadline(start.Add(timeout))
		p, err = rs.recv()
		/* The time taken to receive this packet, plus the SeqTimeout
		 * tolerance, will be the time limit for receiving the next packet.
		 */
		timeout = time.Now().Sub(start) + SeqTimeout
		if err != nil {
			if CoreError(err) == io.EOF {
				/* Connection was closed, go ahead and try reconnecting.
				 */
				return rs.Reconnect()
			}
			if attempt != 0 && wasTimeoutError(err) {
				/* If we get a timeout looking for a consecutive packet,
				 * stop looking, but don't report the timeout as an error.
				 */
				err = nil
			}
			break
		}
		output.Write([]byte(p.str1))
		if p.size < endPacketSize {
			/* If this packet had plenty of empty space,
			 * then there shouldn't be any more.
			 */
			break
		}
	}
	return err
}

func (rs *Session) Close() {
	if rs.conn != nil {
		rs.conn.Close()
	}
}

func (rs *Session) Reconnect() error {
	rs.Close()
	conn, err := net.DialTimeout("tcp", rs.Address, OpenTimeout)
	if err != nil {
		return &RconError{"connect", rs.Address, err}
	}
	rs.conn = conn
	err = rs.auth()
	if err != nil {
		rs.Close()
		rs.conn = nil
	}
	return err
}

func ErrorLevel(err error) int {
	// First check the error against known constants.
	switch err {
	case io.EOF, io.ErrClosedPipe:
		return ProtocolBroken
	}
	// Then try to go by its type.
	switch e := err.(type) {
	case nil:
		return ProtocolOk
	case *RconError:
		return ErrorLevel(e.Err)
	case *ProtocolError:
		return e.Level
	case net.Error:
		return ProtocolBroken
	}
	// If unknown, assume the worst.
	return ProtocolFatal
}

func NewSession(addr, password string) (*Session, error) {
	_, _, err := net.SplitHostPort(addr)
	if err != nil {
		if e, ok := err.(*net.AddrError); ok && e.Err == "missing port in address" {
			addr += ":" + rconPort
		} else {
			return nil, err
		}
	}
	session := &Session{addr, password, 0, nil}
	return session, session.Reconnect()
}
