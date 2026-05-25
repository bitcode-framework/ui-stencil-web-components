# bc-kanban-board

Enterprise kanban board with Trello/Jira-level features — drag-drop cards, column reorder, card detail panel with inline editing, batch save/cancel for assignees and labels, @mention, and full i18n.

## Basic Usage

```html
<bc-kanban-board
  model="task"
  board-title="Sprint Board"
  group-by="stage"
  card-title-field="name"
  columns-config='[{"id":"todo","name":"To Do"},{"id":"progress","name":"In Progress"},{"id":"done","name":"Done"}]'
  comment-model="comment"
  activity-model="activity"
  mention-model="user"
></bc-kanban-board>
```

## 4-Layer Data Fetching

Board supports 4 layers: `localData` → `dataFetcher` → `dataSource` → `model`.

Sub-components (comments, subtasks, attachments, activity) each have their own 4-layer config:

| Prop | Purpose |
|------|---------|
| `model` | Board cards — BitCode API model name |
| `comment-model` | Comments sub-component |
| `subtask-model` | Subtasks sub-component |
| `attachment-model` | Attachments sub-component |
| `activity-model` | Activity timeline sub-component |
| `mention-model` | @Mention user list (also used for assignee picker) |

Each also supports `-data-source`, `-local-data` variants.

## Props

| Prop | Type | Default | Description |
|------|------|---------|-------------|
| `model` | string | `''` | Card data model |
| `board-title` | string | `''` | Board header title |
| `group-by` | string | `'stage'` | Field to group cards into columns |
| `card-title-field` | string | `'name'` | Card title field |
| `card-description-field` | string | `'description'` | Card description field |
| `card-cover-field` | string | `''` | Card cover image field |
| `card-assignees-field` | string | `'assignees'` | Assignees field |
| `card-due-date-field` | string | `'due_date'` | Due date field |
| `card-start-date-field` | string | `'start_date'` | Start date field |
| `card-priority-field` | string | `'priority'` | Priority field |
| `card-labels-field` | string | `'labels'` | Labels field |
| `card-position-field` | string | `'position'` | Sort position field |
| `columns-config` | string (JSON) | `''` | Column definitions: `[{id, name, color?, wip_limit?}]` |
| `allow-add-column` | boolean | `true` | Show "Add Column" button |
| `allow-add-card` | boolean | `true` | Show "Add a card" per column |
| `allow-rename-column` | boolean | `true` | Show rename button on column header |
| `allow-delete-column` | boolean | `false` | Show delete button on column header |
| `allow-reorder-columns` | boolean | `true` | Enable column drag reorder |
| `mention-model` | string | `''` | User model for @mention + assignee picker |
| `mention-data-source` | string | `''` | User API URL for @mention |
| `mention-local-data` | string | `undefined` | Static user list JSON |

## Events

| Event | Detail |
|-------|--------|
| `kanbanCardMove` | `{ cardId, fromColumn, toColumn, toPosition }` |
| `kanbanColumnReorder` | `{ columns: string[] }` |
| `kanbanCardCreate` | `{ column, title, position? }` |
| `kanbanCardUpdate` | `{ cardId, data }` |
| `kanbanCardDelete` | `{ cardId, column }` |
| `kanbanColumnAdd` | `{ name, position? }` |
| `kanbanColumnRename` | `{ columnId, name }` |
| `kanbanColumnDelete` | `{ columnId }` |
| `kanbanError` | `{ message }` |

## Card Detail Inline Editing

Clicking a card opens a slide-in detail panel. Every field is editable with safe cancel/discard behavior:

| Field | Edit Trigger | Save | Cancel |
|-------|-------------|------|--------|
| **Title** | Click title text | Save button or Enter | Cancel button or Escape |
| **Priority** | Click priority badge | Select value from dropdown | ✕ close button in dropdown header |
| **Start Date** | Click date | Pick date (saves) or clear 🗑 | ✕ close button |
| **Due Date** | Click date | Pick date (saves) or clear 🗑 | ✕ close button |
| **Assignees** | Click assignee row | **Done** button (batch) | **Cancel** button (discards all toggles) |
| **Labels** | Click labels row | **Done** button (batch) | **Cancel** button (discards all toggles) |
| **Description** | Click Edit button | **Save** button | **Cancel** button (discards changes) |

### Batch Mode (Assignees & Labels)

Assignees and labels use **batch editing** — toggles only modify a draft state. Changes are only persisted when you click **Done**. Clicking **Cancel** discards all changes. This prevents accidental modifications from miss-clicks.

### Label Creation

While editing labels, clicking a color swatch prompts for a label name and adds it to the draft. Not saved until Done is clicked.

## Features

- **Drag-drop cards** between columns (SortableJS)
- **Column reorder** via drag header
- **Inline dialogs** for add/rename/delete column (no browser `prompt`/`confirm`)
- **Card detail panel** — slide-in side panel with all fields editable
- **Batch edit** for assignees/labels — draft + Done/Cancel pattern
- **@Mention** in comments with dropdown, keyboard nav, and user lookup
- **Start date + Due date** — both editable with clear/cancel
- **Priority badges** — low/medium/high/critical with color coding
- **WIP limit** — visual indicator when column exceeds `wip_limit`
- **Optimistic updates** — local state updated immediately, API in background
- **Full i18n** — all strings use `i18n.t()` with 11 locale keys
- **Dark mode** — CSS custom properties with `[data-bc-theme="dark"]` selectors

## i18n Keys

All keys prefixed with `kanban.`:
`add_column`, `add_card`, `add_a_card`, `card_title`, `column_name`, `new_name`, `delete_column`, `delete_column_confirm`, `delete_column_empty_confirm`, `rename`, `delete`, `menu`, `priority`, `due_date`, `start_date`, `assignees`, `labels`, `description`, `has_description`, `no_description`, `no_priority`, `label_name`, `create_label`, `done`, `comments`, `attachments`, `subtasks`, `checklist`, `add_item`, `add_item_placeholder`, `activity`, `just_now`, `minutes_ago`, `hours_ago`, `send`, `write_comment`, `ctrl_enter_send`, `add_attachment`, `uploading`, `loading`, `mention_users`, `no_users`

## Methods

| Method | Description |
|--------|-------------|
| `refresh()` | Reload columns + cards + reinit sortable |

## CSS Variables

Uses standard `--bc-*` custom properties. Dark mode via `[data-bc-theme="dark"]`.
