# CRMI 1.0.0 notes

## Source

- Home: <https://hl7.org/fhir/uv/crmi/>

Canonical Resource Management Infrastructure (CRMI) 1.0.0 is a universal-realm STU1 implementation guide based on FHIR R4. It defines common infrastructure for authoring, publishing, packaging, distribution, and implementation of FHIR knowledge artifacts.

## Why it matters

CQF Measures 5.0.0 uses CRMI capability profiles for shareable, computable, executable, and publishable artifacts. A spec-compatible dQM implementation needs CRMI concepts for:

- canonical URL and version management,
- dependency tracking,
- artifact lifecycle states,
- packaging operations,
- distribution bundles,
- computable vs executable artifact capabilities.

## Artifact categories

CRMI treats these as knowledge/artifact resources:

- knowledge artifacts: `ActivityDefinition`, `Library`, `Measure`, `PlanDefinition`, `Questionnaire`,
- terminology artifacts: `ValueSet`, `CodeSystem`, `ConceptMap`, `NamingSystem`,
- conformance artifacts: `ImplementationGuide`, `StructureDefinition`, `GraphDefinition`, `StructureMap`,
- profiled domain artifacts where used as definitional knowledge.

## Implementation requirements

- Resolve canonical resources by `url` and `version`, not local file names.
- Keep artifact identity, publication status, experimental flags, jurisdiction, and publisher metadata intact.
- Package only the artifacts and dependencies needed for the requested capability when possible.
- Distinguish computable packages that include source logic from executable packages that include compiled ELM.
- Preserve dependency order and include recursively required libraries and terminology when packaging requests require it.

