package vps

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/obiente/cloud/apps/shared/pkg/database"
	"github.com/obiente/cloud/apps/shared/pkg/logger"
	"golang.org/x/crypto/ssh"
)

const vpsSSHHostKeyCheckTimeout = 5 * time.Second

func newVPSHostKeyCallback(ctx context.Context, vpsID string) ssh.HostKeyCallback {
	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		checkCtx := context.Background()
		if ctx != nil {
			checkCtx = ctx
		}
		checkCtx, cancel := context.WithTimeout(checkCtx, vpsSSHHostKeyCheckTimeout)
		defer cancel()

		fingerprint := ssh.FingerprintSHA256(key)
		if err := database.VerifyOrPinVPSSSHHostKey(vpsID, fingerprint); err != nil {
			logger.Warn("[VPS SSH] Host key verification failed for VPS %s at %s: %v", vpsID, hostname, err)
			return fmt.Errorf("verify SSH host key for VPS %s: %w", vpsID, err)
		}
		if remote != nil {
			logger.Debug("[VPS SSH] Verified host key for VPS %s via %s", vpsID, remote.String())
		}
		return nil
	}
}
