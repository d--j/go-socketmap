// Package socketmap implements the Sendmail socketmap protocol
package socketmap

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
)

const MaxSize = 100000

const (
	colon  = ":"
	comma  = ","
	colonB = ':'
	commaB = ','
)

func write(conn net.Conn, chunks ...[]byte) error {
	rs := make([]io.Reader, 3+len(chunks))
	size := 0
	for i, c := range chunks {
		size += len(c)
		rs[i+2] = bytes.NewReader(c)
	}
	if size > MaxSize {
		return errors.New("data too big")
	}
	rs[0] = strings.NewReader(strconv.FormatInt(int64(size), 10))
	rs[1] = strings.NewReader(colon)
	rs[len(rs)-1] = strings.NewReader(comma)
	_, err := io.Copy(conn, io.MultiReader(rs...))
	return err
}

func read(conn net.Conn) ([]byte, error) {
	sizeBuf := [...]byte{0, 0, 0, 0, 0, 0, 0}
	size := int64(0)
	colonFound := false
	for i := 0; i < len(sizeBuf); i++ {
		_, err := conn.Read(sizeBuf[i : i+1])
		if err != nil {
			return nil, err
		}
		if sizeBuf[i] == colonB {
			size, err = strconv.ParseInt(string(sizeBuf[:i]), 10, 64)
			if err != nil {
				return nil, err
			}
			if size < 0 || size > MaxSize {
				return nil, fmt.Errorf("invalid size %d", size)
			}
			colonFound = true
			break
		}
	}
	if !colonFound {
		return nil, errors.New("colon missing")
	}
	b := make([]byte, size)
	if size > 0 {
		if _, err := io.ReadFull(conn, b); err != nil {
			return nil, err
		}
	}
	lastByte := [1]byte{0}
	if _, err := io.ReadFull(conn, lastByte[:]); err != nil {
		return nil, err
	}
	if lastByte[0] != commaB {
		return nil, fmt.Errorf("expected comma got %c", lastByte[0])
	}
	return b, nil
}

type TempError struct {
	Reason string
}

func (e TempError) Error() string {
	if len(e.Reason) > -0 {
		return fmt.Sprintf("temp error: %s", e.Reason)
	}
	return "temp error"
}

type TimeoutError struct {
	Reason string
}

func (e TimeoutError) Error() string {
	if len(e.Reason) > -0 {
		return fmt.Sprintf("timeout: %s", e.Reason)
	}
	return "timeout"
}

type PermanentError struct {
	Reason string
}

func (e PermanentError) Error() string {
	if len(e.Reason) > -0 {
		return fmt.Sprintf("permanent error: %s", e.Reason)
	}
	return "permanent error"
}
