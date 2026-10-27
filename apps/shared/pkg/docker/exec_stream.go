package docker

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"time"

	"github.com/moby/moby/api/pkg/stdcopy"
	"github.com/moby/moby/client"
)

// ContainerExecStream keeps database dumps out of service memory. Cancellation
// closes the hijacked connection, including a blocked stdin writer.
func (c *Client) ContainerExecStream(ctx context.Context, containerID string, command []string, input io.Reader, output io.Writer, env []string) error {
	if c == nil || c.api == nil {
		return ErrUninitialized
	}
	created, err := c.api.ExecCreate(ctx, containerID, client.ExecCreateOptions{Cmd: command, Env: env, AttachStdin: input != nil, AttachStdout: true, AttachStderr: true})
	if err != nil {
		return fmt.Errorf("create exec: %w", err)
	}
	attached, err := c.api.ExecAttach(ctx, created.ID, client.ExecAttachOptions{})
	if err != nil {
		return fmt.Errorf("attach exec: %w", err)
	}
	defer attached.Close()
	stop := context.AfterFunc(ctx, attached.Close)
	defer stop()
	inputDone := make(chan error, 1)
	go func() {
		var err error
		if input != nil {
			_, err = io.Copy(attached.Conn, input)
		}
		if closeErr := attached.CloseWrite(); err == nil {
			err = closeErr
		}
		inputDone <- err
	}()
	stderr := &boundedDiagnostic{}
	_, copyErr := stdcopy.StdCopy(output, stderr, attached.Reader)
	if copyErr != nil {
		attached.Close()
	}
	inputErr := <-inputDone
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if copyErr != nil {
		return fmt.Errorf("read exec output: %w", copyErr)
	}
	if inputErr != nil {
		return fmt.Errorf("write exec input: %w", inputErr)
	}
	for {
		result, err := c.api.ExecInspect(ctx, created.ID, client.ExecInspectOptions{})
		if err != nil {
			return fmt.Errorf("inspect exec: %w", err)
		}
		if !result.Running {
			if result.ExitCode != 0 {
				return fmt.Errorf("database command exited with status %d: %s", result.ExitCode, stderr.String())
			}
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(20 * time.Millisecond):
		}
	}
}

type boundedDiagnostic struct{ bytes.Buffer }

func (b *boundedDiagnostic) Write(p []byte) (int, error) {
	n := len(p)
	if remaining := 8192 - b.Len(); remaining > 0 {
		if len(p) > remaining {
			p = p[:remaining]
		}
		_, _ = b.Buffer.Write(p)
	}
	return n, nil
}
