package proxy

import (
	"crypto/rand"
	"crypto/sha1"
	"encoding/binary"
	"fmt"
	"io"
	"net"

	"github.com/obiente/cloud/apps/shared/pkg/logger"
)

const (
	mysqlProtocolVersion = 10
	mysqlServerVersion   = "8.0.35-obiente-proxy"
	mysqlCapFlags        = 0x0000_FFFF // All standard capabilities
	mysqlCapFlagsUpper   = 0x0000_00CF // CLIENT_PLUGIN_AUTH | CLIENT_SECURE_CONNECTION | CLIENT_CONNECT_WITH_DB | etc
)

// Capability flags
const (
	clientConnectWithDB     uint32 = 0x00000008
	clientSecureConnection  uint32 = 0x00008000
	clientPluginAuth        uint32 = 0x00080000
	clientPluginAuthLenData uint32 = 0x00200000
)

// handleMySQL handles a MySQL/MariaDB client connection
func (p *Proxy) handleMySQL(clientConn net.Conn) {
	defer clientConn.Close()

	// Generate fake auth challenge
	challenge := make([]byte, 20)
	if _, err := rand.Read(challenge); err != nil {
		logger.Error("Failed to generate MySQL auth challenge: %v", err)
		return
	}

	// Send server greeting to client
	if err := sendMySQLGreeting(clientConn, challenge); err != nil {
		logger.Debug("Failed to send MySQL greeting: %v", err)
		return
	}

	// Read client handshake response
	dbName, username, clientAuthData, clientFlags, err := readMySQLHandshakeResponse(clientConn)
	if err != nil {
		logger.Debug("Failed to read MySQL handshake: %v", err)
		return
	}

	if dbName == "" {
		sendMySQLError(clientConn, 1049, "3D000", "No database specified")
		return
	}

	// Look up route
	route, ok := p.registry.Lookup(dbName)
	if !ok {
		sendMySQLError(clientConn, 1049, "42000", fmt.Sprintf("Unknown database '%s'", dbName))
		return
	}

	// Handle sleeping/stopped databases
	var backendAddr string
	if route.Stopped {
		if route.DBStatus == 5 { // STOPPED - no auto-wake
			sendMySQLError(clientConn, 1049, "HY000", "Database is stopped")
			return
		}
		// SLEEPING (12) - wake on connect
		addr, err := p.wakeAndConnect(route)
		if err != nil {
			logger.Error("Failed to wake database %s: %v", route.DatabaseID, err)
			sendMySQLError(clientConn, 1049, "HY000", "Database failed to start")
			return
		}
		backendAddr = addr
	} else {
		if route.ContainerIP == "" {
			sendMySQLError(clientConn, 1049, "HY000", "Database is not available")
			return
		}
		backendAddr = net.JoinHostPort(route.ContainerIP, fmt.Sprintf("%d", route.InternalPort))
	}

	p.registry.TouchRoute(route.DatabaseID)

	// Connect to backend
	backendConn, err := net.DialTimeout("tcp", backendAddr, dialTimeout)
	if err != nil {
		logger.Error("Failed to connect to MySQL backend %s: %v", backendAddr, err)
		sendMySQLError(clientConn, 1049, "HY000", "Database is temporarily unavailable")
		return
	}
	defer backendConn.Close()

	// Read backend's greeting
	backendGreeting, err := readMySQLPacket(backendConn)
	if err != nil {
		logger.Error("Failed to read backend MySQL greeting: %v", err)
		sendMySQLError(clientConn, 1049, "HY000", "Backend connection failed")
		return
	}

	// Extract backend's auth challenge
	backendChallenge, err := extractMySQLChallenge(backendGreeting)
	if err != nil {
		logger.Error("Failed to parse backend greeting: %v", err)
		sendMySQLError(clientConn, 1049, "HY000", "Backend authentication failed")
		return
	}

	// Decrypt the stored password for the route
	password := ""
	if route.Password != "" && p.secretManager != nil {
		if decrypted, err := p.secretManager.DecryptPassword(route.Password); err == nil {
			password = decrypted
		} else {
			// Might be plaintext
			password = route.Password
		}
	}

	// Re-authenticate with the backend using its challenge
	// Use mysql_native_password: SHA1(password) XOR SHA1(challenge + SHA1(SHA1(password)))
	authResponse := computeMySQLNativeAuth(backendChallenge, password)

	// Build and send handshake response to backend
	if err := sendMySQLHandshakeResponse(backendConn, username, authResponse, dbName, clientFlags); err != nil {
		logger.Error("Failed to send handshake to backend: %v", err)
		sendMySQLError(clientConn, 1049, "HY000", "Backend authentication failed")
		return
	}

	// Read backend auth response
	authResp, err := readMySQLPacket(backendConn)
	if err != nil {
		logger.Error("Failed to read backend auth response: %v", err)
		sendMySQLError(clientConn, 1049, "HY000", "Backend authentication failed")
		return
	}

	// Forward the auth response to client (OK or ERR)
	if err := writeMySQLPacket(clientConn, authResp, 2); err != nil {
		logger.Debug("Failed to forward auth response to client: %v", err)
		return
	}

	// Check if auth succeeded (first byte of payload is 0x00 for OK, 0xFF for ERR)
	if len(authResp) > 0 && authResp[0] == 0xFF {
		return // Auth failed, error already forwarded
	}

	// Ignore client auth data since we re-authenticate with stored credentials
	_ = clientAuthData

	// Bidirectional forwarding
	p.bidirectionalCopy(clientConn, backendConn)
}

// sendMySQLGreeting sends a MySQL server greeting packet
func sendMySQLGreeting(conn net.Conn, challenge []byte) error {
	var payload []byte

	// Protocol version
	payload = append(payload, mysqlProtocolVersion)

	// Server version (null-terminated)
	payload = append(payload, []byte(mysqlServerVersion)...)
	payload = append(payload, 0)

	// Connection ID (4 bytes)
	payload = append(payload, 1, 0, 0, 0)

	// Auth challenge part 1 (8 bytes)
	payload = append(payload, challenge[:8]...)

	// Filler
	payload = append(payload, 0)

	// Capability flags (lower 2 bytes)
	capLower := make([]byte, 2)
	binary.LittleEndian.PutUint16(capLower, uint16(mysqlCapFlags))
	payload = append(payload, capLower...)

	// Character set (utf8mb4 = 45)
	payload = append(payload, 45)

	// Status flags
	payload = append(payload, 0x02, 0x00) // SERVER_STATUS_AUTOCOMMIT

	// Capability flags (upper 2 bytes)
	capUpper := make([]byte, 2)
	binary.LittleEndian.PutUint16(capUpper, uint16(mysqlCapFlagsUpper))
	payload = append(payload, capUpper...)

	// Length of auth data (or 0)
	payload = append(payload, 21) // 8 + 13 = 21

	// Reserved (10 bytes of 0)
	payload = append(payload, make([]byte, 10)...)

	// Auth challenge part 2 (12 bytes + null)
	payload = append(payload, challenge[8:20]...)
	payload = append(payload, 0)

	// Auth plugin name
	payload = append(payload, []byte("mysql_native_password")...)
	payload = append(payload, 0)

	return writeMySQLPacket(conn, payload, 0)
}

// readMySQLHandshakeResponse reads the client's handshake response
// Returns: database name, username, auth data, client flags, error
func readMySQLHandshakeResponse(conn net.Conn) (string, string, []byte, uint32, error) {
	data, err := readMySQLPacket(conn)
	if err != nil {
		return "", "", nil, 0, err
	}

	if len(data) < 32 {
		return "", "", nil, 0, fmt.Errorf("handshake response too short")
	}

	pos := 0

	// Client flags (4 bytes)
	clientFlags := binary.LittleEndian.Uint32(data[pos : pos+4])
	pos += 4

	// Max packet size (4 bytes)
	pos += 4

	// Character set (1 byte)
	pos += 1

	// Reserved (23 bytes)
	pos += 23

	// Username (null-terminated)
	usernameEnd := pos
	for usernameEnd < len(data) && data[usernameEnd] != 0 {
		usernameEnd++
	}
	username := string(data[pos:usernameEnd])
	pos = usernameEnd + 1

	// Auth data
	var authData []byte
	if clientFlags&clientPluginAuthLenData != 0 {
		authLen := int(data[pos])
		pos++
		if pos+authLen <= len(data) {
			authData = data[pos : pos+authLen]
			pos += authLen
		}
	} else if clientFlags&clientSecureConnection != 0 {
		authLen := int(data[pos])
		pos++
		if pos+authLen <= len(data) {
			authData = data[pos : pos+authLen]
			pos += authLen
		}
	} else {
		// Null-terminated auth
		authEnd := pos
		for authEnd < len(data) && data[authEnd] != 0 {
			authEnd++
		}
		authData = data[pos:authEnd]
		pos = authEnd + 1
	}

	// Database name (if CLIENT_CONNECT_WITH_DB)
	var dbName string
	if clientFlags&clientConnectWithDB != 0 && pos < len(data) {
		dbEnd := pos
		for dbEnd < len(data) && data[dbEnd] != 0 {
			dbEnd++
		}
		dbName = string(data[pos:dbEnd])
	}

	return dbName, username, authData, clientFlags, nil
}

// extractMySQLChallenge extracts the 20-byte auth challenge from a greeting packet
func extractMySQLChallenge(greeting []byte) ([]byte, error) {
	if len(greeting) < 1 {
		return nil, fmt.Errorf("empty greeting")
	}

	// Skip protocol version (1 byte)
	pos := 1

	// Skip server version (null-terminated)
	for pos < len(greeting) && greeting[pos] != 0 {
		pos++
	}
	pos++ // skip null

	// Skip connection ID (4 bytes)
	pos += 4

	if pos+8 > len(greeting) {
		return nil, fmt.Errorf("greeting too short for auth challenge")
	}

	// First 8 bytes of challenge
	challenge := make([]byte, 20)
	copy(challenge[:8], greeting[pos:pos+8])
	pos += 8

	// Skip filler (1 byte)
	pos++

	// Skip capability flags lower (2 bytes)
	pos += 2

	// Skip character set (1 byte), status flags (2 bytes), capability flags upper (2 bytes)
	pos += 5

	// Skip auth plugin data len or reserved (1 byte)
	pos++

	// Skip reserved (10 bytes)
	pos += 10

	if pos+12 > len(greeting) {
		return nil, fmt.Errorf("greeting too short for second challenge part")
	}

	// Second part of challenge (12 bytes)
	copy(challenge[8:], greeting[pos:pos+12])

	return challenge, nil
}

// computeMySQLNativeAuth computes mysql_native_password auth response
// SHA1(password) XOR SHA1(challenge + SHA1(SHA1(password)))
func computeMySQLNativeAuth(challenge []byte, password string) []byte {
	if password == "" {
		return nil
	}

	// SHA1(password)
	hash1 := sha1.Sum([]byte(password))

	// SHA1(SHA1(password))
	hash2 := sha1.Sum(hash1[:])

	// SHA1(challenge + SHA1(SHA1(password)))
	h := sha1.New()
	h.Write(challenge)
	h.Write(hash2[:])
	hash3 := h.Sum(nil)

	// XOR SHA1(password) with SHA1(challenge + SHA1(SHA1(password)))
	result := make([]byte, 20)
	for i := 0; i < 20; i++ {
		result[i] = hash1[i] ^ hash3[i]
	}

	return result
}

// sendMySQLHandshakeResponse sends a handshake response to the backend
func sendMySQLHandshakeResponse(conn net.Conn, username string, authData []byte, dbName string, clientFlags uint32) error {
	var payload []byte

	// Client flags
	flags := make([]byte, 4)
	binary.LittleEndian.PutUint32(flags, clientFlags|clientConnectWithDB|clientSecureConnection|clientPluginAuth)
	payload = append(payload, flags...)

	// Max packet size
	maxPkt := make([]byte, 4)
	binary.LittleEndian.PutUint32(maxPkt, 16777216)
	payload = append(payload, maxPkt...)

	// Character set (utf8mb4 = 45)
	payload = append(payload, 45)

	// Reserved (23 bytes)
	payload = append(payload, make([]byte, 23)...)

	// Username (null-terminated)
	payload = append(payload, []byte(username)...)
	payload = append(payload, 0)

	// Auth data (length-encoded)
	if authData != nil {
		payload = append(payload, byte(len(authData)))
		payload = append(payload, authData...)
	} else {
		payload = append(payload, 0)
	}

	// Database name (null-terminated)
	payload = append(payload, []byte(dbName)...)
	payload = append(payload, 0)

	// Auth plugin name
	payload = append(payload, []byte("mysql_native_password")...)
	payload = append(payload, 0)

	return writeMySQLPacket(conn, payload, 1)
}

// readMySQLPacket reads a MySQL protocol packet (4-byte header + payload)
func readMySQLPacket(conn net.Conn) ([]byte, error) {
	header := make([]byte, 4)
	if _, err := io.ReadFull(conn, header); err != nil {
		return nil, err
	}

	length := int(header[0]) | int(header[1])<<8 | int(header[2])<<16
	if length > 1<<24 {
		return nil, fmt.Errorf("MySQL packet too large: %d", length)
	}

	payload := make([]byte, length)
	if _, err := io.ReadFull(conn, payload); err != nil {
		return nil, err
	}

	return payload, nil
}

// writeMySQLPacket writes a MySQL protocol packet
func writeMySQLPacket(conn net.Conn, payload []byte, seqID byte) error {
	header := make([]byte, 4)
	header[0] = byte(len(payload))
	header[1] = byte(len(payload) >> 8)
	header[2] = byte(len(payload) >> 16)
	header[3] = seqID

	if _, err := conn.Write(header); err != nil {
		return err
	}
	_, err := conn.Write(payload)
	return err
}

// sendMySQLError sends a MySQL ERR packet
func sendMySQLError(conn net.Conn, errCode uint16, sqlState string, message string) {
	var payload []byte
	payload = append(payload, 0xFF) // ERR marker

	// Error code (2 bytes LE)
	ec := make([]byte, 2)
	binary.LittleEndian.PutUint16(ec, errCode)
	payload = append(payload, ec...)

	// SQL state marker
	payload = append(payload, '#')
	payload = append(payload, []byte(sqlState)...)

	// Message
	payload = append(payload, []byte(message)...)

	writeMySQLPacket(conn, payload, 2)
}
