package elm

import (
	"bytes"
	"encoding/xml"
	"fmt"
)

// XMLOptions controls XML serialization behavior.
type XMLOptions struct {
	Indent bool
}

// MarshalXML serializes an ELM Library to XML.
func MarshalXML(lib *Library, opts XMLOptions) ([]byte, error) {
	var buf bytes.Buffer
	enc := xml.NewEncoder(&buf)
	if opts.Indent {
		enc.Indent("", "  ")
	}

	buf.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")

	if err := writeLibraryXML(enc, lib); err != nil {
		return nil, err
	}
	if err := enc.Flush(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func writeLibraryXML(enc *xml.Encoder, lib *Library) error {
	attrs := []xml.Attr{
		{Name: xml.Name{Local: "xmlns"}, Value: "urn:hl7-org:elm:r1"},
		{Name: xml.Name{Local: "xmlns:t"}, Value: "urn:hl7-org:elm-types:r1"},
		{Name: xml.Name{Local: "xmlns:xsi"}, Value: "http://www.w3.org/2001/XMLSchema-instance"},
		{Name: xml.Name{Local: "xmlns:xsd"}, Value: "http://www.w3.org/2001/XMLSchema"},
	}
	if lib.LocalID != "" {
		attrs = append(attrs, xml.Attr{Name: xml.Name{Local: "localId"}, Value: lib.LocalID})
	}
	if lib.Locator != "" {
		attrs = append(attrs, xml.Attr{Name: xml.Name{Local: "locator"}, Value: lib.Locator})
	}

	start := xml.StartElement{Name: xml.Name{Local: "library"}, Attr: attrs}
	if err := enc.EncodeToken(start); err != nil {
		return err
	}

	// schemaIdentifier
	if lib.SchemaIdentifier != nil {
		si := xml.StartElement{Name: xml.Name{Local: "schemaIdentifier"}, Attr: []xml.Attr{
			{Name: xml.Name{Local: "id"}, Value: lib.SchemaIdentifier.ID},
		}}
		if lib.SchemaIdentifier.Version != "" {
			si.Attr = append(si.Attr, xml.Attr{Name: xml.Name{Local: "version"}, Value: lib.SchemaIdentifier.Version})
		}
		if err := enc.EncodeToken(si); err != nil {
			return err
		}
		if err := enc.EncodeToken(si.End()); err != nil {
			return err
		}
	}

	// identifier
	if lib.Identifier.ID != "" || lib.Identifier.Version != "" {
		id := xml.StartElement{Name: xml.Name{Local: "identifier"}, Attr: []xml.Attr{
			{Name: xml.Name{Local: "id"}, Value: lib.Identifier.ID},
		}}
		if lib.Identifier.Version != "" {
			id.Attr = append(id.Attr, xml.Attr{Name: xml.Name{Local: "version"}, Value: lib.Identifier.Version})
		}
		if err := enc.EncodeToken(id); err != nil {
			return err
		}
		if err := enc.EncodeToken(id.End()); err != nil {
			return err
		}
	}

	// usings
	if lib.Usings != nil && len(lib.Usings.Def) > 0 {
		usingsEl := xml.StartElement{Name: xml.Name{Local: "usings"}}
		if err := enc.EncodeToken(usingsEl); err != nil {
			return err
		}
		for _, u := range lib.Usings.Def {
			attrs := []xml.Attr{
				{Name: xml.Name{Local: "localIdentifier"}, Value: u.LocalIdentifier},
				{Name: xml.Name{Local: "uri"}, Value: u.URI},
			}
			if u.LocalID != "" {
				attrs = append([]xml.Attr{{Name: xml.Name{Local: "localId"}, Value: u.LocalID}}, attrs...)
			}
			if u.Version != "" {
				attrs = append(attrs, xml.Attr{Name: xml.Name{Local: "version"}, Value: u.Version})
			}
			defEl := xml.StartElement{Name: xml.Name{Local: "def"}, Attr: attrs}
			if err := enc.EncodeToken(defEl); err != nil {
				return err
			}
			if err := enc.EncodeToken(defEl.End()); err != nil {
				return err
			}
		}
		if err := enc.EncodeToken(usingsEl.End()); err != nil {
			return err
		}
	}

	// includes
	if lib.Includes != nil && len(lib.Includes.Def) > 0 {
		incEl := xml.StartElement{Name: xml.Name{Local: "includes"}}
		if err := enc.EncodeToken(incEl); err != nil {
			return err
		}
		for _, inc := range lib.Includes.Def {
			attrs := []xml.Attr{
				{Name: xml.Name{Local: "localIdentifier"}, Value: inc.LocalIdentifier},
				{Name: xml.Name{Local: "path"}, Value: inc.Path},
			}
			if inc.LocalID != "" {
				attrs = append([]xml.Attr{{Name: xml.Name{Local: "localId"}, Value: inc.LocalID}}, attrs...)
			}
			if inc.Version != "" {
				attrs = append(attrs, xml.Attr{Name: xml.Name{Local: "version"}, Value: inc.Version})
			}
			defEl := xml.StartElement{Name: xml.Name{Local: "def"}, Attr: attrs}
			if err := enc.EncodeToken(defEl); err != nil {
				return err
			}
			if err := enc.EncodeToken(defEl.End()); err != nil {
				return err
			}
		}
		if err := enc.EncodeToken(incEl.End()); err != nil {
			return err
		}
	}

	// parameters
	if lib.Parameters != nil && len(lib.Parameters.Def) > 0 {
		paramsEl := xml.StartElement{Name: xml.Name{Local: "parameters"}}
		if err := enc.EncodeToken(paramsEl); err != nil {
			return err
		}
		for _, p := range lib.Parameters.Def {
			attrs := xmlDefAttrs(p.LocalID, p.Locator, p.Name, p.AccessLevel)
			defEl := xml.StartElement{Name: xml.Name{Local: "def"}, Attr: attrs}
			if err := enc.EncodeToken(defEl); err != nil {
				return err
			}
			if p.ParameterTypeSpecifier != nil {
				if err := writeTypeSpecifierXML(enc, "parameterTypeSpecifier", p.ParameterTypeSpecifier); err != nil {
					return err
				}
			}
			if err := enc.EncodeToken(defEl.End()); err != nil {
				return err
			}
		}
		if err := enc.EncodeToken(paramsEl.End()); err != nil {
			return err
		}
	}

	// codeSystems
	if lib.CodeSystems != nil && len(lib.CodeSystems.Def) > 0 {
		csEl := xml.StartElement{Name: xml.Name{Local: "codeSystems"}}
		if err := enc.EncodeToken(csEl); err != nil {
			return err
		}
		for _, cs := range lib.CodeSystems.Def {
			attrs := []xml.Attr{
				{Name: xml.Name{Local: "name"}, Value: cs.Name},
				{Name: xml.Name{Local: "id"}, Value: cs.ID},
			}
			if cs.LocalID != "" {
				attrs = append([]xml.Attr{{Name: xml.Name{Local: "localId"}, Value: cs.LocalID}}, attrs...)
			}
			if cs.Version != "" {
				attrs = append(attrs, xml.Attr{Name: xml.Name{Local: "version"}, Value: cs.Version})
			}
			if cs.AccessLevel != "" {
				attrs = append(attrs, xml.Attr{Name: xml.Name{Local: "accessLevel"}, Value: cs.AccessLevel})
			}
			defEl := xml.StartElement{Name: xml.Name{Local: "def"}, Attr: attrs}
			if err := enc.EncodeToken(defEl); err != nil {
				return err
			}
			if err := enc.EncodeToken(defEl.End()); err != nil {
				return err
			}
		}
		if err := enc.EncodeToken(csEl.End()); err != nil {
			return err
		}
	}

	// valueSets
	if lib.ValueSets != nil && len(lib.ValueSets.Def) > 0 {
		vsEl := xml.StartElement{Name: xml.Name{Local: "valueSets"}}
		if err := enc.EncodeToken(vsEl); err != nil {
			return err
		}
		for _, vs := range lib.ValueSets.Def {
			attrs := []xml.Attr{
				{Name: xml.Name{Local: "name"}, Value: vs.Name},
				{Name: xml.Name{Local: "id"}, Value: vs.ID},
			}
			if vs.LocalID != "" {
				attrs = append([]xml.Attr{{Name: xml.Name{Local: "localId"}, Value: vs.LocalID}}, attrs...)
			}
			if vs.Version != "" {
				attrs = append(attrs, xml.Attr{Name: xml.Name{Local: "version"}, Value: vs.Version})
			}
			if vs.AccessLevel != "" {
				attrs = append(attrs, xml.Attr{Name: xml.Name{Local: "accessLevel"}, Value: vs.AccessLevel})
			}
			defEl := xml.StartElement{Name: xml.Name{Local: "def"}, Attr: attrs}
			if err := enc.EncodeToken(defEl); err != nil {
				return err
			}
			if err := enc.EncodeToken(defEl.End()); err != nil {
				return err
			}
		}
		if err := enc.EncodeToken(vsEl.End()); err != nil {
			return err
		}
	}

	// codes
	if lib.Codes != nil && len(lib.Codes.Def) > 0 {
		codesEl := xml.StartElement{Name: xml.Name{Local: "codes"}}
		if err := enc.EncodeToken(codesEl); err != nil {
			return err
		}
		for _, c := range lib.Codes.Def {
			attrs := []xml.Attr{
				{Name: xml.Name{Local: "name"}, Value: c.Name},
				{Name: xml.Name{Local: "id"}, Value: c.ID},
			}
			if c.LocalID != "" {
				attrs = append([]xml.Attr{{Name: xml.Name{Local: "localId"}, Value: c.LocalID}}, attrs...)
			}
			if c.Display != "" {
				attrs = append(attrs, xml.Attr{Name: xml.Name{Local: "display"}, Value: c.Display})
			}
			if c.AccessLevel != "" {
				attrs = append(attrs, xml.Attr{Name: xml.Name{Local: "accessLevel"}, Value: c.AccessLevel})
			}
			defEl := xml.StartElement{Name: xml.Name{Local: "def"}, Attr: attrs}
			if err := enc.EncodeToken(defEl); err != nil {
				return err
			}
			if c.CodeSystem != nil {
				csAttrs := []xml.Attr{
					{Name: xml.Name{Local: "name"}, Value: c.CodeSystem.Name},
					{Name: xml.Name{Local: "xsi:type"}, Value: "CodeSystemRef"},
				}
				if c.CodeSystem.LibraryName != "" {
					csAttrs = append(csAttrs, xml.Attr{Name: xml.Name{Local: "libraryName"}, Value: c.CodeSystem.LibraryName})
				}
				csEl := xml.StartElement{Name: xml.Name{Local: "codeSystem"}, Attr: csAttrs}
				if err := enc.EncodeToken(csEl); err != nil {
					return err
				}
				if err := enc.EncodeToken(csEl.End()); err != nil {
					return err
				}
			}
			if err := enc.EncodeToken(defEl.End()); err != nil {
				return err
			}
		}
		if err := enc.EncodeToken(codesEl.End()); err != nil {
			return err
		}
	}

	// concepts
	if lib.Concepts != nil && len(lib.Concepts.Def) > 0 {
		conceptsEl := xml.StartElement{Name: xml.Name{Local: "concepts"}}
		if err := enc.EncodeToken(conceptsEl); err != nil {
			return err
		}
		for _, con := range lib.Concepts.Def {
			attrs := []xml.Attr{
				{Name: xml.Name{Local: "name"}, Value: con.Name},
			}
			if con.LocalID != "" {
				attrs = append([]xml.Attr{{Name: xml.Name{Local: "localId"}, Value: con.LocalID}}, attrs...)
			}
			if con.Display != "" {
				attrs = append(attrs, xml.Attr{Name: xml.Name{Local: "display"}, Value: con.Display})
			}
			if con.AccessLevel != "" {
				attrs = append(attrs, xml.Attr{Name: xml.Name{Local: "accessLevel"}, Value: con.AccessLevel})
			}
			defEl := xml.StartElement{Name: xml.Name{Local: "def"}, Attr: attrs}
			if err := enc.EncodeToken(defEl); err != nil {
				return err
			}
			for _, cr := range con.Code {
				crAttrs := []xml.Attr{
					{Name: xml.Name{Local: "name"}, Value: cr.Name},
					{Name: xml.Name{Local: "xsi:type"}, Value: "CodeRef"},
				}
				if cr.LibraryName != "" {
					crAttrs = append(crAttrs, xml.Attr{Name: xml.Name{Local: "libraryName"}, Value: cr.LibraryName})
				}
				crEl := xml.StartElement{Name: xml.Name{Local: "code"}, Attr: crAttrs}
				if err := enc.EncodeToken(crEl); err != nil {
					return err
				}
				if err := enc.EncodeToken(crEl.End()); err != nil {
					return err
				}
			}
			if err := enc.EncodeToken(defEl.End()); err != nil {
				return err
			}
		}
		if err := enc.EncodeToken(conceptsEl.End()); err != nil {
			return err
		}
	}

	// statements
	if lib.Statements != nil && len(lib.Statements.Def) > 0 {
		stmtsEl := xml.StartElement{Name: xml.Name{Local: "statements"}}
		if err := enc.EncodeToken(stmtsEl); err != nil {
			return err
		}
		for _, s := range lib.Statements.Def {
			typeName := "ExpressionDef"
			if s.IsFunction {
				typeName = "FunctionDef"
			}
			attrs := xmlDefAttrs(s.LocalID, s.Locator, s.Name, s.AccessLevel)
			if s.Context != "" {
				attrs = append(attrs, xml.Attr{Name: xml.Name{Local: "context"}, Value: s.Context})
			}
			attrs = append(attrs, xml.Attr{
				Name:  xml.Name{Local: "xsi:type"},
				Value: typeName,
			})
			defEl := xml.StartElement{Name: xml.Name{Local: "def"}, Attr: attrs}
			if err := enc.EncodeToken(defEl); err != nil {
				return err
			}
			for _, op := range s.Operand {
				opAttrs := []xml.Attr{{Name: xml.Name{Local: "name"}, Value: op.Name}}
				if op.LocalID != "" {
					opAttrs = append([]xml.Attr{{Name: xml.Name{Local: "localId"}, Value: op.LocalID}}, opAttrs...)
				}
				opEl := xml.StartElement{Name: xml.Name{Local: "operand"}, Attr: opAttrs}
				if err := enc.EncodeToken(opEl); err != nil {
					return err
				}
				if op.OperandTypeSpecifier != nil {
					if err := writeTypeSpecifierXML(enc, "operandTypeSpecifier", op.OperandTypeSpecifier); err != nil {
						return err
					}
				}
				if err := enc.EncodeToken(opEl.End()); err != nil {
					return err
				}
			}
			if s.Expression != nil {
				if err := writeExpressionXML(enc, "expression", s.Expression); err != nil {
					return err
				}
			}
			if err := enc.EncodeToken(defEl.End()); err != nil {
				return err
			}
		}
		if err := enc.EncodeToken(stmtsEl.End()); err != nil {
			return err
		}
	}

	return enc.EncodeToken(start.End())
}

func xmlDefAttrs(localID, locator, name, accessLevel string) []xml.Attr {
	var attrs []xml.Attr
	if localID != "" {
		attrs = append(attrs, xml.Attr{Name: xml.Name{Local: "localId"}, Value: localID})
	}
	if locator != "" {
		attrs = append(attrs, xml.Attr{Name: xml.Name{Local: "locator"}, Value: locator})
	}
	attrs = append(attrs, xml.Attr{Name: xml.Name{Local: "name"}, Value: name})
	if accessLevel != "" {
		attrs = append(attrs, xml.Attr{Name: xml.Name{Local: "accessLevel"}, Value: accessLevel})
	}
	return attrs
}

func writeTypeSpecifierXML(enc *xml.Encoder, elementName string, ts TypeSpecifier) error {
	switch v := ts.(type) {
	case *NamedTypeSpecifier:
		attrs := []xml.Attr{
			{Name: xml.Name{Local: "xsi:type"}, Value: "NamedTypeSpecifier"},
			{Name: xml.Name{Local: "name"}, Value: v.Name},
		}
		el := xml.StartElement{Name: xml.Name{Local: elementName}, Attr: attrs}
		if err := enc.EncodeToken(el); err != nil {
			return err
		}
		return enc.EncodeToken(el.End())

	case *IntervalTypeSpecifier:
		attrs := []xml.Attr{
			{Name: xml.Name{Local: "xsi:type"}, Value: "IntervalTypeSpecifier"},
		}
		el := xml.StartElement{Name: xml.Name{Local: elementName}, Attr: attrs}
		if err := enc.EncodeToken(el); err != nil {
			return err
		}
		if v.PointType != nil {
			if err := writeTypeSpecifierXML(enc, "pointType", v.PointType); err != nil {
				return err
			}
		}
		return enc.EncodeToken(el.End())

	case *ListTypeSpecifier:
		attrs := []xml.Attr{
			{Name: xml.Name{Local: "xsi:type"}, Value: "ListTypeSpecifier"},
		}
		el := xml.StartElement{Name: xml.Name{Local: elementName}, Attr: attrs}
		if err := enc.EncodeToken(el); err != nil {
			return err
		}
		if v.ElementType != nil {
			if err := writeTypeSpecifierXML(enc, "elementType", v.ElementType); err != nil {
				return err
			}
		}
		return enc.EncodeToken(el.End())

	case *TupleTypeSpecifier:
		attrs := []xml.Attr{
			{Name: xml.Name{Local: "xsi:type"}, Value: "TupleTypeSpecifier"},
		}
		el := xml.StartElement{Name: xml.Name{Local: elementName}, Attr: attrs}
		if err := enc.EncodeToken(el); err != nil {
			return err
		}
		for _, elem := range v.Element {
			elemAttrs := []xml.Attr{{Name: xml.Name{Local: "name"}, Value: elem.Name}}
			elemEl := xml.StartElement{Name: xml.Name{Local: "element"}, Attr: elemAttrs}
			if err := enc.EncodeToken(elemEl); err != nil {
				return err
			}
			if elem.ElementType != nil {
				if err := writeTypeSpecifierXML(enc, "elementType", elem.ElementType); err != nil {
					return err
				}
			}
			if err := enc.EncodeToken(elemEl.End()); err != nil {
				return err
			}
		}
		return enc.EncodeToken(el.End())

	case *ChoiceTypeSpecifier:
		attrs := []xml.Attr{
			{Name: xml.Name{Local: "xsi:type"}, Value: "ChoiceTypeSpecifier"},
		}
		el := xml.StartElement{Name: xml.Name{Local: elementName}, Attr: attrs}
		if err := enc.EncodeToken(el); err != nil {
			return err
		}
		for _, choice := range v.Choice {
			if err := writeTypeSpecifierXML(enc, "choice", choice); err != nil {
				return err
			}
		}
		return enc.EncodeToken(el.End())

	default:
		return fmt.Errorf("unknown TypeSpecifier type: %T", ts)
	}
}

func writeExpressionXML(enc *xml.Encoder, elementName string, expr Expression) error {
	typeName := expressionTypeName(expr)
	attrs := []xml.Attr{
		{Name: xml.Name{Local: "xsi:type"}, Value: typeName},
	}

	switch v := expr.(type) {
	case *LiteralNode:
		attrs = append(attrs,
			xml.Attr{Name: xml.Name{Local: "valueType"}, Value: v.ValueType},
			xml.Attr{Name: xml.Name{Local: "value"}, Value: v.Value},
		)
		el := xml.StartElement{Name: xml.Name{Local: elementName}, Attr: attrs}
		if err := enc.EncodeToken(el); err != nil {
			return err
		}
		return enc.EncodeToken(el.End())

	case *ExpressionRefNode:
		attrs = append(attrs, xml.Attr{Name: xml.Name{Local: "name"}, Value: v.Name})
		if v.LibraryName != "" {
			attrs = append(attrs, xml.Attr{Name: xml.Name{Local: "libraryName"}, Value: v.LibraryName})
		}
		el := xml.StartElement{Name: xml.Name{Local: elementName}, Attr: attrs}
		if err := enc.EncodeToken(el); err != nil {
			return err
		}
		return enc.EncodeToken(el.End())

	default:
		el := xml.StartElement{Name: xml.Name{Local: elementName}, Attr: attrs}
		if err := enc.EncodeToken(el); err != nil {
			return err
		}
		return enc.EncodeToken(el.End())
	}
}

func expressionTypeName(expr Expression) string {
	switch v := expr.(type) {
	case *LiteralNode:
		_ = v
		return "Literal"
	case *NullNode:
		_ = v
		return "Null"
	case *ExpressionRefNode:
		_ = v
		return "ExpressionRef"
	case *ParameterRefNode:
		_ = v
		return "ParameterRef"
	case *FunctionRefNode:
		_ = v
		return "FunctionRef"
	case *OperatorExpressionNode:
		return v.Operator
	case *PropertyNode:
		_ = v
		return "Property"
	case *RetrieveNode:
		_ = v
		return "Retrieve"
	case *UnimplementedNode:
		if v.TypeName != "" {
			return v.TypeName
		}
		return "Unimplemented"
	default:
		return "Unknown"
	}
}

