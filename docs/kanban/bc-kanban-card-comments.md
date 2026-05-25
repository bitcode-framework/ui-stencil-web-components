# bc-kanban-card-comments

Comment list with @mention dropdown, inline file attachments, and full i18n. Renders inside the card detail panel of `bc-kanban-board`.

## Basic Usage

```html
<bc-kanban-card-comments
  card-id="card-001"
  model="comment"
  mention-model="user"
></bc-kanban-card-comments>
```

## 4-Layer Data Fetching

Supports: `localData` → `dataFetcher` → `dataSource` → `model`. Filters by `card_id` (configurable via `filter-by`).

## @Mention System

1. Type `@` in the comment textarea
2. Dropdown appears with users from `mention-model`/`mention-data-source`/`mention-local-data`
3. Filter by name or email as you type
4. Navigate with ↑↓ arrows, select with Enter/Tab
5. Inserts `@Username` at cursor position
6. Mentions are highlighted in rendered comments via `.bc-mention` CSS class

## Comment Attachments

- Click 📎 button to attach files before sending
- Pending files shown as removable chips
- Files uploaded alongside the comment body
- Existing attachments displayed below comment text (icon + name + size)

## Props

| Prop | Type | Default | Description |
|------|------|---------|-------------|
| `card-id` | string | `''` | Parent card ID for filtering |
| `model` | string | `''` | Comment model name |
| `data-source` | string | `''` | Comment API URL |
| `local-data` | string | `undefined` | Static JSON data |
| `filter-by` | string | `'card_id'` | Field to filter by card ID |
| `mention-model` | string | `''` | User model for @mention lookup |
| `mention-data-source` | string | `''` | User API URL for @mention |
| `mention-local-data` | string | `undefined` | Static user list JSON |

## Events

| Event | Detail |
|-------|--------|
| `kanbanCommentCreate` | `{ cardId, body, attachments?: File[] }` |
| `kanbanCommentDelete` | `{ cardId, commentId }` |

## Keyboard Shortcuts

- `Ctrl+Enter` / `Cmd+Enter` — Send comment
- `@` — Trigger mention dropdown
- `↑` / `↓` — Navigate mention suggestions
- `Enter` / `Tab` — Select mention
- `Escape` — Close mention dropdown

## Methods

| Method | Description |
|--------|-------------|
| `refresh()` | Reload comments |
