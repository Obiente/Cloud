package deployments

import (
	"testing"

	deploymentsv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/deployments/v1"
)

func TestValidateUserManagedEnvironmentRejectsPullRequest(t *testing.T) {
	for _, environment := range []deploymentsv1.Environment{
		deploymentsv1.Environment_PRODUCTION,
		deploymentsv1.Environment_STAGING,
		deploymentsv1.Environment_DEVELOPMENT,
	} {
		if err := validateUserManagedEnvironment(environment); err != nil {
			t.Fatalf("expected environment %s to be user-managed: %v", environment, err)
		}
	}

	if err := validateUserManagedEnvironment(deploymentsv1.Environment_PULL_REQUEST); err == nil {
		t.Fatal("pull request environment must only be assigned by preview automation")
	}
	if err := validateUserManagedEnvironment(deploymentsv1.Environment(99)); err == nil {
		t.Fatal("unknown environment must be rejected")
	}
}
