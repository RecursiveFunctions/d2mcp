package remote

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"time"
)

const (
	DefaultMaxRedirects     = 5
	DefaultMaxResponseBytes = 10 << 20
	DefaultRequestTimeout   = 30 * time.Second
	DefaultDialTimeout      = 5 * time.Second
	DefaultTLSHandshake     = 5 * time.Second
	DefaultResponseHeader   = 10 * time.Second
	DefaultIdleConnTimeout  = 30 * time.Second
	DefaultExpectContinue   = time.Second
)

var (
	ErrHTTPSRequired     = errors.New("only HTTPS URLs are allowed")
	ErrUnsafeDestination = errors.New("destination address is not public")
	ErrTooManyRedirects  = errors.New("redirect limit exceeded")
	ErrResponseTooLarge  = errors.New("response body exceeds configured limit")
)

// Resolver is the DNS operation required by Policy.
type Resolver interface {
	LookupNetIP(ctx context.Context, network, host string) ([]netip.Addr, error)
}

// DialContextFunc is compatible with net.Dialer.DialContext.
type DialContextFunc func(ctx context.Context, network, address string) (net.Conn, error)

// Config controls outbound HTTPS policy limits. Zero values select secure defaults.
type Config struct {
	MaxRedirects          int
	MaxResponseBytes      int64
	RequestTimeout        time.Duration
	DialTimeout           time.Duration
	TLSHandshakeTimeout   time.Duration
	ResponseHeaderTimeout time.Duration
	IdleConnTimeout       time.Duration
	ExpectContinueTimeout time.Duration
	Resolver              Resolver
	DialContext           DialContextFunc
}

// Policy validates remote asset requests and owns its security-configured transport.
type Policy struct {
	resolver         Resolver
	dial             DialContextFunc
	maxRedirects     int
	maxResponseBytes int64
	requestTimeout   time.Duration
	transport        *http.Transport
}

// NewPolicy builds a credentialless HTTPS client policy with DNS-aware destination checks.
func NewPolicy(config Config) (*Policy, error) {
	if config.MaxRedirects < 0 {
		return nil, errors.New("max redirects cannot be negative")
	}
	if config.MaxResponseBytes < 0 {
		return nil, errors.New("max response bytes cannot be negative")
	}
	for name, value := range map[string]time.Duration{
		"request timeout":         config.RequestTimeout,
		"dial timeout":            config.DialTimeout,
		"TLS handshake timeout":   config.TLSHandshakeTimeout,
		"response header timeout": config.ResponseHeaderTimeout,
		"idle connection timeout": config.IdleConnTimeout,
		"expect continue timeout": config.ExpectContinueTimeout,
	} {
		if value < 0 {
			return nil, fmt.Errorf("%s cannot be negative", name)
		}
	}

	maxRedirects := config.MaxRedirects
	if maxRedirects == 0 {
		maxRedirects = DefaultMaxRedirects
	}
	maxResponseBytes := config.MaxResponseBytes
	if maxResponseBytes == 0 {
		maxResponseBytes = DefaultMaxResponseBytes
	}
	requestTimeout := durationOr(config.RequestTimeout, DefaultRequestTimeout)
	dialTimeout := durationOr(config.DialTimeout, DefaultDialTimeout)
	tlsHandshakeTimeout := durationOr(config.TLSHandshakeTimeout, DefaultTLSHandshake)
	responseHeaderTimeout := durationOr(config.ResponseHeaderTimeout, DefaultResponseHeader)
	idleConnTimeout := durationOr(config.IdleConnTimeout, DefaultIdleConnTimeout)
	expectContinueTimeout := durationOr(config.ExpectContinueTimeout, DefaultExpectContinue)

	resolver := config.Resolver
	if resolver == nil {
		resolver = net.DefaultResolver
	}
	dial := config.DialContext
	if dial == nil {
		dial = (&net.Dialer{Timeout: dialTimeout, KeepAlive: 30 * time.Second}).DialContext
	}

	policy := &Policy{
		resolver:         resolver,
		dial:             dial,
		maxRedirects:     maxRedirects,
		maxResponseBytes: maxResponseBytes,
		requestTimeout:   requestTimeout,
	}
	policy.transport = &http.Transport{
		Proxy:                 nil,
		DialContext:           policy.dialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          20,
		MaxIdleConnsPerHost:   2,
		IdleConnTimeout:       idleConnTimeout,
		TLSHandshakeTimeout:   tlsHandshakeTimeout,
		ResponseHeaderTimeout: responseHeaderTimeout,
		ExpectContinueTimeout: expectContinueTimeout,
	}
	return policy, nil
}

// NewClient is a convenience constructor for NewPolicy(config).Client().
func NewClient(config Config) (*http.Client, error) {
	policy, err := NewPolicy(config)
	if err != nil {
		return nil, err
	}
	return policy.Client(), nil
}

// Client returns an HTTP client using this policy. The underlying transport is safe for concurrent use.
func (p *Policy) Client() *http.Client {
	return &http.Client{
		Transport: &policyRoundTripper{
			policy: p,
			next:   p.transport,
			limit:  p.maxResponseBytes,
		},
		CheckRedirect: p.checkRedirect,
		Timeout:       p.requestTimeout,
	}
}

// ValidateURL resolves and validates an HTTPS URL without sending a request.
func (p *Policy) ValidateURL(ctx context.Context, target *url.URL) error {
	if target == nil || !strings.EqualFold(target.Scheme, "https") {
		return ErrHTTPSRequired
	}
	if target.Hostname() == "" {
		return errors.New("URL hostname is required")
	}
	_, err := p.resolveAllowed(ctx, target.Hostname())
	return err
}

// CloseIdleConnections closes connections retained by this policy's transport.
func (p *Policy) CloseIdleConnections() {
	p.transport.CloseIdleConnections()
}

// IsAllowedAddress reports whether an address is suitable for a remote HTTPS asset.
func IsAllowedAddress(address netip.Addr) bool {
	if !address.IsValid() {
		return false
	}
	address = address.Unmap()
	if address.IsUnspecified() || address.IsLoopback() || address.IsPrivate() ||
		address.IsLinkLocalUnicast() || address.IsLinkLocalMulticast() || address.IsMulticast() {
		return false
	}
	// RFC 6598 shared carrier space is non-public even though netip does not classify it as private.
	if netip.MustParsePrefix("100.64.0.0/10").Contains(address) {
		return false
	}
	return address.IsGlobalUnicast()
}

func (p *Policy) checkRedirect(request *http.Request, via []*http.Request) error {
	if len(via) > p.maxRedirects {
		return ErrTooManyRedirects
	}
	stripCredentials(request)
	if err := p.ValidateURL(request.Context(), request.URL); err != nil {
		return fmt.Errorf("redirect rejected: %w", err)
	}
	return nil
}

func (p *Policy) dialContext(ctx context.Context, network, address string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, fmt.Errorf("parse dial address: %w", err)
	}
	addresses, err := p.resolveAllowed(ctx, host)
	if err != nil {
		return nil, err
	}

	var dialErrors []error
	for _, destination := range addresses {
		if strings.HasSuffix(network, "4") && !destination.Is4() {
			continue
		}
		if strings.HasSuffix(network, "6") && !destination.Is6() {
			continue
		}
		connection, dialErr := p.dial(ctx, network, net.JoinHostPort(destination.String(), port))
		if dialErr == nil {
			return connection, nil
		}
		dialErrors = append(dialErrors, dialErr)
	}
	if len(dialErrors) == 0 {
		return nil, errors.New("no destination address matched the requested network")
	}
	return nil, fmt.Errorf("dial remote destination: %w", errors.Join(dialErrors...))
}

func (p *Policy) resolveAllowed(ctx context.Context, host string) ([]netip.Addr, error) {
	if parsed, err := netip.ParseAddr(host); err == nil {
		parsed = parsed.Unmap()
		if !IsAllowedAddress(parsed) {
			return nil, fmt.Errorf("%w: %s", ErrUnsafeDestination, parsed)
		}
		return []netip.Addr{parsed}, nil
	}

	addresses, err := p.resolver.LookupNetIP(ctx, "ip", host)
	if err != nil {
		return nil, fmt.Errorf("resolve destination %q: %w", host, err)
	}
	if len(addresses) == 0 {
		return nil, fmt.Errorf("resolve destination %q: no addresses", host)
	}
	allowed := make([]netip.Addr, 0, len(addresses))
	for _, address := range addresses {
		address = address.Unmap()
		if !IsAllowedAddress(address) {
			return nil, fmt.Errorf("%w: %s resolves to %s", ErrUnsafeDestination, host, address)
		}
		allowed = append(allowed, address)
	}
	return allowed, nil
}

type policyRoundTripper struct {
	policy *Policy
	next   http.RoundTripper
	limit  int64
}

func (transport *policyRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	if err := transport.policy.ValidateURL(request.Context(), request.URL); err != nil {
		return nil, err
	}
	request = request.Clone(request.Context())
	request.URL = cloneURL(request.URL)
	stripCredentials(request)

	response, err := transport.next.RoundTrip(request)
	if err != nil {
		return nil, err
	}
	if response.Body != nil {
		response.Body = &boundedBody{body: response.Body, remaining: transport.limit}
	}
	return response, nil
}

func stripCredentials(request *http.Request) {
	request.Header.Del("Authorization")
	request.Header.Del("Proxy-Authorization")
	request.Header.Del("Cookie")
	request.URL = cloneURL(request.URL)
	request.URL.User = nil
}

func cloneURL(value *url.URL) *url.URL {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

type boundedBody struct {
	body      io.ReadCloser
	remaining int64
}

func (body *boundedBody) Read(buffer []byte) (int, error) {
	if len(buffer) == 0 {
		return 0, nil
	}
	if body.remaining == 0 {
		var probe [1]byte
		n, err := body.body.Read(probe[:])
		if n > 0 {
			return 0, ErrResponseTooLarge
		}
		return 0, err
	}
	if int64(len(buffer)) > body.remaining {
		buffer = buffer[:body.remaining]
	}
	n, err := body.body.Read(buffer)
	body.remaining -= int64(n)
	return n, err
}

func (body *boundedBody) Close() error {
	return body.body.Close()
}

func durationOr(value, fallback time.Duration) time.Duration {
	if value == 0 {
		return fallback
	}
	return value
}
