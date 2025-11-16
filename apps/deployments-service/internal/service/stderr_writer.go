package deployments

import (
	"io"
)

// StderrWriter wraps a BuildLogStreamer to write to stderr
type StderrWriter struct {
	streamer *BuildLogStreamer
}

// NewStderrWriter creates a new stderr writer that writes to the streamer's stderr
func NewStderrWriter(streamer *BuildLogStreamer) io.Writer {
	return &StderrWriter{streamer: streamer}
}

// Write implements io.Writer by calling WriteStderr on the streamer
func (w *StderrWriter) Write(p []byte) (n int, err error) {
	return w.streamer.WriteStderr(p)
}
