package parser

import (
	"context"

	"github.com/xdotech/goatlas/internal/indexer/domain"
)

// Language identifies a supported programming language.
type Language string

const (
	LangGo         Language = "go"
	LangTypeScript Language = "typescript"
	LangPython     Language = "python"
	LangJava       Language = "java"
)

// FileResult is the unified output of parsing any source file.
type FileResult struct {
	Symbols      []domain.Symbol
	Imports      []domain.Import
	Endpoints    []domain.APIEndpoint
	Connections  []domain.ServiceConnection
	FuncCalls    []domain.FunctionCall
	TypeUsages   []domain.TypeUsage
	IfaceImpls   []domain.InterfaceImpl
	CompAPICalls []domain.ComponentAPICall
	Module       string
	Framework    *FrameworkHint
}

// LanguageHandler is implemented once per language.
type LanguageHandler interface {
	Language() Language
	Extensions() []string
	Parse(ctx context.Context, filePath string, cfg *PatternConfig) (*FileResult, error)
}

// Registry maps file extensions to language handlers.
type Registry struct {
	handlers map[string]LanguageHandler
}

// NewRegistry constructs a registry with all default language handlers registered.
func NewRegistry() *Registry {
	r := &Registry{handlers: make(map[string]LanguageHandler)}
	r.Register(&GoHandler{})
	r.Register(&TypeScriptHandler{})
	r.Register(&PythonHandler{})
	r.Register(&JavaHandler{})
	return r
}

// Register adds a handler for all extensions it declares.
// Panics if two handlers claim the same extension.
func (r *Registry) Register(h LanguageHandler) {
	for _, ext := range h.Extensions() {
		if _, exists := r.handlers[ext]; exists {
			panic("language registry: duplicate extension " + ext)
		}
		r.handlers[ext] = h
	}
}

// HandlerFor returns the handler for a given file extension, or nil.
func (r *Registry) HandlerFor(ext string) LanguageHandler {
	return r.handlers[ext]
}
