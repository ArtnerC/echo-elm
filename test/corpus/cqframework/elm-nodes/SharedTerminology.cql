// Dependency of CrossLibraryTerminology.cql — a shared terminology library of
// the kind every HEDIS measure includes.
library SharedTerminology version '1.0'

codesystem "SNOMED": 'http://snomed.info/sct'

valueset "Diabetes VS": 'urn:oid:2.16.840.1.113883.3.464.1003.103.12.1001'

code "Diabetes": '73211009' from "SNOMED" display 'Diabetes mellitus'

define "SharedExpression": 1 + 1
