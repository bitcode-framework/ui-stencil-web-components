# @bitcode-framework/ui-components — Documentation

Standalone enterprise Web Component library. Works anywhere — plain HTML, React, Vue, Angular, or any framework. No build step required for consumers.

## Getting Started

```html
<script type="module" src="https://cdn.example.com/bc-components/bc-components.esm.js"></script>

<bc-field-string name="email" label="Email" required clearable hint="We'll never share your email" />
```

See [getting-started.md](getting-started.md) for full setup guide.

## Core Guides

| Guide | Description |
|-------|-------------|
| [bc-setup.md](bc-setup.md) | Global configuration — auth, headers, base URL, theme, validators |
| [theming.md](theming.md) | Theme system — light, dark, system detect, custom themes |
| [data-fetching.md](data-fetching.md) | 4-level data strategy — local, URL, event intercept, custom fetcher |
| [validation.md](validation.md) | 3-level validation — built-in rules, custom JS, server-side |
| [reactivity.md](reactivity.md) | Dependent fields, cascading, cross-field logic |

## Component Reference

### Fields (34)

| Component | Description |
|-----------|-------------|
| [bc-field-string](fields/bc-field-string.md) | Single-line text input |
| [bc-field-text](fields/bc-field-text.md) | Multi-line textarea |
| [bc-field-smalltext](fields/bc-field-smalltext.md) | Small textarea |
| [bc-field-password](fields/bc-field-password.md) | Password input |
| [bc-field-integer](fields/bc-field-integer.md) | Integer number input |
| [bc-field-float](fields/bc-field-float.md) | Float number input |
| [bc-field-decimal](fields/bc-field-decimal.md) | Decimal number input |
| [bc-field-currency](fields/bc-field-currency.md) | Currency input with formatting |
| [bc-field-percent](fields/bc-field-percent.md) | Percentage input |
| [bc-field-date](fields/bc-field-date.md) | Date picker |
| [bc-field-time](fields/bc-field-time.md) | Time picker |
| [bc-field-datetime](fields/bc-field-datetime.md) | DateTime picker |
| [bc-field-duration](fields/bc-field-duration.md) | Duration input |
| [bc-field-checkbox](fields/bc-field-checkbox.md) | Checkbox |
| [bc-field-toggle](fields/bc-field-toggle.md) | Toggle switch |
| [bc-field-select](fields/bc-field-select.md) | Dropdown select |
| [bc-field-radio](fields/bc-field-radio.md) | Radio buttons |
| [bc-field-multicheck](fields/bc-field-multicheck.md) | Multi-checkbox |
| [bc-field-tags](fields/bc-field-tags.md) | Tag input |
| [bc-field-link](fields/bc-field-link.md) | Many2one link field |
| [bc-field-dynlink](fields/bc-field-dynlink.md) | Dynamic link field |
| [bc-field-tableselect](fields/bc-field-tableselect.md) | Table multi-select |
| [bc-field-richtext](fields/bc-field-richtext.md) | Rich text editor (Tiptap) |
| [bc-field-markdown](fields/bc-field-markdown.md) | Markdown editor |
| [bc-field-html](fields/bc-field-html.md) | HTML editor |
| [bc-field-code](fields/bc-field-code.md) | Code editor (CodeMirror) |
| [bc-field-json](fields/bc-field-json.md) | JSON editor |
| [bc-field-file](fields/bc-field-file.md) | File upload |
| [bc-field-image](fields/bc-field-image.md) | Image upload with preview |
| [bc-field-signature](fields/bc-field-signature.md) | Signature pad |
| [bc-field-barcode](fields/bc-field-barcode.md) | Barcode/QR generator |
| [bc-field-color](fields/bc-field-color.md) | Color picker |
| [bc-field-geo](fields/bc-field-geo.md) | Geolocation (Leaflet map) |
| [bc-field-rating](fields/bc-field-rating.md) | Star rating |

### DataTable

| Component | Description |
|-----------|-------------|
| [bc-datatable](datatable/bc-datatable.md) | Full-featured data table |

### Charts (26)

| Component | Description |
|-----------|-------------|
| [bc-chart-bar](charts/bc-chart-bar.md) | Bar chart |
| [bc-chart-line](charts/bc-chart-line.md) | Line chart |
| [bc-chart-pie](charts/bc-chart-pie.md) | Pie/donut chart |
| [bc-chart-area](charts/bc-chart-area.md) | Area chart |
| [bc-chart-scatter](charts/bc-chart-scatter.md) | Scatter/bubble chart |
| [bc-chart-radar](charts/bc-chart-radar.md) | Radar chart |
| [bc-chart-gauge](charts/bc-chart-gauge.md) | Gauge chart |
| [bc-chart-funnel](charts/bc-chart-funnel.md) | Funnel chart |
| [bc-chart-heatmap](charts/bc-chart-heatmap.md) | Heatmap |
| [bc-chart-treemap](charts/bc-chart-treemap.md) | Treemap |
| [bc-chart-sunburst](charts/bc-chart-sunburst.md) | Sunburst chart |
| [bc-chart-candlestick](charts/bc-chart-candlestick.md) | Candlestick/OHLC chart |
| [bc-chart-boxplot](charts/bc-chart-boxplot.md) | Box plot |
| [bc-chart-mixed](charts/bc-chart-mixed.md) | Mixed/combo chart |
| [bc-chart-sankey](charts/bc-chart-sankey.md) | Sankey flow diagram |
| [bc-chart-graph](charts/bc-chart-graph.md) | Network graph |
| [bc-chart-tree](charts/bc-chart-tree.md) | Tree diagram |
| [bc-chart-polar](charts/bc-chart-polar.md) | Polar coordinate chart |
| [bc-chart-parallel](charts/bc-chart-parallel.md) | Parallel coordinates |
| [bc-chart-themeriver](charts/bc-chart-themeriver.md) | Theme river/stream graph |
| [bc-chart-pictorialbar](charts/bc-chart-pictorialbar.md) | Pictorial bar chart |
| [bc-chart-custom](charts/bc-chart-custom.md) | Custom chart (raw ECharts option) |
| [bc-chart-pivot](charts/bc-chart-pivot.md) | Pivot table |
| [bc-chart-kpi](charts/bc-chart-kpi.md) | KPI card |
| [bc-chart-scorecard](charts/bc-chart-scorecard.md) | Scorecard |
| [bc-chart-progress](charts/bc-chart-progress.md) | Progress indicator |

### Layout (10)

bc-row, bc-column, bc-section, bc-tabs, bc-tab, bc-sheet, bc-header, bc-separator, bc-button-box, bc-html-block

### Dialogs (7)

bc-dialog-modal, bc-dialog-confirm, bc-dialog-alert, bc-dialog-prompt, bc-dialog-quickentry, bc-dialog-wizard, bc-toast

| Component | Description |
|-----------|-------------|
| [bc-dialog-confirm](dialogs/bc-dialog-confirm.md) | Confirmation dialog — 4 variants (default, danger, warning, info) |
| [bc-dialog-alert](dialogs/bc-dialog-alert.md) | Alert dialog — 5 variants (default, success, danger, warning, info) |

### Widgets (12)

bc-badge, bc-copy, bc-phone, bc-email, bc-url, bc-progress, bc-statusbar, bc-priority, bc-handle, bc-domain, bc-placeholder, bc-sync-status

### Views (10)

bc-view-list, bc-view-form, bc-view-kanban, bc-view-calendar, bc-view-gantt, bc-view-chart, bc-view-grid, bc-view-pivot, bc-view-gallery, bc-view-tree

| Component | Description |
|-----------|-------------|
| [bc-view-gantt](views/bc-view-gantt.md) | Enterprise Gantt chart — tree hierarchy, drag-and-drop, dependencies, multi-scale timeline |

### Kanban (5)

bc-kanban-board, bc-kanban-card-subtasks, bc-kanban-card-comments, bc-kanban-card-attachments, bc-kanban-card-activity

| Component | Description |
|-----------|-------------|
| [bc-kanban-board](kanban/bc-kanban-board.md) | Enterprise kanban board — drag-drop, column reorder, card detail panel, inline dialogs, @mention, full i18n |
| [bc-kanban-card-comments](kanban/bc-kanban-card-comments.md) | Comments with @mention dropdown, inline file attachments |
| [bc-kanban-card-subtasks](kanban/bc-kanban-card-subtasks.md) | Checklist with progress bar, toggle/create/delete |
| [bc-kanban-card-attachments](kanban/bc-kanban-card-attachments.md) | File list with upload, image thumbnails, delete |
| [bc-kanban-card-activity](kanban/bc-kanban-card-activity.md) | Timeline with relative time formatting |

### Media (8)

bc-viewer-pdf, bc-viewer-image, bc-viewer-document, bc-viewer-youtube, bc-viewer-instagram, bc-viewer-tiktok, bc-viewer-video, bc-viewer-audio
