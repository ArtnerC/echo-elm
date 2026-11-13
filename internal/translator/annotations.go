package translator

// Source annotations, tag annotations and the CQF-mode annotation containers.

import (
	"encoding/json"
	"strings"

	"github.com/artnerc/echo-elm/internal/ast"
)

// cqfAnnotation returns a json.RawMessage for an empty annotation array
// when in CQF mode, nil otherwise.
func (t *Translator) cqfAnnotation() json.RawMessage {
	if t.opts.CQFMode {
		return json.RawMessage("[]")
	}
	return nil
}

// cqfEmptyArrayField returns an empty JSON array when in CQF mode, nil otherwise.
func (t *Translator) cqfEmptyArrayField() json.RawMessage {
	if t.opts.CQFMode {
		return cqfEmptyArray
	}
	return nil
}

// cqfEmptyArray is the JSON raw empty array used for CQF filter fields.
var cqfEmptyArray = json.RawMessage("[]")

// Translate converts an AST library to an ELM library.
// SetSourceText records the original CQL source so that annotation s-trees
// (EnableAnnotations) can extract the literal text spans for each definition.
// Safe to call before Translate.
func (t *Translator) SetSourceText(src string) {
	t.sourceText = src
	t.lineOffsets = computeLineOffsets(src)
}

// computeLineOffsets returns a slice such that lineOffsets[i] is the byte
// offset of the start of line i+1 in src (lines are 1-based externally).
func computeLineOffsets(src string) []int {
	offsets := []int{0}
	for i := 0; i < len(src); i++ {
		if src[i] == '\n' {
			offsets = append(offsets, i+1)
		}
	}
	return offsets
}

// posToOffset converts a 1-based line/column position to a byte offset in
// sourceText. Returns -1 when the position is invalid. Column is 1-based.
func (t *Translator) posToOffset(line, col int) int {
	if line <= 0 || line > len(t.lineOffsets) {
		return -1
	}
	off := t.lineOffsets[line-1] + (col - 1)
	if off < 0 || off > len(t.sourceText) {
		return -1
	}
	return off
}

// sourceSliceWithLeading returns the source text covering loc, extended
// backward to include any preceding contiguous comment-only lines (no blank
// line separator). Mirrors CQF's annotation source-text capture behaviour.
// Carriage returns are stripped to normalize CRLF input to LF (matching CQF).
func (t *Translator) sourceSliceWithLeading(loc ast.Interval) string {
	s := t.posToOffset(loc.Start.Line, loc.Start.Column)
	e := t.posToOffset(loc.Stop.Line, loc.Stop.Column+1)
	if s < 0 || e < 0 || e < s {
		return ""
	}
	s = t.extendStartForLeadingComments(s)
	return strings.ReplaceAll(t.sourceText[s:e], "\r", "")
}

// extendStartForLeadingComments returns the offset of the earliest comment
// character in the trivia run (whitespace + comments) immediately preceding
// the def's first non-trivia character at start. If the trivia contains no
// comment (just whitespace), returns start unchanged.
//
// This mirrors ANTLR HIDDEN-channel token attachment used by CQF: all hidden
// tokens between the previous non-hidden token and the def's first non-hidden
// token are considered the def's leading trivia. CQF strips the LEADING
// whitespace (before any comment) but preserves intra-trivia whitespace.
func (t *Translator) extendStartForLeadingComments(start int) int {
	src := t.sourceText
	if start <= 0 || start > len(src) {
		return start
	}
	// Forward-scan src[0:start] classifying chars; track the position just
	// after the last code character.
	afterLastCode := 0
	n := start
	i := 0
	for i < n {
		c := src[i]
		// Line comment: // ... \n
		if c == '/' && i+1 < len(src) && src[i+1] == '/' {
			for i < n && src[i] != '\n' {
				i++
			}
			continue
		}
		// Block comment: /* ... */
		if c == '/' && i+1 < len(src) && src[i+1] == '*' {
			i += 2
			for i+1 < len(src) && (src[i] != '*' || src[i+1] != '/') {
				i++
			}
			if i+1 < len(src) {
				i += 2
			}
			continue
		}
		// String literal '...'
		if c == '\'' {
			afterLastCode = i + 1
			i++
			for i < n && src[i] != '\'' {
				if src[i] == '\\' && i+1 < len(src) {
					i++
				}
				afterLastCode = i + 1
				i++
			}
			if i < n {
				afterLastCode = i + 1
				i++
			}
			continue
		}
		// Quoted identifier "..."
		if c == '"' {
			afterLastCode = i + 1
			i++
			for i < n && src[i] != '"' {
				if src[i] == '\\' && i+1 < len(src) {
					i++
				}
				afterLastCode = i + 1
				i++
			}
			if i < n {
				afterLastCode = i + 1
				i++
			}
			continue
		}
		if c == ' ' || c == '\t' || c == '\r' || c == '\n' {
			i++
			continue
		}
		afterLastCode = i + 1
		i++
	}
	// Trivia is src[afterLastCode:start]. Find the first non-whitespace char.
	trim := afterLastCode
	for trim < start {
		c := src[trim]
		if c == ' ' || c == '\t' || c == '\r' || c == '\n' {
			trim++
			continue
		}
		break
	}
	if trim == start {
		return start
	}
	return trim
}

// srcAnnotationJSON returns the JSON for a single source-text Annotation entry
// covering loc (with leading comments extended). Returns nil when the slice
// is empty (e.g. missing source mapping).
func (t *Translator) srcAnnotationJSON(loc ast.Interval) json.RawMessage {
	if !t.opts.EnableAnnotations {
		return nil
	}
	text := t.sourceSliceWithLeading(loc)
	if text == "" {
		return nil
	}
	type sNode struct {
		Value []string `json:"value,omitempty"`
	}
	type sBlock struct {
		S []sNode `json:"s,omitempty"`
	}
	type annJSON struct {
		T    json.RawMessage `json:"t,omitempty"`
		S    *sBlock         `json:"s,omitempty"`
		Type string          `json:"type"`
	}
	ann := []annJSON{{Type: "Annotation", T: t.cqfEmptyArrayField(), S: &sBlock{S: []sNode{{Value: []string{text}}}}}}
	b, err := json.Marshal(ann)
	if err != nil {
		return nil
	}
	return json.RawMessage(b)
}

// mergeAnnotations merges a base annotation (e.g. cqfAnnotation [] or @tag
// annotations) with the source-text annotation. When base is empty/nil, returns
// the source annotation. When source is nil, returns base unchanged.
func (t *Translator) mergeAnnotations(base, src json.RawMessage) json.RawMessage {
	if src == nil {
		return base
	}
	if base == nil || string(base) == "[]" || string(base) == "" {
		return src
	}
	var baseArr, srcArr []json.RawMessage //nolint:prealloc
	if err := json.Unmarshal(base, &baseArr); err != nil {
		return src
	}
	if err := json.Unmarshal(src, &srcArr); err != nil {
		return base
	}
	baseArr = append(baseArr, srcArr...)
	b, err := json.Marshal(baseArr)
	if err != nil {
		return base
	}
	return json.RawMessage(b)
}

// isStandardAnnotationTag reports whether a block-comment @tag is emitted into
// the ELM annotation. CQF does not use an allowlist of spec-defined tag names —
// it emits any tag whose name is a valid CQL identifier and silently drops the
// rest, so @measure-component and @dotted.name produce no tag data while
// @author and @customThing do.
func isStandardAnnotationTag(name string) bool {
	if name == "" {
		return false
	}
	for i, r := range name {
		isAlpha := (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || r == '_'
		isDigit := r >= '0' && r <= '9'
		if !isAlpha && (i == 0 || !isDigit) {
			return false
		}
	}
	return true
}

// buildStatementAnnotation builds the ELM annotation for a statement definition.
// In CQFMode:
//   - @tag data (t:[]) is ALWAYS emitted when block-comment @tag annotations exist,
//     regardless of EnableAnnotations (CQF emits tag metadata unconditionally).
//   - source-text s-tree is emitted only when EnableAnnotations=true.
//
// Outside CQFMode: nothing is emitted unless EnableAnnotations is true.
func (t *Translator) buildStatementAnnotation(s *ast.ExpressionDefinition) json.RawMessage {
	if !t.opts.CQFMode && !t.opts.EnableAnnotations {
		return nil
	}
	compat := t.opts.CompatibilityLevel
	emitTagData := compat == "" || compat >= "1.5"

	type sNode struct {
		R     string   `json:"r,omitempty"`
		Value []string `json:"value,omitempty"`
	}
	type sBlock struct {
		R string  `json:"r,omitempty"`
		S []sNode `json:"s,omitempty"`
	}
	type tagJSON struct {
		Name  string `json:"name"`
		Value string `json:"value"`
	}
	type annJSON struct {
		T    []tagJSON `json:"t,omitempty"`
		S    *sBlock   `json:"s,omitempty"`
		Type string    `json:"type"`
	}

	// Build the single Annotation entry that combines @tag annotations and
	// the source-text s-tree (CQF emits both inside one Annotation map).
	// CQF emits t:[...] tag data unconditionally (not gated on --annotations);
	// s-tree is only emitted when EnableAnnotations=true.
	combined := annJSON{Type: "Annotation"}
	if len(s.Annotations) > 0 && emitTagData {
		// CQF merges all @tag pairs across preceding block comments into one
		// flat tag list on the single Annotation entry.
		// CQF only emits standard CQL spec-defined annotation tag names;
		// custom application-specific tags (e.g. @measure-component:) are not emitted.
		for _, ann := range s.Annotations {
			for _, tag := range ann.Tags {
				if isStandardAnnotationTag(tag.Name) {
					combined.T = append(combined.T, tagJSON{Name: tag.Name, Value: tag.Value})
				}
			}
		}
	}
	if t.opts.EnableAnnotations {
		if text := t.sourceSliceWithLeading(s.Loc()); text != "" {
			combined.S = &sBlock{S: []sNode{{Value: []string{text}}}}
		}
	}

	if combined.T == nil && combined.S == nil {
		return t.cqfAnnotation()
	}
	if t.opts.CQFMode && len(combined.T) == 0 {
		// CQF writes the tag container on every Annotation, even when empty.
		type annWithTagContainer struct {
			T    []tagJSON `json:"t"`
			S    *sBlock   `json:"s,omitempty"`
			Type string    `json:"type"`
		}
		b, err := json.Marshal([]annWithTagContainer{{T: []tagJSON{}, S: combined.S, Type: combined.Type}})
		if err != nil {
			return t.cqfAnnotation()
		}
		return json.RawMessage(b)
	}
	b, err := json.Marshal([]annJSON{combined})
	if err != nil {
		return t.cqfAnnotation()
	}
	return json.RawMessage(b)
}

// withTags adds @tag data to an annotation list, as CQF does for statements:
// into the existing Annotation entry when there is one (alongside its s-tree),
// otherwise as a new entry of its own. Tags are emitted with or without
// --annotations, but not below compatibility level 1.5. Returns raw unchanged
// when there are no tags to add.
//
// Statements already did this; parameters and the library header did not, so
// their documentation tags never reached the ELM. (issues/08 M1, #15)
func (t *Translator) withTags(raw json.RawMessage, anns []ast.CQLAnnotation) json.RawMessage {
	if compat := t.opts.CompatibilityLevel; compat != "" && compat < "1.5" {
		return raw
	}
	type tagJSON struct {
		Name  string `json:"name"`
		Value string `json:"value"`
	}
	var tags []tagJSON
	for _, ann := range anns {
		for _, tag := range ann.Tags {
			if isStandardAnnotationTag(tag.Name) {
				tags = append(tags, tagJSON{Name: tag.Name, Value: tag.Value})
			}
		}
	}
	if len(tags) == 0 {
		return raw
	}
	tagBytes, err := json.Marshal(tags)
	if err != nil {
		return raw
	}

	var entries []map[string]json.RawMessage
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &entries); err != nil {
			return raw
		}
	}
	placed := false
	for _, e := range entries {
		if string(e["type"]) == `"Annotation"` {
			e["t"] = tagBytes
			placed = true
			break
		}
	}
	if !placed {
		entries = append(entries, map[string]json.RawMessage{
			"t":    tagBytes,
			"type": json.RawMessage(`"Annotation"`),
		})
	}
	out, err := json.Marshal(entries)
	if err != nil {
		return raw
	}
	return out
}
