package email

import (
	"context"
	"net"
	"testing"
	"time"
)

func TestSMTPExchangeStopsOnTimeoutOrCancellation(t *testing.T) {
	for _, cancelContext := range []bool{false, true} {
		name := "timeout"
		if cancelContext {
			name = "cancellation"
		}
		t.Run(name, func(t *testing.T) {
			listener, err := net.Listen("tcp", "127.0.0.1:0")
			if err != nil {
				t.Fatal(err)
			}
			defer listener.Close()
			accepted := make(chan net.Conn, 1)
			go func() {
				conn, err := listener.Accept()
				if err == nil {
					accepted <- conn
				}
			}()
			address := listener.Addr().(*net.TCPAddr)
			mailer := &SMTPMailer{cfg: SMTPConfig{Host: "127.0.0.1", Port: address.Port, FromAddress: "sender@example.test", Timeout: 100 * time.Millisecond}}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if cancelContext {
				mailer.cfg.Timeout = 10 * time.Second
			}
			result := make(chan error, 1)
			go func() {
				result <- mailer.Send(ctx, &Message{To: []string{"recipient@example.test"}, Subject: "Synthetic timeout test", TextBody: "No external mail is sent."})
			}()
			var conn net.Conn
			select {
			case conn = <-accepted:
				defer conn.Close()
			case <-time.After(2 * time.Second):
				t.Fatal("connection not established")
			}
			// The fake SMTP server deliberately never sends a greeting.
			if cancelContext {
				cancel()
			}
			select {
			case err := <-result:
				if err == nil {
					t.Fatal("stalled SMTP succeeded")
				}
			case <-time.After(2 * time.Second):
				t.Fatal("SMTP exchange ignored timeout/cancellation")
			}
		})
	}
}
