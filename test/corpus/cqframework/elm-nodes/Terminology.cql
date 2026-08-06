// Regression fixture for issues/03-cqf-parity-gaps.md B3 / B4 / B13.
//
// Covers the declaration and reference forms for code systems, value sets,
// codes and concepts. Pins three details the HEDIS comparison flagged: a
// concept's code list holds bare names (unquoted, and with no "type"
// discriminator, unlike a CodeRef expression), a concept carries its display
// clause, and every ValueSetRef expression is stamped preserve=true.
library Terminology version '1.0'

codesystem "SNOMED": 'http://snomed.info/sct'
codesystem "LOINC": 'http://loinc.org' version '2.74'

valueset "Diabetes VS": 'urn:oid:2.16.840.1.113883.3.464.1003.103.12.1001'
valueset "Versioned VS": 'urn:oid:1.2.3' version '20240101'

code "Diabetes": '73211009' from "SNOMED" display 'Diabetes mellitus'
code "Glucose": '2345-7' from "LOINC" display 'Glucose'

concept "Diabetes Concept": { "Diabetes" } display 'Diabetes concept'
concept "Multi Code Concept": { "Diabetes", "Glucose" } display 'Several codes'

define "CodeSystemReference": "SNOMED"
define "CodeReference": "Diabetes"
define "ConceptReference": "Diabetes Concept"
define "ValueSetReference": "Diabetes VS"

define "CodeInValueSet": "Diabetes" in "Diabetes VS"
define "StringInValueSet": 'some-code' in "Versioned VS"
