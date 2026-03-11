# GoAtlas — AI Code Intelligence & Spec Verification System

GoAtlas is an **AI-powered code intelligence platform** that helps LLMs and developers deeply understand large Go/TypeScript codebases. It combines **AST parsing**, a **Neo4j knowledge graph**, **Qdrant vector search**, and **Gemini AI** to provide semantic code search, interactive Q&A, and specification coverage analysis — all exposed via the **Model Context Protocol (MCP)**.

---

## ✨ Key Features

| Feature | Description |
|---------|-------------|
| 🔍 **Code Indexing** | Parses Go/TS/JSX files via AST, extracts symbols (functions, types, methods, interfaces, consts, vars) and API endpoints (HTTP routes) |
| 🧠 **Knowledge Graph** | Builds a Neo4j graph of packages, files, functions, types, and their import relationships |
| 📐 **Vector Embeddings** | Generates Gemini embeddings for all indexed symbols, stored in Qdrant for semantic search |
| 🤖 **AI Agent** | Gemini 2.0 Flash-powered agent with tool-calling (agentic loop up to 20 iterations) for code Q&A |
| 💬 **Interactive Chat** | Multi-turn conversational interface with full conversation history |
| 🔌 **MCP Server** | 10 MCP tools exposed via stdio transport for integration with AI assistants (Cursor, Claude Desktop, etc.) |
| 📊 **Spec Coverage** | Parses feature specification documents and detects implementation coverage in the codebase |

---

## 🏗️ Architecture Overview

```
┌──────────────────────────────────────────────────────────────┐
│                         GoAtlas CLI                          │
│  index │ embed │ build-graph │ ask │ chat │ serve │ coverage │
└────┬──────┬────────┬──────────┬──────┬───────┬──────┬────────┘
     │      │        │          │      │       │      │
     ▼      ▼        ▼          ▼      ▼       ▼      ▼
┌────────┐ ┌──────┐ ┌───────┐ ┌──────────┐ ┌──────┐ ┌────────┐
│Indexer │ │Vector│ │ Graph │ │  Agent   │ │ MCP  │ │Coverage│
│  (AST) │ │Embed │ │Builder│ │(Gemini)  │ │Server│ │Checker │
└───┬────┘ └──┬───┘ └──┬────┘ └────┬─────┘ └──┬───┘ └───┬────┘
    │         │        │           │           │         │
    ▼         ▼        ▼           ▼           ▼         ▼
┌─────────────────────────────────────────────────────────────┐
│                      Data Layer                             │
│  PostgreSQL 17   │   Qdrant   │   Neo4j 5   │  Gemini API  │
│  (files, symbols │   (vector  │   (graph    │  (embeddings │
│   endpoints,     │   search)  │   queries)  │   + chat)    │
│   imports)       │            │             │              │
└─────────────────────────────────────────────────────────────┘
```

---

## 📂 Project Structure

```
goatlas/
├── main.go                          # Entry point → cmd.Execute()
├── go.mod                           # Go 1.25.3 module
├── Makefile                         # Build, test, lint, docker, migrate
├── docker-compose.yml               # PostgreSQL + Qdrant + Neo4j
├── .env.example                     # Environment variable template
│
├── cmd/                             # CLI commands (Cobra)
│   ├── root.go                      # Root command definition
│   ├── index.go                     # `goatlas index <repo-path>`
│   ├── embed.go                     # `goatlas embed`
│   ├── graph.go                     # `goatlas build-graph`
│   ├── ask.go                       # `goatlas ask <question>`
│   ├── chat.go                      # `goatlas chat` (interactive)
│   ├── serve.go                     # `goatlas serve` (MCP stdio)
│   ├── coverage.go                  # `goatlas check-coverage <spec>`
│   └── migrate.go                   # `goatlas migrate`
│
└── internal/                        # Application packages
    ├── config/                      # Configuration (Viper + .env)
    ├── db/                          # PostgreSQL pool + Goose migrations
    │   └── migrations/              # SQL migration files (embedded)
    ├── indexer/                     # Code indexing engine
    │   ├── domain/                  # Domain types (File, Symbol, Endpoint, Import)
    │   ├── parser/                  # AST parsers (Go + JSX/TSX)
    │   ├── repository/postgres/     # PostgreSQL repos (file, symbol, endpoint, import)
    │   ├── usecase/                 # Index repo, search symbols
    │   └── service.go               # Service aggregator
    ├── vector/                      # Vector embedding & search
    │   ├── client.go                # Qdrant gRPC client
    │   ├── embedder.go              # Gemini embedding generator
    │   ├── indexer.go               # Orchestrates embed pipeline
    │   └── searcher.go              # Semantic search queries
    ├── graph/                       # Neo4j knowledge graph
    │   ├── client.go                # Neo4j driver wrapper
    │   ├── builder.go               # Graph construction from indexed data
    │   ├── queries.go               # Cypher query methods
    │   └── types.go                 # Graph node/edge types
    ├── agent/                       # Gemini AI agent
    │   ├── agent.go                 # Agentic loop (Ask + Chat)
    │   ├── tool_bridge.go           # Bridges MCP tools → Gemini function calls
    │   ├── tool_declarations.go     # Gemini FunctionDeclaration schemas
    │   ├── system_prompt.go         # Dynamic system prompt builder
    │   └── types.go                 # AgentConfig, ConversationMessage
    ├── mcp/                         # MCP server implementation
    │   ├── server.go                # Server wiring + stdio transport
    │   ├── domain/tools.go          # MCP tool input types
    │   ├── handler/mcp_handler.go   # 10 MCP tool registrations
    │   └── usecase/                 # Tool use case implementations
    └── coverage/                    # Spec coverage analysis
        ├── parser.go                # Spec file parser (markdown)
        ├── gemini_parser.go         # AI-powered feature extraction
        ├── detector.go              # Implementation detection engine
        ├── reporter.go              # Text/JSON/Markdown report generators
        └── types.go                 # Feature, Component, CoverageReport
```

---

## 🚀 Getting Started

### Prerequisites

- **Go 1.25+**
- **Docker & Docker Compose** (for infrastructure services)
- **Gemini API Key** (for AI features: ask, chat, embed, coverage)

### 1. Start Infrastructure

```bash
# Start PostgreSQL, Qdrant, and Neo4j
make docker-up
```

This starts:
| Service    | Port(s)     | Credentials               |
|------------|-------------|----------------------------|
| PostgreSQL | `5432`      | `goatlas:goatlas/goatlas`   |
| Qdrant     | `6333/6334` | —                          |
| Neo4j      | `7474/7687` | `neo4j:goatlas_neo4j`      |

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
# Embed all indexed symbols into Qdrant for semantic search
go run . embed

# Force re-embed everything
go run . embed --force
```

### 6. (Optional) Build Knowledge Graph

```bash
# Populate Neo4j with package/file/function/type/import relationships
go run . build-graph
```

---

## 💻 Usage

### Ask a Question (Single-Shot)

```bash
goatlas ask "How does the authentication middleware work?"
goatlas ask "What endpoints does the user service expose?"
```

### Interactive Chat

```bash
goatlas chat
# > You: What's the main entry point?
# > Assistant: The main entry point is...
# > You: exit
```

### Spec Coverage Check

```bash
# Check feature spec coverage (with AI extraction)
goatlas check-coverage spec.md --format md

# Without AI (regex-only parsing)
goatlas check-coverage spec.md --no-ai --format json
```

### MCP Server Mode

```bash
# Start as an MCP server for AI assistants
goatlas serve
```

---

## 🔧 MCP Tools Reference

GoAtlas exposes **10 MCP tools** via the stdio transport:

| # | Tool | Description | Key Parameters |
|---|------|-------------|----------------|
| 1 | `search_code` | Search symbols by keyword/semantic/hybrid | `query*`, `limit`, `kind`, `mode` |
| 2 | `read_file` | Read file content with optional line range | `path*`, `start_line`, `end_line` |
| 3 | `find_symbol` | Find a specific symbol by name | `name*`, `kind` |
| 4 | `find_callers` | Find functions referencing a given function | `function_name*` |
| 5 | `list_api_endpoints` | List detected HTTP routes in the codebase | `method`, `service` |
| 6 | `get_file_symbols` | Get all symbols defined in a file | `path*` |
| 7 | `list_services` | List all top-level packages/services | — |
| 8 | `get_service_dependencies` | Get import graph for a service (Neo4j) | `service*` |
| 9 | `get_api_handlers` | Find handler functions matching a pattern (Neo4j) | `pattern*` |
| 10 | `list_components` | List React components, hooks, interfaces, type aliases | `kind`, `limit` |

\* = required parameter

### Search Modes

- **`keyword`** (default) — PostgreSQL full-text search on symbol names and signatures
- **`semantic`** — Qdrant vector similarity search using Gemini embeddings
- **`hybrid`** — Combines keyword + semantic results

---

## 🗄️ Data Model

### PostgreSQL Schema

| Table | Purpose |
|-------|---------|
| `files` | Indexed source files (path, module, hash, last_scanned) |
| `symbols` | Code symbols: functions, types, methods, interfaces, consts, vars |
| `api_endpoints` | Detected HTTP routes (method, path, handler, framework) |
| `imports` | Go import statements per file |

### Neo4j Graph Model

```
(:Package)──[:CONTAINS]──>(:File)──[:DEFINES]──>(:Function)
                                  └──[:DEFINES]──>(:Type)
(:Package)──[:IMPORTS]──>(:Package)
```

Node types: **Package**, **File**, **Function**, **Type**
Edge types: **CONTAINS**, **DEFINES**, **IMPORTS**

### Qdrant Vector Collection

- Collection: code symbol embeddings
- Embedding model: Gemini text-embedding
- Payload: symbol metadata (name, kind, qualified_name, file, signature)

---

## 🤖 AI Agent Architecture

The Gemini agent uses an **agentic tool-calling loop**:

1. Receives user question + dynamic system prompt (includes repo summary, available tools)
2. Gemini decides which MCP tools to call
3. GoAtlas executes the tool calls against indexed data
4. Results are fed back to Gemini
5. Repeats up to **20 iterations** until Gemini provides a final answer

### Configuration

| Parameter | Default | Description |
|-----------|---------|-------------|
| `MaxIterations` | 20 | Max tool-call rounds |
| `Model` | `gemini-2.0-flash` | Gemini model |
| `Temperature` | 0.1 | Response determinism |

---

## 📊 Spec Coverage Checker

The coverage checker analyzes a feature specification document against the indexed codebase:

1. **Parse** — Splits markdown spec into feature sections
2. **Extract** — Uses Gemini AI (or regex fallback) to identify implementable components:
   - API endpoints (`POST /users`)
   - Service methods (`CreateUser`)
   - UI screens (`UserListScreen`)
3. **Detect** — Searches indexed symbols and endpoints for matches
4. **Report** — Generates coverage report with status per feature:
   - ✅ **Implemented** — All components found
   - ⚠️ **Partial** — Some components found
   - ❌ **Missing** — No components found

Output formats: `text`, `json`, `markdown`

---

## 🔨 Development

```bash
# Build binary
make build

# Run tests
make test

# Lint
make lint

# Stop infrastructure
make docker-down

# Clean built binary
make clean
```

---

## ⚙️ Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `DATABASE_DSN` | `postgres://goatlas:goatlas@localhost:5432/goatlas` | PostgreSQL connection string |
| `QDRANT_URL` | `http://localhost:6334` | Qdrant gRPC endpoint |
| `NEO4J_URL` | `bolt://localhost:7687` | Neo4j Bolt endpoint |
| `NEO4J_USER` | `neo4j` | Neo4j username |
| `NEO4J_PASS` | `goatlas_neo4j` | Neo4j password |
| `GEMINI_API_KEY` | — | Google Gemini API key (required for AI features) |
| `REPO_PATH` | — | Default repository path for indexing |
| `HTTP_ADDR` | `:8080` | HTTP server listen address |

---

## 📦 Key Dependencies

| Dependency | Purpose |
|------------|---------|
| `github.com/spf13/cobra` | CLI framework |
| `github.com/spf13/viper` | Configuration management |
| `github.com/jackc/pgx/v5` | PostgreSQL driver & connection pooling |
| `github.com/pressly/goose/v3` | Database migrations |
| `github.com/mark3labs/mcp-go` | MCP server SDK |
| `github.com/google/generative-ai-go` | Gemini AI client |
| `github.com/qdrant/go-client` | Qdrant vector DB client |
| `github.com/neo4j/neo4j-go-driver/v5` | Neo4j graph DB driver |
| `google.golang.org/api` | Google APIs support |

---

## 📜 License

Internal project — All rights reserved.