package deployments

import (
	"context"
	"strings"
	"testing"

	"github.com/obiente/cloud/apps/shared/pkg/auth"
	"github.com/obiente/cloud/apps/shared/pkg/database"

	deploymentsv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/deployments/v1"

	"connectrpc.com/connect"
)

func TestRequestedDeploymentCommitSHARequiresSystemPrincipal(t *testing.T) {
	commitSHA := strings.Repeat("a", 40)

	_, err := requestedDeploymentCommitSHA(context.Background(), commitSHA)
	if connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Fatalf("error code = %s, want %s", connect.CodeOf(err), connect.CodePermissionDenied)
	}

	got, err := requestedDeploymentCommitSHA(auth.WithSystemUser(context.Background()), commitSHA)
	if err != nil {
		t.Fatalf("trusted commit override: %v", err)
	}
	if got != commitSHA {
		t.Fatalf("commit override = %q, want %q", got, commitSHA)
	}
}

func TestRequestedDeploymentCommitSHARejectsInvalidSHA(t *testing.T) {
	_, err := requestedDeploymentCommitSHA(auth.WithSystemUser(context.Background()), "not-a-commit")
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("error code = %s, want %s", connect.CodeOf(err), connect.CodeInvalidArgument)
	}
}

func TestShouldDeployStoredComposeOnlyForPlainCompose(t *testing.T) {
	tests := []struct {
		name       string
		strategy   deploymentsv1.BuildStrategy
		composeYML string
		want       bool
	}{
		{name: "plain compose", strategy: deploymentsv1.BuildStrategy_PLAIN_COMPOSE, composeYML: "services: {}", want: true},
		{name: "repository compose", strategy: deploymentsv1.BuildStrategy_COMPOSE_REPO, composeYML: "services: {}", want: false},
		{name: "empty plain compose", strategy: deploymentsv1.BuildStrategy_PLAIN_COMPOSE, composeYML: "", want: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			deployment := &database.Deployment{
				BuildStrategy: int32(test.strategy),
				ComposeYaml:   test.composeYML,
			}
			if got := shouldDeployStoredCompose(deployment); got != test.want {
				t.Fatalf("shouldDeployStoredCompose() = %t, want %t", got, test.want)
			}
		})
	}
}
