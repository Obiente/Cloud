package proxy

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"

	"github.com/obiente/cloud/apps/shared/pkg/logger"
)

const (
	pgSSLRequestCode = 80877103
	pgProtocolV3     = 196608 // 3.0
)

// handlePostgres handles a PostgreSQL client connection
func (p *Proxy) handlePostgres(clientConn net.Conn) {
	defer clientConn.Close()

	// Read the first message (could be SSLRequest or StartupMessage)
	msg, err := readPgMessage(clientConn)
	if err != nil {
		logger.Debug("Failed to read PostgreSQL startup: %v", err)
		return
	}

	// Handle SSLRequest
	if len(msg) == 8 {
		code := binary.BigEndian.Uint32(msg[4:8])
		if code == pgSSLRequestCode {
			// Respond with 'N' (no SSL)
			if _, err := clientConn.Write([]byte{'N'}); err != nil {
				logger.Debug("Failed to send SSL rejection: %v", err)
				return
			}
			// Now read the actual StartupMessage
			msg, err = readPgMessage(clientConn)
			if err != nil {
				logger.Debug("Failed to read PostgreSQL startup after SSL: %v", err)
				return
			}
		}
	}

	// Parse StartupMessage
	if len(msg) < 8 {
		logger.Debug("PostgreSQL startup message too short")
		sendPgError(clientConn, "protocol error: message too short")
		return
	}

	version := binary.BigEndian.Uint32(msg[4:8])
	if version != pgProtocolV3 {
		logger.Debug("Unsupported PostgreSQL protocol version: %d", version)
		sendPgError(clientConn, "unsupported protocol version")
		return
	}

	// Extract key-value pairs from startup message
	params := parsePgStartupParams(msg[8:])
	dbName, ok := params["database"]
	if !ok || dbName == "" {
		sendPgError(clientConn, "no database specified")
		return
	}

	// Look up route
	route, ok := p.registry.Lookup(dbName)
	if !ok {
		sendPgError(clientConn, fmt.Sprintf("database \"%s\" does not exist", dbName))
		return
	}

	// Handle sleeping/stopped databases
	var backendAddr string
	if route.Stopped {
		if route.DBStatus == 5 { // STOPPED - no auto-wake
			sendPgError(clientConn, "database is stopped")
			return
		}
		// SLEEPING (12) - wake on connect
		addr, err := p.wakeAndConnect(route)
		if err != nil {
			logger.Error("Failed to wake database %s: %v", route.DatabaseID, err)
			sendPgError(clientConn, "database failed to start")
			return
		}
		backendAddr = addr
	} else {
		if route.ContainerIP == "" {
			sendPgError(clientConn, "database is not available")
			return
		}
		backendAddr = net.JoinHostPort(route.ContainerIP, fmt.Sprintf("%d", route.InternalPort))
	}

	p.registry.TouchRoute(route.DatabaseID)

	// Connect to backend
	backendConn, err := net.DialTimeout("tcp", backendAddr, dialTimeout)
	if err != nil {
		logger.Error("Failed to connect to PostgreSQL backend %s: %v", backendAddr, err)
		sendPgError(clientConn, "database is temporarily unavailable")
		return
	}
	defer backendConn.Close()

	// Replay the startup message to backend
	if _, err := backendConn.Write(msg); err != nil {
		logger.Error("Failed to send startup to backend: %v", err)
		sendPgError(clientConn, "backend connection failed")
		return
	}

	// Bidirectional forwarding
	p.bidirectionalCopy(clientConn, backendConn)
}

// readPgMessage reads a PostgreSQL protocol message (length-prefixed)
func readPgMessage(r io.Reader) ([]byte, error) {
	// Read 4-byte length
	lenBuf := make([]byte, 4)
	if _, err := io.ReadFull(r, lenBuf); err != nil {
		return nil, err
	}

	msgLen := binary.BigEndian.Uint32(lenBuf)
	if msgLen < 4 || msgLen > 1<<20 { // sanity check: max 1MB
		return nil, fmt.Errorf("invalid message length: %d", msgLen)
	}

	// Read the rest of the message
	msg := make([]byte, msgLen)
	copy(msg[:4], lenBuf)
	if _, err := io.ReadFull(r, msg[4:]); err != nil {
		return nil, err
	}

	return msg, nil
}

// parsePgStartupParams extracts null-terminated key-value pairs
func parsePgStartupParams(data []byte) map[string]string {
	params := make(map[string]string)
	for len(data) > 1 { // stop at final null byte
		// Find key
		keyEnd := 0
		for keyEnd < len(data) && data[keyEnd] != 0 {
			keyEnd++
		}
		if keyEnd >= len(data) {
			break
		}
		key := string(data[:keyEnd])
		data = data[keyEnd+1:]

		// Find value
		valEnd := 0
		for valEnd < len(data) && data[valEnd] != 0 {
			valEnd++
		}
		if valEnd > len(data) {
			break
		}
		value := string(data[:valEnd])
		if valEnd < len(data) {
			data = data[valEnd+1:]
		} else {
			data = nil
		}

		params[key] = value
	}
	return params
}

// sendPgError sends a PostgreSQL ErrorResponse to the client
func sendPgError(conn net.Conn, message string) {
	// ErrorResponse: 'E' + length + severity + message + terminator
	// Fields: S=FATAL, C=08006, M=message
	var buf []byte
	buf = append(buf, 'E')

	// Build the fields
	var fields []byte
	fields = append(fields, 'S')
	fields = append(fields, []byte("FATAL")...)
	fields = append(fields, 0)
	fields = append(fields, 'C')
	fields = append(fields, []byte("08006")...)
	fields = append(fields, 0)
	fields = append(fields, 'M')
	fields = append(fields, []byte(message)...)
	fields = append(fields, 0)
	fields = append(fields, 0) // terminator

	// Length includes itself (4 bytes) + fields
	length := uint32(4 + len(fields))
	lenBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(lenBytes, length)
	buf = append(buf, lenBytes...)
	buf = append(buf, fields...)

	conn.Write(buf)
}
