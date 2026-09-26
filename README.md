# D2 MCP Server

A Model Context Protocol (MCP) server that provides D2 diagram generation and manipulation capabilities.

D2 is a modern diagram scripting language that turns text to diagrams. This MCP server allows AI assistants like Claude to create, render, export, and save D2 diagrams programmatically.

The server provides 16 MCP tools for complete-source authoring, validation, rendering, embedded D2 guidance, and incremental Oracle editing.

With the new Oracle API integration, AI assistants can now build and modify diagrams incrementally, making it perfect for:
- Converting conversations into architecture diagrams
- Building flowcharts step-by-step as requirements are discussed
- Creating entity relationship diagrams from database schemas
- Generating system diagrams from code analysis
- Refining diagrams based on user feedback without starting over

## Features

### Source-First Diagram Operations
- **d2_create** - Create diagrams from complete D2 source or start empty
- **d2_get_source** - Retrieve the canonical source, including Oracle edits
- **d2_update_source** - Validate then atomically replace complete source
- **d2_format** - Format valid source without storing it
- **d2_validate** - Validate raw or stored source and suggest conservative repairs
- **d2_export** - Export with bundled D2 v0.9 renderers (SVG, PNG, PDF, PPTX, GIF, ASCII)
- **d2_save** - Save an export inside a configured workspace root
- **d2_help** - Search or retrieve embedded D2 v0.9 guidance
- **d2_capabilities** - List the versioned capability catalog

### Oracle API for Incremental Editing
- **d2_oracle_create** - Create shapes and connections incrementally
- **d2_oracle_set** - Set attributes on existing elements
- **d2_oracle_delete** - Delete specific elements from diagrams
- **d2_oracle_move** - Move shapes between containers
- **d2_oracle_rename** - Rename diagram elements
- **d2_oracle_get_info** - Get information about shapes, connections, or containers
- **d2_oracle_serialize** - Get the current D2 text representation of the diagram

### Additional Features
- **Bundled layouts** - Dagre, ELK, and TALA
- **Typed rendering** - Layout, theme overrides, bundled fonts, scale, padding, sketch, animation, board selection, and bounded output options
- **20 themes** - Support for all D2 themes (18 light + 2 dark)

## Project Structure

```
d2mcp/
├── cmd/                  # Application entry point
├── internal/
│   ├── domain/          # Business entities and interfaces
│   │   ├── entity/      # Domain entities
│   │   └── repository/  # Repository interfaces
│   ├── usecase/         # Business logic
│   ├── infrastructure/  # External implementations
│   │   ├── d2/          # D2 library integration
│   │   └── mcp/         # MCP server implementation
│   └── presentation/    # MCP handlers
│       └── handler/     # Tool handlers
└── pkg/                 # Public packages
```

## Prerequisites

- Go 1.27 or higher
- D2 v0.9.0 (included as a Go dependency)

All export formats use bundled Go renderers; no `rsvg-convert`, ImageMagick, or Chromium installation is required.

The embedded capability catalog is based on the official [D2 language tour](https://d2lang.com/tour/intro/), [D2 API documentation](https://d2lang.com/tour/api/), and [D2 v0.9.0 release](https://github.com/d2lang/d2/releases/tag/v0.9.0).

## Installation

### From Source

```bash
# Clone the repository
git clone https://github.com/i2y/d2mcp.git
cd d2mcp

# Build the binary
make build

# Or build for all platforms
make build-all
```

### Using Go Install

```bash
go install github.com/i2y/d2mcp/cmd@latest
```

## Building

```bash
# Simple build
make build

# Run directly
make run

# Cross-platform builds
make build-all
```

## Usage

### With Claude Desktop

Add to your Claude Desktop configuration file:

**macOS**: `~/Library/Application Support/Claude/claude_desktop_config.json`
**Windows**: `%APPDATA%\Claude\claude_desktop_config.json`
**Linux**: `~/.config/Claude/claude_desktop_config.json`

**For STDIO transport (recommended for Claude Desktop):**
```json
{
  "mcpServers": {
    "d2mcp": {
      "command": "/path/to/d2mcp",
      "args": ["-transport=stdio"]
    }
  }
}
```

**For SSE transport:**
```json
{
  "mcpServers": {
    "d2mcp": {
      "command": "/path/to/d2mcp",
      "args": ["-transport=sse", "-addr=:3000"]
    }
  }
}
```

Replace `/path/to/d2mcp` with the actual path to your built binary.

### Standalone

```bash
# Run the MCP server (stdio transport)
./d2mcp -transport=stdio

# Run with SSE transport (default)
./d2mcp
# or explicitly
./d2mcp -transport=sse

# Run with Streamable HTTP transport
./d2mcp -transport=streamable
```

### Transport Options

d2mcp now supports multiple transport protocols:

#### STDIO Transport
The traditional stdio transport for direct process communication:
```bash
./d2mcp -transport=stdio
```

#### SSE Transport (Server-Sent Events)
HTTP-based transport that allows network connectivity:
```bash
# Basic SSE mode (defaults to :3000)
./d2mcp -transport=sse

# Custom configuration
./d2mcp -transport=sse \
  -addr=:8080 \
  -base-url=http://localhost:8080 \
  -base-path=/mcp \
  -keep-alive=30
```

**SSE Configuration Options:**
- `-addr`: Address to listen on (default: ":3000")
- `-base-url`: Base URL for SSE endpoints (auto-generated if not specified)
- `-base-path`: Base path for SSE endpoints (default: "/mcp")
- `-keep-alive`: Keep-alive interval in seconds (default: 30)

**SSE Endpoints:**
When running in SSE mode, the following endpoints are available:
- SSE stream: `http://localhost:3000/mcp/sse`
- Message endpoint: `http://localhost:3000/mcp/message`

#### Streamable HTTP Transport
The modern HTTP-based transport that simplifies bidirectional communication:
```bash
# Basic Streamable HTTP mode
./d2mcp -transport=streamable

# Custom configuration
./d2mcp -transport=streamable \
  -addr=:8080 \
  -endpoint-path=/mcp \
  -heartbeat-interval=30 \
  -stateless
```

**Streamable HTTP Configuration Options:**
- `-addr`: Address to listen on (default: ":3000")
- `-endpoint-path`: Endpoint path for Streamable HTTP (default: "/mcp")
- `-heartbeat-interval`: Heartbeat interval in seconds (default: 30)
- `-stateless`: Enable stateless mode (default: false)

**Streamable HTTP Endpoint:**
When running in Streamable HTTP mode, a single endpoint handles all communication:
- Endpoint: `http://localhost:3000/mcp`

### Workspace and Asset Security

The server confines imports, local images, and saved files to named workspace roots. The default root is the server's current working directory.

```bash
./d2mcp -transport=stdio \
  -workspace-root=docs=/workspace/docs \
  -workspace-root=diagrams=/workspace/diagrams \
  -remote-asset-max-bytes=10485760 \
  -remote-asset-timeout=30s
```

`-workspace-root` is repeatable. Local path traversal and symlink escapes are rejected. Remote assets require HTTPS and are subject to DNS/IP, redirect, timeout, MIME, and response-size checks; loopback and private-network destinations are rejected.

## Tools

### d2_create

Create a new diagram with optional initial content (unified approach):

**Empty diagram (for Oracle API workflow):**
```json
{
  "id": "my-diagram"
}
```

**With initial D2 content and a named workspace for imports/assets:**
```json
{
  "id": "my-diagram",
  "content": "a -> b: Hello\nserver: {shape: cylinder}",
  "workspace_root": "diagrams"
}
```

### d2_export

Export a diagram using a bundled renderer and layout engine:

```json
{
  "diagramId": "my-diagram",
  "format": "png",
  "layout": "tala",
  "themeId": 200,
  "scale": 1.5,
  "maxPixels": 33554432
}
```

Formats are `svg`, `png`, `pdf`, `pptx`, `gif`, and `ascii` (`txt` is an alias). Layouts are `dagre`, `elk`, and `tala`. Typed options cover engine spacing, themes and overrides, bundled fonts, padding, scale, sketch mode, animation, board selection, and resource limits.

PDF, PPTX, GIF, and optionally animated SVG compose bounded descendant boards. PNG and ASCII produce one board; use `boardPath` to select one from a composition.

### d2_save

Save a diagram inside a configured workspace root:

```json
{
  "diagramId": "my-diagram",
  "format": "pdf",
  "workspace_root": "diagrams",
  "path": "exports/output.pdf"
}
```

If `path` is omitted, output is written below `d2mcp_output/` in the selected root.

### d2_validate

Validate raw D2 source or the current source of a stored diagram. Supply exactly one input:

```json
{
  "content": "a: {\n  b"
}
```

```json
{
  "diagram_id": "my-diagram"
}
```

Invalid D2 returns a successful structured result with `valid: false` and compiler diagnostics. Simple, unambiguous terminator errors may also include `repaired_content`; validation never modifies a stored diagram.

### d2_get_source and d2_update_source

Complete D2 source is canonical, so every bundled v0.9 language feature can be authored without waiting for a specialized structured tool.

```json
{"diagram_id": "my-diagram"}
```

```json
{
  "diagram_id": "my-diagram",
  "content": "direction: right\nclient -> api -> database"
}
```

Updates compile first and replace atomically. Invalid drafts return structured diagnostics and leave stored source unchanged.

### d2_format

```json
{"content": "api:{db}"}
```

Returns formatted source without creating or changing a diagram.

### d2_help and d2_capabilities

```json
{"query": "sequence diagrams", "limit": 5}
```

`d2_help` searches the embedded v0.9.0 reference or returns an exact topic by ID. `d2_capabilities` lists its categories and MCP resource URIs. Embedded resources cover the core language, connections, reusable definitions, visuals, rich text, structured/sequence/grid diagrams, composition, layouts, and exports.

### Oracle API Tools

The Oracle API tools enable incremental diagram manipulation without regenerating the entire diagram. These tools are ideal for building diagrams step-by-step or making surgical edits.

#### d2_oracle_create

Create a new shape or connection:

```json
{
  "diagram_id": "my-diagram",
  "key": "server"  // Creates a shape
}
```

```json
{
  "diagram_id": "my-diagram", 
  "key": "server -> database"  // Creates a connection
}
```

#### d2_oracle_set

Set attributes on existing elements:

```json
{
  "diagram_id": "my-diagram",
  "key": "server.shape",
  "value": "cylinder"
}
```

```json
{
  "diagram_id": "my-diagram",
  "key": "server.style.fill",
  "value": "#f0f0f0"
}
```

#### d2_oracle_delete

Delete elements from the diagram:

```json
{
  "diagram_id": "my-diagram",
  "key": "server"  // Deletes the server and its children
}
```

#### d2_oracle_move

Move elements between containers:

```json
{
  "diagram_id": "my-diagram",
  "key": "server",
  "new_parent": "network.internal",  // Moves server into network.internal
  "include_descendants": "true"       // Also moves child elements
}
```

#### d2_oracle_rename

Rename diagram elements:

```json
{
  "diagram_id": "my-diagram",
  "key": "server",
  "new_name": "web_server"
}
```

#### d2_oracle_get_info

Get information about diagram elements:

```json
{
  "diagram_id": "my-diagram",
  "key": "server",
  "info_type": "object"  // Options: "object", "edge", "children"
}
```

#### d2_oracle_serialize

Get the current D2 text representation of the diagram:

```json
{
  "diagram_id": "my-diagram"
}
```

Returns the complete D2 text of the diagram including all modifications made through Oracle API.

### Creating Sequence Diagrams

D2 has built-in support for sequence diagrams. Use `d2_create` with proper D2 sequence diagram syntax:

```json
{
  "id": "api-flow",
  "content": "shape: sequence_diagram\n\nClient -> Server: HTTP Request\nServer -> Database: Query\nDatabase -> Server: Results\nServer -> Client: HTTP Response\n\n# Add styling\nClient -> Server.\"HTTP Request\": {style.stroke-dash: 3}\nDatabase -> Server.\"Results\": {style.stroke-dash: 3}"
}
```

**Example with actors and grouping:**
```json
{
  "id": "auth-flow",
  "content": "shape: sequence_diagram\n\ntitle: Authentication Flow {near: top-center}\n\n# Define actors\nClient: {shape: person}\nAuth Server: {shape: cloud}\nDatabase: {shape: cylinder}\n\n# Interactions\nClient -> Auth Server: Login Request\nAuth Server -> Database: Validate Credentials\nDatabase -> Auth Server: User Data\n\ngroup: Success Case {\n  Auth Server -> Client: Access Token\n  Client -> Auth Server: API Request + Token\n  Auth Server -> Client: API Response\n}\n\ngroup: Failure Case {\n  Auth Server -> Client: 401 Unauthorized\n}"
}
```

### Example Oracle API Workflow

**Starting from scratch:**
```javascript
// 1. Create an empty diagram
d2_create({ id: "architecture" })

// 2. Add shapes incrementally
d2_oracle_create({ diagram_id: "architecture", key: "web" })
d2_oracle_create({ diagram_id: "architecture", key: "api" })
d2_oracle_create({ diagram_id: "architecture", key: "db" })

// 3. Set properties
d2_oracle_set({ diagram_id: "architecture", key: "db.shape", value: "cylinder" })
d2_oracle_set({ diagram_id: "architecture", key: "web.label", value: "Web Server" })

// 4. Create connections
d2_oracle_create({ diagram_id: "architecture", key: "web -> api" })
d2_oracle_create({ diagram_id: "architecture", key: "api -> db" })

// 5. Export final result
d2_export({ diagramId: "architecture", format: "svg" })
```

**Starting with existing content (unified approach):**
```javascript
// 1. Create diagram with initial content
d2_create({ 
  id: "architecture",
  content: "web -> api -> db\ndb: {shape: cylinder}"
})

// 2. Enhance incrementally using Oracle API
d2_oracle_set({ diagram_id: "architecture", key: "web.label", value: "Web Server" })
d2_oracle_create({ diagram_id: "architecture", key: "cache" })
d2_oracle_create({ diagram_id: "architecture", key: "api -> cache" })

// 3. Export final result
d2_export({ diagramId: "architecture", format: "svg" })
```

### When to Use Each Tool

- **d2_create**: Create a stored source document, optionally bound to a named workspace root
- **d2_update_source**: Author or replace any complete D2 v0.9 document atomically
- **d2_oracle_***: Make convenient structured edits; each mutation is revalidated before commit
- **d2_get_source**: Retrieve the canonical source after either editing workflow
- **d2_help / d2_capabilities**: Discover supported syntax, composition, layouts, and exports
- **d2_export / d2_save**: Render in memory or save within an allowed workspace root

## Development

### Running tests

```bash
# Run all tests
make test

# Run with coverage
go test -cover ./...

# Run specific test
go test -v ./internal/presentation/handler
```

### Code Quality

```bash
# Format code
make fmt

# Run linter
make lint

# Clean build artifacts
make clean
```

### Adding new features

1. Define entities in `internal/domain/entity`
2. Add repository interfaces in `internal/domain/repository`
3. Implement business logic in `internal/usecase`
4. Add infrastructure implementations
5. Create MCP handlers in `internal/presentation/handler`
6. Wire dependencies in `cmd/main.go`

### Project Structure

- **cmd/**: Application entry point
- **internal/domain/**: Core business logic and entities
- **internal/infrastructure/**: External service integrations
- **internal/presentation/**: MCP protocol handlers
- **internal/usecase/**: Application business logic

## Troubleshooting

### Export Limit Errors

PNG, PDF, PPTX, and GIF exports are produced by D2's bundled Go raster pipeline. If a diagram exceeds a resource ceiling, reduce its scale or board count, select a specific `boardPath`, or set an allowed bounded `maxWidth`, `maxHeight`, `maxPixels`, `maxFrames`, or `maxOutputBytes` value.

### MCP Connection Issues

1. Ensure the binary has execute permissions: `chmod +x d2mcp`
2. Check Claude Desktop logs for error messages
3. Verify the path in your configuration is absolute

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## Changelog

### v0.5.0 (Latest)
- Added SSE (Server-Sent Events) transport support for network connectivity
- Added Streamable HTTP transport support for modern bidirectional communication
- New command-line flags for transport configuration
- Support for stateful and stateless modes in Streamable HTTP
- Maintained backward compatibility with stdio transport
- Improved error handling and logging for different transport modes

### v0.4.0
- Simplified API to unified `d2_create` for all diagram creation needs
- Enhanced tool descriptions for better AI assistant integration
- Improved Oracle API error handling and validation
- Reduced API surface from 14 to 10 tools
- **Breaking Change**: Removed d2_render, d2_render_to_file, d2_import, d2_from_text - use d2_create instead

### v0.3.0
- Added `d2_oracle_serialize` tool to get current D2 text representation

### v0.2.0
- Added D2 Oracle API integration for incremental diagram manipulation
- 6 new MCP tools for creating, modifying, and querying diagram elements
- Support for stateful diagram editing sessions

### v0.1.0
- Initial release with basic D2 diagram operations
- Support for rendering, creating, exporting, and saving diagrams
- 20 built-in themes
- MCP protocol integration

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
