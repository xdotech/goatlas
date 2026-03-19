# GoAtlas

[![Go](https://img.shields.io/badge/Go_1.25+-00ADD8?style=flat-square&logo=go&logoColor=white)](https://go.dev/) [![Docker](https://img.shields.io/badge/Docker-2496ED?style=flat-square&logo=docker&logoColor=white)](https://hub.docker.com/) [![Neo4j](https://img.shields.io/badge/Neo4j-4581C3?style=flat-square&logo=neo4j&logoColor=white)](https://neo4j.com/) [![PostgreSQL](https://img.shields.io/badge/PostgreSQL-4169E1?style=flat-square&logo=postgresql&logoColor=white)](https://www.postgresql.org/) [![Gemini](https://img.shields.io/badge/Gemini-4285F4?style=flat-square&logo=google&logoColor=white)](https://ai.google.dev/) [![MCP](https://img.shields.io/badge/MCP-Protocol-blueviolet?style=flat-square)](https://modelcontextprotocol.io/)

<p align="center">
  <img src="goatlas.jpg" alt="GoAtlas — AI-Powered Code Intelligence Engine" width="100%">
</p>

**GoAtlas** is an AI-powered code intelligence platform that helps LLMs and developers deeply understand large Go/TypeScript codebases — combining AST parsing, a Neo4j knowledge graph, pgvector semantic search, and Gemini AI, all exposed via the **Model Context Protocol (MCP)**.

## What Makes It Different

- **Deep AST Indexing** — Parses Go/TS/JSX/Python files to extract every symbol (functions, types, methods, interfaces, consts, vars) and HTTP endpoints
- **Knowledge Graph** — Builds a Neo4j graph of packages, files, functions, types, and their import/call/implementation relationships
- **Semantic Search** — Gemini `text-embedding-004` embeddings stored in pgvector (or Qdrant) for meaning-based code discovery
- **Hybrid BM25 + Semantic (RRF)** — Reciprocal Rank Fusion merges keyword and vector results for best-of-both-worlds search
- **Process & Community Detection** — Forward BFS from entry points detects execution flows; Louvain algorithm clusters tightly-connected code communities
- **Confidence Scoring** — 7-tier confidence scores on call-graph edges and interface implementations for precision filtering
- **AI Agent** — Gemini 2.0 Flash agent with agentic tool-calling loop (up to 20 iterations) for code Q&A
- **MCP Server** — 22 MCP tools, 5 resources, 3 prompts via stdio transport for Cursor, Claude Desktop, and any MCP client
- **Claude Code Hooks** — PreToolUse/PostToolUse integration for automatic semantic enrichment and incremental re-indexing
- **Auto-Generated Skills** — AI-powered SKILL.md generation per community cluster for persistent Claude Code context
- **Wiki Generation** — Full Markdown wiki from the knowledge graph (services, communities, architecture)
- **Multi-Repo Support** — Index and query across multiple repositories with a shared repository registry
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
        DET["detect"]
        ASK["ask"]
        CHT["chat"]
        SRV["serve"]
        COV["coverage"]
        HK["hooks"]
        SKL["skills"]
        WK["wiki"]
    end

    subgraph Core["Core Engine"]
        direction TB
        PARSER["AST Parsers<br/>Go · TS/JSX · Python"]
        INDEXER["Code Indexer<br/>symbols · endpoints · imports"]
        PROCDET["Process Detector<br/>BFS from entry points"]
        COMMDET["Community Detector<br/>Louvain algorithm"]
        AGENT["Gemini Agent<br/>agentic tool-calling loop"]
        DOCGEN["Doc Generator<br/>skills · wiki"]
        MCP["MCP Server<br/>tools · resources · prompts"]
        COVERAGE["Coverage Checker<br/>spec vs codebase"]
    end

    subgraph Data["Data Layer"]
        PG["PostgreSQL + pgvector<br/>files · symbols · endpoints<br/>processes · communities<br/>vector embeddings"]
        NEO["Neo4j 5<br/>knowledge graph"]
        GEMINI["Gemini API<br/>embeddings + chat + doc gen"]
        QD["Qdrant (optional)<br/>vector backend"]
    end

    subgraph Clients["AI Assistants"]
        CURSOR["Cursor"]
        CLAUDE["Claude Desktop"]
        HOOKS["Claude Code Hooks"]
        ANY["Any MCP Client"]
    end

    IDX --> PARSER --> INDEXER --> PG
    EMB --> GEMINI --> PG
    GR --> NEO
    DET --> PROCDET & COMMDET --> PG
    ASK & CHT --> AGENT --> MCP
    SKL & WK --> DOCGEN --> GEMINI
    SRV --> MCP
    HK --> HOOKS
    COV --> COVERAGE
    MCP --> PG & NEO & GEMINI
    CURSOR & CLAUDE & HOOKS & ANY --> MCP
```

## Features

### Code Intelligence
- **Multi-Language Parsing** — Go, TypeScript, JSX/TSX, Python via AST
- **Symbol Extraction** — Functions, types, methods, interfaces, constants, variables
- **API Endpoint Detection** — HTTP routes from go-zero, gin, echo, chi, net/http, and more
- **Import Graph** — Full import dependency tracking per file
- **Call Graph** — Function-level call edges with confidence scoring
- **Interface Resolution** — Detects struct-implements-interface relationships with IMPLEMENTS edges

### Search & Discovery
- **Keyword Search** — PostgreSQL full-text search on symbol names and signatures
- **Semantic Search** — Vector similarity search using Gemini embeddings (pgvector or Qdrant)
- **Hybrid Search (RRF)** — Reciprocal Rank Fusion merging BM25 keyword scores with semantic vector scores for best results
- **Symbol Lookup** — Find symbols by exact name with kind filter

### Process & Community Detection
- **Process Detection** — Forward BFS from HTTP handlers, Kafka consumers, and `main()` entry points to trace execution flows
- **Community Detection** — Louvain modularity-based clustering groups tightly-interconnected functions into named communities
- **Confidence Scoring** — 7-tier confidence scoring on call-graph edges and interface implementations (0.0–1.0)

### Knowledge Graph (Neo4j)
- **Package → File → Symbol** relationships
- **Import edges** between packages
- **IMPLEMENTS edges** between types and interfaces
- **Handler pattern matching** for API discovery
- **Service dependency mapping**

### AI Agent
- **Agentic Loop** — Up to 20 tool-calling iterations per question
- **Dynamic System Prompt** — Includes repo summary and available tools
- **Multi-Turn Chat** — Full conversation history support
- **Tool Bridge** — Bridges MCP tools to Gemini function calls

### Auto-Generated Documentation
- **SKILL.md Generation** — AI-generated skill files for each community cluster, providing persistent context for Claude Code
- **Wiki Generation** — Full Markdown wiki from the knowledge graph covering services, communities, and architecture

### Incremental Indexing & Staleness Detection
- **Git-Aware Indexing** — Tracks last indexed commit per repository; `--incremental` flag re-indexes only changed files
- **Staleness Check** — MCP tool and CLI to check if the index is behind `git HEAD`
- **Claude Code Hooks** — PostToolUse hook auto-triggers incremental re-index on file writes

### Spec Coverage
- **Feature Extraction** — AI-powered (Gemini) or regex-based parsing
- **Implementation Detection** — Matches spec components against indexed symbols
- **Coverage Reports** — Text, JSON, or Markdown output with status per feature

## MCP Tools Reference

GoAtlas exposes **22 MCP tools** via the stdio transport:

| Tool | Description | Key Parameters |
|------|-------------|----------------|
| `search_code` | Search symbols by keyword/semantic/hybrid (RRF) | `query*`, `limit`, `kind`, `mode` |
| `read_file` | Read file content with optional line range | `path*`, `start_line`, `end_line` |
| `find_symbol` | Find a specific symbol by name | `name*`, `kind` |
| `find_callers` | Find functions referencing a given function | `function_name*`, `min_confidence` |
| `list_api_endpoints` | List detected HTTP routes in the codebase | `method`, `service` |
| `get_file_symbols` | Get all symbols defined in a file | `path*` |
| `list_services` | List all top-level packages/services | — |
| `get_service_dependencies` | Get import graph for a service (Neo4j) | `service*` |
| `get_api_handlers` | Find handler functions matching a pattern (Neo4j) | `pattern*` |
| `list_components` | List React components, hooks, interfaces, type aliases | `kind`, `limit` |
| `check_staleness` | Check if the index is behind git HEAD | `repo` |
| `list_processes` | List detected execution processes | — |
| `get_process_flow` | Get the ordered call chain for a process | `name*` |
| `list_communities` | List detected code communities (Louvain) | — |
| `detect_processes` | Trigger process + community detection | — |
| `list_repos` | List all indexed repositories with metadata | — |

\* = required parameter

<details>
<summary><strong>Admin & Analysis Tools</strong></summary>

| Tool | Description | Key Parameters |
|------|-------------|----------------|
| `index_repository` | Index or re-index a repository | `path*`, `force`, `incremental` |
| `build_graph` | Build the Neo4j knowledge graph | — |
| `generate_embeddings` | Generate vector embeddings for symbols | `force` |
| `analyze_impact` | Find all affected callers for a function | `symbol*`, `max_depth`, `min_confidence` |
| `trace_type_flow` | Trace data type producers and consumers | `type_name*`, `direction` |
| `get_api_consumers` | Find UI components calling an API endpoint | `api_path*`, `method` |
| `get_component_apis` | Get APIs called by a React component | `component*` |

</details>

### MCP Resources

GoAtlas exposes **5 structured resources** for on-demand data access:

| Resource URI | Description |
|-------------|-------------|
| `goatlas://repositories` | All indexed repositories with metadata |
| `goatlas://endpoints` | Detected API endpoints as structured JSON |
| `goatlas://schema` | Database schema summary with tables, relationships, and example queries |
| `goatlas://communities` | Detected code community clusters |
| `goatlas://processes` | Detected execution flows and entry points |

### MCP Prompts

| Prompt | Description | Arguments |
|--------|-------------|-----------|
| `detect_impact` | Analyze change impact with pre-filled caller context, endpoints, and implementations | `symbol*` |
| `generate_map` | Generate Mermaid architecture diagrams from the indexed codebase graph | `scope` |
| `explain_community` | Explain what a code community cluster does based on its members | `community_name*` |

## Getting Started

### Prerequisites

- **Go 1.25+**
- **Docker & Docker Compose** (for infrastructure services)
- **Gemini API Key** (for AI features: ask, chat, embed, coverage, skills, wiki)

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

# Incremental index — only files changed since last indexed commit
go run . index --incremental /path/to/your/repo
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

### 7. (Optional) Detect Processes & Communities

```bash
go run . detect         # Run process detection (BFS) + community detection (Louvain)
```

### 8. (Optional) Generate SKILL.md Files

```bash
go run . skills generate /path/to/your/repo   # Generate SKILL.md per community cluster
```

### 9. (Optional) Generate Markdown Wiki

```bash
go run . wiki ./wiki-output   # Generate full Markdown wiki from knowledge graph
```

### 10. (Optional) Auto-Index via Git Hook

Set up git hooks to automatically re-index when your working tree changes (after checkout, merge, or rebase):

```bash
# Create the hook script
cat > /path/to/your/repo/.git/hooks/post-merge << 'EOF'
#!/bin/sh
echo "[GoAtlas] Auto-indexing after merge..."
goatlas index "$(git rev-parse --show-toplevel)" &
EOF

# Reuse for other hooks
cp /path/to/your/repo/.git/hooks/post-merge /path/to/your/repo/.git/hooks/post-checkout
cp /path/to/your/repo/.git/hooks/post-merge /path/to/your/repo/.git/hooks/post-rewrite

# Make executable
chmod +x /path/to/your/repo/.git/hooks/post-*
```

> **Tip:** The `&` runs indexing in the background so it doesn't block your git workflow. If you also want to rebuild embeddings or the graph, append `goatlas embed &` or `goatlas build-graph &` to the hook script.

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

## Claude Code Integration

GoAtlas provides first-class Claude Code integration via hooks that enrich tool results and keep the index fresh.

### Install Hooks

```bash
# Auto-configure .claude/settings.json for a repository
goatlas hooks install /path/to/your/repo

# Remove hooks
goatlas hooks uninstall /path/to/your/repo
```

This installs:
- **PreToolUse** — When Claude uses Grep/Glob, GoAtlas enriches results with semantic search hints
- **PostToolUse** — When Claude writes/edits a file, GoAtlas triggers incremental re-indexing in the background

### MCP Server Setup

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
cmd/                          CLI commands (Cobra)
  ask.go                      Single-shot AI question
  chat.go                     Multi-turn interactive chat
  detect.go                   Process & community detection
  hook.go                     Claude Code hook handlers (pre/post)
  hooks.go                    Claude Code hook installer/uninstaller
  index.go                    Repository indexing
  skills.go                   Auto-generated SKILL.md management
  wiki.go                     Markdown wiki generation
  ...
internal/
  config/                     Configuration (Viper + .env)
  db/                         PostgreSQL pool + Goose migrations
    migrations/               SQL migration files (embedded)
  indexer/                    Code indexing engine
    domain/                   Domain types (File, Symbol, Endpoint, Import, Process, Community)
    parser/                   AST parsers (Go + JSX/TSX + Python)
    repository/postgres/      PostgreSQL repos (file, symbol, endpoint, import, process, community)
    usecase/                  Index repo, search symbols
  vector/                     Vector embedding & search
    store.go                  VectorStore interface
    pgvector.go               pgvector implementation (default)
    client.go                 Qdrant implementation (optional)
    embedder.go               Gemini embedding generator
    indexer.go                Embed pipeline orchestrator
    searcher.go               Semantic + hybrid (RRF) search
  graph/                      Neo4j knowledge graph
    client.go                 Neo4j driver wrapper
    builder.go                Graph construction with confidence scoring
    queries.go                Cypher query methods
    process_detector.go       BFS-based execution flow detection
    community_detector.go     Louvain community clustering
    types.go                  Graph domain types
  agent/                      Gemini AI agent
    agent.go                  Agentic loop (Ask + Chat)
    tool_bridge.go            Bridges MCP tools → Gemini function calls
    tool_declarations.go      Gemini FunctionDeclaration schemas
    system_prompt.go          Dynamic system prompt builder
    doc_generator.go          AI-powered SKILL.md + wiki generation
  mcp/                        MCP server implementation
    server.go                 Server wiring + stdio transport
    domain/tools.go           MCP tool input types
    handler/
      mcp_handler.go          22 tool registrations
      resources.go            5 MCP resource definitions
      prompts.go              3 MCP prompt templates
    registry/
      repo_registry.go        Multi-repo management
    usecase/                  Tool use case implementations
  coverage/                   Spec coverage analysis
    parser.go                 Spec file parser (markdown)
    gemini_parser.go          AI-powered feature extraction
    detector.go               Implementation detection engine
    reporter.go               Report generators (text/json/md)
scripts/
  goatlas-hook.sh             Git post-commit hook installer
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
| `repositories` | Indexed repository roots with last commit tracking |
| `files` | Indexed source files (path, module, hash, last_scanned) |
| `symbols` | Code symbols: functions, types, methods, interfaces, consts, vars |
| `function_calls` | Call graph edges between functions with confidence scores |
| `interface_impls` | Interface implementation relationships with confidence |
| `api_endpoints` | Detected HTTP routes (method, path, handler, framework) |
| `type_usages` | Type input/output flow tracking (direction: input/output) |
| `component_api_calls` | Frontend component to backend API mapping |
| `imports` | Go import statements per file |
| `symbol_embeddings` | Vector embeddings for semantic search (pgvector, 768-dim, HNSW index) |
| `processes` | Detected execution flows with entry points |
| `process_steps` | Ordered steps in execution flows |
| `communities` | Detected code community clusters (Louvain) with modularity scores |
| `community_members` | Functions belonging to a community |

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

## Contributing

Contributions are what make the open source community such an amazing place to learn, inspire, and create. Any contributions you make are **greatly appreciated**.

1. Fork the Project
2. Create your Feature Branch (`git checkout -b feature/AmazingFeature`)
3. Commit your Changes (`git commit -m 'feat: Add some AmazingFeature'`)
4. Push to the Branch (`git push origin feature/AmazingFeature`)
5. Open a Pull Request

## License

MIT