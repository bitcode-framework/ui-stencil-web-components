# bc-kanban-card-attachments

File list with upload, image thumbnails, delete. Renders inside the card detail panel of `bc-kanban-board`.

## Basic Usage

```html
<bc-kanban-card-attachments
  card-id="card-001"
  model="attachment"
></bc-kanban-card-attachments>
```

## 4-Layer Data Fetching

Supports: `localData` → `dataFetcher` → `dataSource` → `model`. Filters by `card_id`.

## Props

| Prop | Type | Default | Description |
|------|------|---------|-------------|
| `card-id` | string | `''` | Parent card ID |
| `model` | string | `''` | Attachment model name |
| `data-source` | string | `''` | Attachment API URL |
| `local-data` | string | `undefined` | Static JSON data |
| `filter-by` | string | `'card_id'` | Field to filter by card ID |

## Events

| Event | Detail |
|-------|--------|
| `kanbanAttachmentUpload` | `{ cardId, files: File[] }` |
| `kanbanAttachmentDelete` | `{ cardId, attachmentId }` |

## Methods

| Method | Description |
|--------|-------------|
| `refresh()` | Reload attachments |
