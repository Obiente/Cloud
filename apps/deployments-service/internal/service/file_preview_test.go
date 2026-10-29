package deployments

import (
	"bytes"
	"encoding/base64"
	"testing"

	deploymentsv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/deployments/v1"
	"google.golang.org/protobuf/proto"
)

func TestContainerFilePreviewEncoding(t *testing.T) {
	for _, test := range []struct {
		name     string
		data     []byte
		encoding string
	}{
		{"empty", nil, "text"},
		{"unicode", []byte("saved: café"), "text"},
		{"binary", []byte{0xff, 0x80, 0, 1}, "base64"},
	} {
		t.Run(test.name, func(t *testing.T) {
			metadata := &deploymentsv1.ContainerFile{Name: "record", Path: "/record", VolumeName: proto.String("data")}
			response := containerFileResponse(test.data, metadata)
			if response.Size != int64(len(test.data)) || response.Encoding != test.encoding || response.Metadata != metadata || response.GetTruncated() {
				t.Fatalf("wrong preview contract: %v", response)
			}
			decoded := []byte(response.Content)
			if response.Encoding == "base64" {
				var err error
				decoded, err = base64.StdEncoding.DecodeString(response.Content)
				if err != nil {
					t.Fatal(err)
				}
			}
			if !bytes.Equal(decoded, test.data) {
				t.Fatalf("file bytes changed: %v", decoded)
			}
		})
	}
}
