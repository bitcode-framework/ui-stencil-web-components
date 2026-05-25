# bc-kanban-card-activity

Timeline with dot-style markers, relative time formatting. Read-only activity log. Renders inside the card detail panel of `bc-kanban-board`.

## Basic Usage

```html
<bc-kanban-card-activity
  card-id="card-001"
  model="activity"
></bc-kanban-card-activity>
```

## 4-Layer Data Fetching

Supports: `localData` → `dataFetcher` → `dataSource` → `model`. Filters by `card_id`.

## Props

| Prop | Type | Default | Description |
|------|------|---------|-------------|
| `card-id` | string | `''` | Parent card ID |
| `model` | string | `''` | Activity model name |
| `data-source` | string | `''` | Activity API URL |
| `local-data` | string | `undefined` | Static JSON data |
| `filter-by` | string | `'card_id'` | Field to filter by card ID |

## Methods

| Method | Description |
|--------|-------------|
| `refresh()` | Reload activities |
