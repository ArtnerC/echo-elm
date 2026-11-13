package translator

// Type specifiers: translating declared types and rendering type names.

import (
	"sort"
	"strings"

	"github.com/artnerc/echo-elm/internal/ast"
	"github.com/artnerc/echo-elm/internal/elm"
	"github.com/artnerc/echo-elm/internal/typesystem"
)

// modelURIVersioned returns the ELM namespace URI for a model name + optional version.
// For versioned models like QDM, the URI includes the version (e.g. urn:healthit-gov:qdm:v5_3).
func (t *Translator) modelURIVersioned(name, version string) string {
	if name == "QDM" && version != "" {
		v := strings.ReplaceAll(version, ".", "_")
		return "urn:healthit-gov:qdm:v" + v
	}
	if uri, ok := typesystem.ModelURIByName[name]; ok {
		return uri
	}
	return name
}

// dataNamespaceForModelURI returns the ELM data-type namespace for a model URI.
// FHIR-profiled models (QICore/QUICK/USCore) use the FHIR base namespace for ELM data types.
func dataNamespaceForModelURI(modelName, modelURI string) string {
	if modelName == "QUICK" {
		return "http://hl7.org/fhir"
	}
	return modelURI
}

// templateIDForModel generates the ELM templateId for a given model name, URI and type.
// FHIR uses the StructureDefinition URL pattern. QUICK uses the legacy QI-Core pattern.
func templateIDForModel(modelName, modelURI, typeName string) string {
	switch modelName {
	case "QUICK":
		lc := strings.ToLower(typeName)
		return lc + "-qicore-qicore-" + lc
	default:
		if modelURI == "http://hl7.org/fhir" {
			return "http://hl7.org/fhir/StructureDefinition/" + typeName
		}
	}
	return ""
}

// qualifyDataType returns the model-namespace-qualified data type and optional templateId
// for a retrieve expression. The raw type name may be unqualified ("Encounter") or
// model-prefixed ("FHIR.Encounter"). When the primary declared model is FHIR, the
// templateId follows the StructureDefinition URL pattern.
func (t *Translator) qualifyDataType(rawType string) (dataType, templateID string) {
	if strings.HasPrefix(rawType, "{") {
		return rawType, "" // already namespace-qualified
	}

	// The first segment is a model alias only if a model is declared under that
	// name. Otherwise the whole dotted name is a type in the primary model: an
	// unqualified backbone type such as Claim.Item, whose "Claim" is part of the
	// type name, not a model.
	modelAlias, typeName := "", rawType
	if before, after, ok := strings.Cut(rawType, "."); ok {
		if _, isModel := t.modelsByAlias[before]; isModel {
			modelAlias, typeName = before, after
		}
	}

	var (
		modelName string
		uri       string
	)
	if modelAlias != "" {
		uri = t.modelsByAlias[modelAlias]
		modelName = modelAlias
	} else {
		modelName = t.primaryModelName
		uri = t.primaryModelURI
	}

	if uri == "" {
		return rawType, ""
	}

	// For models that profile FHIR (QUICK), use the FHIR data type namespace.
	dataNS := dataNamespaceForModelURI(modelName, uri)
	dt := "{" + dataNS + "}" + typeName
	templateID = templateIDForModel(modelName, dataNS, typeName)
	return dt, templateID
}

// declResultType returns the fixed system type CQF records on a terminology
// declaration under --result-types, or "" when result types are off.
func (t *Translator) declResultType(typeName string) string {
	if !t.opts.EnableResultTypes {
		return ""
	}
	return typeName
}

// translateTypeSpecifier converts an AST TypeSpecifier to an ELM TypeSpecifier.
func (t *Translator) translateTypeSpecifier(ts ast.TypeSpecifier) elm.TypeSpecifier {
	if ts == nil {
		return nil
	}
	ann := t.cqfAnnotation()
	switch v := ts.(type) {
	case *ast.NamedTypeSpecifier:
		name := v.Name
		if v.Qualifier != "" {
			name = v.Qualifier + "." + v.Name
		}
		// Model types render as {modelUri}LocalName, the same form retrieves use.
		// resolveTypeName only maps System names, so an alias-qualified
		// `FHIR.Encounter` or a bare `Encounter` would otherwise reach the ELM
		// as written and give one type two more spellings.
		name = t.qualifyTypeName(resolveTypeName(name))
		nts := &elm.NamedTypeSpecifier{Annotation: ann, Name: name}
		if t.opts.EnableLocators {
			nts.Locator = locatorStr(v.Loc())
		}
		return nts
	case *ast.IntervalTypeSpecifier:
		its := &elm.IntervalTypeSpecifier{
			Annotation: ann,
			PointType:  t.translateTypeSpecifier(v.PointType),
		}
		if t.opts.EnableLocators {
			its.Locator = locatorStr(v.Loc())
		}
		return its
	case *ast.ListTypeSpecifier:
		lts := &elm.ListTypeSpecifier{
			Annotation:  ann,
			ElementType: t.translateTypeSpecifier(v.ElementType),
		}
		if t.opts.EnableLocators {
			lts.Locator = locatorStr(v.Loc())
		}
		return lts
	case *ast.TupleTypeSpecifier:
		tts := &elm.TupleTypeSpecifier{Annotation: ann}
		if t.opts.EnableLocators {
			tts.Locator = locatorStr(v.Loc())
		}
		for _, elem := range v.Elements {
			ted := &elm.TupleElementDefinition{
				Annotation:  t.cqfAnnotation(),
				Name:        elem.Name,
				ElementType: t.translateTypeSpecifier(elem.Type),
			}
			if t.opts.EnableLocators {
				ted.Locator = locatorStr(elem.Loc())
			}
			tts.Element = append(tts.Element, ted)
		}
		return tts
	case *ast.ChoiceTypeSpecifier:
		cts := &elm.ChoiceTypeSpecifier{Annotation: ann}
		if t.opts.EnableLocators {
			cts.Locator = locatorStr(v.Loc())
		}
		for _, ct := range v.Types {
			cts.Choice = append(cts.Choice, t.translateTypeSpecifier(ct))
		}
		sortChoiceSpecifiers(cts.Choice)
		return cts
	default:
		return &elm.NamedTypeSpecifier{Annotation: ann, Name: typesystem.TypeAny}
	}
}

// sortChoiceSpecifiers orders choice alternatives the way CQF does, which is by
// the Java toString() of each type: named types print as Model.Name and the
// collection types in lower case, as interval<…> and list<…>. That puts
// FHIR.Age before FHIR.dateTime before System.Integer before interval<…> before
// list<…> — which is why a declared Choice<FHIR.dateTime, FHIR.Period, …> came
// back reordered. Declaration order is not preserved. (issues/06)
func sortChoiceSpecifiers(specs []elm.TypeSpecifier) {
	sort.SliceStable(specs, func(i, j int) bool {
		return choiceSortKey(specs[i]) < choiceSortKey(specs[j])
	})
}

// choiceSortKey renders a type specifier the way the CQF type model prints it.
func choiceSortKey(s elm.TypeSpecifier) string {
	switch v := s.(type) {
	case *elm.NamedTypeSpecifier:
		return modelQualifiedName(v.Name)
	case *elm.ListTypeSpecifier:
		return "list<" + choiceSortKey(v.ElementType) + ">"
	case *elm.IntervalTypeSpecifier:
		return "interval<" + choiceSortKey(v.PointType) + ">"
	case *elm.ChoiceTypeSpecifier:
		parts := make([]string, len(v.Choice))
		for i, c := range v.Choice {
			parts[i] = choiceSortKey(c)
		}
		return "choice<" + strings.Join(parts, ",") + ">"
	case *elm.TupleTypeSpecifier:
		return "tuple{}"
	}
	return ""
}

// modelQualifiedName renders {uri}Name as Model.Name for the two models CQF
// names; any other namespace keeps its URI, which still sorts consistently.
func modelQualifiedName(name string) string {
	if !strings.HasPrefix(name, "{") {
		return name
	}
	end := strings.Index(name, "}")
	if end < 0 {
		return name
	}
	uri, local := name[1:end], name[end+1:]
	switch uri {
	case typesystem.SystemURI:
		return "System." + local
	case "http://hl7.org/fhir":
		return "FHIR." + local
	}
	return uri + "." + local
}

// knownTS returns ts, or nil when it carries no information — unresolved, or
// Any. A null is only cast to a type something else actually established.
func knownTS(ts typeSpec) typeSpec {
	if ts == nil {
		return nil
	}
	if n, ok := ts.(namedTS); ok && (n.name == "" || n.name == anyTS.name) {
		return nil
	}
	return ts
}

// resolveTypeName maps a bare CQL type name to its ELM-qualified form.
func resolveTypeName(name string) string {
	switch name {
	case "Boolean":
		return typesystem.TypeBoolean
	case "Integer":
		return typesystem.TypeInteger
	case "Long":
		return typesystem.TypeLong
	case "Decimal":
		return typesystem.TypeDecimal
	case "String":
		return typesystem.TypeString
	case "Date":
		return typesystem.TypeDate
	case "DateTime":
		return typesystem.TypeDateTime
	case "Time":
		return typesystem.TypeTime
	case "Quantity":
		return typesystem.TypeQuantity
	case "Ratio":
		return typesystem.TypeRatio
	case "Any":
		return typesystem.TypeAny
	case "Code":
		return typesystem.TypeCode
	case "Concept":
		return typesystem.TypeConcept
	case "Vocabulary":
		return typesystem.TypeVocabulary
	}
	// Already qualified (e.g. "FHIR.Patient") or unknown — keep as-is.
	return name
}

// systemClassTypes are the System structured types an Instance selector can
// name without a qualifier; their class type is always the System one.
var systemClassTypes = map[string]bool{
	"Code": true, "Concept": true, "Quantity": true, "Ratio": true,
	"ValueSet": true, "CodeSystem": true, "Vocabulary": true,
}
