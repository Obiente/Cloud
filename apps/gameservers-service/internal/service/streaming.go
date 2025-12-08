package gameservers

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"

	"gameservers-service/internal/orchestrator"

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
	if err := s.checkGameServerPermission(ctx, gameServerID, "read"); err != nil {
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

// GetGameServerLogs is deprecated - use StreamGameServerLogs instead
// This endpoint blocks until all logs are fetched, which can cause timeouts
func (s *Service) GetGameServerLogs(ctx context.Context, req *connect.Request[gameserversv1.GetGameServerLogsRequest]) (*connect.Response[gameserversv1.GetGameServerLogsResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, fmt.Errorf("GetGameServerLogs is deprecated. Please use StreamGameServerLogs instead, which connects immediately and streams logs without blocking"))
}

// StreamGameServerLogs streams logs for a game server
// This connects immediately and streams historical logs first (if requested), then continues with live logs
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
	if err := s.checkGameServerPermission(ctx, gameServerID, "read"); err != nil {
		logger.Error("[StreamGameServerLogs] Permission check failed for game server %s: %v", gameServerID, err)
		return err
	}

	// Get logs from Docker container
	manager, err := s.getGameServerManager()
	if err != nil {
		logger.Error("[StreamGameServerLogs] Failed to get game server manager: %v", err)
		return connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get game server manager: %w", err))
	}

	// Parse request parameters
	tail := req.Msg.Tail
	if tail == nil || *tail == 0 {
		tailVal := int32(100) // Default to last 100 lines
		tail = &tailVal
	}

	follow := req.Msg.Follow
	if follow == nil {
		followVal := true // Default to following logs
		follow = &followVal
	}

	// Parse since/until timestamps for historical logs
	var sinceTime *time.Time
	if req.Msg.Since != nil {
		since := req.Msg.Since.AsTime()
		sinceTime = &since
	}

	var untilTime *time.Time
	if req.Msg.Until != nil {
		until := req.Msg.Until.AsTime()
		untilTime = &until
	} else if sinceTime != nil {
		// For lazy loading (scrolling up), we want logs BEFORE the since timestamp
		untilTime = sinceTime
		sinceTime = nil
	}

	// Get search query if provided
	searchQuery := ""
	if req.Msg.SearchQuery != nil && *req.Msg.SearchQuery != "" {
		searchQuery = strings.ToLower(strings.TrimSpace(*req.Msg.SearchQuery))
	}

	logger.Info("[StreamGameServerLogs] Streaming logs for %s (tail=%d, follow=%v, since=%v, until=%v, search=%s)",
		gameServerID, *tail, *follow, sinceTime, untilTime, func() string {
			if searchQuery != "" {
				return searchQuery
			}
			return "none"
		}())

	// Step 1: Stream historical logs first
	// Always fetch initial tail as historical logs to ensure connection is established immediately
	// If since/until is provided, use those; otherwise just get the tail
	logger.Info("[StreamGameServerLogs] Fetching historical logs for %s", gameServerID)
	if err := s.streamHistoricalLogs(ctx, manager, gameServerID, stream, *tail, sinceTime, untilTime, searchQuery); err != nil {
		logger.Warn("[StreamGameServerLogs] Error streaming historical logs: %v", err)
		// Continue to live streaming even if historical fails
	}

	// Step 2: Continue with live streaming if follow is true
	if !*follow {
		logger.Info("[StreamGameServerLogs] Follow disabled, ending stream after historical logs")
		return nil
	}

	// For live streaming, we only want NEW logs (follow mode)
	// If we already sent historical logs, we don't want to duplicate them
	// So we use tail=0 to skip the initial tail and only get new logs
	logger.Info("[StreamGameServerLogs] Starting live log streaming for %s (following new logs only)", gameServerID)
	logsReader, err := manager.GetGameServerLogs(ctx, gameServerID, "0", true, nil, nil)
	if err != nil {
		return connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get game server logs: %w", err))
	}
	defer logsReader.Close()

	// Stream live logs
	return s.streamLiveLogs(ctx, logsReader, stream, searchQuery)
}

// streamHistoricalLogs fetches and streams historical logs (non-following)
func (s *Service) streamHistoricalLogs(ctx context.Context, manager *orchestrator.GameServerManager, gameServerID string, stream *connect.ServerStream[gameserversv1.GameServerLogLine], limit int32, sinceTime *time.Time, untilTime *time.Time, searchQuery string) error {
	// Calculate how many lines to fetch
	// For historical logs, we fetch more than requested to account for filtering
	dockerTailLimit := int(limit)
	if searchQuery != "" {
		dockerTailLimit = int(limit) * 3 // Fetch more when searching
	} else if untilTime != nil {
		dockerTailLimit = int(limit) * 2 // Fetch more for historical pagination
	}

	// Cap to prevent memory issues
	const maxHistoricalLines = 5000
	if dockerTailLimit > maxHistoricalLines {
		dockerTailLimit = maxHistoricalLines
	}

	logger.Info("[streamHistoricalLogs] Fetching %d lines for %s (since=%v, until=%v, search=%s)", dockerTailLimit, gameServerID, sinceTime, untilTime, searchQuery)

	// Hybrid approach: Try follow=false first (fast, returns immediately), then fallback to follow=true
	tailParam := fmt.Sprintf("%d", dockerTailLimit)
	
	// Parse and stream historical logs
	// For until queries, Docker returns logs in chronological order (oldest to newest)
	// We need to keep only the last N lines, so we use a sliding window
	header := make([]byte, 8)
	var allLines []*gameserversv1.GameServerLogLine
	useSlidingWindow := untilTime != nil
	linesRead := 0
	linesSent := 0

	logger.Info("[streamHistoricalLogs] Attempting to read historical logs (useSlidingWindow=%v)", useSlidingWindow)

	// CRITICAL FIX: User confirmed Docker CLI works (docker logs shows logs), but our API returns EOF
	// Docker's tail parameter can be unreliable with follow=false. Use "all" to get all logs, then limit client-side
	// This ensures we actually get logs even if Docker's tail calculation fails
	effectiveTail := "all"
	limitClientSide := true // When using "all", we need to limit on our side
	logger.Info("[streamHistoricalLogs] Using tail='all' to ensure we get logs (Docker tail can be unreliable with follow=false), will limit to %d lines client-side", dockerTailLimit)

	// Try follow=false first - it's faster and more reliable for historical logs
	// According to Docker docs, follow=false with tail returns the last N lines and closes immediately
	logger.Info("[streamHistoricalLogs] Trying follow=false with tail=%s - Docker should return logs immediately or EOF", effectiveTail)
	
	logsReader, err := manager.GetGameServerLogs(ctx, gameServerID, effectiveTail, false, sinceTime, untilTime)
	if err != nil {
		logger.Error("[streamHistoricalLogs] Failed to get logs reader with follow=false: %v", err)
		return fmt.Errorf("failed to get historical logs: %w", err)
	}
	defer logsReader.Close()
	
	logger.Info("[streamHistoricalLogs] Successfully obtained logs reader (follow=false), reading until EOF...")

	// Read all logs until EOF with follow=false
	// Docker returns logs and closes the stream
	readAnyLogs := false
	useFollow := false
	for {
		// Check context cancellation
		select {
		case <-ctx.Done():
			logger.Info("[streamHistoricalLogs] Context cancelled, read %d lines, sent %d lines", linesRead, linesSent)
			return nil
		default:
		}

		// Read header - with follow=false, Docker returns logs immediately or EOF
		// With follow=true, we use timeout to detect when Docker has finished sending tail
		var err error
		if useFollow {
			// With follow=true, use timeout to detect when Docker has finished sending tail
			// Docker sends tail immediately, then waits for new logs
			headerChan := make(chan error, 1)
			go func() {
				_, readErr := io.ReadFull(logsReader, header)
				headerChan <- readErr
			}()
			
			select {
			case err = <-headerChan:
				// Got result, continue reading
				logger.Debug("[streamHistoricalLogs] Read header with follow=true, continuing...")
			case <-time.After(5 * time.Second):
				// Timeout - Docker has finished sending historical logs
				logger.Info("[streamHistoricalLogs] Timeout (5s) after reading %d lines with follow=true (Docker finished sending tail), stopping", linesSent)
				if linesSent == 0 {
					logger.Warn("[streamHistoricalLogs] No historical logs were read with follow=true - container may have no logs")
				}
				return nil
			}
		} else {
			// With follow=false, read directly (Docker returns logs immediately or EOF)
			// No timeout needed - Docker either sends logs immediately or returns EOF
			logger.Debug("[streamHistoricalLogs] Reading header with follow=false (no timeout)...")
			_, err = io.ReadFull(logsReader, header)
			logger.Debug("[streamHistoricalLogs] Read header result: err=%v", err)
		}
		
		if err != nil {
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				logger.Info("[streamHistoricalLogs] EOF reached (follow=%v), read %d lines total, sent %d lines", useFollow, linesRead, linesSent)
				if !readAnyLogs && !useFollow {
					// Got EOF immediately with follow=false and no logs - try follow=true as fallback
					logger.Warn("[streamHistoricalLogs] Got EOF immediately with follow=false and no logs, trying follow=true as fallback")
					logsReader.Close()
					
					// Retry with follow=true and longer timeout
					logsReader, err = manager.GetGameServerLogs(ctx, gameServerID, tailParam, true, sinceTime, untilTime)
					if err != nil {
						logger.Error("[streamHistoricalLogs] Failed to get logs reader with follow=true: %v", err)
						return fmt.Errorf("failed to get historical logs with follow=true: %w", err)
					}
					useFollow = true
					logger.Info("[streamHistoricalLogs] Retrying with follow=true, will use timeout to detect when Docker finishes sending tail")
					
					// Continue to next iteration to read with follow=true
					continue
				} else if !readAnyLogs {
					logger.Warn("[streamHistoricalLogs] WARNING: Got EOF immediately - no historical logs available. Container may have no logs yet.")
				}
				break // Done reading historical logs
			}
			logger.Error("[streamHistoricalLogs] Error reading header: %v (linesSent=%d, follow=%v)", err, linesSent, useFollow)
			// If we've read some logs but hit an error, that's okay - we got what we could
			if linesSent > 0 {
				logger.Info("[streamHistoricalLogs] Error after reading %d lines, stopping", linesSent)
				return nil
			}
			return fmt.Errorf("failed to read log header: %w", err)
		}
		
		readAnyLogs = true

		streamType := header[0]
		payloadLength := int(binary.BigEndian.Uint32(header[4:8]))
		if payloadLength == 0 {
			logger.Debug("[streamHistoricalLogs] Header with zero payload length, skipping")
			continue
		}
		logger.Info("[streamHistoricalLogs] Header: streamType=%d, payloadLength=%d bytes", streamType, payloadLength)

		// Read payload
		payload := make([]byte, payloadLength)
		bytesRead, err := io.ReadFull(logsReader, payload)
		if err != nil {
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				logger.Warn("[streamHistoricalLogs] EOF while reading payload (expected %d bytes, read %d bytes). This might indicate Docker closed the stream early.", payloadLength, bytesRead)
				// If we read some data, try to parse what we have
				if bytesRead > 0 {
					logger.Info("[streamHistoricalLogs] Attempting to parse partial payload (%d bytes)", bytesRead)
					payload = payload[:bytesRead]
					// Continue to parsing below
				} else {
					logger.Info("[streamHistoricalLogs] No payload data read, read %d lines total, sent %d lines", linesRead, linesSent)
					break
				}
			} else {
				return fmt.Errorf("failed to read log payload: %w", err)
			}
		}

		// Parse log lines
		lines := s.parseLogPayload(payload, streamType == 2)
		logger.Info("[streamHistoricalLogs] Parsed %d lines from payload (size=%d bytes, linesSent=%d)", len(lines), payloadLength, linesSent)
		if len(lines) > 0 && len(lines[0].Line) > 0 {
			previewLen := 50
			if len(lines[0].Line) < previewLen {
				previewLen = len(lines[0].Line)
			}
			logger.Info("[streamHistoricalLogs] First line preview: %q", lines[0].Line[:previewLen])
		}
		
		for _, line := range lines {
			linesRead++
			
			// Apply search filter if provided
			if searchQuery != "" && !strings.Contains(strings.ToLower(line.Line), searchQuery) {
				continue
			}

			if useSlidingWindow {
				// For until queries, use sliding window to keep only last N lines
				allLines = append(allLines, line)
				if len(allLines) > int(limit)+100 { // Keep a bit more for filtering
					allLines = allLines[1:] // Remove oldest
				}
			} else {
				// When using "all", we need to buffer and only send the last N lines
				if limitClientSide {
					allLines = append(allLines, line)
					// Keep only the last N lines in memory
					if len(allLines) > int(dockerTailLimit)+100 { // Keep a bit extra for filtering
						allLines = allLines[1:] // Remove oldest
					}
				} else {
					// Stream immediately to client
					previewLen := 50
					if len(line.Line) < previewLen {
						previewLen = len(line.Line)
					}
					logger.Debug("[streamHistoricalLogs] Sending log line to client: %q", line.Line[:previewLen])
					if err := stream.Send(line); err != nil {
						logger.Error("[streamHistoricalLogs] Failed to send log line to client: %v", err)
						return err
					}
					linesSent++
					if linesSent == 1 {
						firstLinePreview := 100
						if len(line.Line) < firstLinePreview {
							firstLinePreview = len(line.Line)
						}
						logger.Info("[streamHistoricalLogs] ✓ Successfully sent first log line to client! Line: %q", line.Line[:firstLinePreview])
					}
					if linesSent%100 == 0 {
						logger.Info("[streamHistoricalLogs] Sent %d lines so far", linesSent)
					}
				}
			}
		}
	}

	// For until queries or client-side limiting, send the last N lines
	if useSlidingWindow || limitClientSide {
		startIdx := 0
		limitToUse := int(limit)
		if limitClientSide {
			limitToUse = dockerTailLimit
		}
		if len(allLines) > limitToUse {
			startIdx = len(allLines) - limitToUse
		}
		sentCount := 0
		for i := startIdx; i < len(allLines); i++ {
			if err := stream.Send(allLines[i]); err != nil {
				logger.Warn("[streamHistoricalLogs] Failed to send log line from buffer: %v", err)
				return err
			}
			sentCount++
			linesSent++
		}
		logger.Info("[streamHistoricalLogs] Streamed %d historical log lines (from %d total read) for %s", sentCount, len(allLines), gameServerID)
		if sentCount == 0 && linesRead == 0 {
			logger.Warn("[streamHistoricalLogs] WARNING: No historical logs were read or sent! This might indicate an issue with Docker log retrieval.")
		}
	} else {
		logger.Info("[streamHistoricalLogs] Completed streaming historical logs for %s: read %d lines, sent %d lines (no sliding window used)", gameServerID, linesRead, linesSent)
		if linesSent == 0 && linesRead == 0 {
			logger.Warn("[streamHistoricalLogs] WARNING: No historical logs were read or sent! This might indicate an issue with Docker log retrieval.")
		}
	}

	return nil
}

// streamLiveLogs streams live logs (following)
func (s *Service) streamLiveLogs(ctx context.Context, logsReader io.ReadCloser, stream *connect.ServerStream[gameserversv1.GameServerLogLine], searchQuery string) error {
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
				if ctx.Err() != nil {
					return nil
				}
				// Brief pause before retrying
				select {
				case <-ctx.Done():
					return nil
				case <-time.After(100 * time.Millisecond):
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
		payloadLength := int(binary.BigEndian.Uint32(header[4:8]))
		if payloadLength == 0 {
			continue
		}

		// Read payload
		payload := make([]byte, payloadLength)
		_, err = io.ReadFull(logsReader, payload)
		if err != nil {
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				// Partial payload, try to send what we have
				if len(payload) > 0 {
					lines := s.parseLogPayload(payload, streamType == 2)
					for _, line := range lines {
						if searchQuery == "" || strings.Contains(strings.ToLower(line.Line), searchQuery) {
							if sendErr := stream.Send(line); sendErr != nil {
								if ctx.Err() != nil {
									return nil
								}
								return sendErr
							}
						}
					}
				}
				if ctx.Err() != nil {
					return nil
				}
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

		// Parse and send log lines
		lines := s.parseLogPayload(payload, streamType == 2)
		for _, line := range lines {
			// Apply search filter if provided
			if searchQuery != "" && !strings.Contains(strings.ToLower(line.Line), searchQuery) {
				continue
			}

			if sendErr := stream.Send(line); sendErr != nil {
				if ctx.Err() != nil {
					return nil
				}
				return sendErr
			}
		}
	}
}

// parseLogPayload parses a Docker log payload and returns log lines
func (s *Service) parseLogPayload(payload []byte, isStderr bool) []*gameserversv1.GameServerLogLine {
	// Sanitize to valid UTF-8 and strip ANSI sequences
	rawText := strings.ToValidUTF8(string(payload), "")
	rawText = stripansi.Strip(rawText)
	rawText = stripAnsiEscapeSequences(rawText)

	// Split by newlines to handle multiple lines in one payload
	lines := strings.Split(rawText, "\n")
	var logLines []*gameserversv1.GameServerLogLine

	for _, lineText := range lines {
		lineText = strings.TrimRight(lineText, "\r")
		if lineText == "" {
			continue
		}

		// Strip timestamps from log lines
		lineText = stripTimestampFromLine(lineText)

		// Detect log level from content
		logLevel := detectLogLevelFromContent(lineText, isStderr)

		logLines = append(logLines, &gameserversv1.GameServerLogLine{
			Line:      lineText,
			Timestamp: timestamppb.Now(),
			Level:     &logLevel,
		})
	}

	return logLines
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
		
		// Remove escape sequences that might appear as text: <--ERE], <--ERE, <--, etc.
		// These are often malformed escape sequences that got converted to text
		// Do this FIRST before other patterns to catch it early
		// Match <-- followed by any characters (including letters, numbers, etc.) up to and including ]
		line = regexp.MustCompile(`<--[A-Z0-9]*\]`).ReplaceAllString(line, "") // Match <--ERE] specifically
		line = regexp.MustCompile(`<--[A-Z]*`).ReplaceAllString(line, "")     // Match <--ERE (without bracket)
		line = regexp.MustCompile(`<--`).ReplaceAllString(line, "")           // Match <-- alone
		
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
	
	// Final cleanup: remove any remaining <-- patterns (should have been caught earlier, but be safe)
	// Be very aggressive here - match <-- followed by anything up to ]
	line = regexp.MustCompile(`<--[A-Z0-9]*\]`).ReplaceAllString(line, "") // Match <--ERE] specifically
	line = regexp.MustCompile(`<--[A-Z]*`).ReplaceAllString(line, "")     // Match <--ERE (without bracket)
	line = regexp.MustCompile(`<--`).ReplaceAllString(line, "")           // Match <-- alone
	
	// Also remove standalone ERE] patterns that might remain (leftover from incomplete stripping)
	line = regexp.MustCompile(`^ERE\]`).ReplaceAllString(line, "")
	line = regexp.MustCompile(`\s+ERE\]`).ReplaceAllString(line, "")
	line = regexp.MustCompile(`ERE\]`).ReplaceAllString(line, "") // Remove anywhere
	
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

