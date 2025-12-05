package orchestrator

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"math/big"
	"net"
	"os"
	"strings"
	"time"

	"github.com/obiente/cloud/apps/shared/pkg/database"
	"github.com/obiente/cloud/apps/shared/pkg/logger"
	"golang.org/x/crypto/ssh"
)

// Cloud-init operations

// GenerateRandomPassword generates a random password
// Exported for use in password reset functionality
func GenerateRandomPassword(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*"
	b := make([]byte, length)
	charsetLen := big.NewInt(int64(len(charset)))
	for i := range b {
		n, _ := rand.Int(rand.Reader, charsetLen)
		b[i] = charset[n.Int64()]
	}
	return string(b)
}


func GenerateCloudInitUserData(config *VPSConfig) string {
	return generateCloudInitUserData(config)
}


func generateCloudInitUserData(config *VPSConfig) string {
	userData := "#cloud-config\n\n"

	// SSH configuration
	sshInstallServer := true
	sshAllowPW := true
	if config.CloudInit != nil {
		if config.CloudInit.SSHInstallServer != nil {
			sshInstallServer = *config.CloudInit.SSHInstallServer
		}
		if config.CloudInit.SSHAllowPW != nil {
			sshAllowPW = *config.CloudInit.SSHAllowPW
		}
	}
	userData += "ssh:\n"
	userData += fmt.Sprintf("  install-server: %v\n", sshInstallServer)
	userData += fmt.Sprintf("  allow-pw: %v\n", sshAllowPW)
	userData += "\n"

	// Disable cloud-init network configuration - we use Proxmox's ipconfig0 instead
	userData += "network:\n"
	userData += "  config: disabled\n"
	userData += "\n"

	// Hostname
	if config.CloudInit != nil && config.CloudInit.Hostname != nil && *config.CloudInit.Hostname != "" {
		userData += fmt.Sprintf("hostname: %s\n", *config.CloudInit.Hostname)
		userData += fmt.Sprintf("fqdn: %s\n", *config.CloudInit.Hostname)
		userData += "\n"
	}

	// Timezone
	if config.CloudInit != nil && config.CloudInit.Timezone != nil && *config.CloudInit.Timezone != "" {
		userData += fmt.Sprintf("timezone: %s\n\n", *config.CloudInit.Timezone)
	}

	// Locale
	if config.CloudInit != nil && config.CloudInit.Locale != nil && *config.CloudInit.Locale != "" {
		// Quote locale to handle special characters safely
		escapedLocale := strings.ReplaceAll(*config.CloudInit.Locale, "'", "''")
		userData += fmt.Sprintf("locale: '%s'\n\n", escapedLocale)
	}

	// Users configuration
	userData += "users:\n"

	// Add root user (always included)
	userData += "  - name: root\n"

	// Root password (from config or auto-generated)
	// Quote the password to handle special YAML characters safely
	if config.RootPassword != nil && *config.RootPassword != "" {
		// Escape any single quotes in the password and wrap in single quotes
		escapedPassword := strings.ReplaceAll(*config.RootPassword, "'", "''")
		userData += fmt.Sprintf("    passwd: '%s'\n", escapedPassword)
	}

	// Add SSH keys for root (includes both VPS-specific and org-wide keys, plus bastion and terminal keys)
	rootSSHKeys := []string{}

	// Add bastion SSH key (required for SSH bastion host connections)
	if config.OrganizationID != "" && config.VPSID != "" {
		bastionKey, err := database.GetVPSBastionKey(config.VPSID)
		if err == nil {
			rootSSHKeys = append(rootSSHKeys, strings.TrimSpace(bastionKey.PublicKey))
			logger.Debug("[ProxmoxClient] Added bastion key to cloud-init for VPS %s (fingerprint: %s)", config.VPSID, bastionKey.Fingerprint)
		} else {
			logger.Warn("[ProxmoxClient] Failed to get bastion key for VPS %s: %v (SSH bastion may not work)", config.VPSID, err)
		}
	}

	// Add web terminal SSH key (optional - only if it exists, allows disabling web terminal)
	if config.OrganizationID != "" && config.VPSID != "" {
		terminalKey, err := database.GetVPSTerminalKey(config.VPSID)
		if err == nil {
			rootSSHKeys = append(rootSSHKeys, strings.TrimSpace(terminalKey.PublicKey))
		} else {
			logger.Debug("[ProxmoxClient] Terminal key not found for VPS %s: %v (web terminal disabled)", config.VPSID, err)
		}
	}

	// Add user-provided SSH keys (VPS-specific and org-wide)
	if config.OrganizationID != "" {
		sshKeys, err := database.GetSSHKeysForVPS(config.OrganizationID, config.VPSID)
		if err == nil {
			for _, key := range sshKeys {
				rootSSHKeys = append(rootSSHKeys, strings.TrimSpace(key.PublicKey))
			}
		}
	}

	// Also check if root user is in custom users list and merge SSH keys
	if config.CloudInit != nil {
		for _, user := range config.CloudInit.Users {
			if user.Name == "root" {
				rootSSHKeys = append(rootSSHKeys, user.SSHAuthorizedKeys...)
				break
			}
		}
	}

	if len(rootSSHKeys) > 0 {
		userData += "    ssh_authorized_keys:\n"
		for _, key := range rootSSHKeys {
			userData += fmt.Sprintf("      - %s\n", key)
		}
	}

	userData += "    sudo: ALL=(ALL) NOPASSWD:ALL\n"

	// Add custom users (excluding root if already added)
	if config.CloudInit != nil {
		for _, user := range config.CloudInit.Users {
			if user.Name == "root" {
				continue // Root already handled above
			}
			userData += fmt.Sprintf("  - name: %s\n", user.Name)

			if user.Password != nil && *user.Password != "" {
				// Escape any single quotes in the password and wrap in single quotes
				escapedPassword := strings.ReplaceAll(*user.Password, "'", "''")
				userData += fmt.Sprintf("    passwd: '%s'\n", escapedPassword)
			}

			if len(user.SSHAuthorizedKeys) > 0 {
				userData += "    ssh_authorized_keys:\n"
				for _, key := range user.SSHAuthorizedKeys {
					userData += fmt.Sprintf("      - %s\n", strings.TrimSpace(key))
				}
			}

			if user.Sudo != nil && *user.Sudo {
				if user.SudoNopasswd != nil && *user.SudoNopasswd {
					userData += "    sudo: ALL=(ALL) NOPASSWD:ALL\n"
				} else {
					userData += "    sudo: ALL=(ALL) ALL\n"
				}
			}

			if len(user.Groups) > 0 {
				userData += fmt.Sprintf("    groups: %s\n", strings.Join(user.Groups, ","))
			}

			if user.Shell != nil && *user.Shell != "" {
				userData += fmt.Sprintf("    shell: %s\n", *user.Shell)
			}

			if user.LockPasswd != nil {
				userData += fmt.Sprintf("    lock_passwd: %v\n", *user.LockPasswd)
			}

			if user.Gecos != nil && *user.Gecos != "" {
				userData += fmt.Sprintf("    gecos: %s\n", *user.Gecos)
			}
		}
	}

	userData += "\n"

	// Package management
	packageUpdate := false
	packageUpgrade := false
	if config.CloudInit != nil {
		if config.CloudInit.PackageUpdate != nil {
			packageUpdate = *config.CloudInit.PackageUpdate
		}
		if config.CloudInit.PackageUpgrade != nil {
			packageUpgrade = *config.CloudInit.PackageUpgrade
		}
	}

	userData += fmt.Sprintf("package_update: %v\n", packageUpdate)
	userData += fmt.Sprintf("package_upgrade: %v\n", packageUpgrade)
	packages := []string{"curl", "wget", "htop", "openssh-server", "qemu-guest-agent"}
	if config.CloudInit != nil && len(config.CloudInit.Packages) > 0 {
		packages = append(packages, config.CloudInit.Packages...)
	}

	if len(packages) > 0 {
		userData += "packages:\n"
		for _, pkg := range packages {
			userData += fmt.Sprintf("  - %s\n", pkg)
		}
	}
	userData += "\n"

	// Write files - merge Obiente Cloud configs with user configs
	hasWriteFiles := false
	if config.CloudInit != nil && len(config.CloudInit.WriteFiles) > 0 {
		hasWriteFiles = true
	}

	// Obiente Cloud required file paths
	sshConfigPath := "/etc/ssh/sshd_config.d/99-obiente-cloud.conf"
	pamConfigPath := "/etc/pam.d/sshd"
	lastlogScriptPath := "/usr/local/bin/obiente-update-lastlog.sh"

	// Check which Obiente Cloud files user has provided
	userSSHConfig := -1
	userPAMConfig := -1
	userLastlogScript := -1
	if hasWriteFiles {
		for i, file := range config.CloudInit.WriteFiles {
			switch file.Path {
			case sshConfigPath:
				userSSHConfig = i
			case pamConfigPath:
				userPAMConfig = i
			case lastlogScriptPath:
				userLastlogScript = i
			}
		}
	}

	// Track if we need to restart SSH (if we added/modified SSH config)
	needsSSHRestart := false

	// Always include write_files section if we have Obiente Cloud files or user files
	if hasWriteFiles || userSSHConfig == -1 || userPAMConfig == -1 || userLastlogScript == -1 {
		userData += "write_files:\n"

		// SSH config: merge AcceptEnv if user provided their own, otherwise create new
		if userSSHConfig != -1 {
			// User provided SSH config - merge AcceptEnv into it
			userSSHFile := config.CloudInit.WriteFiles[userSSHConfig]
			userData += fmt.Sprintf("  - path: %s\n", sshConfigPath)
			userData += "    content: |\n"

			// Add user's content
			lines := strings.Split(userSSHFile.Content, "\n")
			for _, line := range lines {
				userData += fmt.Sprintf("      %s\n", line)
			}

			// Check if AcceptEnv already exists
			hasAcceptEnv := false
			for _, line := range lines {
				if strings.Contains(strings.ToLower(line), "acceptenv") && strings.Contains(line, "SSH_CLIENT") {
					hasAcceptEnv = true
					break
				}
			}

			// Add our AcceptEnv if not present
			if !hasAcceptEnv {
				userData += "      # Obiente Cloud: Accept environment variables for real IP forwarding\n"
				userData += "      AcceptEnv SSH_CLIENT SSH_CONNECTION SSH_CLIENT_REAL\n"
				needsSSHRestart = true
			}

			if userSSHFile.Owner != nil && *userSSHFile.Owner != "" {
				userData += fmt.Sprintf("    owner: %s\n", *userSSHFile.Owner)
			} else {
				userData += "    owner: root:root\n"
			}

			if userSSHFile.Permissions != nil && *userSSHFile.Permissions != "" {
				userData += fmt.Sprintf("    permissions: %s\n", *userSSHFile.Permissions)
			} else {
				userData += "    permissions: '0644'\n"
			}

			if userSSHFile.Append != nil && *userSSHFile.Append {
				userData += "    append: true\n"
			}

			if userSSHFile.Defer != nil && *userSSHFile.Defer {
				userData += "    defer: true\n"
			}
		} else {
			// Create new SSH config
			userData += "  - path: /etc/ssh/sshd_config.d/99-obiente-cloud.conf\n"
			userData += "    content: |\n"
			userData += "      # Obiente Cloud: Accept environment variables for real IP forwarding\n"
			userData += "      # This allows the SSH proxy to forward the client's real IP address\n"
			userData += "      AcceptEnv SSH_CLIENT SSH_CONNECTION SSH_CLIENT_REAL\n"
			userData += "    owner: root:root\n"
			userData += "    permissions: '0644'\n"
			needsSSHRestart = true
		}

		// PAM config: merge our lastlog script if user provided their own, otherwise create new
		if userPAMConfig != -1 {
			// User provided PAM config - merge our lastlog script into it
			userPAMFile := config.CloudInit.WriteFiles[userPAMConfig]
			userData += fmt.Sprintf("  - path: %s\n", pamConfigPath)
			userData += "    content: |\n"

			// Check if our lastlog script is already present
			hasLastlogScript := false
			hasPamLastlog := false
			lines := strings.Split(userPAMFile.Content, "\n")
			for _, line := range lines {
				if strings.Contains(line, "obiente-update-lastlog.sh") {
					hasLastlogScript = true
				}
				if strings.Contains(line, "pam_lastlog.so") {
					hasPamLastlog = true
				}
			}

			// Add user's content
			for _, line := range lines {
				userData += fmt.Sprintf("      %s\n", line)
			}

			// Add our lastlog script before pam_lastlog if not present
			if !hasLastlogScript {
				if hasPamLastlog {
					// Insert before pam_lastlog
					userData += "      # Obiente Cloud: Update lastlog with real client IP before standard lastlog\n"
					userData += "      session    optional     pam_exec.so quiet /usr/local/bin/obiente-update-lastlog.sh\n"
				} else {
					// Add at the end of session section
					userData += "      # Obiente Cloud: Update lastlog with real client IP\n"
					userData += "      session    optional     pam_exec.so quiet /usr/local/bin/obiente-update-lastlog.sh\n"
					userData += "      session    optional     pam_lastlog.so\n"
				}
			}

			if userPAMFile.Owner != nil && *userPAMFile.Owner != "" {
				userData += fmt.Sprintf("    owner: %s\n", *userPAMFile.Owner)
			} else {
				userData += "    owner: root:root\n"
			}

			if userPAMFile.Permissions != nil && *userPAMFile.Permissions != "" {
				userData += fmt.Sprintf("    permissions: %s\n", *userPAMFile.Permissions)
			} else {
				userData += "    permissions: '0644'\n"
			}

			if userPAMFile.Append != nil && *userPAMFile.Append {
				userData += "    append: true\n"
			}

			if userPAMFile.Defer != nil && *userPAMFile.Defer {
				userData += "    defer: true\n"
			}
		} else {
			// Create new PAM config
			userData += "  - path: /etc/pam.d/sshd\n"
			userData += "    content: |\n"
			userData += "      # Obiente Cloud: PAM configuration for SSH with real IP forwarding\n"
			userData += "      @include common-auth\n"
			userData += "      account    required     pam_nologin.so\n"
			userData += "      account    include      common-account\n"
			userData += "      password   include      common-password\n"
			userData += "      session    optional     pam_keyinit.so revoke\n"
			userData += "      session    required     pam_limits.so\n"
			userData += "      session    include      common-session\n"
			userData += "      # Update lastlog with real client IP before standard lastlog\n"
			userData += "      session    optional     pam_exec.so quiet /usr/local/bin/obiente-update-lastlog.sh\n"
			userData += "      session    optional     pam_lastlog.so\n"
			userData += "    owner: root:root\n"
			userData += "    permissions: '0644'\n"
		}

		// Lastlog script: use user's if provided, otherwise create new
		if userLastlogScript != -1 {
			// User provided their own script - use it as-is
			userScriptFile := config.CloudInit.WriteFiles[userLastlogScript]
			userData += fmt.Sprintf("  - path: %s\n", lastlogScriptPath)
			userData += "    content: |\n"
			lines := strings.Split(userScriptFile.Content, "\n")
			for _, line := range lines {
				userData += fmt.Sprintf("      %s\n", line)
			}

			if userScriptFile.Owner != nil && *userScriptFile.Owner != "" {
				userData += fmt.Sprintf("    owner: %s\n", *userScriptFile.Owner)
			} else {
				userData += "    owner: root:root\n"
			}

			if userScriptFile.Permissions != nil && *userScriptFile.Permissions != "" {
				userData += fmt.Sprintf("    permissions: %s\n", *userScriptFile.Permissions)
			} else {
				userData += "    permissions: '0755'\n"
			}

			if userScriptFile.Append != nil && *userScriptFile.Append {
				userData += "    append: true\n"
			}

			if userScriptFile.Defer != nil && *userScriptFile.Defer {
				userData += "    defer: true\n"
			}
		} else {
			// Create new lastlog script
			userData += "  - path: /usr/local/bin/obiente-update-lastlog.sh\n"
			userData += "    content: |\n"
			userData += "      #!/bin/bash\n"
			userData += "      # Obiente Cloud: Update lastlog with real client IP from SSH proxy\n"
			userData += "      \n"
			userData += "      if [ \"$PAM_TYPE\" != \"open_session\" ] || [ -z \"$PAM_USER\" ]; then\n"
			userData += "        exit 0\n"
			userData += "      fi\n"
			userData += "      \n"
			userData += "      # Get real client IP from environment and export for Python\n"
			userData += "      if [ -n \"$SSH_CLIENT_REAL\" ]; then\n"
			userData += "        export SSH_CLIENT_REAL=\"$SSH_CLIENT_REAL\"\n"
			userData += "      elif [ -n \"$SSH_CLIENT\" ]; then\n"
			userData += "        export SSH_CLIENT_REAL=$(echo \"$SSH_CLIENT\" | awk '{print $1}')\n"
			userData += "      else\n"
			userData += "        exit 0\n"
			userData += "      fi\n"
			userData += "      \n"
			userData += "      # Update lastlog database with real client IP\n"
			userData += "      python3 << 'PYTHON_SCRIPT'\n"
			// Python script lines must be indented for YAML parsing (6 spaces to match literal block)
			// The heredoc will preserve this indentation, but Python at module level doesn't care about leading whitespace
			userData += "      import struct\n"
			userData += "      import os\n"
			userData += "      import pwd\n"
			userData += "      import time\n"
			userData += "      import sys\n"
			userData += "      \n"
			userData += "      try:\n"
			userData += "          user = os.environ.get('PAM_USER')\n"
			userData += "          client_ip = os.environ.get('SSH_CLIENT_REAL') or (os.environ.get('SSH_CLIENT', '').split()[0] if os.environ.get('SSH_CLIENT') else '')\n"
			userData += "          \n"
			userData += "          if not user or not client_ip:\n"
			userData += "              sys.exit(0)\n"
			userData += "          \n"
			userData += "          pw = pwd.getpwnam(user)\n"
			userData += "          uid = pw.pw_uid\n"
			userData += "          lastlog_path = '/var/log/lastlog'\n"
			userData += "          \n"
			userData += "          if not os.path.exists(lastlog_path) or not os.access(lastlog_path, os.W_OK):\n"
			userData += "              sys.exit(0)\n"
			userData += "          \n"
			userData += "          # lastlog format: struct lastlog {\n"
			userData += "          #     int32_t ll_time;      // 4 bytes (or 8 for 64-bit)\n"
			userData += "          #     char ll_line[UT_LINESIZE];  // 32 bytes\n"
			userData += "          #     char ll_host[UT_HOSTSIZE]; // 256 bytes\n"
			userData += "          # };\n"
			userData += "          # Total: 292 bytes (32-bit) or 296 bytes (64-bit)\n"
			userData += "          \n"
			userData += "          # Detect if time_t is 64-bit (check struct size)\n"
			userData += "          import ctypes\n"
			userData += "          time_t_size = ctypes.sizeof(ctypes.c_time_t) if hasattr(ctypes, 'c_time_t') else 8\n"
			userData += "          record_size = 32 + 256 + time_t_size  # ll_line + ll_host + ll_time\n"
			userData += "          \n"
			userData += "          with open(lastlog_path, 'r+b') as f:\n"
			userData += "              f.seek(uid * record_size)\n"
			userData += "              \n"
			userData += "              # Prepare data\n"
			userData += "              current_time = int(time.time())\n"
			userData += "              host_bytes = client_ip.encode('utf-8')[:255].ljust(256, b'\\0')\n"
			userData += "              line_bytes = ('pts/0').encode('utf-8')[:31].ljust(32, b'\\0')\n"
			userData += "              \n"
			userData += "              # Write time_t (little-endian)\n"
			userData += "              if time_t_size == 8:\n"
			userData += "                  f.write(struct.pack('<Q', current_time))  # 64-bit\n"
			userData += "              else:\n"
			userData += "                  f.write(struct.pack('<I', current_time))  # 32-bit\n"
			userData += "              \n"
			userData += "              # Write line and host\n"
			userData += "              f.write(line_bytes)\n"
			userData += "              f.write(host_bytes)\n"
			userData += "              \n"
			userData += "      except Exception:\n"
			userData += "          pass\n"
			userData += "      PYTHON_SCRIPT\n"
			userData += "      \n"
			userData += "      exit 0\n"
			userData += "    owner: root:root\n"
			userData += "    permissions: '0755'\n"
		}

		// Add user's other write files (excluding Obiente Cloud files we already handled)
		if hasWriteFiles {
			for i, file := range config.CloudInit.WriteFiles {
				// Skip Obiente Cloud files we already handled
				if i == userSSHConfig || i == userPAMConfig || i == userLastlogScript {
					continue
				}

				userData += fmt.Sprintf("  - path: %s\n", file.Path)
				userData += "    content: |\n"
				lines := strings.Split(file.Content, "\n")
				for _, line := range lines {
					userData += fmt.Sprintf("      %s\n", line)
				}

				if file.Owner != nil && *file.Owner != "" {
					userData += fmt.Sprintf("    owner: %s\n", *file.Owner)
				}

				if file.Permissions != nil && *file.Permissions != "" {
					userData += fmt.Sprintf("    permissions: %s\n", *file.Permissions)
				}

				if file.Append != nil && *file.Append {
					userData += "    append: true\n"
				}

				if file.Defer != nil && *file.Defer {
					userData += "    defer: true\n"
				}
			}
		}
		userData += "\n"
	}

	// Runcmd - install SSH + guest agent and run any user commands
	userData += "runcmd:\n"

	// Generate OS-specific runcmd commands based on image type
	// Image enum: 1=Ubuntu 22.04, 2=Ubuntu 24.04, 3=Debian 12, 4=Debian 13, 5=Rocky 9, 6=Alma 9
	switch config.Image {
	case 1, 2, 3, 4: // Ubuntu/Debian (apt-based)
		userData += generateUbuntuDebianRuncmd(config, needsSSHRestart)
	case 5, 6: // Rocky/Alma Linux (yum/dnf-based)
		userData += generateRockyAlmaRuncmd(config, needsSSHRestart)
	default: // Generic/fallback (try to detect package manager)
		userData += generateGenericRuncmd(config, needsSSHRestart)
	}

	// Custom runcmd commands
	if config.CloudInit != nil && len(config.CloudInit.Runcmd) > 0 {
		for _, cmd := range config.CloudInit.Runcmd {
			userData += fmt.Sprintf("  - %s\n", cmd)
		}
	}

	return userData
}


func (pc *ProxmoxClient) CreateCloudInitSnippet(ctx context.Context, nodeName string, storage string, vmID int, userData string) (string, error) {
	return pc.createCloudInitSnippet(ctx, nodeName, storage, vmID, userData)
}


func (pc *ProxmoxClient) createCloudInitSnippet(ctx context.Context, nodeName string, storage string, vmID int, userData string) (string, error) {
	// Proxmox snippets are stored in: <storage>/snippets/
	// Snippets require directory-type storage (dir, nfs, cifs, etc.), not block storage (lvm, zfs)
	snippetFilename := fmt.Sprintf("vm-%d-user-data", vmID)

	// Check if storage supports snippets (must be directory-type storage with snippets content type)
	storageInfo, err := pc.getStorageInfo(ctx, nodeName, storage)
	if err != nil {
		logger.Warn("[ProxmoxClient] Failed to get storage info for '%s': %v, proceeding anyway", storage, err)
	} else if storageInfo != nil {
		storageType, ok := storageInfo["type"].(string)
		if ok {
			// Snippets are only supported on directory-type storage
			// Block storage types (lvm, lvm-thin, zfs, zfspool) don't support snippets
			supportsSnippets := storageType == "dir" || storageType == "directory" ||
				storageType == "nfs" || storageType == "cifs" || storageType == "glusterfs"

			if !supportsSnippets {
				return "", fmt.Errorf("storage '%s' (type: %s) does not support snippets. Snippets require directory-type storage (dir, nfs, cifs). Please set PROXMOX_SNIPPET_STORAGE to a directory-type storage pool (e.g., 'local'). You can use PROXMOX_STORAGE_POOL for VM disks and PROXMOX_SNIPPET_STORAGE for snippets separately", storage, storageType)
			}

			// Check if storage has "snippets" in its content types
			// Proxmox storage must have "snippets" enabled in content types to accept snippet uploads
			if contentVal, ok := storageInfo["content"].(string); ok && contentVal != "" {
				// Content is a comma-separated list like "images,iso,vztmpl,snippets"
				if !strings.Contains(contentVal, "snippets") {
					return "", fmt.Errorf("storage '%s' does not have 'snippets' enabled in its content types. Current content types: %s. Please enable 'snippets' in the storage configuration (Datacenter → Storage → Edit storage → Content: check 'Snippets')", storage, contentVal)
				}
			} else {
				// Content might be an array in some Proxmox versions
				if contentArr, ok := storageInfo["content"].([]interface{}); ok {
					hasSnippets := false
					for _, ct := range contentArr {
						if ctStr, ok := ct.(string); ok && ctStr == "snippets" {
							hasSnippets = true
							break
						}
					}
					if !hasSnippets {
						return "", fmt.Errorf("storage '%s' does not have 'snippets' enabled in its content types. Please enable 'snippets' in the storage configuration (Datacenter → Storage → Edit storage → Content: check 'Snippets')", storage)
					}
				}
			}

			logger.Info("[ProxmoxClient] Storage '%s' (type: %s) supports snippets", storage, storageType)
		}
	}

	// SSH is required for snippet writing - no fallback to upload endpoint
	if pc.config.SSHUser == "" {
		return "", fmt.Errorf("SSH is required for snippet writing. Please configure PROXMOX_SSH_USER environment variable. See https://docs.obiente.cloud/guides/proxmox-ssh-user-setup for setup instructions")
	}
	if pc.config.SSHKeyPath == "" && pc.config.SSHKeyData == "" {
		return "", fmt.Errorf("SSH key is required for snippet writing. Please configure either PROXMOX_SSH_KEY_PATH or PROXMOX_SSH_KEY_DATA environment variable. See https://docs.obiente.cloud/guides/proxmox-ssh-user-setup for setup instructions")
	}

	// Write snippet via SSH (only method supported)
	snippetPath, err := pc.writeSnippetViaSSH(ctx, nodeName, storage, snippetFilename, userData)
	if err != nil {
		return "", fmt.Errorf("failed to create snippet via SSH: %w. Ensure SSH is properly configured and the SSH user has write permissions to the snippets directory. See https://docs.obiente.cloud/guides/proxmox-ssh-user-setup for troubleshooting", err)
	}

	logger.Info("[ProxmoxClient] Successfully created snippet via SSH: %s", snippetPath)
	return snippetPath, nil
}


func (pc *ProxmoxClient) writeSnippetViaSSH(ctx context.Context, nodeName string, storage string, filename string, content string) (string, error) {
	// Check if we have a way to resolve SSH endpoint (via node mapping)
	sshEndpoint := resolveSSHEndpoint(nodeName, pc.config)
	if sshEndpoint == "" {
		return "", fmt.Errorf("SSH endpoint not configured for node %s (configure PROXMOX_NODE_ENDPOINTS or PROXMOX_NODE_SSH_ENDPOINTS)", nodeName)
	}
	if pc.config.SSHUser == "" {
		return "", fmt.Errorf("SSH user not configured (PROXMOX_SSH_USER)")
	}

	// Load SSH private key
	var signer ssh.Signer
	var err error

	if pc.config.SSHKeyData != "" {
		// Use key data from environment variable
		// Support both raw key data and base64-encoded key data
		keyData := []byte(pc.config.SSHKeyData)

		// Try to decode as base64 first (if it fails, assume it's raw key data)
		if decoded, err := base64.StdEncoding.DecodeString(pc.config.SSHKeyData); err == nil {
			// Successfully decoded as base64 - check if it looks like a valid SSH key
			if strings.Contains(string(decoded), "BEGIN") || strings.Contains(string(decoded), "PRIVATE KEY") {
				keyData = decoded
			}
			// If base64 decode succeeded but doesn't look like a key, try raw anyway
		}

		signer, err = ssh.ParsePrivateKey(keyData)
		if err != nil {
			return "", fmt.Errorf("failed to parse SSH key data: %w", err)
		}
	} else if pc.config.SSHKeyPath != "" {
		// Read key from file
		keyData, err := os.ReadFile(pc.config.SSHKeyPath)
		if err != nil {
			return "", fmt.Errorf("failed to read SSH key file: %w", err)
		}
		signer, err = ssh.ParsePrivateKey(keyData)
		if err != nil {
			return "", fmt.Errorf("failed to parse SSH key: %w", err)
		}
	} else {
		return "", fmt.Errorf("SSH key not configured (PROXMOX_SSH_KEY_PATH or PROXMOX_SSH_KEY_DATA)")
	}

	// Create SSH client config
	sshConfig := &ssh.ClientConfig{
		User:            pc.config.SSHUser,
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(signer)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // TODO: Consider validating host keys for production
		Timeout:         10 * time.Second,
	}

	// Connect to Proxmox node via SSH
	// Resolve SSH endpoint from node name using PROXMOX_NODE_SSH_ENDPOINTS or PROXMOX_NODE_ENDPOINTS mapping
	// (sshEndpoint was already resolved at the start of the function)
	if sshEndpoint == "" {
		return "", fmt.Errorf("SSH endpoint not configured for node %s (configure PROXMOX_NODE_ENDPOINTS or PROXMOX_NODE_SSH_ENDPOINTS)", nodeName)
	}
	
	sshHost := sshEndpoint
	sshPort := "22"
	if strings.Contains(sshEndpoint, ":") {
		// Port is included in endpoint
		parts := strings.Split(sshEndpoint, ":")
		sshHost = parts[0]
		sshPort = parts[1]
	}

	conn, err := ssh.Dial("tcp", fmt.Sprintf("%s:%s", sshHost, sshPort), sshConfig)
	if err != nil {
		return "", fmt.Errorf("failed to connect to Proxmox node via SSH: %w", err)
	}
	defer conn.Close()

	// Determine snippets directory path
	// Try to get storage path via API first, fallback to default
	storageInfo, err := pc.getStorageInfo(ctx, nodeName, storage)
	var snippetsPath string
	if err == nil && storageInfo != nil {
		if pathVal, ok := storageInfo["path"].(string); ok && pathVal != "" {
			snippetsPath = fmt.Sprintf("%s/snippets", pathVal)
		}
	}

	// Fallback to default path for local storage
	if snippetsPath == "" {
		snippetsPath = "/var/lib/vz/snippets"
	}

	filePath := fmt.Sprintf("%s/%s", snippetsPath, filename)
	logger.Debug("[ProxmoxClient] Writing snippet file to: %s", filePath)

	// Write file using dd via stdin
	session, err := conn.NewSession()
	if err != nil {
		return "", fmt.Errorf("failed to create SSH session: %w", err)
	}
	defer session.Close()

	var stderr bytes.Buffer
	session.Stderr = &stderr

	stdin, err := session.StdinPipe()
	if err != nil {
		return "", fmt.Errorf("failed to get stdin pipe: %w", err)
	}

	cmd := fmt.Sprintf("/bin/sh -c 'dd of=\"%s\" bs=8192 2>/dev/null'", filePath)
	if err := session.Start(cmd); err != nil {
		stdin.Close()
		return "", fmt.Errorf("failed to start dd command: %w", err)
	}

	// Write content to stdin
	if _, err := stdin.Write([]byte(content)); err != nil {
		stdin.Close()
		session.Wait()
		return "", fmt.Errorf("failed to write file content: %w", err)
	}
	stdin.Close()

	// Wait for command to complete
	if err := session.Wait(); err != nil {
		// Even if dd returns an error, verify if file was created
		verifySession, _ := conn.NewSession()
		verifyCmd := fmt.Sprintf("/bin/sh -c 'test -f \"%s\"'", filePath)
		if verifySession.Run(verifyCmd) != nil {
			verifySession.Close()
			return "", fmt.Errorf("failed to write file to %s: %w (stderr: %s)", filePath, err, stderr.String())
		}
		verifySession.Close()
	}

	// Verify file exists (dd may succeed but file might not be created)
	verifySession, err := conn.NewSession()
	if err != nil {
		return "", fmt.Errorf("failed to create verification session: %w", err)
	}
	verifyCmd := fmt.Sprintf("/bin/sh -c 'test -f \"%s\"'", filePath)
	if err := verifySession.Run(verifyCmd); err != nil {
		verifySession.Close()
		return "", fmt.Errorf("file write completed but file %s does not exist", filePath)
	}
	verifySession.Close()

	// Set file permissions (non-critical)
	chmodSession, _ := conn.NewSession()
	chmodSession.Run(fmt.Sprintf("/bin/sh -c 'chmod 644 \"%s\"'", filePath))
	chmodSession.Close()

	logger.Info("[ProxmoxClient] Successfully wrote snippet file via SSH: %s", filePath)

	// Return the cicustom path
	snippetPath := fmt.Sprintf("user=%s:snippets/%s", storage, filename)
	return snippetPath, nil
}


func (pc *ProxmoxClient) deleteSnippetViaSSH(ctx context.Context, nodeName string, storage string, filename string) error {
	// Check if we have a way to resolve SSH endpoint (via node mapping)
	sshEndpoint := resolveSSHEndpoint(nodeName, pc.config)
	if sshEndpoint == "" {
		return fmt.Errorf("SSH endpoint not configured for node %s (configure PROXMOX_NODE_ENDPOINTS or PROXMOX_NODE_SSH_ENDPOINTS)", nodeName)
	}
	if pc.config.SSHUser == "" {
		return fmt.Errorf("SSH user not configured (PROXMOX_SSH_USER)")
	}
	if pc.config.SSHKeyPath == "" && pc.config.SSHKeyData == "" {
		return fmt.Errorf("SSH key not configured (PROXMOX_SSH_KEY_PATH or PROXMOX_SSH_KEY_DATA)")
	}

	// Load SSH private key
	var signer ssh.Signer
	var err error

	if pc.config.SSHKeyData != "" {
		keyData := []byte(pc.config.SSHKeyData)
		if decoded, err := base64.StdEncoding.DecodeString(pc.config.SSHKeyData); err == nil {
			if strings.Contains(string(decoded), "BEGIN") || strings.Contains(string(decoded), "PRIVATE KEY") {
				keyData = decoded
			}
		}
		signer, err = ssh.ParsePrivateKey(keyData)
		if err != nil {
			return fmt.Errorf("failed to parse SSH key data: %w", err)
		}
	} else if pc.config.SSHKeyPath != "" {
		keyData, err := os.ReadFile(pc.config.SSHKeyPath)
		if err != nil {
			return fmt.Errorf("failed to read SSH key file: %w", err)
		}
		signer, err = ssh.ParsePrivateKey(keyData)
		if err != nil {
			return fmt.Errorf("failed to parse SSH key: %w", err)
		}
	} else {
		return fmt.Errorf("SSH key not configured (PROXMOX_SSH_KEY_PATH or PROXMOX_SSH_KEY_DATA)")
	}

	// Create SSH client config
	sshConfig := &ssh.ClientConfig{
		User:            pc.config.SSHUser,
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(signer)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}

	// Connect to Proxmox node via SSH
	// Use resolved SSH endpoint (from node mapping)
	sshHost := sshEndpoint
	if sshHost == "" {
		return fmt.Errorf("SSH endpoint not configured for node %s (configure PROXMOX_NODE_ENDPOINTS or PROXMOX_NODE_SSH_ENDPOINTS)", nodeName)
	}
	
	sshPort := "22"
	if strings.Contains(sshHost, ":") {
		parts := strings.Split(sshHost, ":")
		sshHost = parts[0]
		sshPort = parts[1]
	}

	conn, err := ssh.Dial("tcp", fmt.Sprintf("%s:%s", sshHost, sshPort), sshConfig)
	if err != nil {
		return fmt.Errorf("failed to connect to Proxmox node via SSH: %w", err)
	}
	defer conn.Close()

	// Determine snippets directory path
	storageInfo, err := pc.getStorageInfo(ctx, nodeName, storage)
	var snippetsPath string
	if err == nil && storageInfo != nil {
		if pathVal, ok := storageInfo["path"].(string); ok && pathVal != "" {
			snippetsPath = fmt.Sprintf("%s/snippets", pathVal)
		}
	}

	if snippetsPath == "" {
		snippetsPath = "/var/lib/vz/snippets"
	}

	filePath := fmt.Sprintf("%s/%s", snippetsPath, filename)

	// Delete the file
	session, err := conn.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create SSH session: %w", err)
	}
	defer session.Close()

	cmd := fmt.Sprintf("/bin/sh -c 'rm -f \"%s\"'", filePath)
	if err := session.Run(cmd); err != nil {
		// Check if file exists - if it doesn't, that's fine (already deleted)
		verifySession, _ := conn.NewSession()
		verifyCmd := fmt.Sprintf("/bin/sh -c 'test -f \"%s\"'", filePath)
		if verifySession.Run(verifyCmd) == nil {
			verifySession.Close()
			return fmt.Errorf("failed to delete snippet file %s: %w", filePath, err)
		}
		verifySession.Close()
		// File doesn't exist - already deleted, that's fine
		logger.Debug("[ProxmoxClient] Snippet file %s does not exist (may have been already deleted)", filePath)
		return nil
	}

	logger.Info("[ProxmoxClient] Successfully deleted snippet file via SSH: %s", filePath)
	return nil
}


func (pc *ProxmoxClient) ReadSnippetViaSSH(ctx context.Context, nodeName string, storage string, filename string) (string, error) {
	// Check if we have a way to resolve SSH endpoint (via node mapping)
	sshEndpoint := resolveSSHEndpoint(nodeName, pc.config)
	if sshEndpoint == "" {
		return "", fmt.Errorf("SSH endpoint not configured for node %s (configure PROXMOX_NODE_ENDPOINTS or PROXMOX_NODE_SSH_ENDPOINTS)", nodeName)
	}
	if pc.config.SSHUser == "" {
		return "", fmt.Errorf("SSH user not configured (PROXMOX_SSH_USER)")
	}
	if pc.config.SSHKeyPath == "" && pc.config.SSHKeyData == "" {
		return "", fmt.Errorf("SSH key not configured (PROXMOX_SSH_KEY_PATH or PROXMOX_SSH_KEY_DATA)")
	}

	// Load SSH private key
	var signer ssh.Signer
	var err error

	if pc.config.SSHKeyData != "" {
		keyData := []byte(pc.config.SSHKeyData)
		if decoded, err := base64.StdEncoding.DecodeString(pc.config.SSHKeyData); err == nil {
			if strings.Contains(string(decoded), "BEGIN") || strings.Contains(string(decoded), "PRIVATE KEY") {
				keyData = decoded
			}
		}
		signer, err = ssh.ParsePrivateKey(keyData)
		if err != nil {
			return "", fmt.Errorf("failed to parse SSH key data: %w", err)
		}
	} else if pc.config.SSHKeyPath != "" {
		keyData, err := os.ReadFile(pc.config.SSHKeyPath)
		if err != nil {
			return "", fmt.Errorf("failed to read SSH key file: %w", err)
		}
		signer, err = ssh.ParsePrivateKey(keyData)
		if err != nil {
			return "", fmt.Errorf("failed to parse SSH key: %w", err)
		}
	} else {
		return "", fmt.Errorf("SSH key not configured (PROXMOX_SSH_KEY_PATH or PROXMOX_SSH_KEY_DATA)")
	}

	// Create SSH client config
	sshConfig := &ssh.ClientConfig{
		User:            pc.config.SSHUser,
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(signer)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}

	// Connect to Proxmox node via SSH
	// Use resolved SSH endpoint (from node mapping)
	sshHost := sshEndpoint
	if sshHost == "" {
		return "", fmt.Errorf("SSH endpoint not configured for node %s (configure PROXMOX_NODE_ENDPOINTS or PROXMOX_NODE_SSH_ENDPOINTS)", nodeName)
	}
	
	sshPort := "22"
	if strings.Contains(sshHost, ":") {
		parts := strings.Split(sshHost, ":")
		sshHost = parts[0]
		sshPort = parts[1]
	}

	conn, err := ssh.Dial("tcp", fmt.Sprintf("%s:%s", sshHost, sshPort), sshConfig)
	if err != nil {
		return "", fmt.Errorf("failed to connect to Proxmox node via SSH: %w", err)
	}
	defer conn.Close()

	// Determine snippets directory path
	storageInfo, err := pc.getStorageInfo(ctx, nodeName, storage)
	var snippetsPath string
	if err == nil && storageInfo != nil {
		if pathVal, ok := storageInfo["path"].(string); ok && pathVal != "" {
			snippetsPath = fmt.Sprintf("%s/snippets", pathVal)
		}
	}

	if snippetsPath == "" {
		snippetsPath = "/var/lib/vz/snippets"
	}

	filePath := fmt.Sprintf("%s/%s", snippetsPath, filename)
	logger.Debug("[ProxmoxClient] Reading snippet file from: %s", filePath)

	// Read file using cat
	session, err := conn.NewSession()
	if err != nil {
		return "", fmt.Errorf("failed to create SSH session: %w", err)
	}
	defer session.Close()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	session.Stdout = &stdout
	session.Stderr = &stderr

	cmd := fmt.Sprintf("/bin/sh -c 'cat \"%s\"'", filePath)
	if err := session.Run(cmd); err != nil {
		return "", fmt.Errorf("failed to read snippet file %s: %w (stderr: %s)", filePath, err, stderr.String())
	}

	content := stdout.String()
	if content == "" {
		return "", fmt.Errorf("snippet file %s is empty", filePath)
	}

	logger.Debug("[ProxmoxClient] Successfully read snippet file via SSH: %s (%d bytes)", filePath, len(content))
	return content, nil
}

// generateUbuntuDebianRuncmd generates runcmd commands for Ubuntu and Debian (apt-based)
func generateUbuntuDebianRuncmd(config *VPSConfig, needsSSHRestart bool) string {
	userData := ""

	userData += "  - |\n"
	userData += "    echo \"Installing openssh-server and qemu-guest-agent (Ubuntu/Debian)...\"\n"
	userData += "    apt-get install -y --no-update openssh-server qemu-guest-agent 2>/dev/null || {\n"
	userData += "      echo \"Direct install failed, updating package lists and retrying...\"\n"
	userData += "      sleep 5\n"
	userData += "      apt-get update || echo \"WARNING: apt-get update failed, continuing anyway\"\n"
	userData += "      apt-get install -y openssh-server || echo \"WARNING: openssh-server installation failed\"\n"
	userData += "      apt-get install -y qemu-guest-agent || echo \"WARNING: qemu-guest-agent not available\"\n"
	userData += "    }\n"
	userData += "    if dpkg -l qemu-guest-agent 2>/dev/null | grep -q '^ii'; then\n"
	userData += "      echo \"qemu-guest-agent package verified installed\"\n"
	userData += "    else\n"
	userData += "      echo \"WARNING: qemu-guest-agent package not installed\"\n"
	userData += "    fi\n"
	userData += "  \n"

	// Enable and start SSH
	userData += "  - |\n"
	userData += "    echo \"Enabling and starting SSH (Ubuntu/Debian)...\"\n"
	userData += "    systemctl enable ssh || systemctl enable sshd || true\n"
	userData += "    systemctl start ssh || systemctl start sshd || true\n"
	userData += "    echo \"SSH service configuration completed\"\n"
	userData += "  \n"

	// Restart SSH if needed
	if needsSSHRestart {
		userData += "  - systemctl restart sshd || systemctl restart ssh || service ssh restart || service sshd restart || true\n"
	}

	// Configure qemu-guest-agent with more robust startup
	userData += "  - |\n"
	userData += "    echo \"Configuring qemu-guest-agent (Ubuntu/Debian)...\"\n"
	userData += "    # Wait for systemd to be fully ready\n"
	userData += "    sleep 2\n"
	userData += "    if systemctl list-unit-files 2>/dev/null | grep -q qemu-guest-agent; then\n"
	userData += "      systemctl enable qemu-guest-agent 2>/dev/null || echo \"WARNING: Failed to enable qemu-guest-agent\"\n"
	userData += "      # Stop if running to ensure clean start\n"
	userData += "      systemctl stop qemu-guest-agent 2>/dev/null || true\n"
	userData += "      sleep 1\n"
	userData += "      systemctl start qemu-guest-agent 2>/dev/null || echo \"WARNING: Failed to start qemu-guest-agent\"\n"
	userData += "      # Verify it's running\n"
	userData += "      sleep 2\n"
	userData += "      if systemctl is-active --quiet qemu-guest-agent 2>/dev/null; then\n"
	userData += "        echo \"qemu-guest-agent is running successfully\"\n"
	userData += "      else\n"
	userData += "        echo \"WARNING: qemu-guest-agent is not running, checking status...\"\n"
	userData += "        systemctl status qemu-guest-agent 2>&1 || true\n"
	userData += "        # One more retry\n"
	userData += "        sleep 3\n"
	userData += "        systemctl start qemu-guest-agent 2>/dev/null || echo \"WARNING: qemu-guest-agent still not running\"\n"
	userData += "      fi\n"
	userData += "    else\n"
	userData += "      echo \"WARNING: qemu-guest-agent service not found (package may not be installed)\"\n"
	userData += "      echo \"Checking if package exists...\"\n"
	userData += "      dpkg -l | grep -i qemu || echo \"No qemu packages found\"\n"
	userData += "    fi\n"
	userData += "    echo \"Guest agent configuration completed\"\n"
	userData += "  \n"

	return userData
}

// generateRockyAlmaRuncmd generates runcmd commands for Rocky Linux and Alma Linux (yum/dnf-based)
func generateRockyAlmaRuncmd(config *VPSConfig, needsSSHRestart bool) string {
	userData := ""

	// Update package lists
	userData += "  - |\n"
	userData += "    echo \"Updating package lists (Rocky/Alma Linux)...\"\n"
	userData += "    if command -v dnf >/dev/null 2>&1; then\n"
	userData += "      dnf update -y || true\n"
	userData += "    elif command -v yum >/dev/null 2>&1; then\n"
	userData += "      yum update -y || true\n"
	userData += "    fi\n"
	userData += "    echo \"Package lists updated\"\n"
	userData += "  \n"

	// Install SSH server and guest agent
	userData += "  - |\n"
	userData += "    echo \"Installing openssh-server and qemu-guest-agent (Rocky/Alma Linux)...\"\n"
	userData += "    if command -v dnf >/dev/null 2>&1; then\n"
	userData += "      dnf install -y openssh-server qemu-guest-agent && echo \"Packages installed\" || echo \"WARNING: Package installation failed\"\n"
	userData += "    elif command -v yum >/dev/null 2>&1; then\n"
	userData += "      yum install -y openssh-server qemu-guest-agent && echo \"Packages installed\" || echo \"WARNING: Package installation failed\"\n"
	userData += "    fi\n"
	userData += "  \n"

	// Enable and start SSH
	userData += "  - |\n"
	userData += "    echo \"Enabling and starting SSH (Rocky/Alma Linux)...\"\n"
	userData += "    systemctl enable sshd || systemctl enable ssh || true\n"
	userData += "    systemctl start sshd || systemctl start ssh || true\n"
	userData += "    echo \"SSH service configuration completed\"\n"
	userData += "  \n"

	// Restart SSH if needed
	if needsSSHRestart {
		userData += "  - systemctl restart sshd || systemctl restart ssh || service ssh restart || service sshd restart || true\n"
	}

	// Configure qemu-guest-agent
	userData += "  - |\n"
	userData += "    echo \"Configuring qemu-guest-agent (Rocky/Alma Linux)...\"\n"
	userData += "    if systemctl list-unit-files 2>/dev/null | grep -q qemu-guest-agent; then\n"
	userData += "      systemctl enable qemu-guest-agent 2>/dev/null || echo \"WARNING: Failed to enable qemu-guest-agent\"\n"
	userData += "      systemctl start qemu-guest-agent 2>/dev/null || echo \"WARNING: Failed to start qemu-guest-agent\"\n"
	userData += "      if systemctl is-active --quiet qemu-guest-agent 2>/dev/null; then\n"
	userData += "        echo \"qemu-guest-agent is running\"\n"
	userData += "      else\n"
	userData += "        echo \"WARNING: qemu-guest-agent is not running (this is non-critical)\"\n"
	userData += "        sleep 2\n"
	userData += "        systemctl start qemu-guest-agent 2>/dev/null || echo \"WARNING: qemu-guest-agent still not running\"\n"
	userData += "      fi\n"
	userData += "    else\n"
	userData += "      echo \"WARNING: qemu-guest-agent service not found (package may not be installed)\"\n"
	userData += "    fi\n"
	userData += "    echo \"Guest agent configuration completed\"\n"
	userData += "  \n"

	return userData
}

// generateGenericRuncmd generates runcmd commands for unknown OS (tries to detect package manager)
func generateGenericRuncmd(config *VPSConfig, needsSSHRestart bool) string {
	userData := ""

	// Update package lists - try to detect package manager
	userData += "  - |\n"
	userData += "    echo \"Updating package lists (generic OS detection)...\"\n"
	userData += "    if command -v apt-get >/dev/null 2>&1; then\n"
	userData += "      # Ensure universe repository is enabled on Ubuntu\n"
	userData += "      if [ -f /etc/apt/sources.list ]; then\n"
	userData += "        sed -i '/^#.*universe/s/^#//' /etc/apt/sources.list || true\n"
	userData += "        sed -i '/^#.*universe/s/^#//' /etc/apt/sources.list.d/*.list 2>/dev/null || true\n"
	userData += "      fi\n"
	userData += "      # Skip apt-get update by default to avoid DNS/network issues\n"
	userData += "    elif command -v yum >/dev/null 2>&1; then\n"
	userData += "      yum update -y || true\n"
	userData += "    elif command -v dnf >/dev/null 2>&1; then\n"
	userData += "      dnf update -y || true\n"
	userData += "    fi\n"
	userData += "    echo \"Package lists updated (or skipped if not needed)\"\n"
	userData += "  \n"

	// Install SSH server and guest agent
	userData += "  - |\n"
	userData += "    echo \"Installing openssh-server and qemu-guest-agent (generic OS detection)...\"\n"
	userData += "    INSTALLED=0\n"
	userData += "    if command -v apt-get >/dev/null 2>&1; then\n"
	userData += "      apt-get install -y openssh-server && INSTALLED=1 || echo \"WARNING: openssh-server installation failed\"\n"
	userData += "      apt-get install -y qemu-guest-agent && echo \"qemu-guest-agent installed\" || echo \"WARNING: qemu-guest-agent not available\"\n"
	userData += "    elif command -v yum >/dev/null 2>&1; then\n"
	userData += "      yum install -y openssh-server qemu-guest-agent && INSTALLED=1 || echo \"WARNING: Package installation failed\"\n"
	userData += "    elif command -v dnf >/dev/null 2>&1; then\n"
	userData += "      dnf install -y openssh-server qemu-guest-agent && INSTALLED=1 || echo \"WARNING: Package installation failed\"\n"
	userData += "    else\n"
	userData += "      echo \"WARNING: No package manager found (apt-get, yum, or dnf) - packages may already be installed\"\n"
	userData += "    fi\n"
	userData += "    if [ $INSTALLED -eq 1 ]; then\n"
	userData += "      echo \"Successfully installed openssh-server\"\n"
	userData += "    else\n"
	userData += "      echo \"Packages may already be installed or installation failed - continuing anyway\"\n"
	userData += "    fi\n"
	userData += "  \n"

	// Enable and start SSH
	userData += "  - |\n"
	userData += "    echo \"Enabling and starting SSH (generic OS detection)...\"\n"
	userData += "    systemctl enable ssh || systemctl enable sshd || true\n"
	userData += "    systemctl start ssh || systemctl start sshd || true\n"
	userData += "    echo \"SSH service configuration completed\"\n"
	userData += "  \n"

	// Restart SSH if needed
	if needsSSHRestart {
		userData += "  - systemctl restart sshd || systemctl restart ssh || service ssh restart || service sshd restart || true\n"
	}

	// Configure qemu-guest-agent
	userData += "  - |\n"
	userData += "    echo \"Configuring qemu-guest-agent (generic OS detection)...\"\n"
	userData += "    if command -v qemu-ga >/dev/null 2>&1 || systemctl list-unit-files 2>/dev/null | grep -q qemu-guest-agent; then\n"
	userData += "      echo \"qemu-guest-agent is already installed\"\n"
	userData += "    else\n"
	userData += "      echo \"WARNING: qemu-guest-agent package not found, attempting to install...\"\n"
	userData += "      if command -v apt-get >/dev/null 2>&1; then\n"
	userData += "        if [ -f /etc/apt/sources.list ]; then\n"
	userData += "          sed -i '/^#.*universe/s/^#//' /etc/apt/sources.list || true\n"
	userData += "          sed -i '/^#.*universe/s/^#//' /etc/apt/sources.list.d/*.list 2>/dev/null || true\n"
	userData += "          # Skip apt-get update by default to avoid DNS/network issues\n"
	userData += "        fi\n"
	userData += "        apt-get install -y qemu-guest-agent || echo \"WARNING: Failed to install qemu-guest-agent\"\n"
	userData += "      elif command -v yum >/dev/null 2>&1; then\n"
	userData += "        yum install -y qemu-guest-agent || echo \"WARNING: Failed to install qemu-guest-agent\"\n"
	userData += "      elif command -v dnf >/dev/null 2>&1; then\n"
	userData += "        dnf install -y qemu-guest-agent || echo \"WARNING: Failed to install qemu-guest-agent\"\n"
	userData += "      fi\n"
	userData += "    fi\n"
	userData += "    if systemctl list-unit-files 2>/dev/null | grep -q qemu-guest-agent; then\n"
	userData += "      systemctl enable qemu-guest-agent 2>/dev/null || echo \"WARNING: Failed to enable qemu-guest-agent\"\n"
	userData += "      systemctl start qemu-guest-agent 2>/dev/null || echo \"WARNING: Failed to start qemu-guest-agent\"\n"
	userData += "      if systemctl is-active --quiet qemu-guest-agent 2>/dev/null; then\n"
	userData += "        echo \"qemu-guest-agent is running\"\n"
	userData += "      else\n"
	userData += "        echo \"WARNING: qemu-guest-agent is not running (this is non-critical)\"\n"
	userData += "        sleep 2\n"
	userData += "        systemctl start qemu-guest-agent 2>/dev/null || echo \"WARNING: qemu-guest-agent still not running\"\n"
	userData += "      fi\n"
	userData += "    else\n"
	userData += "      echo \"WARNING: qemu-guest-agent service not found (package may not be installed)\"\n"
	userData += "    fi\n"
	userData += "    echo \"Guest agent configuration completed\"\n"
	userData += "  \n"

	return userData
}


// UpdateCloudInitUserDataWithStaticIP updates the cloud-init userData snippet to add a public IP address
// alongside the existing internal DHCP IP. The public IP uses its own gateway for routing.
// This function reads the existing userData and adds the public IP configuration without removing DHCP.
func (pc *ProxmoxClient) UpdateCloudInitUserDataWithStaticIP(ctx context.Context, nodeName string, vmID int, publicIP string, publicGateway string, netmask string) error {
	// Get VM config to find the cicustom path
	vmConfig, err := pc.GetVMConfig(ctx, nodeName, vmID)
	if err != nil {
		return fmt.Errorf("failed to get VM config: %w", err)
	}

	// Check if VM uses cloud-init snippets
	cicustom, ok := vmConfig["cicustom"].(string)
	if !ok || cicustom == "" {
		return fmt.Errorf("VM %d does not use cloud-init snippets (cicustom not set). Cannot update IP configuration", vmID)
	}

	// Parse cicustom to get storage and filename
	// Format: "user=<storage>:snippets/<filename>"
	var storage, filename string
	if strings.HasPrefix(cicustom, "user=") {
		parts := strings.TrimPrefix(cicustom, "user=")
		storageAndFile := strings.SplitN(parts, ":", 2)
		if len(storageAndFile) == 2 {
			storage = storageAndFile[0]
			filepath := storageAndFile[1]
			// Extract filename from path (e.g., "snippets/vm-123-user-data" -> "vm-123-user-data")
			if strings.HasPrefix(filepath, "snippets/") {
				filename = strings.TrimPrefix(filepath, "snippets/")
			} else {
				filename = filepath
			}
		} else {
			return fmt.Errorf("invalid cicustom format: %s", cicustom)
		}
	} else {
		return fmt.Errorf("unsupported cicustom format: %s (expected 'user=<storage>:snippets/<filename>')", cicustom)
	}

	if storage == "" || filename == "" {
		return fmt.Errorf("failed to parse storage and filename from cicustom: %s", cicustom)
	}

	// Read existing userData snippet
	existingUserData, err := pc.ReadSnippetViaSSH(ctx, nodeName, storage, filename)
	if err != nil {
		return fmt.Errorf("failed to read existing cloud-init snippet: %w", err)
	}

	// Update userData to add public IP alongside existing DHCP configuration
	updatedUserData := updateUserDataWithPublicIP(existingUserData, publicIP, publicGateway, netmask)

	// Write updated snippet back
	_, err = pc.writeSnippetViaSSH(ctx, nodeName, storage, filename, updatedUserData)
	if err != nil {
		return fmt.Errorf("failed to write updated cloud-init snippet: %w", err)
	}

	logger.Info("[ProxmoxClient] Successfully updated cloud-init userData for VM %d with public IP %s", vmID, publicIP)
	return nil
}

// updateUserDataWithPublicIP adds a public IP address to cloud-init userData alongside the existing DHCP configuration
// The public IP uses its own gateway for routing, while the internal IP continues to use DHCP via Proxmox ipconfig0
func updateUserDataWithPublicIP(userData string, publicIP string, publicGateway string, netmask string) string {
	// Default values if not provided
	if netmask == "" {
		netmask = "24" // Default to /24
	}
	if publicGateway == "" {
		// Calculate default gateway from IP (typically .1 in the subnet)
		ip := net.ParseIP(publicIP)
		if ip != nil {
			// For /24, set last octet to 1
			ip4 := ip.To4()
			if ip4 != nil {
				ip4[3] = 1
				publicGateway = ip4.String()
			}
		}
		// If we still don't have a gateway, this means the IP is invalid
		// This should not happen as IPs should be validated before calling this function
		if publicGateway == "" {
			logger.Error("[ProxmoxClient] Failed to calculate gateway for public IP %s: invalid IP format", publicIP)
			// Return userData unchanged - the network config will be invalid but we won't use a wrong gateway
			return userData
		}
	}

	// Parse the IP to validate it
	ip := net.ParseIP(publicIP)
	if ip == nil {
		logger.Warn("[ProxmoxClient] Invalid IP address format: %s, using default netmask", publicIP)
	}

	// Check if the public IP is already configured
	if strings.Contains(userData, publicIP) {
		logger.Info("[ProxmoxClient] Public IP %s already configured in userData", publicIP)
		return userData
	}

	// Check if network config is disabled (using Proxmox ipconfig0 for DHCP)
	networkDisabled := strings.Contains(userData, "network:\n  config: disabled")
	
	// We need to enable network config to add the public IP, but keep DHCP working
	// Proxmox ipconfig0 will still work for the primary interface, we're just adding an additional address
	if networkDisabled {
		// Remove the disabled network config line
		userData = strings.Replace(userData, "network:\n  config: disabled\n", "", 1)
	}

	// Find where to insert network config (after SSH config, before hostname)
	insertPoint := strings.Index(userData, "hostname:")
	if insertPoint == -1 {
		insertPoint = strings.Index(userData, "users:")
	}
	if insertPoint == -1 {
		insertPoint = strings.Index(userData, "package_update:")
	}
	if insertPoint == -1 {
		// Insert at the end of SSH config
		sshIndex := strings.Index(userData, "ssh:\n")
		if sshIndex != -1 {
			nextDoubleNewline := strings.Index(userData[sshIndex:], "\n\n")
			if nextDoubleNewline != -1 {
				insertPoint = sshIndex + nextDoubleNewline + 2
			}
		}
	}

	// Calculate the public IP subnet for routing
	publicIPSubnet := ""
	if ip != nil {
		ip4 := ip.To4()
		if ip4 != nil {
			// For /24, use the first 3 octets
			publicIPSubnet = fmt.Sprintf("%d.%d.%d.0/24", ip4[0], ip4[1], ip4[2])
		}
	}
	if publicIPSubnet == "" {
		publicIPSubnet = "0.0.0.0/0" // Fallback to default route
	}

	// Network configuration that adds public IP alongside DHCP
	// DHCP will continue to work via Proxmox ipconfig0, this adds the public IP as an additional address
	networkConfig := fmt.Sprintf(`network:
  version: 2
  ethernets:
    eth0:
      # Keep DHCP for internal IP (configured via Proxmox ipconfig0)
      dhcp4: true
      # Add public IP as additional address
      addresses:
        - %s/%s
      # Routing configuration for public IP
      routes:
        # Route for public IP subnet via public gateway
        - to: %s
          via: %s
          metric: 100
      nameservers:
        addresses: [1.1.1.1, 1.0.0.1]

`, publicIP, netmask, publicIPSubnet, publicGateway)

	if insertPoint > 0 && insertPoint < len(userData) {
		userData = userData[:insertPoint] + networkConfig + userData[insertPoint:]
	} else {
		// Append at the beginning (after #cloud-config)
		cloudConfigIndex := strings.Index(userData, "#cloud-config\n")
		if cloudConfigIndex != -1 {
			insertPoint = cloudConfigIndex + len("#cloud-config\n") + 1
			userData = userData[:insertPoint] + networkConfig + userData[insertPoint:]
		} else {
			// Prepend
			userData = networkConfig + userData
		}
	}

	return userData
}
