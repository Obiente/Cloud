package gameservers

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"

	"github.com/acarl005/stripansi"
	"github.com/obiente/cloud/apps/shared/pkg/logger"
	v1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/common/v1" // Import with v1 alias to match generated code
	gameserversv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/gameservers/v1"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// StreamGameServerStatus streams status updates for a game server
func (s *Service) StreamGameServerStatus(ctx context.Context, req *connect.Request[gameserversv1.StreamGameServerStatusRequest], stream *connect.ServerStream[gameserversv1.GameServerStatusUpdate]) error {
	// Ensure authenticated - use the returned context
	var err error
	ctx, err = s.ensureAuthenticated(ctx, req)
	if err != nil {
		return err
	}

	gameServerID := req.Msg.GetGameServerId()
	if err := s.checkGameServerPermission(ctx, gameServerID, "view"); err != nil {
		return err
	}

	// TODO: Implement actual streaming
	// For now, return current status
	gameServer, err := s.repo.GetByID(ctx, gameServerID)
	if err != nil {
		return connect.NewError(connect.CodeNotFound, fmt.Errorf("game server %s not found", gameServerID))
	}

	update := &gameserversv1.GameServerStatusUpdate{
		GameServerId: gameServerID,
		Status:       gameserversv1.GameServerStatus(gameServer.Status),
		Timestamp:    timestamppb.Now(),
	}

	return stream.Send(update)
}

// GetGameServerLogs retrieves logs for a game server
func (s *Service) GetGameServerLogs(ctx context.Context, req *connect.Request[gameserversv1.GetGameServerLogsRequest]) (*connect.Response[gameserversv1.GetGameServerLogsResponse], error) {
	gameServerID := req.Msg.GetGameServerId()
	if gameServerID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("game server ID is required"))
	}
	if err := s.checkGameServerPermission(ctx, gameServerID, "view"); err != nil {
		return nil, err
	}

	// Get logs from Docker container
	manager, err := s.getGameServerManager()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get game server manager: %w", err))
	}

	limit := req.Msg.Limit
	if limit == nil || *limit == 0 {
		limitVal := int32(100) // Default to last 100 lines
		limit = &limitVal
	}

	// Parse since timestamp if provided
	var sinceTime *time.Time
	if req.Msg.Since != nil {
		since := req.Msg.Since.AsTime()
		sinceTime = &since
	}

	// Parse until timestamp if provided (for historical loading)
	var untilTime *time.Time
	if req.Msg.Until != nil {
		until := req.Msg.Until.AsTime()
		untilTime = &until
	} else if sinceTime != nil {
		// For lazy loading (scrolling up), we want logs BEFORE the since timestamp
		// Docker's "until" parameter gets logs before a timestamp
		// Use until to get logs before the since timestamp
		untilTime = sinceTime
		sinceTime = nil // Clear since when using until
	}

	// Get search query if provided
	searchQuery := ""
	if req.Msg.SearchQuery != nil && *req.Msg.SearchQuery != "" {
		searchQuery = strings.ToLower(strings.TrimSpace(*req.Msg.SearchQuery))
	}
	
	// Log the request for debugging
	logger.Info("[GetGameServerLogs] Request received for game server %s (limit=%d, since=%v, until=%v, search=%s)", 
		gameServerID, 
		func() int32 { if limit != nil { return *limit } else { return 0 } }(),
		sinceTime,
		untilTime,
		func() string { if searchQuery != "" { return searchQuery } else { return "none" } }())

	// For proper pagination, always use a reasonable limit
	// When using until for historical loading, we want to fetch in chunks
	// If search is active, we may need to fetch more to account for filtering
	dockerTailLimit := *limit
	if searchQuery != "" {
		// When searching, fetch more logs to account for filtering
		// But cap it at a reasonable maximum to prevent timeouts
		dockerTailLimit = *limit * 5
		if dockerTailLimit > 1000 {
			dockerTailLimit = 1000 // Cap at 1000 to prevent timeouts
		}
	} else if untilTime != nil {
		// For historical loading without search, use the requested limit
		// This ensures proper pagination - we get exactly what was requested
		dockerTailLimit = *limit
	}
	
	// Add a timeout to the Docker API call itself to prevent hanging
	// Docker API calls can hang if the daemon is slow or the container is in a bad state
	dockerCtx, dockerCancel := context.WithTimeout(ctx, 10*time.Second)
	defer dockerCancel()
	
	// Use dockerTailLimit for Docker query
	logger.Debug("[GetGameServerLogs] Calling Docker API for game server %s (tail=%d, since=%v, until=%v)", 
		gameServerID, dockerTailLimit, sinceTime, untilTime)
	logsReader, err := manager.GetGameServerLogs(dockerCtx, gameServerID, fmt.Sprintf("%d", dockerTailLimit), false, sinceTime, untilTime)
	if err != nil {
		// Check if it's a timeout
		if dockerCtx.Err() == context.DeadlineExceeded {
			logger.Error("[GetGameServerLogs] Docker API call timed out after 10 seconds for game server %s: %v", gameServerID, err)
			return nil, connect.NewError(connect.CodeDeadlineExceeded, fmt.Errorf("docker API call timed out after 10 seconds: %w", err))
		}
		logger.Error("[GetGameServerLogs] Failed to get game server logs for %s: %v", gameServerID, err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get game server logs: %w", err))
	}
	defer logsReader.Close()
	logger.Debug("[GetGameServerLogs] Successfully obtained logs reader for game server %s", gameServerID)

	// Add a timeout to prevent indefinite blocking on reads
	// For non-following logs, Docker should return quickly, but we add a safety timeout
	readCtx, readCancel := context.WithTimeout(ctx, 10*time.Second)
	defer readCancel()

	// Parse Docker multiplexed log format
	// Docker logs format: [8-byte header][payload]
	// Header: [stream_type(1)][reserved(3)][size(4 bytes, big-endian)]
	// stream_type: 1=stdout, 2=stderr
	var logLines []*gameserversv1.GameServerLogLine
	header := make([]byte, 8)
	
	// Create a channel to signal when read completes
	type readResult struct {
		n   int
		err error
	}
	
	for {
		// Check if context is cancelled
		select {
		case <-readCtx.Done():
			// Timeout or cancellation - return what we have
			goto done
		case <-ctx.Done():
			goto done
		default:
		}

		// Read 8-byte header with timeout
		headerChan := make(chan readResult, 1)
		go func() {
			n, err := io.ReadFull(logsReader, header)
			headerChan <- readResult{n: n, err: err}
		}()
		
		var err error
		select {
		case <-readCtx.Done():
			// Timeout - return what we have
			goto done
		case result := <-headerChan:
			err = result.err
		}
		if err != nil {
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				// EOF means we've read all available logs
				break
			}
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to read log header: %w", err))
		}

		streamType := header[0]
		// Read payload length (bytes 4-7, big-endian)
		payloadLength := int(binary.BigEndian.Uint32(header[4:8]))

		if payloadLength == 0 {
			continue
		}

		// Read payload with timeout
		payload := make([]byte, payloadLength)
		payloadChan := make(chan readResult, 1)
		go func() {
			n, err := io.ReadFull(logsReader, payload)
			payloadChan <- readResult{n: n, err: err}
		}()
		
		select {
		case <-readCtx.Done():
			// Timeout - return what we have
			goto done
		case result := <-payloadChan:
			err = result.err
		}
		
		if err != nil {
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				// Partial payload, process what we have
				if len(payload) > 0 {
					// Strip ANSI sequences from raw payload first
					rawText := strings.ToValidUTF8(string(payload), "")
					rawText = stripansi.Strip(rawText)
					rawText = stripAnsiEscapeSequences(rawText) // Additional cleanup for edge cases
					sanitizedLine := stripTimestampFromLine(rawText)
					if sanitizedLine != "" {
						logLevel := detectLogLevelFromContent(sanitizedLine, streamType == 2)
						line := &gameserversv1.GameServerLogLine{
							Line:      sanitizedLine,
							Timestamp: timestamppb.Now(),
							Level:     &logLevel,
						}
						logLines = append(logLines, line)
					}
				}
				break
			}
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to read log payload: %w", err))
		}

		// Extract timestamp from original payload before processing
		// Docker logs with timestamps have format: 2025-01-01T12:00:00.123456789Z <log content>
		var logTimestamp *timestamppb.Timestamp
		timestampRegex := regexp.MustCompile(`^(\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d+)?Z)\s+`)
		originalPayloadText := strings.ToValidUTF8(string(payload), "")
		if matches := timestampRegex.FindStringSubmatch(originalPayloadText); len(matches) > 1 {
			if parsedTime, err := time.Parse(time.RFC3339Nano, matches[1]); err == nil {
				logTimestamp = timestamppb.New(parsedTime)
			}
		}

		// Sanitize to valid UTF-8 and strip ANSI sequences from raw payload
		rawText := originalPayloadText
		rawText = stripansi.Strip(rawText) // Use library to strip ANSI
		rawText = stripAnsiEscapeSequences(rawText) // Additional cleanup for edge cases
		
		// Split by newlines to handle multiple lines in one payload
		lines := strings.Split(rawText, "\n")
		for _, lineText := range lines {
			lineText = strings.TrimRight(lineText, "\r")
			if lineText == "" {
				continue
			}

			// Strip timestamps from log lines (e.g., [01:57:06] or 2025-11-05T01:57:06.052Z)
			lineText = stripTimestampFromLine(lineText)
			
			// Detect log level from content
			logLevel := detectLogLevelFromContent(lineText, streamType == 2)
			
			// If no timestamp extracted, use current time
			if logTimestamp == nil {
				logTimestamp = timestamppb.Now()
			}
			
			line := &gameserversv1.GameServerLogLine{
				Line:      lineText,
				Timestamp: logTimestamp,
				Level:     &logLevel,
			}
			
			// Apply search filter if provided
			if searchQuery == "" || strings.Contains(strings.ToLower(lineText), searchQuery) {
				logLines = append(logLines, line)
				
				// If we have enough matching lines, we can stop early
				if len(logLines) >= int(*limit) {
					goto done
				}
			}
		}
		
		// Stop if we've processed enough raw lines (accounting for filtering)
		// Count all processed lines, not just matching ones
		if len(logLines) >= int(dockerTailLimit) {
			break
		}
	}
	
done:
	// When using until parameter, Docker returns logs in chronological order (oldest to newest)
	// up to the until timestamp. For historical loading, we want the MOST RECENT logs before
	// that timestamp, so we need to take the last N lines, not the first N.
	if untilTime != nil && len(logLines) > int(*limit) {
		// Take the last N lines (most recent before until timestamp)
		logLines = logLines[len(logLines)-int(*limit):]
	} else if len(logLines) > int(*limit) {
		// For normal queries (no until), take the first N lines
		logLines = logLines[:int(*limit)]
	}

	logger.Info("[GetGameServerLogs] Returning %d log lines for game server %s", len(logLines), gameServerID)
	res := connect.NewResponse(&gameserversv1.GetGameServerLogsResponse{
		Lines: logLines,
	})
	return res, nil
}

// StreamGameServerLogs streams logs for a game server
func (s *Service) StreamGameServerLogs(ctx context.Context, req *connect.Request[gameserversv1.StreamGameServerLogsRequest], stream *connect.ServerStream[gameserversv1.GameServerLogLine]) error {
	logger.Info("[StreamGameServerLogs] Request received")
	// Ensure authenticated - use the returned context
	var err error
	ctx, err = s.ensureAuthenticated(ctx, req)
	if err != nil {
		logger.Error("[StreamGameServerLogs] Authentication failed: %v", err)
		return err
	}

	gameServerID := req.Msg.GetGameServerId()
	logger.Info("[StreamGameServerLogs] Request for game server %s", gameServerID)
	if err := s.checkGameServerPermission(ctx, gameServerID, "view"); err != nil {
		logger.Error("[StreamGameServerLogs] Permission check failed for game server %s: %v", gameServerID, err)
		return err
	}

	// Get logs from Docker container
	manager, err := s.getGameServerManager()
	if err != nil {
		logger.Error("[StreamGameServerLogs] Failed to get game server manager: %v", err)
		return connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get game server manager: %w", err))
	}

	tail := req.Msg.Tail
	if tail == nil || *tail == 0 {
		tailVal := int32(100) // Default to last 100 lines
		tail = &tailVal
	}

	// Always follow logs when streaming - client context will handle disconnection
	// This ensures logs are always streamed while the tab is open
	follow := true

	logsReader, err := manager.GetGameServerLogs(ctx, gameServerID, fmt.Sprintf("%d", *tail), follow, nil, nil)
	if err != nil {
		return connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get game server logs: %w", err))
	}
	defer logsReader.Close()

	// Parse Docker multiplexed log format
	// Docker logs format: [8-byte header][payload]
	// Header: [stream_type(1)][reserved(3)][size(4 bytes, big-endian)]
	// stream_type: 1=stdout, 2=stderr
	header := make([]byte, 8)
	for {
		// Check if context is cancelled
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		// Read 8-byte header
		_, err := io.ReadFull(logsReader, header)
		if err != nil {
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				// EOF is normal when following logs - container might have paused or no new logs
				// Check if context is cancelled before waiting
				if ctx.Err() != nil {
					return nil
				}
				// Brief pause before retrying to avoid tight loop
				// This allows the stream to continue when new logs arrive
				select {
				case <-ctx.Done():
					return nil
				case <-time.After(100 * time.Millisecond):
					// Reset header and retry
					header = make([]byte, 8)
					continue
				}
			}
			if ctx.Err() != nil {
				return nil
			}
			return connect.NewError(connect.CodeInternal, fmt.Errorf("failed to read log header: %w", err))
		}

		streamType := header[0]
		// Read payload length (bytes 4-7, big-endian)
		payloadLength := int(binary.BigEndian.Uint32(header[4:8]))

		if payloadLength == 0 {
			continue
		}

		// Read payload
		payload := make([]byte, payloadLength)
		_, err = io.ReadFull(logsReader, payload)
		if err != nil {
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				// Partial payload, send what we have
				if len(payload) > 0 {
					// Strip ANSI sequences from raw payload first
					rawText := strings.ToValidUTF8(string(payload), "")
					rawText = stripansi.Strip(rawText)
					rawText = stripAnsiEscapeSequences(rawText) // Additional cleanup for edge cases
					sanitizedLine := stripTimestampFromLine(rawText)
					if sanitizedLine != "" {
						logLevel := detectLogLevelFromContent(sanitizedLine, streamType == 2)
						line := &gameserversv1.GameServerLogLine{
							Line:      sanitizedLine,
							Timestamp: timestamppb.Now(),
							Level:     &logLevel,
						}

						if sendErr := stream.Send(line); sendErr != nil {
							if ctx.Err() != nil {
								return nil
							}
							return sendErr
						}
					}
				}
				// EOF is normal when following - check context and retry
				if ctx.Err() != nil {
					return nil
				}
				// Brief pause before retrying
				select {
				case <-ctx.Done():
					return nil
				case <-time.After(100 * time.Millisecond):
					continue
				}
			}
			if ctx.Err() != nil {
				return nil
			}
			return connect.NewError(connect.CodeInternal, fmt.Errorf("failed to read log payload: %w", err))
		}

		// Sanitize to valid UTF-8 and strip ANSI sequences from raw payload
		rawText := strings.ToValidUTF8(string(payload), "")
		rawText = stripansi.Strip(rawText) // Use library to strip ANSI
		rawText = stripAnsiEscapeSequences(rawText) // Additional cleanup for edge cases
		
		// Split by newlines to handle multiple lines in one payload
		lines := strings.Split(rawText, "\n")
		for _, lineText := range lines {
			lineText = strings.TrimRight(lineText, "\r")
			if lineText == "" {
				continue
			}

			// Strip timestamps from log lines (e.g., [01:57:06] or 2025-11-05T01:57:06.052Z)
			// Minecraft logs often have timestamps embedded in them
			lineText = stripTimestampFromLine(lineText)
			
			// Detect log level from content
			logLevel := detectLogLevelFromContent(lineText, streamType == 2)
			
			line := &gameserversv1.GameServerLogLine{
				Line:      lineText,
				Timestamp: timestamppb.Now(),
				Level:     &logLevel,
			}

			if sendErr := stream.Send(line); sendErr != nil {
				// Check if context was cancelled (client disconnected)
				if ctx.Err() != nil {
					return nil
				}
				return sendErr
			}
		}
	}
}

// stripAnsiEscapeSequences removes ANSI escape sequences from log lines
// These sequences are used for terminal formatting (colors, cursor control, etc.)
func stripAnsiEscapeSequences(line string) string {
	// Comprehensive ANSI escape sequence removal
	// Pattern 1: Standard ANSI escape sequences with ESC prefix (\x1b or \033)
	// Matches: ESC[ followed by parameters and command letter
	ansiRegex := regexp.MustCompile(`\x1b\[[0-9;?]*[a-zA-Z]|\x1b[=<>]|\033\[[0-9;?]*[a-zA-Z]|\033[=<>]`)
	line = ansiRegex.ReplaceAllString(line, "")
	
	// Pattern 2: CSI sequences that may appear without ESC prefix (common in malformed logs)
	// These sequences can appear concatenated like [?1h[=[?2004h
	// We need to be careful not to match valid log content like [INFO] or [21:45:50]
	
	// Iteratively remove escape sequences until no more changes occur
	// This handles cases where sequences are concatenated without spaces
	for {
		original := line
		
		// Remove terminal mode sequences: [?1h, [?2004h, etc.
		line = regexp.MustCompile(`\[\?[0-9]+[hHlLmM]`).ReplaceAllString(line, "")
		
		// Remove application keypad mode sequences: [=, [>, [<
		line = regexp.MustCompile(`\[[=<>]`).ReplaceAllString(line, "")
		
		// Remove single-character CSI sequences: [K (clear to end of line), [H (cursor home), etc.
		// Common CSI single-letter commands: A-H, J, K, m, s, u
		// Match [K specifically first since it's very common
		line = regexp.MustCompile(`\[K`).ReplaceAllString(line, "")
		line = regexp.MustCompile(`\[[A-HJmsu]`).ReplaceAllString(line, "")
		
		// Remove formatting codes: [0m, [1m, [4m, [3m, [30m, etc.
		// Match [ followed by digits and 'm' (SGR - Select Graphic Rendition)
		// But exclude timestamps like [21:45:50] by requiring the 'm' suffix
		line = regexp.MustCompile(`\[[0-9;]+m`).ReplaceAllString(line, "")
		
		// Remove prompt continuation indicators like ">...." anywhere in the line
		// These can appear at the start or after escape sequences
		line = regexp.MustCompile(`>\.+`).ReplaceAllString(line, "")
		
		// Remove any remaining malformed escape patterns
		line = regexp.MustCompile(`\[=[?0-9]*`).ReplaceAllString(line, "")
		
		// If no changes were made, we're done
		if line == original {
			break
		}
	}
	
	// Final pass: remove any remaining control sequences that might have been missed
	// This is a more aggressive pass that catches edge cases
	// Remove patterns like [ followed by non-alphanumeric characters that aren't part of valid log content
	// But be very careful not to remove valid log brackets like [INFO] or [21:45:50]
	// We only remove if it's clearly an escape sequence pattern
	line = regexp.MustCompile(`^\[\?[0-9]*[hHlLmM]?`).ReplaceAllString(line, "") // At start of line
	line = regexp.MustCompile(`\[K$`).ReplaceAllString(line, "")                // At end of line
	line = regexp.MustCompile(`\[K\s`).ReplaceAllString(line, " ")              // Before whitespace
	
	return strings.TrimSpace(line)
}

// stripTimestampFromLine removes embedded timestamps from log lines
// Examples: [01:57:06] or 2025-11-05T01:57:06.052Z
func stripTimestampFromLine(line string) string {
	// First strip ANSI escape sequences
	line = stripAnsiEscapeSequences(line)
	
	// Remove Minecraft-style timestamps: [HH:MM:SS]
	line = regexp.MustCompile(`^\[\d{2}:\d{2}:\d{2}\]\s*`).ReplaceAllString(line, "")
	
	// Remove ISO timestamps: 2025-11-05T01:57:06.052Z at start of line
	line = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d+)?Z\s+`).ReplaceAllString(line, "")
	
	// Remove init/other timestamps: [init] or similar
	line = regexp.MustCompile(`^\[init\]\s+`).ReplaceAllString(line, "")
	
	return strings.TrimSpace(line)
}

// detectLogLevelFromContent detects log level from log line content
// For game servers, we prioritize content-based detection over stderr/stdout
func detectLogLevelFromContent(line string, isStderr bool) v1.LogLevel {
	lineLower := strings.ToLower(strings.TrimSpace(line))
	
	// Priority 1: Check for Minecraft/Java server log patterns (most specific)
	// Format: [HH:MM:SS] [Thread/LEVEL]: message
	// Examples: [Server thread/INFO]: Done, [Server thread/WARN]: Warning, [Server thread/ERROR]: Error
	// Use regex-like matching with word boundaries to avoid false matches
	if matched, _ := regexp.MatchString(`/\s*(info|information)\s*]:`, lineLower); matched {
		return v1.LogLevel_LOG_LEVEL_INFO
	}
	if matched, _ := regexp.MatchString(`/\s*(warn|warning)\s*]:`, lineLower); matched {
		return v1.LogLevel_LOG_LEVEL_WARN
	}
	if matched, _ := regexp.MatchString(`/\s*(error|fatal)\s*]:`, lineLower); matched {
		return v1.LogLevel_LOG_LEVEL_ERROR
	}
	if matched, _ := regexp.MatchString(`/\s*(debug|trace)\s*]:`, lineLower); matched {
		return v1.LogLevel_LOG_LEVEL_DEBUG
	}
	
	// Priority 2: Check for standalone log level markers at start of line
	// Examples: INFO mc-server-runner, WARN something, ERROR something
	// Use word boundaries to ensure we match whole words, not parts of other words
	if matched, _ := regexp.MatchString(`^(info|information)\s+`, lineLower); matched {
		return v1.LogLevel_LOG_LEVEL_INFO
	}
	if matched, _ := regexp.MatchString(`^(warn|warning)\s+`, lineLower); matched {
		return v1.LogLevel_LOG_LEVEL_WARN
	}
	if matched, _ := regexp.MatchString(`^(error|fatal|failed)\s+`, lineLower); matched {
		return v1.LogLevel_LOG_LEVEL_ERROR
	}
	if matched, _ := regexp.MatchString(`^(debug|trace)\s+`, lineLower); matched {
		return v1.LogLevel_LOG_LEVEL_DEBUG
	}
	
	// Priority 3: Check for bracketed log level markers
	// Examples: [INFO], [WARN], [ERROR], [DEBUG]
	if matched, _ := regexp.MatchString(`\[(info|information)\]`, lineLower); matched {
		return v1.LogLevel_LOG_LEVEL_INFO
	}
	if matched, _ := regexp.MatchString(`\[(warn|warning)\]`, lineLower); matched {
		return v1.LogLevel_LOG_LEVEL_WARN
	}
	if matched, _ := regexp.MatchString(`\[(error|fatal)\]`, lineLower); matched {
		return v1.LogLevel_LOG_LEVEL_ERROR
	}
	if matched, _ := regexp.MatchString(`\[(debug|trace)\]`, lineLower); matched {
		return v1.LogLevel_LOG_LEVEL_DEBUG
	}
	
	// Priority 4: Check for log level markers with colon
	// Examples: INFO:, WARN:, ERROR:, DEBUG:
	if matched, _ := regexp.MatchString(`^(info|information):`, lineLower); matched {
		return v1.LogLevel_LOG_LEVEL_INFO
	}
	if matched, _ := regexp.MatchString(`^(warn|warning):`, lineLower); matched {
		return v1.LogLevel_LOG_LEVEL_WARN
	}
	if matched, _ := regexp.MatchString(`^(error|fatal):`, lineLower); matched {
		return v1.LogLevel_LOG_LEVEL_ERROR
	}
	if matched, _ := regexp.MatchString(`^(debug|trace):`, lineLower); matched {
		return v1.LogLevel_LOG_LEVEL_DEBUG
	}
	
	// Priority 5: Check for error/fatal patterns in content (but be careful not to match "error" in "information")
	// Only match if "error" or "fatal" appears as a standalone word or in specific contexts
	if matched, _ := regexp.MatchString(`\berror\b|\bfatal\b|\bfailed\b`, lineLower); matched {
		// But exclude if it's part of "information" or other false positives
		if !strings.Contains(lineLower, "information") && !strings.Contains(lineLower, "inferior") {
			return v1.LogLevel_LOG_LEVEL_ERROR
		}
	}
	
	// Priority 6: Check for warning patterns
	if matched, _ := regexp.MatchString(`\bwarn(ing)?\b`, lineLower); matched {
		return v1.LogLevel_LOG_LEVEL_WARN
	}
	
	// Priority 7: Game server specific patterns (usually INFO)
	if strings.Contains(lineLower, "[server]") || strings.Contains(lineLower, "[minecraft]") ||
		strings.Contains(lineLower, "[console]") ||
		strings.Contains(lineLower, "starting") || strings.Contains(lineLower, "stopping") ||
		strings.Contains(lineLower, "joined") || strings.Contains(lineLower, "left") ||
		strings.Contains(lineLower, "logged") || strings.Contains(lineLower, "saving") ||
		strings.Contains(lineLower, "done") || strings.Contains(lineLower, "waiting") {
		return v1.LogLevel_LOG_LEVEL_INFO
	}
	
	// Priority 8: Default behavior
	// For game servers, most logs are INFO regardless of stderr/stdout
	// Many game servers send all logs to stderr, so we don't use isStderr as a strong indicator
	return v1.LogLevel_LOG_LEVEL_INFO
}

