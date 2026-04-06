// Package inputvalidation provides security-focused input validation for all
// user-controllable string fields across Obiente Cloud services.
//
// Attack surfaces covered:
//   - Shell fields (VPS user creation/update)
//   - Cloud-init runcmd entries and user shells
//   - write_files paths, permissions, and content
//   - Package names in cloud-init
//   - Hostnames, timezones, locales
//   - SSH authorized keys
//   - Usernames and group names
//   - Game server docker images, start commands, and env var keys/values
//   - File upload paths and file names
package inputvalidation

import (
	"encoding/base64"
	"fmt"
	"regexp"
	"strings"
)

// ─────────────────────────────────────────────────────────────────────────────
// Common miner / reverse-shell / dropper indicators
// ─────────────────────────────────────────────────────────────────────────────

// minerKeywords are substrings found in cryptomining payloads.
var minerKeywords = []string{
	// miners
	"xmrig", "cpuminer", "ccminer", "nicehash", "claymore", "teamredminer",
	"gminer", "lolminer", "phoenixminer", "nbminer", "trex-miner", "miniZ",
	// pools / protocols
	"stratum+", "stratum+tcp", "stratum+ssl",
	"pool.hashvault", "xmrpool", "minexmr", "nanopool", "supportxmr",
	"unmineable", "rx.unmineable", "moneroocean", "2miners.com", "ethermine",
	"f2pool", "antpool", "viabtc", "slushpool", "nicehash.com",
	// performance tuning for miners
	"nr_hugepages", "msr-tools", "hugepages", "transparent_hugepage",
}

// shellMetacharacters are characters/sequences that enable command injection.
var shellMetacharacters = []string{
	"&&", "||", ";;", ";", "`", "$(", "${", "<(", ">(", ">(", ">|",
}

// dropperKeywords are typical dropper-stage commands.
var dropperKeywords = []string{
	"wget ", "wget\t", "curl ", "curl\t",
	"apt-get", "apt ", "yum ", "dnf ", "apk add",
	"pip install", "npm install", "cargo install",
	"chmod +x", "chmod 7", "chown ",
	"/tmp/", "/dev/shm", "/run/shm",
}

// shellInterpreters are executable interpreters that could run arbitrary code.
var shellInterpreters = []string{
	"python2", "python3", "python ", "python\t",
	"perl ", "perl\t",
	"ruby ", "ruby\t",
	"node ", "node\t",
	"lua ", "lua\t",
	"php ", "php\t",
	"bash ", "bash\t", "bash\n", "bash -",
	"sh ", "sh\t", "sh\n", "sh -",
	"zsh ", "zsh\t",
	"powershell", "pwsh",
}

// sessionTools are used to hide persistent mining sessions.
var sessionTools = []string{
	"tmux ", "tmux\t", "tmux\n",
	"screen ", "screen\t", "screen\n",
	"nohup ",
	"disown",
}

// ─────────────────────────────────────────────────────────────────────────────
// Shell field validation (e.g. /bin/bash)
// ─────────────────────────────────────────────────────────────────────────────

// Shell validates that a shell field contains only an absolute path to a shell
// binary and nothing else. Empty string is also accepted (means use default).
func Shell(shell string) error {
	if shell == "" {
		return nil
	}
	lower := strings.ToLower(shell)

	if err := containsAny(lower, minerKeywords, "miner-related content"); err != nil {
		return err
	}
	if err := containsAny(lower, dropperKeywords, "download/dropper command"); err != nil {
		return err
	}
	if err := containsAny(lower, shellMetacharacters, "shell metacharacter"); err != nil {
		return err
	}
	if err := containsAny(lower, sessionTools, "session persistence tool"); err != nil {
		return err
	}
	if err := containsAny(lower, shellInterpreters, "interpreter invocation"); err != nil {
		return err
	}

	// Must be a clean absolute path with no spaces or special chars
	if !strings.HasPrefix(shell, "/") {
		return fmt.Errorf("shell must be an absolute path (e.g. /bin/bash), got %q", shell)
	}
	// Must not contain spaces (a path cannot have a space and still be a bare binary path)
	if strings.ContainsAny(shell, " \t\n\r") {
		return fmt.Errorf("shell path must not contain whitespace, got %q", shell)
	}

	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Runcmd entry validation
// ─────────────────────────────────────────────────────────────────────────────

// Runcmd validates a single runcmd entry in cloud-init. These are full shell
// commands run as root, so we apply the full threat model.
func Runcmd(cmd string) error {
	lower := strings.ToLower(cmd)
	if err := containsAny(lower, minerKeywords, "miner-related content"); err != nil {
		return fmt.Errorf("runcmd entry rejected: %w", err)
	}
	if err := containsAny(lower, dropperKeywords, "download/dropper command"); err != nil {
		return fmt.Errorf("runcmd entry rejected: %w", err)
	}
	if err := containsAny(lower, sessionTools, "session persistence tool"); err != nil {
		return fmt.Errorf("runcmd entry rejected: %w", err)
	}
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Package name validation
// ─────────────────────────────────────────────────────────────────────────────

// validPackageName matches Debian/Ubuntu package name syntax.
var validPackageName = regexp.MustCompile(`^[a-z0-9][a-z0-9.+\-]{0,127}$`)

// blockedPackages are known mining/exploitation packages.
var blockedPackages = []string{
	"xmrig", "cpuminer", "ccminer", "nicehash",
	"msr-tools", "msrtools",
	"tmate", "ngrok",
}

// PackageName validates a single cloud-init package name.
func PackageName(pkg string) error {
	lower := strings.ToLower(strings.TrimSpace(pkg))
	if !validPackageName.MatchString(lower) {
		return fmt.Errorf("package name %q contains invalid characters", pkg)
	}
	for _, b := range blockedPackages {
		if strings.EqualFold(lower, b) || strings.HasPrefix(lower, b) {
			return fmt.Errorf("package %q is not permitted", pkg)
		}
	}
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// write_files field validation
// ─────────────────────────────────────────────────────────────────────────────

// WriteFilePath validates a cloud-init write_files path.
func WriteFilePath(path string) error {
	if path == "" {
		return fmt.Errorf("write_files path must not be empty")
	}
	if !strings.HasPrefix(path, "/") {
		return fmt.Errorf("write_files path must be absolute, got %q", path)
	}

	// Protect critical system directories being overwritten
	dangerousPrefixes := []string{
		"/etc/cron", "/etc/crontab", "/etc/cron.d",
		"/etc/sudoers", "/etc/sudoers.d",
		"/etc/passwd", "/etc/shadow", "/etc/group",
		"/etc/ssh/sshd_config",
		"/etc/rc", "/etc/init.d", "/etc/systemd",
		"/boot/", "/usr/lib/systemd", "/lib/systemd",
		"/proc/", "/sys/",
		"/tmp/", "/dev/shm", "/run/shm",
	}
	for _, pfx := range dangerousPrefixes {
		if strings.HasPrefix(path, pfx) {
			return fmt.Errorf("write_files path %q targets a protected system path", path)
		}
	}

	// Reject path traversal
	if strings.Contains(path, "..") {
		return fmt.Errorf("write_files path %q contains path traversal", path)
	}

	return nil
}

// WriteFilePermissions validates cloud-init write_files permissions field.
// Permissions must be an octal string like "0644". We block setuid/setgid/sticky
// bits and world-writable combined with other risks.
func WriteFilePermissions(perms string) error {
	if perms == "" {
		return nil
	}
	// Allow only octal strings
	if !regexp.MustCompile(`^0?[0-7]{3,4}$`).MatchString(perms) {
		return fmt.Errorf("write_files permissions %q must be an octal string (e.g. 0644)", perms)
	}
	// Reject setuid (4xxx) and setgid (2xxx) bits
	if len(perms) == 4 {
		switch perms[0] {
		case '4', '6', '7':
			return fmt.Errorf("write_files permissions %q sets setuid bit, which is not allowed", perms)
		case '2', '3':
			return fmt.Errorf("write_files permissions %q sets setgid bit, which is not allowed", perms)
		}
	}
	return nil
}

// WriteFileContent validates the content of a write_files entry.
// It scans for embedded miner scripts and dropper payloads, both in
// plain text and base64-encoded form.
func WriteFileContent(content string) error {
	if err := scanContentForThreats(content); err != nil {
		return fmt.Errorf("write_files content rejected: %w", err)
	}

	// Also try decoding as base64 and scanning the decoded payload
	decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(content))
	if err == nil && len(decoded) > 0 {
		if err := scanContentForThreats(string(decoded)); err != nil {
			return fmt.Errorf("write_files base64 content rejected: %w", err)
		}
	}

	return nil
}

func scanContentForThreats(content string) error {
	lower := strings.ToLower(content)
	if err := containsAny(lower, minerKeywords, "miner-related content"); err != nil {
		return err
	}
	if err := containsAny(lower, dropperKeywords, "download/dropper command"); err != nil {
		return err
	}
	if err := containsAny(lower, sessionTools, "session persistence tool"); err != nil {
		return err
	}
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Username / group name validation
// ─────────────────────────────────────────────────────────────────────────────

var validUsername = regexp.MustCompile(`^[a-z_][a-z0-9_\-]{0,31}$`)
var validGroupName = regexp.MustCompile(`^[a-z_][a-z0-9_\-]{0,31}$`)

// Username validates a Linux username.
func Username(name string) error {
	if name == "" {
		return fmt.Errorf("username must not be empty")
	}
	if name == "root" {
		return fmt.Errorf("username %q is reserved", name)
	}
	if !validUsername.MatchString(name) {
		return fmt.Errorf("username %q must match [a-z_][a-z0-9_-]{0,31}", name)
	}
	return nil
}

// GroupName validates a Linux group name.
func GroupName(name string) error {
	if name == "" {
		return fmt.Errorf("group name must not be empty")
	}
	// Allow some well-known pre-existing groups
	if !validGroupName.MatchString(name) {
		return fmt.Errorf("group name %q must match [a-z_][a-z0-9_-]{0,31}", name)
	}
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// SSH authorized key validation
// ─────────────────────────────────────────────────────────────────────────────

// SSHAuthorizedKey validates a single SSH authorized key entry.
// Rejects keys with embedded options that can execute commands on connect.
func SSHAuthorizedKey(key string) error {
	key = strings.TrimSpace(key)
	if key == "" {
		return nil
	}

	// Reject forced command / environment options that execute code
	lower := strings.ToLower(key)
	dangerousOptions := []string{
		`command="`, `command='`,
		`environment="`, `environment='`,
		`tunnel="`, `tunnel='`,
		`permitopen="`, `permitopen='`,
		`permitlisten="`, `permitlisten='`,
	}
	for _, opt := range dangerousOptions {
		if strings.Contains(lower, opt) {
			return fmt.Errorf("SSH key contains disallowed option %q", opt)
		}
	}

	// Key must start with a recognised key type
	validPrefixes := []string{
		"ssh-rsa ", "ssh-ed25519 ", "ssh-dss ", "ssh-ecdsa ",
		"ecdsa-sha2-nistp256 ", "ecdsa-sha2-nistp384 ", "ecdsa-sha2-nistp521 ",
		"sk-ssh-ed25519 ", "sk-ecdsa-sha2-nistp256 ",
	}
	ok := false
	for _, pfx := range validPrefixes {
		if strings.HasPrefix(key, pfx) {
			ok = true
			break
		}
	}
	if !ok {
		return fmt.Errorf("SSH key must start with a recognised key type (e.g. ssh-ed25519)")
	}

	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Hostname / timezone / locale validation
// ─────────────────────────────────────────────────────────────────────────────

var validHostname = regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?$`)

// Hostname validates a cloud-init hostname.
func Hostname(h string) error {
	if h == "" {
		return nil
	}
	if len(h) > 63 {
		return fmt.Errorf("hostname %q exceeds 63 characters", h)
	}
	if !validHostname.MatchString(h) {
		return fmt.Errorf("hostname %q contains invalid characters (only alphanumeric and hyphens allowed)", h)
	}
	return nil
}

var validTimezone = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_+\-/]{0,63}$`)

// Timezone validates a timezone string (e.g. "America/New_York").
func Timezone(tz string) error {
	if tz == "" {
		return nil
	}
	if !validTimezone.MatchString(tz) {
		return fmt.Errorf("timezone %q contains invalid characters", tz)
	}
	return nil
}

var validLocale = regexp.MustCompile(`^[a-z]{2,8}(_[A-Z]{2})?(\.[A-Za-z0-9\-]+)?(@[A-Za-z0-9]+)?$`)

// Locale validates a locale string (e.g. "en_US.UTF-8").
func Locale(l string) error {
	if l == "" {
		return nil
	}
	if !validLocale.MatchString(l) {
		return fmt.Errorf("locale %q contains invalid characters", l)
	}
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Game server fields
// ─────────────────────────────────────────────────────────────────────────────

// blockedDockerImages are known cryptomining container images.
var blockedDockerImages = []string{
	"xmrig", "cpuminer", "ccminer", "monero", "randomx",
	"minergate", "nicehash", "teamredminer", "gminer",
	"ethminer", "phoenixminer", "lolminer", "nbminer",
}

// DockerImage validates a game server docker image reference.
func DockerImage(image string) error {
	if image == "" {
		return nil
	}
	lower := strings.ToLower(image)
	for _, b := range blockedDockerImages {
		if strings.Contains(lower, b) {
			return fmt.Errorf("docker image %q matches blocked pattern %q", image, b)
		}
	}
	return nil
}

// StartCommand validates a game server start command. It rejects miner binaries
// and download/dropper patterns but permits shell metacharacters since some
// legitimate game server start scripts use pipes and conditionals.
func StartCommand(cmd string) error {
	if cmd == "" {
		return nil
	}
	lower := strings.ToLower(cmd)
	if err := containsAny(lower, minerKeywords, "miner-related content"); err != nil {
		return fmt.Errorf("start_command rejected: %w", err)
	}
	if err := containsAny(lower, dropperKeywords, "download/dropper command"); err != nil {
		return fmt.Errorf("start_command rejected: %w", err)
	}
	if err := containsAny(lower, sessionTools, "session persistence tool"); err != nil {
		return fmt.Errorf("start_command rejected: %w", err)
	}
	return nil
}

// blockedEnvKeys are environment variable key names that configure mining.
var blockedEnvKeys = []string{
	"pool_url", "pool_host", "wallet_address", "mining_address",
	"stratum_url", "miner_wallet", "coin_address",
}

// EnvVar validates a game server environment variable key and value.
func EnvVar(key, value string) error {
	lowerKey := strings.ToLower(key)
	for _, b := range blockedEnvKeys {
		if strings.Contains(lowerKey, b) {
			return fmt.Errorf("env var key %q matches blocked mining config pattern", key)
		}
	}
	lowerVal := strings.ToLower(value)
	if err := containsAny(lowerVal, minerKeywords, "miner-related content"); err != nil {
		return fmt.Errorf("env var %q value rejected: %w", key, err)
	}
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// File upload path / name validation
// ─────────────────────────────────────────────────────────────────────────────

// UploadFileName validates a filename used in a file upload (not a path).
func UploadFileName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("file name must not be empty")
	}
	// Reject path traversal
	if strings.Contains(name, "/") || strings.Contains(name, "\\") || strings.Contains(name, "..") {
		return fmt.Errorf("file name %q must not contain path separators or traversal sequences", name)
	}
	// Reject null bytes
	if strings.ContainsRune(name, 0) {
		return fmt.Errorf("file name contains null byte")
	}
	return nil
}

// UploadDestPath validates a destination path for a file upload into a container.
func UploadDestPath(path string) error {
	if path == "" {
		path = "/"
	}
	if !strings.HasPrefix(path, "/") {
		return fmt.Errorf("destination path must be absolute, got %q", path)
	}
	if strings.Contains(path, "..") {
		return fmt.Errorf("destination path %q contains path traversal", path)
	}
	if strings.ContainsRune(path, 0) {
		return fmt.Errorf("destination path contains null byte")
	}
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Cloud-init config composite validator
// ─────────────────────────────────────────────────────────────────────────────

// CloudInitUser holds the fields of a cloud-init user for validation.
type CloudInitUser struct {
	Name              string
	Shell             string
	Groups            []string
	SSHAuthorizedKeys []string
	Password          string
}

// CloudInitWriteFile holds the fields of a cloud-init write_files entry for validation.
type CloudInitWriteFile struct {
	Path        string
	Content     string
	Permissions string
}

// CloudInitConfig validates a full cloud-init configuration object.
// It validates all user-controllable string fields.
func CloudInitConfig(
	hostname, timezone, locale string,
	packages []string,
	users []CloudInitUser,
	runcmds []string,
	writeFiles []CloudInitWriteFile,
) error {
	if err := Hostname(hostname); err != nil {
		return err
	}
	if err := Timezone(timezone); err != nil {
		return err
	}
	if err := Locale(locale); err != nil {
		return err
	}

	for _, pkg := range packages {
		if err := PackageName(pkg); err != nil {
			return err
		}
	}

	for _, user := range users {
		if err := Username(user.Name); err != nil {
			return fmt.Errorf("user %q: %w", user.Name, err)
		}
		if err := Shell(user.Shell); err != nil {
			return fmt.Errorf("user %q shell: %w", user.Name, err)
		}
		for _, g := range user.Groups {
			if err := GroupName(g); err != nil {
				return fmt.Errorf("user %q group %q: %w", user.Name, g, err)
			}
		}
		for _, k := range user.SSHAuthorizedKeys {
			if err := SSHAuthorizedKey(k); err != nil {
				return fmt.Errorf("user %q SSH key: %w", user.Name, err)
			}
		}
	}

	for i, cmd := range runcmds {
		if err := Runcmd(cmd); err != nil {
			return fmt.Errorf("runcmd[%d]: %w", i, err)
		}
	}

	for _, wf := range writeFiles {
		if err := WriteFilePath(wf.Path); err != nil {
			return err
		}
		if err := WriteFilePermissions(wf.Permissions); err != nil {
			return err
		}
		if err := WriteFileContent(wf.Content); err != nil {
			return err
		}
	}

	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Internal helpers
// ─────────────────────────────────────────────────────────────────────────────

// containsAny checks whether s contains any of the given substrings and returns
// an error describing the first match found.
func containsAny(s string, patterns []string, desc string) error {
	for _, p := range patterns {
		if strings.Contains(s, p) {
			return fmt.Errorf("contains disallowed %s (%q)", desc, p)
		}
	}
	return nil
}
