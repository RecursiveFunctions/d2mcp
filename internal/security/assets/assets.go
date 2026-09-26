package assets

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"html"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/i2y/d2mcp/internal/security/workspace"
)

const DefaultMaxBytes int64 = 10 << 20

var (
	ErrUnsupportedAsset = errors.New("unsupported asset URL")
	ErrAssetTooLarge    = errors.New("asset exceeds configured limit")
)

var (
	imageTagPattern = regexp.MustCompile(`(?is)<image\b[^>]*>`)
	hrefPattern     = regexp.MustCompile(`(?i)\b(?:href|xlink:href)\s*=\s*"([^"]*)"`)
)

// Resolver confines local assets to workspace roots and fetches remote assets through an injected client.
type Resolver struct {
	workspaces *workspace.Roots
	client     *http.Client
	maxBytes   int64
}

func New(workspaces *workspace.Roots, client *http.Client, maxBytes int64) (*Resolver, error) {
	if workspaces == nil {
		return nil, errors.New("workspace roots are required")
	}
	if client == nil {
		return nil, errors.New("HTTP client is required")
	}
	if maxBytes < 0 {
		return nil, errors.New("max asset bytes cannot be negative")
	}
	if maxBytes == 0 {
		maxBytes = DefaultMaxBytes
	}
	return &Resolver{workspaces: workspaces, client: client, maxBytes: maxBytes}, nil
}

// DataURI resolves one local or remote image into a bounded data URI.
func (resolver *Resolver) DataURI(ctx context.Context, rootName, reference string) (string, error) {
	return resolver.resolve(ctx, rootName, reference)
}

// EmbedSVGImages replaces generated SVG image references with bounded data URIs.
func (resolver *Resolver) EmbedSVGImages(ctx context.Context, rootName string, source []byte) ([]byte, error) {
	matches := imageTagPattern.FindAllIndex(source, -1)
	if len(matches) == 0 {
		return source, nil
	}

	var output bytes.Buffer
	start := 0
	for _, match := range matches {
		output.Write(source[start:match[0]])
		tag := source[match[0]:match[1]]
		href := hrefPattern.FindSubmatchIndex(tag)
		if href == nil {
			output.Write(tag)
			start = match[1]
			continue
		}

		value := html.UnescapeString(string(tag[href[2]:href[3]]))
		dataURI, err := resolver.resolve(ctx, rootName, value)
		if err != nil {
			return nil, fmt.Errorf("resolve SVG image %q: %w", value, err)
		}
		output.Write(tag[:href[2]])
		output.WriteString(dataURI)
		output.Write(tag[href[3]:])
		start = match[1]
	}
	output.Write(source[start:])
	return output.Bytes(), nil
}

func (resolver *Resolver) resolve(ctx context.Context, rootName, reference string) (string, error) {
	if strings.HasPrefix(strings.ToLower(reference), "data:") {
		if !strings.HasPrefix(strings.ToLower(reference), "data:image/") {
			return "", ErrUnsupportedAsset
		}
		if int64(len(reference)) > resolver.maxBytes*2+1024 {
			return "", ErrAssetTooLarge
		}
		return reference, nil
	}
	if strings.HasPrefix(reference, "#") {
		return reference, nil
	}

	parsed, err := url.Parse(reference)
	if err != nil {
		return "", fmt.Errorf("parse asset URL: %w", err)
	}
	switch strings.ToLower(parsed.Scheme) {
	case "https":
		return resolver.fetchRemote(ctx, parsed)
	case "http":
		return "", ErrUnsupportedAsset
	case "file":
		if parsed.Host != "" || parsed.RawQuery != "" || parsed.Fragment != "" {
			return "", ErrUnsupportedAsset
		}
		return resolver.readLocal(rootName, parsed.Path)
	case "":
		if parsed.Host != "" || parsed.RawQuery != "" || parsed.Fragment != "" {
			return "", ErrUnsupportedAsset
		}
		return resolver.readLocal(rootName, parsed.Path)
	default:
		return "", ErrUnsupportedAsset
	}
}

func (resolver *Resolver) fetchRemote(ctx context.Context, target *url.URL) (string, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, target.String(), nil)
	if err != nil {
		return "", fmt.Errorf("create asset request: %w", err)
	}
	response, err := resolver.client.Do(request)
	if err != nil {
		return "", fmt.Errorf("fetch remote asset: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return "", fmt.Errorf("fetch remote asset: unexpected status %s", response.Status)
	}
	data, err := readBounded(response.Body, resolver.maxBytes)
	if err != nil {
		return "", err
	}
	mediaType, err := imageMediaType(data, target.Path, response.Header.Get("Content-Type"))
	if err != nil {
		return "", err
	}
	return dataURI(mediaType, data), nil
}

func (resolver *Resolver) readLocal(rootName, path string) (string, error) {
	resolved, err := resolver.workspaces.ResolveRead(rootName, filepath.FromSlash(path))
	if err != nil {
		return "", err
	}
	file, err := os.Open(resolved)
	if err != nil {
		return "", fmt.Errorf("open local asset: %w", err)
	}
	defer file.Close()
	data, err := readBounded(file, resolver.maxBytes)
	if err != nil {
		return "", err
	}
	mediaType, err := imageMediaType(data, resolved, "")
	if err != nil {
		return "", err
	}
	return dataURI(mediaType, data), nil
}

func readBounded(reader io.Reader, limit int64) ([]byte, error) {
	limited := &io.LimitedReader{R: reader, N: limit + 1}
	data, err := io.ReadAll(limited)
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, ErrAssetTooLarge
	}
	return data, nil
}

func imageMediaType(data []byte, path, header string) (string, error) {
	if header != "" {
		mediaType, _, err := mime.ParseMediaType(header)
		if err == nil && strings.HasPrefix(strings.ToLower(mediaType), "image/") {
			return mediaType, nil
		}
	}
	if extensionType := mime.TypeByExtension(strings.ToLower(filepath.Ext(path))); extensionType != "" {
		mediaType, _, err := mime.ParseMediaType(extensionType)
		if err == nil && strings.HasPrefix(strings.ToLower(mediaType), "image/") {
			return mediaType, nil
		}
	}
	detected := http.DetectContentType(data)
	mediaType, _, err := mime.ParseMediaType(detected)
	if err != nil || !strings.HasPrefix(strings.ToLower(mediaType), "image/") {
		return "", errors.New("asset content is not an image")
	}
	return mediaType, nil
}

func dataURI(mediaType string, data []byte) string {
	return "data:" + mediaType + ";base64," + base64.StdEncoding.EncodeToString(data)
}
