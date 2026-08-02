package deployments

import (
	"context"
	"testing"

	"github.com/obiente/cloud/apps/shared/pkg/auth"

	deploymentsv1 "github.com/obiente/cloud/apps/shared/proto/obiente/cloud/deployments/v1"

	"connectrpc.com/connect"
)

func TestInternalServiceAuthInterceptorSetsSystemPrincipal(t *testing.T) {
	interceptor := NewInternalServiceAuthInterceptor("shared-secret")
	called := false
	next := func(ctx context.Context, _ connect.AnyRequest) (connect.AnyResponse, error) {
		called = true
		user, err := auth.GetUserFromContext(ctx)
		if err != nil {
			t.Fatalf("get internal principal: %v", err)
		}
		if user.Id != "system" {
			t.Fatalf("internal principal = %q, want system", user.Id)
		}
		return connect.NewResponse(&deploymentsv1.TriggerDeploymentResponse{}), nil
	}

	req := connect.NewRequest(&deploymentsv1.TriggerDeploymentRequest{})
	req.Header().Set(internalServiceSecretHeader, "shared-secret")
	if _, err := interceptor.WrapUnary(next)(context.Background(), req); err != nil {
		t.Fatalf("authenticate internal request: %v", err)
	}
	if !called {
		t.Fatal("authenticated internal request did not reach handler")
	}
}

func TestInternalServiceAuthInterceptorRejectsInvalidSecret(t *testing.T) {
	interceptor := NewInternalServiceAuthInterceptor("shared-secret")
	req := connect.NewRequest(&deploymentsv1.TriggerDeploymentRequest{})
	req.Header().Set(internalServiceSecretHeader, "wrong-secret")

	_, err := interceptor.WrapUnary(func(context.Context, connect.AnyRequest) (connect.AnyResponse, error) {
		t.Fatal("invalid internal request reached handler")
		return nil, nil
	})(context.Background(), req)
	if connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("error code = %s, want %s", connect.CodeOf(err), connect.CodeUnauthenticated)
	}
}

func TestTriggerDeploymentForwardHeadersAuthenticatesSystemCall(t *testing.T) {
	t.Setenv("INTERNAL_SERVICE_SECRET", "shared-secret")
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
