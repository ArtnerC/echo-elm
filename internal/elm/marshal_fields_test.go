package elm

import (
	"encoding/json"
	"os"
	"reflect"
	"regexp"
	"strings"
	"testing"
)

// expressionNodeTypes lists every concrete Expression implementation. A node
// added without an entry here is caught by TestAllExpressionTypesCovered.
func expressionNodeTypes() []Expression {
	return []Expression{
		&OperandRefNode{}, &LiteralNode{}, &NullNode{}, &ExpressionRefNode{},
		&ParameterRefNode{}, &ValueSetRefNode{}, &InValueSetNode{}, &CalculateAgeNode{},
		&ToConceptNode{}, &SplitNode{}, &LastNode{}, &CodeSystemRefNode{},
		&CodeRefNode{}, &ConceptRefNode{}, &FunctionRefNode{}, &OperatorExpressionNode{},
		&PropertyNode{}, &RetrieveNode{}, &SingletonFromNode{}, &QuantityNode{},
		&RatioNode{}, &UnimplementedNode{}, &UnaryExpressionNode{}, &AggregateExpressionNode{},
		&NamedOperatorExpressionNode{}, &DateNode{}, &DateTimeNode{}, &TimeNode{},
		&IfNode{}, &CaseNode{}, &IsNode{}, &AsNode{}, &ConvertNode{}, &IntervalNode{},
		&ListNode{}, &TupleNode{}, &InstanceNode{}, &CodeNode{}, &ConceptNode{},
		&QueryThisRefNode{}, &IdentifierRefNode{}, &AliasRefNode{}, &LetRefNode{},
		&ExternalConstantNode{}, &PrecisionOperatorNode{}, &UnaryPrecisionOperatorNode{},
		&QueryNode{}, &MinMaxValueNode{},
	}
}

// TestMarshalPreservesResultType guards against a whole class of defect: a node
// whose MarshalJSON hand-builds its output silently drops any field added to the
// struct later. Three nodes did exactly that with resultTypeName. Every
// expression node must round-trip a result type through its own marshaller.
func TestMarshalPreservesResultType(t *testing.T) {
	const wantName = "{urn:hl7-org:elm-types:r1}Integer"

	for _, node := range expressionNodeTypes() {
		typeName := reflect.TypeOf(node).Elem().Name()
		t.Run(typeName, func(t *testing.T) {
			SetResultType(node, wantName, nil)
			if !HasResultType(node) {
				t.Fatalf("%s: SetResultType did not take effect", typeName)
			}
			b, err := json.Marshal(node)
			if err != nil {
				t.Fatalf("%s: marshal: %v", typeName, err)
			}
			var got map[string]interface{}
			if err := json.Unmarshal(b, &got); err != nil {
				t.Fatalf("%s: unmarshal %s: %v", typeName, b, err)
			}
			if got["resultTypeName"] != wantName {
				t.Errorf("%s: resultTypeName lost in MarshalJSON\n  got: %s", typeName, b)
			}

			// The structured form has to survive too.
			SetResultType(node, "", &ListTypeSpecifier{ElementType: &NamedTypeSpecifier{Name: wantName}})
			b, err = json.Marshal(node)
			if err != nil {
				t.Fatalf("%s: marshal with specifier: %v", typeName, err)
			}
			if !strings.Contains(string(b), `"resultTypeSpecifier"`) {
				t.Errorf("%s: resultTypeSpecifier lost in MarshalJSON\n  got: %s", typeName, b)
			}
		})
	}
}

// TestAllExpressionTypesCovered keeps expressionNodeTypes honest: it must list
// one instance per Expression implementation in this package.
func TestAllExpressionTypesCovered(t *testing.T) {
	listed := map[string]bool{}
	for _, n := range expressionNodeTypes() {
		listed[reflect.TypeOf(n).Elem().Name()] = true
	}
	// Names are declared in model.go via `func (*X) isExpression() {}`; this
	// mirrors that list so a new node type fails here rather than silently
	// escaping the marshal guard above.
	for _, name := range declaredExpressionTypeNames(t) {
		if !listed[name] {
			t.Errorf("%s implements Expression but is not covered by TestMarshalPreservesResultType", name)
		}
	}
}

// declaredExpressionTypeNames scans model.go for isExpression implementations.
// Reading the source keeps the list authoritative without codegen.
func declaredExpressionTypeNames(t *testing.T) []string {
	t.Helper()
	src, err := os.ReadFile("model.go")
	if err != nil {
		t.Fatalf("read model.go: %v", err)
	}
	re := regexp.MustCompile(`func \(\*([A-Za-z]+)\) isExpression\(\)`)
	var names []string
	for _, m := range re.FindAllStringSubmatch(string(src), -1) {
		names = append(names, m[1])
	}
	if len(names) == 0 {
		t.Fatal("no Expression implementations found in model.go")
	}
	return names
}
