# Go Public API Contract — `pkg/echoelm`

> Frozen at v0.1.0 at end of Phase 6. Breaking changes require a major bump.

## Package layout

```go
package echoelm // import "github.com/<org>/echo-elm/pkg/echoelm"
```

## Core types

```go
// Translator translates CQL libraries to ELM.
type Translator struct { /* opaque */ }

// New constructs a Translator with the supplied options and resolvers.
// Use DefaultOptions, DefaultCLIOptions, or DefaultFHIRArtifactOptions rather
// than relying on the zero value.
func New(opts Options, r Resolvers) (*Translator, error)

// Translate parses and compiles a single CQL source into ELM.
func (t *Translator) Translate(ctx context.Context, src Source) (*Result, error)

// TranslateLibrary resolves and compiles a named library plus its includes.
func (t *Translator) TranslateLibrary(ctx context.Context, id VersionedIdentifier) (*Result, error)
```

## Options

```go
type Options struct {
    CompatibilityLevel string         // "1.5" in v0; "2.0" future experimental
    SignatureLevel     SignatureLevel // None|Differing|Overloads|All
    ErrorLevel         Severity       // Trace|Info|Warning|Error
    ValidateUnits      bool
    Flags              CompilerFlags  // bitset of CqlCompilerOptions.Options
    Format             Format         // XML | JSON | Both
    ReportSelectivity  bool           // 4.x-only knob
}

// DefaultOptions returns modern ELM-generation defaults for echo-elm consumers.
func DefaultOptions() Options

// DefaultFHIRArtifactOptions returns Using CQL with FHIR recommended defaults.
func DefaultFHIRArtifactOptions() Options

// DefaultCLIOptions returns the main echo-elm translate defaults.
func DefaultCLIOptions() Options

// DefaultCQFCLIOptions returns CQFramework-compatible CLI defaults.
func DefaultCQFCLIOptions(profile CQFProfile) Options

type CompilerFlags uint32 // mirrors CqlCompilerOptions.Options exactly
const (
    EnableDateRangeOptimization CompilerFlags = 1 << iota
    EnableAnnotations
    EnableLocators
    EnableResultTypes
    EnableDetailedErrors
    DisableListTraversal
    DisableListDemotion
    DisableListPromotion
    EnableIntervalDemotion
    EnableIntervalPromotion
    DisableMethodInvocation
    RequireFromKeyword
    DisableDefaultModelInfoLoad
)

// PresetDebug and PresetStrict mirror the CLI --debug and --strict aliases.
func PresetDebug() CompilerFlags
func PresetStrict() CompilerFlags
```

## Source & result

```go
type Source struct {
    Path     string         // optional; used for diagnostics & banners
    Content  []byte
    Identity VersionedIdentifier
}

func SourceFromBytes(path string, content []byte) Source
func SourceFromReader(path string, r io.Reader) (Source, error)
func SourcesFromFS(fsys fs.FS, root string) ([]Source, error)
func SourcesFromDir(root string) ([]Source, error)
func SourceFromFHIRLibraryContent(contentType string, data []byte) (Source, error)

type VersionedIdentifier struct {
    System  string
    Name    string
    Version string
}

type Result struct {
    Library     *Library          // ELM model
    XML         []byte            // when Format includes XML
    JSON        []byte            // when Format includes JSON
    Diagnostics []Diagnostic
    Stats       Stats             // timings, counts
}

type Diagnostic struct {
    Severity Severity
    Message  string
    Locator  Locator              // zero value means n/a
    Code     string               // e.g. "CQL-1234"
}

type Locator struct{ StartLine, StartCol, EndLine, EndCol int }
```

## Resolvers

```go
type Resolvers struct {
    LibrarySource       LibrarySource
    ModelInfoProvider   ModelInfoProvider
    TerminologyProvider TerminologyProvider
    OptionsProvider     OptionsProvider
    UcumService         UcumService
}

type LibrarySource interface {
    Open(ctx context.Context, id VersionedIdentifier) (io.ReadCloser, error)
}

func LibrarySourceFromFS(fsys fs.FS, root string) LibrarySource
func LibrarySourceFromDir(root string) LibrarySource

type ModelInfoProvider interface {
    Load(ctx context.Context, model VersionedIdentifier) (*ModelInfo, error)
}

type TerminologyProvider interface {
    Expand(ctx context.Context, vs ValueSetRef) (*ValueSet, error)
    Lookup(ctx context.Context, c CodeRef) (*Code, error)
}

type OptionsProvider interface {
    LoadOptions(ctx context.Context, source Source) (Options, bool, error)
}

type UcumService interface {
    ValidateUnit(ctx context.Context, unit string) error
}
```

## Stability rules

- Adding fields to structs: minor.
- Adding methods to non-sealed interfaces (`LibrarySource`, `ModelInfoProvider`, `TerminologyProvider`): **major** — these are user-implementable. Add new optional interfaces instead.
- Bit values in `CompilerFlags`: never re-numbered.
- Default behaviour is constructor-driven. Do not infer semantics from the
  zero value of `Options`.

## Test surface

Every exported symbol must have at least one `Example*` test and one
behaviour test. Test files live next to the package in `pkg/echoelm/`.
