package main

/* Simple teletype interface to rcon protocol
 */

import (
	"./rcon"
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
)

func readln(f io.Reader) (string, error) {
	var line string
	r := bufio.NewReader(f)
	for pre := true; pre; {
		var buf []byte
		var err error
		buf, pre, err = r.ReadLine()
		if err != nil {
			return line, err
		}
		line += string(buf)
	}
	return line, nil
}
func reconnect(session *rcon.Session, retries int) error {
	var err error
	for i := 0; i < retries; i++ {
		fmt.Fprintf(os.Stderr, "attempting to reconnect... %d / %d\n", i+1, retries)
		err = session.Reconnect()
		if err == nil {
			break
		}
	}
	return err
}
func command(cmd string, session *rcon.Session) error {
retry:
	err := session.Command(cmd, os.Stdout)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		switch rcon.ErrorLevel(err) {
		case rcon.ProtocolBroken:
			err = reconnect(session, 5)
			if err != nil {
				fmt.Fprintln(os.Stderr, "failed to reconnect")
			} else {
				goto retry
			}
		}
	}
	return err
}
func getinfo() (string, string, error) {
	var err error
	var address, password string
	flag.StringVar(&address, "address", "", "IP or hostname of server")
	flag.StringVar(&password, "password", "", "RCON password")
	flag.Parse()
	if address == "" {
		fmt.Print("[address]: ")
		address, err = readln(os.Stdin)
	}
	if err == nil && password == "" {
		fmt.Print("[password]: ")
		password, err = readln(os.Stdin)
	}
	return address, password, err
}
func main() {
	address, password, err := getinfo()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	session, err := rcon.NewSession(address, password)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	defer session.Close()
	for {
		line, err := readln(os.Stdin)
		if err != nil {
			if err != io.EOF {
				fmt.Fprintln(os.Stderr, err)
			}
			break
		}
		err = command(line, session)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			break
		} else {
			fmt.Println()
		}
	}
}
