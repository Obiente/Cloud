package vps

import (
	"bytes"
	"encoding/binary"
	"net"
	"testing"
)

func TestParseProxyProtocolHeaderV1(t *testing.T) {
	payload := []byte("SSH-2.0-test\r\n")
	frame := append([]byte("PROXY TCP4 203.0.113.9 10.0.0.2 12345 22\r\n"), payload...)

	consumed, realIP, ok := parseProxyProtocolHeader(frame)
	if !ok {
		t.Fatalf("expected v1 header to be parsed")
	}
	if realIP != "203.0.113.9" {
		t.Fatalf("expected source IP to be parsed, got %q", realIP)
	}
	if !bytes.Equal(frame[consumed:], payload) {
		t.Fatalf("expected parser to consume only the proxy header")
	}
}

func TestParseProxyProtocolHeaderV2(t *testing.T) {
	srcIP := net.ParseIP("203.0.113.9").To4()
	dstIP := net.ParseIP("10.0.0.2").To4()
	addressBlock := make([]byte, 12)
	copy(addressBlock[0:4], srcIP)
	copy(addressBlock[4:8], dstIP)
	binary.BigEndian.PutUint16(addressBlock[8:10], 12345)
	binary.BigEndian.PutUint16(addressBlock[10:12], 22)

	payload := []byte("SSH-2.0-test\r\n")
	header := append([]byte{}, proxyProtocolV2Signature...)
	header = append(header, 0x21, 0x11)
	length := make([]byte, 2)
	binary.BigEndian.PutUint16(length, uint16(len(addressBlock)))
	header = append(header, length...)
	frame := append(append(header, addressBlock...), payload...)

	consumed, realIP, ok := parseProxyProtocolHeader(frame)
	if !ok {
		t.Fatalf("expected v2 header to be parsed")
	}
	if realIP != "203.0.113.9" {
		t.Fatalf("expected source IP to be parsed, got %q", realIP)
	}
	if !bytes.Equal(frame[consumed:], payload) {
		t.Fatalf("expected parser to consume only the proxy header")
	}
}

func TestParseProxyProtocolHeaderUnknownV1(t *testing.T) {
	frame := []byte("PROXY UNKNOWN\r\nSSH-2.0-test\r\n")

	consumed, realIP, ok := parseProxyProtocolHeader(frame)
	if !ok {
		t.Fatalf("expected UNKNOWN v1 header to be parsed")
	}
	if realIP != "" {
		t.Fatalf("expected no client IP for UNKNOWN header, got %q", realIP)
	}
	if !bytes.Equal(frame[consumed:], []byte("SSH-2.0-test\r\n")) {
		t.Fatalf("expected parser to consume the proxy header")
	}
}
