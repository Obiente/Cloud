package gameservers

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"

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

	// For lazy loading (scrolling up), we want logs BEFORE the since timestamp
	// Docker's "until" parameter gets logs before a timestamp
	var untilTime *time.Time
	if sinceTime != nil {
		// Use until to get logs before the since timestamp
		untilTime = sinceTime
		sinceTime = nil // Clear since when using until
	}

	logsReader, err := manager.GetGameServerLogs(ctx, gameServerID, fmt.Sprintf("%d", *limit), false, sinceTime, untilTime)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to get game server logs: %w", err))
	}
	defer logsReader.Close()

	// Parse Docker multiplexed log format
	// Docker logs format: [8-byte header][payload]
	// Header: [stream_type(1)][reserved(3)][size(4 bytes, big-endian)]
	// stream_type: 1=stdout, 2=stderr
	var logLines []*gameserversv1.GameServerLogLine
	header := make([]byte, 8)
	
	for {
		// Check if context is cancelled
		select {
		case <-ctx.Done():
			break
		default:
		}

		// Read 8-byte header
		_, err := io.ReadFull(logsReader, header)
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

		// Read payload
		payload := make([]byte, payloadLength)
		_, err = io.ReadFull(logsReader, payload)
		if err != nil {
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				// Partial payload, process what we have
				if len(payload) > 0 {
					sanitizedLine := stripTimestampFromLine(strings.ToValidUTF8(string(payload), ""))
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

		// Sanitize to valid UTF-8
		sanitizedLine := strings.ToValidUTF8(string(payload), "")
		
		// Split by newlines to handle multiple lines in one payload
		lines := strings.Split(sanitizedLine, "\n")
		for _, lineText := range lines {
			lineText = strings.TrimRight(lineText, "\r")
			if lineText == "" {
				continue
			}

			// Strip timestamps from log lines (e.g., [01:57:06] or 2025-11-05T01:57:06.052Z)
			lineText = stripTimestampFromLine(lineText)
			
			// Detect log level from content
			logLevel := detectLogLevelFromContent(lineText, streamType == 2)
			
			// Try to extract timestamp from Docker log line if timestamps are enabled
			// Docker logs with timestamps have format: 2025-01-01T12:00:00.123456789Z <log content>
			var logTimestamp *timestamppb.Timestamp
			timestampRegex := regexp.MustCompile(`^(\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d+)?Z)\s+`)
			// Check the original payload before stripping timestamps
			if matches := timestampRegex.FindStringSubmatch(string(payload)); len(matches) > 1 {
				if parsedTime, err := time.Parse(time.RFC3339Nano, matches[1]); err == nil {
					logTimestamp = timestamppb.New(parsedTime)
				}
			}
			
			// If no timestamp extracted, use current time
			if logTimestamp == nil {
				logTimestamp = timestamppb.Now()
			}
			
			line := &gameserversv1.GameServerLogLine{
				Line:      lineText,
				Timestamp: logTimestamp,
				Level:     &logLevel,
			}
			logLines = append(logLines, line)
			
			// Limit the number of lines returned
			if len(logLines) >= int(*limit) {
				break
			}
		}
		
		if len(logLines) >= int(*limit) {
			break
		}
	}

	res := connect.NewResponse(&gameserversv1.GetGameServerLogsResponse{
		Lines: logLines,
	})
	return res, nil
}

// StreamGameServerLogs streams logs for a game server
func (s *Service) StreamGameServerLogs(ctx context.Context, req *connect.Request[gameserversv1.StreamGameServerLogsRequest], stream *connect.ServerStream[gameserversv1.GameServerLogLine]) error {
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

	// Get logs from Docker container
	manager, err := s.getGameServerManager()
	if err != nil {
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
					sanitizedLine := stripTimestampFromLine(strings.ToValidUTF8(string(payload), ""))
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

		// Sanitize to valid UTF-8
		sanitizedLine := strings.ToValidUTF8(string(payload), "")
		
		// Split by newlines to handle multiple lines in one payload
		lines := strings.Split(sanitizedLine, "\n")
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

// stripTimestampFromLine removes embedded timestamps from log lines
// Examples: [01:57:06] or 2025-11-05T01:57:06.052Z
func stripTimestampFromLine(line string) string {
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

