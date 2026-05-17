# Spec compatibility checklist

## Compatibility matrix

| Layer | Baseline | Compatibility rule |
| --- | --- | --- |
| CQL language | 1.5.3 | Default parser and semantics target; CQL 2 requires opt-in |
| ELM physical model | ELM R1 from CQL 1.5.3 | Validate XML/JSON output and namespace/content-type rules |
| FHIR base | R4 4.0.1 | Required for current CQL/FHIR quality IGs |
| FHIR CQL packaging | Using CQL with FHIR 2.0.0 | Apply ELM suitability and modelinfo rules |
| Artifact lifecycle/packaging | CRMI 1.0.0 | Use canonical resource capability profiles and dependency packaging rules |
| Quality measures | CQF Measures 5.0.0 | Use Measure/Library packaging and profiles |
| Reporting | DEQM 5.0.0 | Use DEQM report profiles and gaps-in-care structures when applicable |
| Data model | QI-Core 7.0.2 + US Core as package requires | Resolve package dependencies; do not assume latest US Core if measure depends on older alignment |
| Legacy model | QDM 5.6 | Separate adapter and tests |
| Translator tool parity (current) | `org.cqframework:cql-to-elm-jvm:4.8.0` | Reference oracle for CLI semantics and option defaults; see `cqframework-compatibility.md` |
| Translator tool parity (legacy) | `info.cqframework:cql-to-elm:3.29.0` | Compatibility surface for older pinned consumers; same flag set |
| JS runtime reference | cql-execution 3.3.0 | Useful but not complete enough to define full dQM behavior |

## Artifact validation checklist

For every measure package:

1. Resolve all canonical URLs and versions.
2. Confirm the Measure has a primary Library.
3. Confirm all directly referenced criteria names exist in the primary Library or are properly qualified.
4. Decode every Library attachment and validate declared content type.
5. Confirm CQL and ELM are semantically paired, or retranslate from CQL.
6. Check ELM suitability options before executing packaged ELM.
7. Confirm model declarations include versions.
8. Confirm value set and code system references can be resolved or are included.
9. Build data requirements and compare to packaged/declared requirements.
10. Validate resources against FHIR base and IG profiles required by the package.
11. Run test case Bundles when included.
12. Emit OperationOutcome diagnostics for any incompatibility; do not silently skip.

For terminology-specific checks, use `references\implementation\terminology.md`.
For CQFramework `cql-to-elm` CLI / option parity, use
`references\implementation\cqframework-compatibility.md`.

## Folder update process for new spec versions

1. Add `references\specs\<spec>\<new-version>\README.md`.
2. Keep old version folders unchanged.
3. Update `references\README.md` only if the new version becomes the baseline.
4. Update skill references if workflow guidance changes.
5. Add compatibility notes explaining migration risks.

