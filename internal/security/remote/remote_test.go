package remote

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestIsAllowedAddress(t *testing.T) {
	blocked := []string{
		"0.0.0.0",
		"127.0.0.1",
		"10.0.0.1",
		"169.254.1.1",
		"224.0.0.1",
		"100.64.0.1",
		"::",
		"::1",
		"fc00::1",
		"fe80::1",
		"ff02::1",
		"::ffff:127.0.0.1",
	}
	for _, value := range blocked {
		if IsAllowedAddress(netip.MustParseAddr(value)) {
			t.Errorf("IsAllowedAddress(%s) = true, want false", value)
		}
	}
	for _, value := range []string{"8.8.8.8", "2606:4700:4700::1111"} {
		if !IsAllowedAddress(netip.MustParseAddr(value)) {
			t.Errorf("IsAllowedAddress(%s) = false, want true", value)
		}
	}
}

func TestPolicyRequiresHTTPSAndPublicResolution(t *testing.T) {
	policy := mustPolicy(t, Config{Resolver: fakeResolver{addresses: map[string][]netip.Addr{
		"public.example": {netip.MustParseAddr("8.8.8.8")},
		"mixed.example":  {netip.MustParseAddr("8.8.8.8"), netip.MustParseAddr("127.0.0.1")},
	}}})

	if err := policy.ValidateURL(context.Background(), mustURL(t, "http://public.example/asset")); !errors.Is(err, ErrHTTPSRequired) {
		t.Fatalf("HTTP validation error = %v, want ErrHTTPSRequired", err)
	}
	if err := policy.ValidateURL(context.Background(), mustURL(t, "https://127.0.0.1/asset")); !errors.Is(err, ErrUnsafeDestination) {
		t.Fatalf("loopback validation error = %v, want ErrUnsafeDestination", err)
	}
	if err := policy.ValidateURL(context.Background(), mustURL(t, "https://mixed.example/asset")); !errors.Is(err, ErrUnsafeDestination) {
		t.Fatalf("mixed DNS validation error = %v, want ErrUnsafeDestination", err)
	}
	if err := policy.ValidateURL(context.Background(), mustURL(t, "https://public.example/asset")); err != nil {
		t.Fatalf("public HTTPS validation error = %v", err)
	}
}

func TestDialRevalidatesDestinationAndUsesResolvedIP(t *testing.T) {
	var dialed string
	dialFailure := errors.New("dial stopped")
	policy := mustPolicy(t, Config{
		Resolver: fakeResolver{addresses: map[string][]netip.Addr{
			"public.example":  {netip.MustParseAddr("8.8.8.8")},
			"private.example": {netip.MustParseAddr("10.0.0.1")},
		}},
		DialContext: func(_ context.Context, _, address string) (net.Conn, error) {
			dialed = address
			return nil, dialFailure
		},
	})

	if _, err := policy.dialContext(context.Background(), "tcp", "private.example:443"); !errors.Is(err, ErrUnsafeDestination) {
		t.Fatalf("private dial error = %v, want ErrUnsafeDestination", err)
	}
	if dialed != "" {
		t.Fatalf("private destination reached dialer as %q", dialed)
	}
	if _, err := policy.dialContext(context.Background(), "tcp", "public.example:443"); !errors.Is(err, dialFailure) {
		t.Fatalf("public dial error = %v, want dial failure", err)
	}
	if dialed != "8.8.8.8:443" {
		t.Fatalf("dial address = %q, want %q", dialed, "8.8.8.8:443")
	}
}

func TestRedirectPolicyStripsCredentialsAndEnforcesLimits(t *testing.T) {
	policy := mustPolicy(t, Config{
		MaxRedirects: 1,
		Resolver: fakeResolver{addresses: map[string][]netip.Addr{
			"public.example":  {netip.MustParseAddr("8.8.8.8")},
			"private.example": {netip.MustParseAddr("192.168.1.2")},
		}},
	})
	redirect := &http.Request{
		URL:    mustURL(t, "https://user:password@public.example/next"),
		Header: http.Header{"Authorization": {"Bearer secret"}, "Proxy-Authorization": {"secret"}, "Cookie": {"session=secret"}},
	}
	if err := policy.checkRedirect(redirect, []*http.Request{{}}); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"Authorization", "Proxy-Authorization", "Cookie"} {
		if value := redirect.Header.Get(name); value != "" {
			t.Errorf("redirect retained %s: %q", name, value)
		}
	}
	if redirect.URL.User != nil {
		t.Error("redirect retained URL credentials")
	}

	redirect.URL = mustURL(t, "http://public.example/next")
	if err := policy.checkRedirect(redirect, []*http.Request{{}}); !errors.Is(err, ErrHTTPSRequired) {
		t.Fatalf("HTTP redirect error = %v, want ErrHTTPSRequired", err)
	}
	redirect.URL = mustURL(t, "https://private.example/next")
	if err := policy.checkRedirect(redirect, []*http.Request{{}}); !errors.Is(err, ErrUnsafeDestination) {
		t.Fatalf("private redirect error = %v, want ErrUnsafeDestination", err)
	}
	redirect.URL = mustURL(t, "https://public.example/next")
	if err := policy.checkRedirect(redirect, []*http.Request{{}, {}}); !errors.Is(err, ErrTooManyRedirects) {
		t.Fatalf("redirect limit error = %v, want ErrTooManyRedirects", err)
	}
}

func TestRoundTripperStripsCredentialsAndBoundsResponse(t *testing.T) {
	policy := mustPolicy(t, Config{Resolver: fakeResolver{addresses: map[string][]netip.Addr{
		"public.example": {netip.MustParseAddr("8.8.8.8")},
	}}})
	capture := &captureRoundTripper{body: "123456"}
	transport := &policyRoundTripper{policy: policy, next: capture, limit: 5}
	request, err := http.NewRequest(http.MethodGet, "https://user:password@public.example/asset", nil)
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Authorization", "Bearer secret")
	request.Header.Set("Proxy-Authorization", "secret")
	request.Header.Set("Cookie", "session=secret")

	response, err := transport.RoundTrip(request)
	if err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(response.Body)
	if !errors.Is(err, ErrResponseTooLarge) {
		t.Fatalf("ReadAll() error = %v, want ErrResponseTooLarge", err)
	}
	if string(body) != "12345" {
		t.Fatalf("ReadAll() = %q, want %q", body, "12345")
	}
	if capture.request.URL.User != nil {
		t.Error("transport retained URL credentials")
	}
	for _, name := range []string{"Authorization", "Proxy-Authorization", "Cookie"} {
		if value := capture.request.Header.Get(name); value != "" {
			t.Errorf("transport retained %s: %q", name, value)
		}
	}
	if request.Header.Get("Authorization") == "" || request.URL.User == nil {
		t.Error("transport mutated the caller's request")
	}
}

func TestBoundedBodyAllowsExactLimit(t *testing.T) {
	body := &boundedBody{body: io.NopCloser(strings.NewReader("12345")), remaining: 5}
	got, err := io.ReadAll(body)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "12345" {
		t.Fatalf("ReadAll() = %q, want %q", got, "12345")
	}
}

func TestClientAppliesRequestTimeout(t *testing.T) {
	client, err := NewClient(Config{
		RequestTimeout: 2 * time.Second,
		Resolver:       fakeResolver{},
	})
	if err != nil {
		t.Fatal(err)
	}
	if client.Timeout != 2*time.Second {
		t.Fatalf("client timeout = %v, want 2s", client.Timeout)
	}
	if client.Jar != nil {
		t.Fatal("client has a cookie jar")
	}
}

type fakeResolver struct {
	addresses map[string][]netip.Addr
	err       error
}

func (resolver fakeResolver) LookupNetIP(_ context.Context, _, host string) ([]netip.Addr, error) {
	if resolver.err != nil {
		return nil, resolver.err
	}
	return resolver.addresses[host], nil
}

type captureRoundTripper struct {
	request *http.Request
	body    string
}

func (transport *captureRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	transport.request = request
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(transport.body)),
		Request:    request,
	}, nil
}

func mustPolicy(t *testing.T, config Config) *Policy {
	t.Helper()
	policy, err := NewPolicy(config)
	if err != nil {
		t.Fatal(err)
	}
	return policy
}

func mustURL(t *testing.T, value string) *url.URL {
	t.Helper()
	parsed, err := url.Parse(value)
	if err != nil {
		t.Fatal(err)
	}
	return parsed
}
