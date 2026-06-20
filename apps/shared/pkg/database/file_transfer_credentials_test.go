package database

import "testing"

func TestNormalizeFileTransferScopes(t *testing.T) {
	got := NormalizeFileTransferScopes(" sftp:read, sftp:write, rw, read ")
	if got != "read,write" {
		t.Fatalf("NormalizeFileTransferScopes() = %q, want read,write", got)
	}
}

func TestFileTransferCredentialHasScope(t *testing.T) {
	if !FileTransferCredentialHasScope("sftp:*", FileTransferScopeRead) {
		t.Fatal("sftp:* should include read")
	}
	if !FileTransferCredentialHasScope("sftp:*", FileTransferScopeWrite) {
		t.Fatal("sftp:* should include write")
	}
	if FileTransferCredentialHasScope("read", FileTransferScopeWrite) {
		t.Fatal("read should not include write")
	}
}

func TestNormalizeFileTransferResourceType(t *testing.T) {
	if got := NormalizeFileTransferResourceType("game-server"); got != FileTransferResourceGameServer {
		t.Fatalf("NormalizeFileTransferResourceType() = %q", got)
	}
}
