// Package cqlparser contains the ANTLR4-generated CQL parser for Go.
// Grammar source: grammar/cql.g4 (CQL 1.5.3, cqframework v3.29.0)
//
// To regenerate from grammar:
//
//	task generate
//
//go:generate java -jar ../../../tools/antlr/antlr-4.13.2-complete.jar -Dlanguage=Go -package cqlparser -o . ../.././../grammar/cql.g4
package cqlparser
