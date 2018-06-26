package wapsnmp

import (
	"encoding/hex"
	"net"
	"testing"
	"time"
)

// Internal structure to take care of responses.
type expectAndRespond struct {
	expect  string
	respond []string
}

/* A udpStub is a UDP stubbing tool.

   You test UDP programs by using

   NewUdpStub().Expect("aabbcc").andReturn([]string("ddeeff")

   This will return a net.Conn that will simulate receiving a "ddeeff"
   packet when you send a "aabbcc" packet (expressed as hexdumps for
   readability).
*/
type udpStub struct {
	expectedPacketsAndResponses map[string][]byte
	ignoreUnknownPackets        bool
	expectResponses             []*expectAndRespond
	queuedPackets               []string

	t      *testing.T
	closed bool
}

// NewUdpStub creates a new udpStub.
func NewUdpStub(t *testing.T) *udpStub {
	return &udpStub{t: t}
}

// Expect declares that you expect this connection to be sent a hex-encoded string.
func (u *udpStub) Expect(packet string) *expectAndRespond {
	e := &expectAndRespond{packet, []string{}}
	u.expectResponses = append(u.expectResponses, e)
	return e
}

// AndRespond registers a response for a string you set to be expected with Expect.
func (e *expectAndRespond) AndRespond(packets []string) *expectAndRespond {
	for _, packet := range packets {
		e.respond = append(e.respond, packet)
	}
	return e
}

/* Read reads bytes from the connection.

   Only returns stuff you put in the object with the AndRespond method.
*/
func (u *udpStub) Read(b []byte) (n int, err error) {
	if len(u.queuedPackets) > 0 {
		val, err := hex.DecodeString(u.queuedPackets[0])
		if err != nil {
			u.t.Fatalf("Error while decoding expected packet: '%v'", err)
		}

		for idx, vb := range val {
			b[idx] = vb
		}
		u.queuedPackets = u.queuedPackets[1:]
		return len(val), nil
	}
	return 0, nil
}

/* Write writes bytes to the connection.

   If the bytes were expected, it can trigger responses.
   If an unexpected packet is written it will trigger an error.
*/
func (u *udpStub) Write(b []byte) (n int, err error) {
	// We're expecting the first packet in the expectResponses array.
	realPacket := hex.EncodeToString(b)
	expectedPacket := u.expectResponses[0].expect

	if realPacket == expectedPacket {
		for _, response := range u.expectResponses[0].respond {
			u.queuedPackets = append(u.queuedPackets, response)
		}
		u.expectResponses = u.expectResponses[1:]
	} else {
		if !u.ignoreUnknownPackets {
			u.t.Errorf("error: received  '%v'\n        expected '%v'", realPacket, expectedPacket)
		}
	}

	return len(b), nil
}

/* Close closes the udpStub.

   This sets a boolean flag so you can check the connection was really closed.
*/
func (u *udpStub) Close() error {
	u.closed = true
	return nil
}

// CheckClosed checks if the udpStub was closed, signaling an error if it wasn't.
func (u *udpStub) CheckClosed() {
	if !u.closed {
		u.t.Errorf("Connection was not closed")
	}
}

// LocalAddr so udpStub implements the net.conn interface, but doesn't actually return anything.
func (u *udpStub) LocalAddr() net.Addr {
	return nil
}

// RemoteAddr so udpStub implements the net.conn interface, but doesn't actually return anything.
func (u *udpStub) RemoteAddr() net.Addr {
	return nil
}

// SetDeadline so udpStub implements the net.conn interface, but doesn't actually return anything.
func (u *udpStub) SetDeadline(t time.Time) error {
	return nil
}

// SetReadDeadline so udpStub implements the net.conn interface, but doesn't actually return anything.
func (u *udpStub) SetReadDeadline(t time.Time) error {
	return nil
}

// SetWriteDeadline so udpStub implements the net.conn interface, but doesn't actually return anything.
func (u *udpStub) SetWriteDeadline(t time.Time) error {
	return nil
}
