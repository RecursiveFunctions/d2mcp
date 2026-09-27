# Distribution Pipeline

`server.json` is the canonical source for the server name, version, repository, OCI image, transport, and workspace mount. `distribution/targets.yaml` contains stable client-specific metadata. Do not hand-edit files under `distribution/generated` or `integrations/vscode/src/release.ts`.

## Local workflow

```bash
make generate-adapters
make verify-adapters
go test -race ./...
go vet ./...

cd integrations/vscode
npm ci
npm run package
```

The companion extension uses VS Code's MCP server definition provider API. It launches the published Docker image and mounts each workspace at `/workspace`; it does not bundle the Go binary or modify user configuration.

## Release ordering

A protected `vX.Y.Z` tag starts `.github/workflows/release.yml`:

1. Verify the tag, `server.json`, runtime version, OCI tag, generated adapters, and extension manifest agree.
2. Test the Go server and package the companion VSIX.
3. Publish the immutable `linux/amd64` and `linux/arm64` image to GHCR.
4. Publish and verify the Official MCP Registry entry.
5. Publish the identical VSIX to configured extension marketplaces.
6. Create a GitHub Release containing the VSIX, adapter bundle, release metadata, and checksums.
7. Open or update supported curator submissions.

Ordinary pushes and pull requests validate artifacts but never publish them. Version changes are intentional and manual; the pipeline does not choose or bump versions.

## Repository configuration

Marketplace jobs safely skip destinations whose credentials are absent.

| Name | Kind | Purpose |
|---|---|---|
| `OVSX_PAT` | Secret | Publish `RecursiveFunctions.d2mcp` to Open VSX |
| `VSCE_PAT` | Secret | Publish the same VSIX to Visual Studio Marketplace |
| `MARKETPLACE_GITHUB_TOKEN` | Secret | Push to the bot-owned Kilo fork and create the Cline submission issue |
| `KILO_FORK` | Variable | Bot-owned fork in `owner/kilo-marketplace` form |
| `CLINE_LOGO_URL` | Variable | Public URL for Cline's required 400x400 PNG |

Before adding `OVSX_PAT`, create or claim the `RecursiveFunctions` Open VSX namespace, connect an Eclipse account, and sign the Publisher Agreement. Create the matching Visual Studio Marketplace publisher before adding `VSCE_PAT`.

Use narrowly scoped credentials. For stronger controls, move the secrets into protected `extension-marketplaces` and `upstream-catalogs` environments and add those environment names to the corresponding workflow jobs.

## Curated destinations

- Kilo accepts pull requests. The workflow maintains the `automation/d2mcp-release` branch in the configured fork and opens one upstream PR.
- Cline accepts submission issues, not catalog pull requests. The workflow creates the issue only when `CLINE_LOGO_URL` is configured and no matching issue already exists.
- Cursor, Windsurf, Trae, and Antigravity do not document public marketplace publishing APIs. Their generated configurations and install links ship in each GitHub Release.
- Zed has announced migration from context-server extensions to the Official MCP Registry. The pipeline publishes the Registry entry and a manual adapter instead of maintaining a temporary Rust extension.

## Recovery

Never move or rebuild an existing OCI version tag. If a downstream publication fails, verify the existing GHCR digest and Official Registry record, then rerun only the missing downstream command using the VSIX and adapter bundle from the GitHub Actions artifacts. A recovery must fail if any existing artifact has a different checksum or digest.

## Adding a client

1. Add its stable workspace expression and metadata to `distribution/targets.yaml`.
2. Add a structured renderer in `internal/distribution`.
3. Add round-trip and immutable-image assertions.
4. Regenerate committed outputs.
5. Add the client to the README table and scheduled compatibility checks.

Do not automate private or undocumented marketplace endpoints.