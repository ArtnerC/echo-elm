# layered/ provenance

Synthetic fixtures for the shapes recorded in `issues/07-post-main-follow-up-inputs.md`,
written for this repository. They mirror construct *shapes* observed in measure content;
no measure text is included.

`FHIRHelpers.cql` is `org/hl7/fhir/FHIRHelpers-4.0.1.cql` from the cqframework
`quick-5.0.0.jar` (Apache-2.0, https://github.com/cqframework/clinical_quality_language),
copied unchanged. It is here — rather than left unresolved as in the other corpus
directories — because the layered-include behavior only reproduces when every include
in the chain resolves, as it does in a real measure bundle.
