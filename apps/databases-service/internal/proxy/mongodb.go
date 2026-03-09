package proxy

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"

	"github.com/obiente/cloud/apps/shared/pkg/logger"
)

const (
	mongoOpMsg    = 2013
	mongoHeaderSz = 16
)

// handleMongoDB handles a MongoDB client connection
func (p *Proxy) handleMongoDB(clientConn net.Conn) {
	defer clientConn.Close()

	// Read the first OP_MSG from the client
	header, payload, err := readMongoMessage(clientConn)
	if err != nil {
		logger.Debug("Failed to read MongoDB message: %v", err)
		return
	}

	opcode := binary.LittleEndian.Uint32(header[12:16])
	if opcode != mongoOpMsg {
		// For non-OP_MSG opcodes, we can't extract $db - reject
		sendMongoError(clientConn, header, "unsupported opcode, use MongoDB 3.6+ wire protocol")
		return
	}

	// Parse OP_MSG to extract $db field
	dbName, err := extractMongoDBName(payload)
	if err != nil {
		sendMongoError(clientConn, header, fmt.Sprintf("failed to parse message: %v", err))
		return
	}

	if dbName == "" {
		sendMongoError(clientConn, header, "no $db field in message")
		return
	}

	// The $db field in MongoDB might be "admin" for initial isMaster/hello.
	// We need to look at the actual database being connected to.
	// For routing, we use the database name from the connection string which
	// MongoDB drivers send in the initial handshake's $db field.
	route, ok := p.registry.Lookup(dbName)
	if !ok {
		sendMongoError(clientConn, header, fmt.Sprintf("database \"%s\" not found", dbName))
		return
	}

	// Handle sleeping/stopped databases
	var backendAddr string
	if route.Stopped {
		if route.DBStatus == 5 { // STOPPED - no auto-wake
			sendMongoError(clientConn, header, "database is stopped")
			return
		}
		// SLEEPING (12) - wake on connect
		addr, err := p.wakeAndConnect(route)
		if err != nil {
			logger.Error("Failed to wake database %s: %v", route.DatabaseID, err)
			sendMongoError(clientConn, header, "database failed to start")
			return
		}
		backendAddr = addr
	} else {
		if route.ContainerIP == "" {
			sendMongoError(clientConn, header, "database is not available")
			return
		}
		backendAddr = net.JoinHostPort(route.ContainerIP, fmt.Sprintf("%d", route.InternalPort))
	}

	p.registry.TouchRoute(route.DatabaseID)

	// Connect to backend
	backendConn, err := net.DialTimeout("tcp", backendAddr, dialTimeout)
	if err != nil {
		logger.Error("Failed to connect to MongoDB backend %s: %v", backendAddr, err)
		sendMongoError(clientConn, header, "database is temporarily unavailable")
		return
	}
	defer backendConn.Close()

	// Replay the first message to backend
	fullMsg := append(header, payload...)
	if _, err := backendConn.Write(fullMsg); err != nil {
		logger.Error("Failed to replay message to backend: %v", err)
		sendMongoError(clientConn, header, "backend connection failed")
		return
	}

	// Bidirectional forwarding
	p.bidirectionalCopy(clientConn, backendConn)
}

// readMongoMessage reads a MongoDB wire protocol message
// Returns the 16-byte header and the remaining payload
func readMongoMessage(conn net.Conn) ([]byte, []byte, error) {
	header := make([]byte, mongoHeaderSz)
	if _, err := io.ReadFull(conn, header); err != nil {
		return nil, nil, err
	}

	msgLen := int(binary.LittleEndian.Uint32(header[0:4]))
	if msgLen < mongoHeaderSz || msgLen > 48*1024*1024 { // Max 48MB
		return nil, nil, fmt.Errorf("invalid MongoDB message length: %d", msgLen)
	}

	payload := make([]byte, msgLen-mongoHeaderSz)
	if _, err := io.ReadFull(conn, payload); err != nil {
		return nil, nil, err
	}

	return header, payload, nil
}

// extractMongoDBName extracts the $db field from an OP_MSG payload
func extractMongoDBName(payload []byte) (string, error) {
	if len(payload) < 5 {
		return "", fmt.Errorf("OP_MSG payload too short")
	}

	// Skip flagBits (4 bytes)
	pos := 4

	// Read sections
	for pos < len(payload) {
		if pos >= len(payload) {
			break
		}

		kind := payload[pos]
		pos++

		if kind == 0 {
			// Type 0: Body - single BSON document
			if pos+4 > len(payload) {
				break
			}
			docLen := int(binary.LittleEndian.Uint32(payload[pos : pos+4]))
			if docLen < 5 || pos+docLen > len(payload) {
				break
			}

			// Parse BSON document to find $db field
			dbName := findBSONString(payload[pos:pos+docLen], "$db")
			if dbName != "" {
				return dbName, nil
			}
			pos += docLen
		} else if kind == 1 {
			// Type 1: Document Sequence - skip
			if pos+4 > len(payload) {
				break
			}
			seqLen := int(binary.LittleEndian.Uint32(payload[pos : pos+4]))
			if seqLen < 4 {
				break
			}
			pos += seqLen
		} else {
			break
		}
	}

	return "", nil
}

// findBSONString finds a string field in a BSON document by key
// Simple BSON parser - only handles string type (0x02)
func findBSONString(doc []byte, key string) string {
	if len(doc) < 5 {
		return ""
	}

	pos := 4 // Skip document length

	for pos < len(doc)-1 {
		if doc[pos] == 0 { // End of document
			break
		}

		elemType := doc[pos]
		pos++

		// Read element name (null-terminated)
		nameEnd := pos
		for nameEnd < len(doc) && doc[nameEnd] != 0 {
			nameEnd++
		}
		if nameEnd >= len(doc) {
			break
		}
		name := string(doc[pos:nameEnd])
		pos = nameEnd + 1

		if elemType == 0x02 && name == key { // String type
			if pos+4 > len(doc) {
				break
			}
			strLen := int(binary.LittleEndian.Uint32(doc[pos : pos+4]))
			pos += 4
			if strLen < 1 || pos+strLen > len(doc) {
				break
			}
			return string(doc[pos : pos+strLen-1]) // -1 to strip null terminator
		}

		// Skip value based on type
		pos = skipBSONValue(doc, pos, elemType)
		if pos < 0 {
			break
		}
	}

	return ""
}

// skipBSONValue advances past a BSON value of the given type
func skipBSONValue(doc []byte, pos int, elemType byte) int {
	switch elemType {
	case 0x01: // Double
		return pos + 8
	case 0x02, 0x0D, 0x0E: // String, JavaScript, Symbol
		if pos+4 > len(doc) {
			return -1
		}
		strLen := int(binary.LittleEndian.Uint32(doc[pos : pos+4]))
		return pos + 4 + strLen
	case 0x03, 0x04: // Document, Array
		if pos+4 > len(doc) {
			return -1
		}
		docLen := int(binary.LittleEndian.Uint32(doc[pos : pos+4]))
		return pos + docLen
	case 0x05: // Binary
		if pos+4 > len(doc) {
			return -1
		}
		binLen := int(binary.LittleEndian.Uint32(doc[pos : pos+4]))
		return pos + 4 + 1 + binLen // length + subtype + data
	case 0x06, 0x0A: // Undefined, Null
		return pos
	case 0x07: // ObjectId
		return pos + 12
	case 0x08: // Boolean
		return pos + 1
	case 0x09, 0x11, 0x12: // DateTime, Timestamp, Int64
		return pos + 8
	case 0x10: // Int32
		return pos + 4
	case 0x13: // Decimal128
		return pos + 16
	case 0xFF, 0x7F: // MinKey, MaxKey
		return pos
	default:
		return -1
	}
}

// sendMongoError sends a MongoDB OP_MSG error response
func sendMongoError(conn net.Conn, requestHeader []byte, message string) {
	// Build a simple BSON error document: {ok: 0, errmsg: "...", code: 26}
	doc := buildBSONErrorDoc(message)

	// Build OP_MSG response
	var payload []byte
	// flagBits (4 bytes)
	payload = append(payload, 0, 0, 0, 0)
	// Section kind 0 (body)
	payload = append(payload, 0)
	// BSON document
	payload = append(payload, doc...)

	// Build header
	msgLen := uint32(mongoHeaderSz + len(payload))
	header := make([]byte, mongoHeaderSz)
	binary.LittleEndian.PutUint32(header[0:4], msgLen)

	// RequestID
	binary.LittleEndian.PutUint32(header[4:8], 1)

	// ResponseTo - use the request's RequestID
	if len(requestHeader) >= 8 {
		copy(header[8:12], requestHeader[4:8])
	}

	// Opcode OP_MSG
	binary.LittleEndian.PutUint32(header[12:16], mongoOpMsg)

	conn.Write(header)
	conn.Write(payload)
}

// buildBSONErrorDoc builds a BSON document: {ok: 0.0, errmsg: "...", code: 26}
func buildBSONErrorDoc(message string) []byte {
	var doc []byte

	// Placeholder for document length
	doc = append(doc, 0, 0, 0, 0)

	// ok: 0.0 (double, type 0x01)
	doc = append(doc, 0x01)
	doc = append(doc, []byte("ok")...)
	doc = append(doc, 0)
	okVal := make([]byte, 8) // 0.0 as float64
	doc = append(doc, okVal...)

	// errmsg: "..." (string, type 0x02)
	doc = append(doc, 0x02)
	doc = append(doc, []byte("errmsg")...)
	doc = append(doc, 0)
	strLen := make([]byte, 4)
	binary.LittleEndian.PutUint32(strLen, uint32(len(message)+1))
	doc = append(doc, strLen...)
	doc = append(doc, []byte(message)...)
	doc = append(doc, 0)

	// code: 26 (int32, type 0x10)
	doc = append(doc, 0x10)
	doc = append(doc, []byte("code")...)
	doc = append(doc, 0)
	codeVal := make([]byte, 4)
	binary.LittleEndian.PutUint32(codeVal, 26) // NamespaceNotFound
	doc = append(doc, codeVal...)

	// Document terminator
	doc = append(doc, 0)

	// Set document length
	binary.LittleEndian.PutUint32(doc[0:4], uint32(len(doc)))

	return doc
}
