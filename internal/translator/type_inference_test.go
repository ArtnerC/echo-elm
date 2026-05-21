package translator_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/artnerc/echo-elm/internal/translator"
)

func toJSON(t *testing.T, lib interface{}) string {
	t.Helper()
	b, err := json.Marshal(lib)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// Implicit Integer→Decimal promotion: Add/Subtract/Multiply/Modulo with one
// Integer literal and one Decimal literal wraps the Integer in ToDecimal.
func TestImplicitDecimalPromotion_Add(t *testing.T) {
	r := translate(t, `library X version '1'
define V: 1 + 2.5`, func(o *translator.Options) { *o = translator.CQFDefaultOptions() })
	s := toJSON(t, r.Library)
	if !strings.Contains(s, `"ToDecimal"`) {
		t.Errorf("expected ToDecimal wrap on Integer side: %s", s)
	}
	if !strings.Contains(s, `"Add"`) {
		t.Errorf("expected Add operator: %s", s)
	}
}

func TestImplicitDecimalPromotion_Multiply_RightInt(t *testing.T) {
	r := translate(t, `library X version '1'
define V: 2.5 * 3`, func(o *translator.Options) { *o = translator.CQFDefaultOptions() })
	s := toJSON(t, r.Library)
	if !strings.Contains(s, `"ToDecimal"`) {
		t.Errorf("expected ToDecimal wrap on right Integer: %s", s)
	}
}

// No promotion when both sides are the same kind.
func TestNoPromotion_BothIntegers(t *testing.T) {
	r := translate(t, `library X version '1'
define V: 1 + 2`, func(o *translator.Options) { *o = translator.CQFDefaultOptions() })
	s := toJSON(t, r.Library)
	if strings.Contains(s, `"ToDecimal"`) {
		t.Errorf("did not expect ToDecimal: %s", s)
	}
}

// If/Case else-null typing: when else is bare null, infer type from then branch
// and wrap null in As(<inferredType>, null).
func TestConditionalNullTyping_IfElse(t *testing.T) {
	r := translate(t, `library X version '1'
define V: if true then 1 else null`, func(o *translator.Options) { *o = translator.CQFDefaultOptions() })
	s := toJSON(t, r.Library)
	if !strings.Contains(s, `"As"`) {
		t.Errorf("expected As wrap on null else: %s", s)
	}
	if !strings.Contains(s, "Integer") {
		t.Errorf("expected Integer type on As: %s", s)
	}
}

func TestConditionalNullTyping_Case(t *testing.T) {
	r := translate(t, `library X version '1'
define V:
  case
    when true then 'yes'
    else null
  end`, func(o *translator.Options) { *o = translator.CQFDefaultOptions() })
	s := toJSON(t, r.Library)
	if !strings.Contains(s, `"As"`) {
		t.Errorf("expected As wrap on case else null: %s", s)
	}
	if !strings.Contains(s, "String") {
		t.Errorf("expected String as As type: %s", s)
	}
}
