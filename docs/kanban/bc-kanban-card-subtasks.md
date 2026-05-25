# bc-kanban-card-subtasks

Checklist with progress bar, toggle/create/delete subtasks. Renders inside the card detail panel of `bc-kanban-board`.

## Basic Usage

```html
<bc-kanban-card-subtasks
  card-id="card-001"
  model="subtask"
></bc-kanban-card-subtasks>
```

## 4-Layer Data Fetching

Supports: `localData` → `dataFetcher` → `dataSource` → `model`. Filters by `card_id`.

## Props

| Prop | Type | Default | Description |
|------|------|---------|-------------|
| `card-id` | string | `''` | Parent card ID |
| `model` | string | `''` | Subtask model name |
| `data-source` | string | `''` | Subtask API URL |
| `local-data` | string | `undefined` | Static JSON data |
| `filter-by` | string | `'card_id'` | Field to filter by card ID |

## Events

| Event | Detail |
|-------|--------|
| `kanbanSubtaskToggle` | `{ cardId, subtaskId, done }` |
| `kanbanSubtaskCreate` | `{ cardId, title }` |
| `kanbanSubtaskDelete` | `{ cardId, subtaskId }` |

## Methods

| Method | Description |
|--------|-------------|
| `refresh()` | Reload subtasks |
