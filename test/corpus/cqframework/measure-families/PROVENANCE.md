# measure-families provenance

Synthetic fixtures written for this repository, moved here from `issues/08-repros/` as the
difference families they reproduce were fixed. Each reproduces a construct *shape* that the
three-release fresh-CQF remeasurement (`issues/08-three-release-fresh-cqf-remeasurement.md`)
found in licensed measure content. They contain no measure text, value set or code system
identifiers, measure logic, or patient data; library, definition and function names are
invented or generic.

`FHIRHelpers.cql` is `org/hl7/fhir/FHIRHelpers-4.0.1.cql` from the cqframework
`quick-5.0.0.jar` (Apache-2.0, https://github.com/cqframework/clinical_quality_language),
copied unchanged. It is byte-identical to `../layered/FHIRHelpers.cql`.

The goldens under `test/goldens/cqf/measure-bundle/measure-families/` are reference ELM from
the pinned CQF 5.0.0 CLI under the `measure-bundle` option profile.

`FunctionOrder.cql`, `SetOperationChains.cql`, `AnyBranchCast.cql` and `AnyBranchHelpers.cql`
were written for this repository after reviewing the issue 08 first-difference appendix. They
reproduce three shapes the appendix showed in licensed content (statement order around forward
function references, regrouped union chains, and an `if` branch calling an included helper that
returns Any) from one-sentence construct descriptions. They contain no measure content.
