package orchestrator

import (
	"bytes"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"

	"github.com/obiente/cloud/apps/shared/pkg/logger"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

var proxmoxSSHFallbackWarnings sync.Map

func newProxmoxHostKeyCallback(nodeName, endpoint string) (ssh.HostKeyCallback, error) {
	knownHostsPath := strings.TrimSpace(os.Getenv("PROXMOX_SSH_KNOWN_HOSTS_PATH"))
	if knownHostsPath != "" {
		callback, err := knownhosts.New(knownHostsPath)
		if err != nil {
			return nil, fmt.Errorf("load PROXMOX_SSH_KNOWN_HOSTS_PATH: %w", err)
		}
		return callback, nil
	}

	hostKeysEnv := strings.TrimSpace(os.Getenv("PROXMOX_NODE_SSH_HOST_KEYS"))
	if hostKeysEnv != "" {
		hostKeys, err := parseProxmoxSSHHostKeyMapping(hostKeysEnv)
		if err != nil {
			return nil, err
		}

		expectedKey, ok := hostKeys[nodeName]
		if !ok {
			return nil, fmt.Errorf("missing SSH host key for node %s in PROXMOX_NODE_SSH_HOST_KEYS", nodeName)
		}

		return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			if !bytes.Equal(key.Marshal(), expectedKey.Marshal()) {
				return fmt.Errorf("unexpected SSH host key for node %s at %s", nodeName, endpoint)
			}
			return nil
		}, nil
	}

	warnKey := fmt.Sprintf("%s|%s", nodeName, endpoint)
	if _, loaded := proxmoxSSHFallbackWarnings.LoadOrStore(warnKey, struct{}{}); !loaded {
		logger.Warn("[ProxmoxClient] SSH host key verification disabled for node %s at %s; configure PROXMOX_SSH_KNOWN_HOSTS_PATH or PROXMOX_NODE_SSH_HOST_KEYS", nodeName, endpoint)
	}

	return ssh.InsecureIgnoreHostKey(), nil
}

func parseProxmoxSSHHostKeyMapping(raw string) (map[string]ssh.PublicKey, error) {
	keys := make(map[string]ssh.PublicKey)

	for _, entry := range strings.Split(raw, ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}

		parts := strings.SplitN(entry, ":", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid PROXMOX_NODE_SSH_HOST_KEYS entry %q", entry)
		}

		nodeName := strings.TrimSpace(parts[0])
		keyData := strings.TrimSpace(parts[1])
		if nodeName == "" || keyData == "" {
			return nil, fmt.Errorf("invalid PROXMOX_NODE_SSH_HOST_KEYS entry %q", entry)
		}

		publicKey, _, _, _, err := ssh.ParseAuthorizedKey([]byte(keyData))
		if err != nil {
			return nil, fmt.Errorf("parse SSH host key for node %s: %w", nodeName, err)
		}

		keys[nodeName] = publicKey
	}

	if len(keys) == 0 {
		return nil, fmt.Errorf("PROXMOX_NODE_SSH_HOST_KEYS did not contain any valid entries")
	}

	return keys, nil
}
