package parser

import "github.com/codeatlas/codeatlas/internal/model"

type EntitySeed struct {
	Key           string
	Kind          string
	Name          string
	QualifiedName string
	Range         model.Range
	Metadata      map[string]any
}

type ReferenceSeed struct {
	FromKey    string
	Target     string
	Kind       string
	Range      model.Range
	Confidence float64
	Metadata   map[string]any
}

type ImportSeed struct {
	FromKey   string
	Specifier string
	Range     model.Range
}

type ParseResult struct {
	Path            string
	Language        string
	FileKey         string
	Entities        []EntitySeed
	References      []ReferenceSeed
	Imports         []ImportSeed
	HasSyntaxErrors bool
}
