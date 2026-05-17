# Agent: echo-elm translator

You are working on **echo-elm**, the Go CQL→ELM translator. Read
`AGENTS.md` at the repo root first; this file narrows scope to
translator-core work and inlines the rules you need most often.

## Scope owned by this agent

- ANTLR4 CQL 1.5.3 grammar binding (Go target) and parser.
- Symbol tables, scopes, name resolution.
- CQL type system: type lattice, generics, conversions, promotion /
  demotion, choice types, intervals/lists/tuples/class types, overload
  resolution honoring `SignatureLevel`.
- ELM AST and builder. Emit `resultTypeName` / `resultTypeSpecifier`,
  signatures, annotations, locators per options.
- Serializers: ELM XML and ELM JSON.
- Resolver interfaces (`LibrarySourceProvider`, `ModelInfoProvider`,
  `TerminologyProvider`, `UcumService`, `OptionsProvider`) and their
  default filesystem-backed implementations.
- Diagnostics with stable severity + locators.
- `pkg/echoelm` public Go API surface and stability.

## Out of scope (belongs in echo-qm or another module)

- Executing ELM, computing measure populations, expanding value sets,
  fetching FHIR data, or producing MeasureReports.
- Building FHIR Bundles or rendering DEQM artifacts.

## Inputs you accept (default Go API)

- `text/cql` from a `[]byte`, `io.Reader`, file path, or directory.
- FHIR `Library.content` with media type `text/cql`.
- Canonical URL resolved through a `LibrarySourceProvider`.

## Outputs you produce

- ELM XML (default) or JSON, conformant to the CQL 1.5.3 physical
  representation, with the ELM `Library` header recording: translator
  version, compatibility level, signature level, and the semantically
  relevant options (so downstream consumers can check ELM suitability).
- A structured `Diagnostics` collection with severity, message, and
  `[startLine:startChar, endLine:endChar]` locators.

## Hard rules

- Default `compatibilityLevel = "1.5"`. Never let CQL 2.0.0-ballot
  features affect 1.5 output; CQL 2 support is future experimental work.
- Default options for FHIR artifacts match
  `CqlCompilerOptions.defaultOptions()` — see `AGENTS.md` for the list.
- `SignatureLevel` defaults: library API and modern `translate` `Overloads`;
  `echo-elm cqf translate` `None` for CQFramework parity.
- Treat the ModelInfo strategy (profile-informed vs. derived) as part
  of artifact identity. Record it; never silently mix.
- Never silently fall back when a library, modelinfo, value set, code
  system, code, or unit cannot be resolved — emit a diagnostic and
  refuse to emit ELM (matches CQFramework behavior).
- Preserve library identifier, version, and `using`/`include` aliases
  byte-exact from source.
- Refuse to silently pick an overload when ambiguity exists; emit a
  diagnostic.

## References to keep open

- `references\implementation\echo-elm.md` — architecture + acceptance.
- `references\specs\cql\1.5.3\README.md` — CQL/ELM baseline.
- `references\specs\using-cql-with-fhir\2.0.0\README.md` — option
  defaults, ELM suitability, ModelInfo guidance.
- `references\implementation\cqframework-compatibility.md` — CLI flag
  parity.
- `references\implementation\terminology.md` — terminology resolver
  contract (echo-elm exposes the interface; resolution itself is
  pluggable).
