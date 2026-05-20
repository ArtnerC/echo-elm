// lib/cql-locator.ts — CodeMirror locator helpers
import type { EditorView } from '@codemirror/view';
import { StateEffect, StateField } from '@codemirror/state';
import { Decoration, type DecorationSet } from '@codemirror/view';

export interface Locator {
  startLine: number;
  startChar: number;
  endLine: number;
  endChar: number;
}

const highlightEffect = StateEffect.define<{ from: number; to: number } | null>();

const highlightMark = Decoration.mark({ class: 'cm-locator-highlight' });

export const locatorHighlightField = StateField.define<DecorationSet>({
  create: () => Decoration.none,
  update(deco, tr) {
    deco = deco.map(tr.changes);
    for (const e of tr.effects) {
      if (e.is(highlightEffect)) {
        if (e.value === null) {
          deco = Decoration.none;
        } else {
          deco = Decoration.set([highlightMark.range(e.value.from, e.value.to)]);
        }
      }
    }
    return deco;
  },
  provide: (f) => EditorView.decorations.from(f)
});

/** Jump the editor to a CQL locator position and transiently highlight it. */
export function jumpToLocator(view: EditorView, loc: Locator): void {
  const doc = view.state.doc;
  // CQL locators are 1-based line, 0-based char.
  const startLine = Math.min(loc.startLine, doc.lines);
  const endLine = Math.min(loc.endLine, doc.lines);
  const lineStart = doc.line(startLine);
  const lineEnd = doc.line(endLine);
  const from = Math.min(lineStart.from + loc.startChar, lineStart.to);
  const to = Math.min(lineEnd.from + loc.endChar, lineEnd.to);

  view.dispatch({
    effects: [
      highlightEffect.of({ from, to }),
      EditorView.scrollIntoView(from, { y: 'center' })
    ]
  });

  // Clear highlight after 3 s.
  setTimeout(() => {
    view.dispatch({ effects: highlightEffect.of(null) });
  }, 3000);
}
