# GoAtlas

[![Go](https://img.shields.io/badge/Go_1.25+-00ADD8?style=flat-square&logo=go&logoColor=white)](https://go.dev/) [![Docker](https://img.shields.io/badge/Docker-2496ED?style=flat-square&logo=docker&logoColor=white)](https://hub.docker.com/) [![Neo4j](https://img.shields.io/badge/Neo4j-4581C3?style=flat-square&logo=neo4j&logoColor=white)](https://neo4j.com/) [![PostgreSQL](https://img.shields.io/badge/PostgreSQL-4169E1?style=flat-square&logo=postgresql&logoColor=white)](https://www.postgresql.org/) [![Gemini](https://img.shields.io/badge/Gemini-4285F4?style=flat-square&logo=google&logoColor=white)](https://ai.google.dev/) [![MCP](https://img.shields.io/badge/MCP-Protocol-blueviolet?style=flat-square)](https://modelcontextprotocol.io/)

**GoAtlas** is an AI-powered code intelligence platform that helps LLMs and developers deeply understand large Go/TypeScript codebases — combining AST parsing, a Neo4j knowledge graph, pgvector semantic search, and Gemini AI, all exposed via the **Model Context Protocol (MCP)**.

## What Makes It Different

- **Deep AST Indexing** — Parses Go/TS/JSX/Python files to extract every symbol (functions, types, methods, interfaces, consts, vars) and HTTP endpoints
- **Knowledge Graph** — Builds a Neo4j graph of packages, files, functions, types, and their import/call relationships
- **Semantic Search** — Gemini `text-embedding-004` embeddings stored in pgvector (or Qdrant) for meaning-based code discovery
- **AI Agent** — Gemini 2.0 Flash agent with agentic tool-calling loop (up to 20 iterations) for code Q&A
- **MCP Server** — 10+ MCP tools via stdio transport for seamless integration with Cursor, Claude Desktop, and other AI assistants
- **Spec Coverage** — Parses feature specs and detects implementation coverage in the codebase
- **Interactive Chat** — Multi-turn conversational interface with full conversation history

## Architecture

```mermaid
graph TB
    subgraph CLI["GoAtlas CLI"]
        direction LR
        IDX["index"]
        EMB["embed"]
        GR["build-graph"]
        ASK["ask"]
        CHT["chat"]
        SRV["serve"]
        COV["coverage"]
    end

    subgraph Core["Core Engine"]
        direction TB
        PARSER["AST Parsers<br/>Go · TS/JSX · Python"]
        INDEXER["Code Indexer<br/>symbols · endpoints · imports"]
        AGENT["Gemini Agent<br/>agentic tool-calling loop"]
        MCP["MCP Server<br/>stdio transport"]
        COVERAGE["Coverage Checker<br/>spec vs codebase"]
    end

    subgraph Data["Data Layer"]
        PG["PostgreSQL + pgvector<br/>files · symbols · endpoints<br/>imports · vector embeddings"]
        NEO["Neo4j 5<br/>knowledge graph"]
        GEMINI["Gemini API<br/>embeddings + chat"]
        QD["Qdrant (optional)<br/>vector backend"]
    end

    subgraph Clients["AI Assistants"]
        CURSOR["Cursor"]
        CLAUDE["Claude Desktop"]
        ANY["Any MCP Client"]
    end

    IDX --> PARSER --> INDEXER --> PG
    EMB --> GEMINI --> PG
    GR --> NEO
    ASK & CHT --> AGENT --> MCP
    SRV --> MCP
    COV --> COVERAGE
    MCP --> PG & NEO & GEMINI
    CURSOR & CLAUDE & ANY --> MCP
```

## Features

### Code Intelligence
- **Multi-Language Parsing** — Go, TypeScript, JSX/TSX, Python via AST
- **Symbol Extraction** — Functions, types, methods, interfaces, constants, variables
- **API Endpoint Detection** — HTTP routes from go-zero, gin, echo, chi, net/http, and more
- **Import Graph** — Full import dependency tracking per file

### Search & Discovery
- **Keyword Search** — PostgreSQL full-text search on symbol names and signatures
- **Semantic Search** — Vector similarity search using Gemini embeddings (pgvector or Qdrant)
- **Hybrid Search** — Combines keyword + semantic for best results
- **Symbol Lookup** — Find symbols by exact name with kind filter

### Knowledge Graph (Neo4j)
- **Package → File → Symbol** relationships
- **Import edges** between packages
- **Handler pattern matching** for API discovery
- **Service dependency mapping**

### AI Agent
- **Agentic Loop** — Up to 20 tool-calling iterations per question
- **Dynamic System Prompt** — Includes repo summary and available tools
- **Multi-Turn Chat** — Full conversation history support
- **Tool Bridge** — Bridges MCP tools to Gemini function calls

### Spec Coverage
- **Feature Extraction** — AI-powered (Gemini) or regex-based parsing
- **Implementation Detection** — Matches spec components against indexed symbols
- **Coverage Reports** — Text, JSON, or Markdown output with status per feature

## MCP Tools Reference

GoAtlas exposes **10+ MCP tools** via the stdio transport:

| Tool | Description | Key Parameters |
|------|-------------|----------------|
| `search_code` | Search symbols by keyword/semantic/hybrid | `query*`, `limit`, `kind`, `mode` |
| `read_file` | Read file content with optional line range | `path*`, `start_line`, `end_line` |
| `find_symbol` | Find a specific symbol by name | `name*`, `kind` |
| `find_callers` | Find functions referencing a given function | `function_name*` |
| `list_api_endpoints` | List detected HTTP routes in the codebase | `method`, `service` |
| `get_file_symbols` | Get all symbols defined in a file | `path*` |
| `list_services` | List all top-level packages/services | — |
| `get_service_dependencies` | Get import graph for a service (Neo4j) | `service*` |
| `get_api_handlers` | Find handler functions matching a pattern (Neo4j) | `pattern*` |
| `list_components` | List React components, hooks, interfaces, type aliases | `kind`, `limit` |

\* = required parameter

<details>
<summary><strong>Additional MCP Tools</strong></summary>

| Tool | Description | Key Parameters |
|------|-------------|----------------|
| `index_repository` | Index or re-index a repository | `path*`, `force` |
| `build_graph` | Build the Neo4j knowledge graph | — |
| `generate_embeddings` | Generate vector embeddings for symbols | `force` |
| `analyze_impact` | Find all affected callers for a function | `symbol*`, `max_depth` |
| `trace_type_flow` | Trace data type producers and consumers | `type_name*`, `direction` |
| `get_api_consumers` | Find UI components calling an API endpoint | `api_path*`, `method` |
| `get_component_apis` | Get APIs called by a React component | `component*` |

</details>

## Getting Started

### Prerequisites

- **Go 1.25+**
- **Docker & Docker Compose** (for infrastructure services)
- **Gemini API Key** (for AI features: ask, chat, embed, coverage)

### 1. Start Infrastructure

```bash
# Start PostgreSQL (pgvector), Qdrant, and Neo4j
make docker-up
```

This starts:
| Service    | Port(s)     | Credentials               | Notes |
|------------|-------------|----------------------------|-------|
| PostgreSQL | `5432`      | `goatlas:goatlas/goatlas`   | Uses `pgvector/pgvector:pg17` image |
| Qdrant     | `6333/6334` | —                          | Optional (only needed if `QDRANT_URL` is set) |
| Neo4j      | `7474/7687` | `neo4j:goatlas_neo4j`      | Optional (for graph tools) |

#### Using an existing PostgreSQL instance

If you already have a PostgreSQL container running (e.g. `postgres:14`), install pgvector:

```bash
# Install pgvector extension inside the running container
docker exec <container_name> bash -c "apt-get update && apt-get install -y postgresql-14-pgvector"

# Enable the extension
docker exec <container_name> psql -U <user> -d <database> -c "CREATE EXTENSION IF NOT EXISTS vector;"
```

> **Note:** Replace `postgresql-14-pgvector` with the matching version for your PostgreSQL. This installation is lost when the container is recreated — for persistence, switch the image to `pgvector/pgvector:pgXX`.

### 2. Configure Environment

```bash
cp .env.example .env
# Edit .env and set:
#   GEMINI_API_KEY=your_key_here
#   REPO_PATH=/path/to/your/go/repo
```

### 3. Run Database Migrations

```bash
make migrate
# or: go run . migrate
```

### 4. Index a Repository

```bash
# Index a Go/TS codebase
make run-index REPO_PATH=/path/to/your/repo
# or: go run . index /path/to/your/repo

# Force re-index all files
go run . index --force /path/to/your/repo
```

### 5. (Optional) Generate Embeddings

```bash
go run . embed          # Embed all indexed symbols (pgvector)
go run . embed --force  # Force re-embed everything
```

### 6. (Optional) Build Knowledge Graph

```bash
go run . build-graph    # Populate Neo4j graph
```

## Usage

**Single-Shot Question:**
```bash
goatlas ask "How does the authentication middleware work?"
goatlas ask "What endpoints does the user service expose?"
```

**Interactive Chat:**
```bash
goatlas chat
# > You: What's the main entry point?
# > Assistant: The main entry point is...
# > You: exit
```

**Spec Coverage:**
```bash
goatlas check-coverage spec.md --format md
goatlas check-coverage spec.md --no-ai --format json
```

**MCP Server:**
```bash
goatlas serve
```

## CLI Tool Integration

**Cursor** — `~/.cursor/mcp.json`:
```json
{
  "mcpServers": {
    "goatlas": {
      "command": "goatlas",
      "args": ["serve"]
    }
  }
}
```

**Claude Desktop** — `claude_desktop_config.json`:
```json
{
  "mcpServers": {
    "goatlas": {
      "command": "goatlas",
      "args": ["serve"]
    }
  }
}
```

## Project Structure

```
cmd/goatlas/              Entrypoint (Cobra CLI)
internal/
  config/                 Configuration (Viper + .env)
  db/                     PostgreSQL pool + Goose migrations
    migrations/           SQL migration files (embedded)
  indexer/                Code indexing engine
    domain/               Domain types (File, Symbol, Endpoint, Import)
    parser/               AST parsers (Go + JSX/TSX + Python)
    repository/postgres/  PostgreSQL repos (file, symbol, endpoint, import)
    usecase/              Index repo, search symbols
  vector/                 Vector embedding & search
    store.go              VectorStore interface
    pgvector.go           pgvector implementation (default)
    client.go             Qdrant implementation (optional)
    embedder.go           Gemini embedding generator
    indexer.go            Embed pipeline orchestrator
    searcher.go           Semantic search queries
  graph/                  Neo4j knowledge graph
    client.go             Neo4j driver wrapper
    builder.go            Graph construction from indexed data
    queries.go            Cypher query methods
  agent/                  Gemini AI agent
    agent.go              Agentic loop (Ask + Chat)
    tool_bridge.go        Bridges MCP tools → Gemini function calls
    tool_declarations.go  Gemini FunctionDeclaration schemas
    system_prompt.go      Dynamic system prompt builder
  mcp/                    MCP server implementation
    server.go             Server wiring + stdio transport
    domain/tools.go       MCP tool input types
    handler/              Tool registrations
    usecase/              Tool use case implementations
  coverage/               Spec coverage analysis
    parser.go             Spec file parser (markdown)
    gemini_parser.go      AI-powered feature extraction
    detector.go           Implementation detection engine
    reporter.go           Report generators (text/json/md)
```

## Development

```bash
make build        # compile binary
make run          # build + run
make test         # run tests with race detector
make lint         # run linter
make docker-up    # start infrastructure
make docker-down  # stop infrastructure
make migrate      # run database migrations
make clean        # remove build artifacts
```

<details>
<summary><strong>Environment Variables</strong></summary>

| Variable | Default | Description |
|----------|---------|-------------|
| `DATABASE_DSN` | `postgres://goatlas:goatlas@localhost:5432/goatlas` | PostgreSQL connection string |
| `QDRANT_URL` | — (empty) | Qdrant gRPC endpoint. If set, uses Qdrant for vectors. If empty, uses pgvector |
| `NEO4J_URL` | `bolt://localhost:7687` | Neo4j Bolt endpoint |
| `NEO4J_USER` | `neo4j` | Neo4j username |
| `NEO4J_PASS` | `goatlas_neo4j` | Neo4j password |
| `GEMINI_API_KEY` | — | Google Gemini API key (required for AI features) |
| `REPO_PATH` | — | Default repository path for indexing |
| `HTTP_ADDR` | `:8080` | HTTP server listen address |

</details>

<details>
<summary><strong>Data Model</strong></summary>

#### PostgreSQL Schema

| Table | Purpose |
|-------|---------|
| `files` | Indexed source files (path, module, hash, last_scanned) |
| `symbols` | Code symbols: functions, types, methods, interfaces, consts, vars |
| `api_endpoints` | Detected HTTP routes (method, path, handler, framework) |
| `imports` | Go import statements per file |
| `symbol_embeddings` | Vector embeddings for semantic search (pgvector, 768-dim, HNSW index) |

#### Neo4j Graph Model

```
(:Package)──[:CONTAINS]──>(:File)──[:DEFINES]──>(:Function)
                                  └──[:DEFINES]──>(:Type)
(:Package)──[:IMPORTS]──>(:Package)
(:Type)──[:IMPLEMENTS]──>(:Interface)
```

Node types: **Package**, **File**, **Function**, **Type**
Edge types: **CONTAINS**, **DEFINES**, **IMPORTS**, **IMPLEMENTS**

#### Vector Storage

| Backend | When Used | Storage |
|---------|-----------|--------|
| **pgvector** (default) | `QDRANT_URL` not set | PostgreSQL `symbol_embeddings` table with HNSW index |
| **Qdrant** (optional) | `QDRANT_URL` is set | Qdrant `code_symbols` collection |

Embedding model: Gemini `text-embedding-004` (768 dimensions), Cosine similarity

</details>

## Roadmap

- [x] Phase 1 — AST indexing engine (Go + TypeScript)
- [x] Phase 2 — PostgreSQL storage + pgvector embeddings
- [x] Phase 3 — Neo4j knowledge graph
- [x] Phase 4 — Gemini AI agent with tool-calling
- [x] Phase 5 — MCP server (10 tools, stdio transport)
- [x] Phase 6 — Spec coverage checker
- [x] Phase 7 — Interactive chat mode
- [x] Phase 8 — Python parser (tree-sitter)
- [x] Phase 9 — Interface method resolution + IMPLEMENTS edges
- [x] Phase 10 — Impact analysis + type flow tracing
- [ ] Phase 11 — HTTP server mode (SSE transport)
- [ ] Phase 12 — Dashboard UI
- [ ] Phase 13 — Multi-repo support

## Contributing

Contributions are what make the open source community such an amazing place to learn, inspire, and create. Any contributions you make are **greatly appreciated**.

1. Fork the Project
2. Create your Feature Branch (`git checkout -b feature/AmazingFeature`)
3. Commit your Changes (`git commit -m 'feat: Add some AmazingFeature'`)
4. Push to the Branch (`git push origin feature/AmazingFeature`)
5. Open a Pull Request

## License

MIT