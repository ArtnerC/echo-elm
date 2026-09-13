package typesystem

// This file holds the FHIR knowledge that does NOT come from the ModelInfo:
// which conversions FHIRHelpers actually defines, and the handful of retrieve
// shapes echo-elm special-cases. The ModelInfo-derived tables — property types,
// list-valued properties, primitive value types and primary code paths — are
// generated into fhir_modelinfo_gen.go by internal/tools/modelinfogen.

// FHIRComplexCoercion maps a FHIR complex type to the FHIRHelpers function that
// converts it to a CQL system type. Primitives and binding types are not listed:
// their conversion follows from the System type of their "value" element, which
// the ModelInfo records (see FHIRPrimitiveValueType).
var FHIRComplexCoercion = map[string]string{
	"Period":          "ToInterval",
	"Range":           "ToInterval",
	"Quantity":        "ToQuantity",
	"Ratio":           "ToRatio",
	"CodeableConcept": "ToConcept",
	"Coding":          "ToCode",
}

// systemTypeCoercion maps the System type of a FHIR primitive's "value" element
// to the FHIRHelpers function that produces it.
var systemTypeCoercion = map[string]string{
	"String":   "ToString",
	"Boolean":  "ToBoolean",
	"Integer":  "ToInteger",
	"Long":     "ToLong",
	"Decimal":  "ToDecimal",
	"DateTime": "ToDateTime",
	"Date":     "ToDate",
	"Time":     "ToTime",
	"Quantity": "ToQuantity",
	"Ratio":    "ToRatio",
	"Concept":  "ToConcept",
	"Code":     "ToCode",
}

// FHIRCoercionFor returns the FHIRHelpers function that converts a value of the
// given FHIR type to its CQL system type, or "" when no conversion applies.
//
// Complex types convert through an explicitly defined FHIRHelpers overload;
// primitives and binding types (FHIR.code, FHIR.AdministrativeGender) convert
// according to the System type of their "value" element.
func FHIRCoercionFor(fhirType string) string {
	if fhirType == "" {
		return ""
	}
	if fn, ok := FHIRComplexCoercion[fhirType]; ok {
		return fn
	}
	return systemTypeCoercion[FHIRPrimitiveValueType[fhirType]]
}

// IsFHIRListProperty reports whether "TypeName.propertyName" is list-valued.
// CQF converts a list of FHIR primitives by lifting the conversion into a
// per-element query rather than wrapping the list, so echo-elm leaves these
// uncoerced instead of emitting a conversion that would be wrong.
func IsFHIRListProperty(sourceType, path string) bool {
	return FHIRListProperty[sourceType+"."+path]
}

// IsFHIRDateTimeType returns true when fhirType is a FHIR type that
// FHIRHelpers converts to a CQL DateTime (point, not interval). This is
// used to decide whether `during` (IncludedIn) should be emitted as In
// instead of IncludedIn.
func IsFHIRDateTimeType(fhirType string) bool {
	return FHIRPrimitiveValueType[fhirType] == "DateTime"
}

// FHIRReferenceChoiceCodeProperty maps FHIR resource type names to the target
// resource type when the code property is a choice of CodeableConcept|Reference(X).
// For these resources, a retrieve using the code property must be expanded into a
// Union of a direct code retrieve and a reference-resolution sub-query.
//
// Key: FHIR resource type (e.g. "MedicationRequest")
// Value: target reference resource type (e.g. "Medication")
var FHIRReferenceChoiceCodeProperty = map[string]string{
	"MedicationRequest":        "Medication",
	"MedicationAdministration": "Medication",
	"MedicationDispense":       "Medication",
	"MedicationStatement":      "Medication",
}

// FHIRPropertyTypeOf returns the declared FHIR type of typeName.property,
// walking the base-type chain.
//
// FHIR elements are inherited: Condition.id is declared on Resource, and
// Encounter.meta on Resource too, so indexing FHIRPropertyType directly answers
// "no such property" for every inherited element. That silently produced no
// result type at all on those nodes, which then collapsed enclosing queries to
// List<Any>.
//
// The chain is finite and acyclic in the ModelInfo, but the walk is bounded
// anyway: a malformed table should degrade to "unknown", not hang.
func FHIRPropertyTypeOf(typeName, property string) (string, bool) {
	for depth := 0; typeName != "" && depth < 32; depth++ {
		if t, ok := FHIRPropertyType[typeName+"."+property]; ok {
			return t, true
		}
		typeName = FHIRBaseType[typeName]
	}
	return "", false
}

// IsFHIRListPropertyOf reports whether typeName.property is list-valued,
// walking the base-type chain the same way FHIRPropertyTypeOf does.
func IsFHIRListPropertyOf(typeName, property string) bool {
	for depth := 0; typeName != "" && depth < 32; depth++ {
		if _, ok := FHIRPropertyType[typeName+"."+property]; ok {
			return FHIRListProperty[typeName+"."+property]
		}
		typeName = FHIRBaseType[typeName]
	}
	return false
}
