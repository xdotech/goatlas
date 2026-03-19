# GoAtlas

[![Go](https://img.shields.io/badge/Go_1.25+-00ADD8?style=flat-square&logo=go&logoColor=white)](https://go.dev/) [![Docker](https://img.shields.io/badge/Docker-2496ED?style=flat-square&logo=docker&logoColor=white)](https://hub.docker.com/) [![Neo4j](https://img.shields.io/badge/Neo4j-4581C3?style=flat-square&logo=neo4j&logoColor=white)](https://neo4j.com/) [![PostgreSQL](https://img.shields.io/badge/PostgreSQL-4169E1?style=flat-square&logo=postgresql&logoColor=white)](https://www.postgresql.org/) [![MCP](https://img.shields.io/badge/MCP-Protocol-blueviolet?style=flat-square)](https://modelcontextprotocol.io/)

<p align="center">
  <img src="goatlas.jpg" alt="GoAtlas — AI-Powered Code Intelligence Engine" width="100%">
</p>

**GoAtlas** is an AI-powered code intelligence platform that helps LLMs and developers deeply understand large codebases — combining multi-language AST parsing, a Neo4j knowledge graph, pgvector semantic search, and a pluggable LLM backend, all exposed via the **Model Context Protocol (MCP)**.

## What Makes It Different

- **Multi-Language** — Go, TypeScript/JSX, Python, and Java parsed via AST (tree-sitter); symbols, endpoints, imports extracted per language
- **Zero-Config Auto-Discovery** — Reads `go.mod`, `package.json`, `requirements.txt`, `pom.xml` to automatically activate connection detection patterns for gRPC, Kafka, HTTP clients, and more
- **Cross-Service Connection Detection** — Detects inter-service connections (gRPC, Kafka, HTTP) across Go, TS, Python, and Java with no config required
- **Knowledge Graph** — Neo4j graph of packages, files, functions, types with import/call/implementation edges
- **Hybrid BM25 + Semantic Search** — Reciprocal Rank Fusion merges keyword and vector results; semantic search via Gemini or Ollama embeddings
- **Process & Community Detection** — BFS from entry points traces execution flows; Louvain clustering groups code communities
- **Pluggable LLM** — Supports Gemini (`gemini-2.0-flash`) and Ollama (any local model) for both embeddings and agentic Q&A
- **MCP Server** — 22 tools, 5 resources, 3 prompts via stdio for Cursor, Claude Desktop, and any MCP client
- **Claude Code Hooks** — PreToolUse/PostToolUse integration for semantic enrichment and incremental re-indexing
- **AI-Generated Docs** — SKILL.md per community cluster and full Markdown wiki from the knowledge graph

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
        SKL["skills"]
        WK["wiki"]
    end

    subgraph Core["Core Engine"]
        direction TB
        PARSER["AST Parsers\nGo · TS/JSX · Python · Java"]
        INDEXER["Code Indexer\nsymbols · endpoints · connections"]
        CATALOG["Auto-Discovery\ngo.mod · package.json · pom.xml"]
        PROCDET["Process Detector\nBFS from entry points"]
        COMMDET["Community Detector\nLouvain algorithm"]
        AGENT["LLM Agent\nGemini or Ollama"]
        DOCGEN["Doc Generator\nskills · wiki"]
        MCP["MCP Server\ntools · resources · prompts"]
    end

    subgraph Data["Data Layer"]
        PG["PostgreSQL + pgvector\nsymbols · endpoints · embeddings"]
        NEO["Neo4j\nknowledge graph"]
        LLM["LLM Backend\nGemini API or Ollama"]
        QD["Qdrant (optional)\nvector backend"]
    end

    subgraph Clients["AI Assistants"]
        CURSOR["Cursor"]
        CLAUDE["Claude Desktop"]
        HOOKS["Claude Code Hooks"]
        ANY["Any MCP Client"]
    end

    IDX --> PARSER & CATALOG --> INDEXER --> PG
    EMB --> LLM --> PG
    GR --> NEO
    DET --> PROCDET & COMMDET --> PG
    ASK & CHT --> AGENT --> MCP
    SKL & WK --> DOCGEN --> LLM
    SRV --> MCP
    MCP --> PG & NEO & LLM
    CURSOR & CLAUDE & HOOKS & ANY --> MCP
```

## Features

### Code Intelligence
- **Multi-Language Parsing** — Go, TypeScript/JSX, Python, Java via AST (tree-sitter for Python/Java)
- **Symbol Extraction** — Functions, types, methods, interfaces, constants, variables per file
- **API Endpoint Detection** — HTTP routes from go-zero, gin, echo, chi, net/http, Spring MVC, and more
- **Cross-Service Connection Detection** — Auto-detects gRPC, Kafka, HTTP clients across all 4 languages
- **Zero-Config Auto-Discovery** — Reads `go.mod`, `package.json`, `requirements.txt`, `pom.xml` to activate connection patterns automatically
- **Call Graph** — Function-level call edges with 7-tier confidence scoring
- **Interface Resolution** — Detects struct-implements-interface relationships

### Search & Discovery
- **Keyword Search** — PostgreSQL full-text search on symbol names and signatures
- **Semantic Search** — Vector similarity via Gemini or Ollama embeddings (pgvector or Qdrant backend)
- **Hybrid Search (RRF)** — Reciprocal Rank Fusion merging BM25 + vector scores for best results
- **Symbol Lookup** — Find symbols by name with kind filter

### Process & Community Detection
- **Process Detection** — Forward BFS from HTTP handlers, Kafka consumers, and `main()` entry points
- **Community Detection** — Louvain modularity clustering groups tightly-connected code into named communities
- **Confidence Scoring** — 7-tier scores on call-graph edges and interface implementations

### Knowledge Graph (Neo4j)
- Package → File → Symbol relationships
- Import edges between packages
- IMPLEMENTS edges between types and interfaces
- Service dependency mapping

### AI Agent
- **Pluggable LLM** — Gemini (`gemini-2.0-flash`) or Ollama (any local model)
- **Agentic Loop** — Up to 20 tool-calling iterations per question
- **Multi-Turn Chat** — Full conversation history support
- **Dynamic System Prompt** — Includes repo summary, services, and available tools

### Auto-Generated Documentation
- **SKILL.md Generation** — AI-generated skill files per community cluster for persistent Claude Code context
- **Wiki Generation** — Full Markdown wiki (services, communities, architecture) from the knowledge graph

### Incremental Indexing
- **Git-Aware** — Tracks last indexed commit; `--incremental` re-indexes only changed files
- **Staleness Detection** — Check if index is behind `git HEAD`
- **Claude Code Hooks** — PostToolUse hook auto-triggers incremental re-index on file writes

## Quick Start

### 1. Install

```bash
go install github.com/xdotech/goatlas@latest
```

Or build from source:
```bash
git clone https://github.com/xdotech/goatlas
cd goatlas && make build
```

### 2. Start Infrastructure

```bash
make docker-up
```

Starts PostgreSQL (pgvector), Qdrant (optional), and Neo4j (optional).

### 3. Configure

```bash
cp .env.example .env
```

Minimum required:
```env
DATABASE_DSN=postgres://goatlas:goatlas@localhost:5432/goatlas
REPO_PATH=/path/to/your/repo

# Pick a provider (at least one)
GEMINI_API_KEY=your_key_here   # for Gemini
# or
LLM_PROVIDER=ollama            # for local Ollama
EMBED_PROVIDER=ollama
```

### 4. Migrate & Index

```bash
goatlas migrate
goatlas index /path/to/your/repo
```

### 5. Start MCP Server

```bash
goatlas serve
```

### 6. Ask Questions

```bash
goatlas ask "How does the payment service connect to Kafka?"
goatlas chat    # interactive multi-turn session
```

## LLM Providers

GoAtlas supports **Gemini** (default) and **Ollama** (local) for both chat and embeddings, configured independently.

| Variable | Default | Description |
|----------|---------|-------------|
| `LLM_PROVIDER` | `gemini` | Chat/agent provider: `gemini` \| `ollama` |
| `EMBED_PROVIDER` | `gemini` | Embedding provider: `gemini` \| `ollama` |
| `GEMINI_API_KEY` | — | Required when using Gemini |
| `OLLAMA_URL` | `http://localhost:11434` | Ollama server URL |
| `OLLAMA_MODEL` | `llama3.2` | Ollama chat model |
| `OLLAMA_EMBED_MODEL` | `nomic-embed-text` | Ollama embedding model |

**Ollama example** — run entirely locally:
```env
LLM_PROVIDER=ollama
EMBED_PROVIDER=ollama
OLLAMA_URL=http://localhost:11434
OLLAMA_MODEL=llama3.2
OLLAMA_EMBED_MODEL=nomic-embed-text
```

> **Note:** Switching embedding models requires re-running `go run . embed --force` since vector dimensions differ (Gemini `text-embedding-004` = 768 dims, `nomic-embed-text` = 768 dims, `mxbai-embed-large` = 1024 dims).

## Multi-Language Support

GoAtlas auto-discovers connection patterns by reading your dependency files — no manual config needed.

| Language | Source | Detected Connections |
|----------|--------|---------------------|
| **Go** | `go.mod` | gRPC, Kafka (segmentio, sarama, confluent), HTTP (resty, standard), Redis, NATS |
| **TypeScript** | `package.json` | KafkaJS, gRPC-JS, Axios, ioredis |
| **Python** | `requirements.txt` | grpcio, kafka-python, httpx, requests, aiohttp, redis |
| **Java** | `pom.xml` | io.grpc, Spring Kafka, OpenFeign, Spring Web, kafka-clients, Lettuce, JNATS |

Custom patterns can be added via `patterns.yaml` in the repo root — they merge additively with catalog defaults.

## MCP Tools

GoAtlas exposes **22 MCP tools** via stdio transport:

| Tool | Description |
|------|-------------|
| `search_code` | Keyword, semantic, or hybrid (RRF) symbol search |
| `find_symbol` | Lookup a symbol by name and kind |
| `read_file` | Read any indexed file with optional line range |
| `find_callers` | Find all callers of a function |
| `list_api_endpoints` | List detected HTTP routes |
| `get_file_symbols` | All symbols defined in a file |
| `list_services` | Top-level packages/services in the repo |
| `get_service_dependencies` | Import graph for a service (Neo4j) |
| `get_api_handlers` | Handler functions for an endpoint pattern (Neo4j) |
| `list_components` | React components, hooks, interfaces, type aliases |
| `list_processes` | Detected execution flows |
| `get_process_flow` | Ordered call chain for a process |
| `list_communities` | Louvain code community clusters |
| `detect_processes` | Trigger process + community detection |
| `check_staleness` | Check if index is behind git HEAD |
| `list_repos` | All indexed repositories |
| `index_repository` | Index or re-index a repository |
| `build_graph` | Build the Neo4j knowledge graph |
| `generate_embeddings` | Generate vector embeddings |
| `analyze_impact` | All callers affected by a function change |
| `trace_type_flow` | Trace data type producers and consumers |
| `get_component_apis` | APIs called by a React component |

**Resources** — `goatlas://repositories`, `goatlas://endpoints`, `goatlas://communities`, `goatlas://processes`, `goatlas://schema`

**Prompts** — `detect_impact`, `generate_map`, `explain_community`

## Claude Code / Cursor Integration

**Cursor** — `~/.cursor/mcp.json`:
```json
{
  "mcpServers": {
    "goatlas": { "command": "goatlas", "args": ["serve"] }
  }
}
```

**Claude Desktop** — `claude_desktop_config.json`:
```json
{
  "mcpServers": {
    "goatlas": { "command": "goatlas", "args": ["serve"] }
  }
}
```

**Claude Code Hooks** — auto-enriches Grep/Glob results and triggers incremental re-indexing on file writes:
```bash
goatlas hooks install /path/to/your/repo
```

## CLI Reference

```bash
goatlas index <path>              # index a repository
goatlas index --force <path>      # force re-index all files
goatlas index --incremental <path> # only changed files since last commit
goatlas embed                     # generate vector embeddings
goatlas embed --force             # re-embed everything
goatlas build-graph               # populate Neo4j knowledge graph
goatlas detect                    # process + community detection
goatlas ask "<question>"          # single-shot AI Q&A
goatlas chat                      # interactive multi-turn session
goatlas serve                     # start MCP server (stdio)
goatlas skills generate <path>    # generate SKILL.md per community
goatlas wiki <output-dir>         # generate Markdown wiki
goatlas check-coverage spec.md    # spec implementation coverage
goatlas hooks install <path>      # install Claude Code hooks
goatlas migrate                   # run database migrations
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `DATABASE_DSN` | `postgres://goatlas:goatlas@localhost:5432/goatlas` | PostgreSQL DSN |
| `QDRANT_URL` | — | Qdrant gRPC endpoint (empty = use pgvector) |
| `NEO4J_URL` | `bolt://localhost:7687` | Neo4j Bolt endpoint |
| `NEO4J_USER` | `neo4j` | Neo4j username |
| `NEO4J_PASS` | `goatlas_neo4j` | Neo4j password |
| `GEMINI_API_KEY` | — | Google Gemini API key |
| `LLM_PROVIDER` | `gemini` | LLM backend: `gemini` \| `ollama` |
| `EMBED_PROVIDER` | `gemini` | Embedding backend: `gemini` \| `ollama` |
| `OLLAMA_URL` | `http://localhost:11434` | Ollama server URL |
| `OLLAMA_MODEL` | `llama3.2` | Ollama chat model |
| `OLLAMA_EMBED_MODEL` | `nomic-embed-text` | Ollama embedding model |
| `REPO_PATH` | cwd | Default repository path |

## Development

```bash
make build        # compile binary
make test         # run tests
make lint         # run linter
make docker-up    # start infrastructure
make docker-down  # stop infrastructure
make migrate      # run migrations
```

## License

MIT
