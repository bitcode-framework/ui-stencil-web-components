import { Extension } from '@codemirror/state';
import { HighlightStyle, syntaxHighlighting } from '@codemirror/language';
import { EditorView } from '@codemirror/view';
import { tags as t } from '@lezer/highlight';

const bitcodeEditorHighlightStyle = HighlightStyle.define([
  { tag: [t.keyword, t.modifier, t.controlKeyword, t.moduleKeyword], color: 'var(--bc-editor-keyword, #c792ea)' },
  { tag: [t.string, t.special(t.string)], color: 'var(--bc-editor-string, #ecc48d)' },
  { tag: [t.number, t.integer, t.float, t.bool, t.null], color: 'var(--bc-editor-number, #f78c6c)' },
  { tag: [t.comment, t.lineComment, t.blockComment, t.docComment], color: 'var(--bc-editor-comment, #7f8c98)', fontStyle: 'italic' },
  { tag: [t.variableName, t.propertyName, t.attributeName], color: 'var(--bc-editor-property, #82aaff)' },
  { tag: [t.typeName, t.className, t.namespace, t.labelName], color: 'var(--bc-editor-type, #4ec9b0)' },
  { tag: [t.function(t.variableName), t.function(t.propertyName), t.definition(t.function(t.variableName))], color: 'var(--bc-editor-function, #82aaff)' },
  { tag: [t.operator, t.separator, t.punctuation, t.bracket, t.angleBracket], color: 'var(--bc-editor-punctuation, #89ddff)' },
  { tag: [t.regexp, t.escape], color: 'var(--bc-editor-regex, #addb67)' },
  { tag: [t.invalid], color: 'var(--bc-editor-error, #ff5370)' },
]);

const bitcodeEditorViewTheme = EditorView.theme({
  '&': {
    color: 'var(--bc-editor-fg, var(--bc-text, #111827))',
    backgroundColor: 'var(--bc-editor-bg, var(--bc-bg, #ffffff))',
    fontFamily: 'var(--bc-font-mono, Consolas, monospace)',
    fontSize: '13px',
    lineHeight: '1.65',
  },
  '.cm-scroller': {
    fontFamily: 'inherit',
    overflow: 'auto',
  },
  '.cm-content': {
    caretColor: 'var(--bc-editor-caret, var(--bc-primary, #4f46e5))',
    padding: '12px 0',
  },
  '.cm-line': {
    padding: '0 16px 0 12px',
  },
  '.cm-gutters': {
    backgroundColor: 'var(--bc-editor-gutter-bg, var(--bc-bg-secondary, #f9fafb))',
    color: 'var(--bc-editor-gutter-fg, var(--bc-text-secondary, #6b7280))',
    borderRight: '1px solid var(--bc-editor-gutter-border, var(--bc-border-color, #e5e7eb))',
    minWidth: '44px',
  },
  '.cm-gutter': {
    minWidth: '44px',
  },
  '.cm-lineNumbers .cm-gutterElement': {
    padding: '0 10px 0 12px',
    fontVariantNumeric: 'tabular-nums',
  },
  '.cm-activeLine': {
    backgroundColor: 'var(--bc-editor-active-line, rgba(79, 70, 229, 0.08))',
  },
  '.cm-activeLineGutter': {
    backgroundColor: 'var(--bc-editor-active-gutter, rgba(79, 70, 229, 0.12))',
    color: 'var(--bc-editor-active-gutter-fg, var(--bc-text, #111827))',
  },
  '.cm-selectionBackground, &.cm-focused .cm-selectionBackground, ::selection': {
    backgroundColor: 'var(--bc-editor-selection, rgba(79, 70, 229, 0.22))',
  },
  '.cm-cursor, .cm-dropCursor': {
    borderLeftColor: 'var(--bc-editor-caret, var(--bc-primary, #4f46e5))',
  },
  '&.cm-focused': {
    outline: 'none',
  },
  '.cm-foldPlaceholder': {
    backgroundColor: 'var(--bc-editor-fold-bg, rgba(148, 163, 184, 0.16))',
    border: '1px solid transparent',
    color: 'var(--bc-editor-fold-fg, var(--bc-text-secondary, #6b7280))',
    borderRadius: '4px',
  },
  '.cm-tooltip': {
    border: '1px solid var(--bc-editor-tooltip-border, var(--bc-border-color, #e5e7eb))',
    backgroundColor: 'var(--bc-editor-tooltip-bg, var(--bc-bg-secondary, #f9fafb))',
    color: 'var(--bc-editor-tooltip-fg, var(--bc-text, #111827))',
  },
  '.cm-panels': {
    backgroundColor: 'var(--bc-editor-panel-bg, var(--bc-bg-secondary, #f9fafb))',
    color: 'var(--bc-editor-panel-fg, var(--bc-text, #111827))',
  },
  '.cm-searchMatch': {
    backgroundColor: 'var(--bc-editor-search-match, rgba(245, 158, 11, 0.22))',
    outline: '1px solid rgba(245, 158, 11, 0.35)',
  },
  '.cm-searchMatch.cm-searchMatch-selected': {
    backgroundColor: 'var(--bc-editor-search-match-selected, rgba(245, 158, 11, 0.34))',
  },
  '.cm-matchingBracket, .cm-nonmatchingBracket': {
    backgroundColor: 'rgba(130, 170, 255, 0.14)',
    outline: '1px solid rgba(130, 170, 255, 0.2)',
  },
}, { dark: false });

export function getCodeEditorThemeExtensions(): Extension[] {
  return [
    bitcodeEditorViewTheme,
    syntaxHighlighting(bitcodeEditorHighlightStyle),
  ];
}
