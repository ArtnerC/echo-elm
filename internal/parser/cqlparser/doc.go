// Package cqlparser contains the ANTLR4-generated CQL parser for Go.
// Grammar source: grammar/cql.g4 (CQL 1.5.3, cqframework v3.29.0)
//
// To regenerate from grammar (downloads ANTLR JAR if needed):
//
//	task generate:grammar
//
// The ANTLR JAR is NOT committed; it is downloaded on demand to avoid
// committing Java binaries that carry a large CVE surface.
//
//go:generate java -jar ../../../tools/antlr/antlr-4.13.2-complete.jar -Dlanguage=Go -package cqlparser -o . ../../../grammar/cql.g4
package cqlparser
