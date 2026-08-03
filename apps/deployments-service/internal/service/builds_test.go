package deployments

import "testing"

func TestInvalidBuildID(t *testing.T) {
	t.Parallel()

	for _, buildID := range []string{"", " ", "undefined", "UNDEFINED", "null"} {
		if !invalidBuildID(buildID) {
			t.Errorf("invalidBuildID(%q) = false, want true", buildID)
		}
	}
	for _, buildID := range []string{"build-123", "01JBUILD"} {
		if invalidBuildID(buildID) {
			t.Errorf("invalidBuildID(%q) = true, want false", buildID)
		}
	}
}
