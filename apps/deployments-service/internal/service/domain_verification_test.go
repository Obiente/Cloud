package deployments

import "testing"

func TestVerificationTXTDomain(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		domain string
		want   string
	}{
		{name: "exact domain", domain: "example.com", want: "example.com"},
		{name: "wildcard domain", domain: "*.example.com", want: "example.com"},
		{name: "wildcard domain normalized", domain: "  *.Example.Com. ", want: "example.com"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := verificationTXTDomain(tc.domain); got != tc.want {
				t.Fatalf("verificationTXTDomain(%q) = %q, want %q", tc.domain, got, tc.want)
			}
		})
	}
}

func TestCustomDomainsConflict(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		left  string
		right string
		want  bool
	}{
		{name: "exact match", left: "example.com", right: "example.com", want: true},
		{name: "wildcard covers one subdomain", left: "*.example.com", right: "foo.example.com", want: true},
		{name: "wildcard does not cover apex", left: "*.example.com", right: "example.com", want: false},
		{name: "wildcard does not cover nested subdomain", left: "*.example.com", right: "foo.bar.example.com", want: false},
		{name: "different domains", left: "*.example.net", right: "foo.example.com", want: false},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := customDomainsConflict(tc.left, tc.right); got != tc.want {
				t.Fatalf("customDomainsConflict(%q, %q) = %v, want %v", tc.left, tc.right, got, tc.want)
			}
		})
	}
}
