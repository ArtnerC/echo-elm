// Helper for FluentCrossLibrary: a fluent function reached through an
// expression receiver rather than an include alias.
//
// Deliberately System-typed. A FHIR-typed fluent function would also exercise
// type-name rendering and implicit FHIRHelpers conversion (issues/05 3.1, 3.2),
// which are still open — this fixture is meant to pin libraryName resolution
// alone, so it fails for one reason or none.
library FluentHelpers version '1.0'

define fluent function "Doubled"(n Integer): n * 2
