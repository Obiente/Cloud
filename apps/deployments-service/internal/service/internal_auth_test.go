package deployments

import (
	"context"
	"testing"

	"github.com/obiente/cloud/apps/shared/pkg/auth"

	deploymentsv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/deployments/v1"
	deploymentsv1connect "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/deployments/v1/deploymentsv1connect"

	"connectrpc.com/connect"
)

func TestInternalServiceAuthInterceptorSetsSystemPrincipal(t *testing.T) {
	interceptor := NewInternalServiceAuthInterceptor("shared-secret")

	ctx, err := interceptor.authenticateForwardedTrigger(
		context.Background(),
		"shared-secret",
		deploymentsv1connect.DeploymentServiceTriggerDeploymentProcedure,
		"node-two",
	)
	if err != nil {
		t.Fatalf("authenticate internal request: %v", err)
	}
	user, err := auth.GetUserFromContext(ctx)
	if err != nil {
		t.Fatalf("get internal principal: %v", err)
	}
	if user.Id != "system" {
		t.Fatalf("internal principal = %q, want system", user.Id)
	}
}

func TestInternalServiceAuthInterceptorRejectsInvalidSecret(t *testing.T) {
	interceptor := NewInternalServiceAuthInterceptor("shared-secret")

	_, err := interceptor.authenticateForwardedTrigger(
		context.Background(),
		"wrong-secret",
		deploymentsv1connect.DeploymentServiceTriggerDeploymentProcedure,
		"node-two",
	)
	if connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("error code = %s, want %s", connect.CodeOf(err), connect.CodeUnauthenticated)
	}
}

func TestInternalServiceAuthInterceptorRejectsOtherProcedures(t *testing.T) {
	interceptor := NewInternalServiceAuthInterceptor("shared-secret")

	_, err := interceptor.authenticateForwardedTrigger(
		context.Background(),
		"shared-secret",
		deploymentsv1connect.DeploymentServiceUpdateDeploymentProcedure,
		"node-two",
	)
	if connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Fatalf("error code = %s, want %s", connect.CodeOf(err), connect.CodePermissionDenied)
	}
}

func TestInternalServiceAuthInterceptorRequiresForwardingTarget(t *testing.T) {
	interceptor := NewInternalServiceAuthInterceptor("shared-secret")

	_, err := interceptor.authenticateForwardedTrigger(
		context.Background(),
		"shared-secret",
		deploymentsv1connect.DeploymentServiceTriggerDeploymentProcedure,
		"",
	)
	if connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Fatalf("error code = %s, want %s", connect.CodeOf(err), connect.CodePermissionDenied)
	}
}

func TestInternalServiceAuthInterceptorAllowsForwardedDeletion(t *testing.T) {
	interceptor := NewInternalServiceAuthInterceptor("shared-secret")
	ctx, err := interceptor.authenticateForwardedTrigger(
		context.Background(),
		"shared-secret",
		deploymentsv1connect.DeploymentServiceDeleteDeploymentProcedure,
		"node-two",
	)
	if err != nil {
		t.Fatalf("authenticate forwarded deletion: %v", err)
	}
	user, err := auth.GetUserFromContext(ctx)
	if err != nil || user == nil || user.Id != "system" {
		t.Fatalf("forwarded deletion did not receive system identity: user=%#v err=%v", user, err)
	}
}

func TestTriggerDeploymentForwardHeadersAuthenticatesSystemCall(t *testing.T) {
	t.Setenv(deploymentsInternalServiceSecretEnv, "shared-secret")
	req := connect.NewRequest(&deploymentsv1.TriggerDeploymentRequest{})

	headers, err := triggerDeploymentForwardHeaders(auth.WithSystemUser(context.Background()), req, "node-two")
	if err != nil {
		t.Fatalf("build forwarding headers: %v", err)
	}
	if headers[internalServiceSecretHeader] != "shared-secret" {
		t.Fatalf("internal secret header = %q", headers[internalServiceSecretHeader])
	}
	if headers["X-Obiente-Target-Node"] != "node-two" {
		t.Fatalf("target node header = %q", headers["X-Obiente-Target-Node"])
	}
}
