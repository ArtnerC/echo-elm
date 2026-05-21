// Package typesystem provides CQL/ELM type constants and a simple type registry.
package typesystem

// System URI constants.
const (
	SystemURI     = "urn:hl7-org:elm-types:r1"
	SystemELMName = "{urn:hl7-org:elm-types:r1}"
)

// System type ELM names.
const (
	TypeBoolean    = SystemELMName + "Boolean"
	TypeInteger    = SystemELMName + "Integer"
	TypeLong       = SystemELMName + "Long"
	TypeDecimal    = SystemELMName + "Decimal"
	TypeString     = SystemELMName + "String"
	TypeDate       = SystemELMName + "Date"
	TypeDateTime   = SystemELMName + "DateTime"
	TypeTime       = SystemELMName + "Time"
	TypeQuantity   = SystemELMName + "Quantity"
	TypeRatio      = SystemELMName + "Ratio"
	TypeAny        = SystemELMName + "Any"
	TypeVocabulary = SystemELMName + "Vocabulary"
	TypeCode       = SystemELMName + "Code"
	TypeConcept    = SystemELMName + "Concept"
	TypeCodeSystem = SystemELMName + "CodeSystem"
	TypeValueSet   = SystemELMName + "ValueSet"
)

// ModelURIByName maps well-known model names to their namespace URIs.
var ModelURIByName = map[string]string{
	"System": SystemURI,
	"FHIR":   "http://hl7.org/fhir",
	"QDM":    "urn:healthit-gov:qdm:v5_6",
	"QICore": "http://hl7.org/fhir/us/qicore",
	"USCore": "http://hl7.org/fhir/us/core",
	"QUICK":  "http://hl7.org/fhir",
}

// InferLiteralType returns the ELM type name for a literal given its kind.
// kind is one of: "boolean", "integer", "long", "decimal", "string", "date",
// "datetime", "time", "quantity", "ratio", "null".
func InferLiteralType(kind string) string {
	switch kind {
	case "boolean":
		return TypeBoolean
	case "integer":
		return TypeInteger
	case "long":
		return TypeLong
	case "decimal":
		return TypeDecimal
	case "string":
		return TypeString
	case "date":
		return TypeDate
	case "datetime":
		return TypeDateTime
	case "time":
		return TypeTime
	case "quantity":
		return TypeQuantity
	case "ratio":
		return TypeRatio
	default:
		return TypeAny
	}
}
