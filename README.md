# @bitcode-framework/ui-components

Enterprise-grade Stencil Web Components for business applications. 127 components covering forms, charts, data tables, layout, dialogs, media viewers, kanban board, and widgets. Works in any HTML page, no framework required.

[![MIT License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![npm version](https://img.shields.io/npm/v/@bitcode-framework/ui-components.svg)](https://www.npmjs.com/package/@bitcode-framework/ui-components)
[![Live Demo](https://img.shields.io/badge/demo-live-brightgreen.svg)](https://bitcode-framework.github.io/ui-stencil-web-components/demo/)

**[Live Demo →](https://bitcode-framework.github.io/ui-stencil-web-components/demo/)** — interactive component gallery with all 127 components.

## What is @bitcode-framework/ui-components?

127 Web Components built with [Stencil.js](https://stenciljs.com/). They compile to standard Custom Elements, so they run anywhere HTML runs: plain pages, React, Vue, Angular, Svelte, or any framework that renders to the DOM.

Built for the [BitCode](https://github.com/bitcode-framework) low-code platform, but fully standalone. No BitCode server or runtime dependency. Drop a `<script>` tag and start using components.

**Framework-agnostic. No build step for consumers. Tree-shakeable.**

## Component Overview

| Category | Count | Components |
|----------|-------|------------|
| Fields | 35 | text, textarea, small text, password, integer, float, decimal, currency, percent, date, time, datetime, duration, checkbox, toggle, select, radio, multi-checkbox, tags, many2one link, dynamic link, table select, morph, rich text (Tiptap), markdown, HTML editor, code (CodeMirror), JSON, file upload, image upload, signature pad, barcode/QR, color picker, geolocation (Leaflet), rating |
| Charts | 26 | bar, line, pie/donut, area, scatter/bubble, radar, gauge, funnel, heatmap, treemap, sunburst, candlestick/OHLC, boxplot, mixed/combo, sankey, network graph, tree, polar, parallel coordinates, theme river, pictorial bar, custom (raw ECharts), pivot table, KPI card, scorecard, progress |
| DataTable | 3 | data table with server-side pagination/sorting/filtering, filter builder, lookup modal |
| Views | 10 | form, list, kanban, calendar, gantt, tree, map, activity, report, editor |
| Layout | 10 | row, column, section, tabs, tab, sheet, header, separator, button-box, html-block |
| Dialogs | 5 | modal, confirm, quick-entry, wizard, toast |
| Widgets | 19 | badge, copy-to-clipboard, phone, email, URL, progress, status bar, priority, drag handle, domain, sync status, PDF viewer, image viewer, document viewer, YouTube embed, Instagram embed, TikTok embed, video player, audio player |
| Search | 4 | search, filter bar, filter panel, favorites |
| Social | 3 | activity feed, chatter, timeline |
| Print | 3 | print, export, report link |
| Table | 1 | child table (editable sub-table for forms) |

All 26 charts are powered by [Apache ECharts](https://echarts.apache.org/). Pass raw ECharts options to `bc-chart-custom` for anything not covered by the dedicated chart components.

## Quick Start

### CDN (no build step)

```html
<!DOCTYPE html>
<html>
<head>
  <script type="module" src="https://unpkg.com/@bitcode-framework/ui-components/dist/bc-components/bc-components.esm.js"></script>
</head>
<body>
  <bc-field-string name="email" label="Email" required placeholder="you@example.com"></bc-field-string>
  <bc-field-select name="country" label="Country"
    options='[{"label":"Indonesia","value":"ID"},{"label":"Japan","value":"JP"}]'>
  </bc-field-select>
</body>
</html>
```

### NPM

```bash
npm install @bitcode-framework/ui-components
```

```javascript
import { defineCustomElements } from '@bitcode-framework/ui-components/loader';
defineCustomElements();
```

### Global Configuration (optional)

Components work with zero config. When you need API integration, auth, or theming, configure once:

```javascript
import { BcSetup } from '@bitcode-framework/ui-components';

BcSetup.configure({
  baseUrl: '/api',
  auth: { type: 'bearer', token: () => localStorage.getItem('jwt') },
  theme: 'system',
  locale: 'en'
});
```

Config can also be set via meta tags for server-rendered pages:

```html
<meta name="bc-base-url" content="/api">
<meta name="bc-auth-token" content="eyJhbG...">
<meta name="bc-theme" content="dark">
```

## Usage Examples

### Form Fields

```html
<bc-field-string name="name" label="Full Name" required clearable></bc-field-string>
<bc-field-integer name="age" label="Age" min="0" max="150"></bc-field-integer>
<bc-field-currency name="price" label="Price" currency="USD"></bc-field-currency>
<bc-field-date name="birthday" label="Birthday" format="YYYY-MM-DD"></bc-field-date>
<bc-field-toggle name="active" label="Active"></bc-field-toggle>
<bc-field-rating name="score" label="Rating" max="5"></bc-field-rating>
<bc-field-geo name="location" label="Location" lat="-6.2" lng="106.8"></bc-field-geo>
```

### API-Connected Select

```html
<bc-field-select
  name="city"
  label="City"
  data-source="/api/cities"
  data-text-field="name"
  data-value-field="id"
  searchable
>
</bc-field-select>
```

### DataTable

```html
<bc-datatable
  columns='[
    {"field":"name","header":"Name","sortable":true},
    {"field":"email","header":"Email"},
    {"field":"status","header":"Status","filterable":true}
  ]'
  data-source="/api/users"
  server-side
  pagination
  page-size="20"
  selectable
>
</bc-datatable>
```

### Charts

```html
<bc-chart-bar
  data='[{"category":"Q1","revenue":120000},{"category":"Q2","revenue":180000}]'
  x-field="category"
  y-field="revenue"
  title="Quarterly Revenue"
></bc-chart-bar>

<bc-chart-kpi title="Active Users" value="12483" trend="up" trend-value="12.5%"></bc-chart-kpi>

<bc-chart-pie
  data='[{"name":"Desktop","value":60},{"name":"Mobile","value":40}]'
  name-field="name"
  value-field="value"
></bc-chart-pie>
```

### Dark Mode

Apply to any element or the body:

```html
<body data-bc-theme="dark">
```

Or auto-detect system preference:

```javascript
BcSetup.configure({ theme: 'system' });
```

## Features

### 4-Level Data Fetching

Components that load data support four strategies, controlled via attributes or `BcSetup`:

1. **Local data** - pass data directly via attributes (`options`, `data`, `columns`)
2. **URL endpoint** - set `data-source="/api/users"` and the component fetches automatically
3. **Event intercept** - listen for `bcDataFetch` to modify requests before they go out
4. **Custom fetcher** - register a function in `BcSetup` for full control over the request/response cycle

### 3-Level Validation

1. **Built-in rules** - `required`, `min`, `max`, `minlength`, `maxlength`, `pattern`, `email`, `url`, and more via attributes
2. **Custom JS validators** - register named validators via `BcSetup.registerValidator()`
3. **Server-side** - components send validation requests and display server errors

```javascript
BcSetup.registerValidator('no-competitor', async (value) => {
  if (String(value).endsWith('@competitor.com')) return 'Competitor emails not allowed';
  return null;
});
```

```html
<bc-field-string name="email" label="Email" validate="required email no-competitor"></bc-field-string>
```

### Theming

Four theme modes: `light`, `dark`, `system` (auto-detect OS preference), and custom. All colors use CSS custom properties, so you override at any granularity.

```css
:root {
  --bc-primary: #6366f1;
  --bc-border-radius: 8px;
  --bc-font-family: 'Inter', sans-serif;
}
```

### i18n

11 languages built in. Set via `BcSetup.configure({ locale: 'ja' })` or the `locale` attribute on individual components.

| Code | Language |
|------|----------|
| `en` | English |
| `id` | Bahasa Indonesia |
| `ar` | Arabic |
| `de` | German |
| `es` | Spanish |
| `fr` | French |
| `ja` | Japanese |
| `ko` | Korean |
| `pt-BR` | Portuguese (Brazil) |
| `ru` | Russian |
| `zh-CN` | Chinese (Simplified) |

### Reactivity

Fields can react to changes in other fields. Register rules that run when a field value changes:

```javascript
BcSetup.reactivity({
  'customer_type': (value, form) => {
    if (value === 'company') {
      form.setRequired('tax_id', true);
      form.setVisible('company_name', true);
    } else {
      form.setRequired('tax_id', false);
      form.setVisible('company_name', false);
    }
  }
});
```

### Offline Support

When used inside the [BitCode Tauri shell](https://github.com/bitcode-framework/ui-tauri), components automatically route CRUD operations to a local SQLite database for models marked `mode: "offline"`. No code changes to the components themselves.

## Framework Integration

Stencil compiles to standard Custom Elements, so integration is straightforward in any framework.

### React

```jsx
import { defineCustomElements } from '@bitcode-framework/ui-components/loader';
defineCustomElements();

function ContactForm() {
  return (
    <form>
      <bc-field-string name="name" label="Name" required></bc-field-string>
      <bc-field-string name="email" label="Email" required></bc-field-string>
      <bc-field-select name="country" label="Country"
        options={JSON.stringify([
          { label: 'Indonesia', value: 'ID' },
          { label: 'Japan', value: 'JP' }
        ])}>
      </bc-field-select>
    </form>
  );
}
```

### Vue

```javascript
import { defineCustomElements } from '@bitcode-framework/ui-components/loader';
defineCustomElements();

// vite.config.js or vue.config.js
export default {
  compilerOptions: {
    isCustomElement: (tag) => tag.startsWith('bc-')
  }
};
```

```vue
<template>
  <bc-field-string name="name" label="Name" required></bc-field-string>
  <bc-datatable :columns="columns" data-source="/api/users" server-side></bc-datatable>
</template>
```

### Angular

```typescript
// app.module.ts
import { CUSTOM_ELEMENTS_SCHEMA, NgModule } from '@angular/core';

import { defineCustomElements } from '@bitcode-framework/ui-components/loader';
defineCustomElements();

@NgModule({
  schemas: [CUSTOM_ELEMENTS_SCHEMA]
})
export class AppModule {}
```

```html
<!-- contact.component.html -->
<bc-field-string name="name" label="Name" required></bc-field-string>
<bc-field-date name="birthday" label="Birthday"></bc-field-date>
```

## Tech Stack

| Library | Purpose |
|---------|---------|
| [Stencil.js](https://stenciljs.com/) | Web Component compiler |
| [Apache ECharts](https://echarts.apache.org/) | Charts (26 chart types) |
| [Tiptap](https://tiptap.dev/) | Rich text editor |
| [CodeMirror](https://codemirror.net/) | Code, JSON, HTML, SQL, CSS editors |
| [Leaflet](https://leafletjs.com/) | Maps and geolocation |
| [FullCalendar](https://fullcalendar.io/) | Calendar views |
| [markdown-it](https://github.com/markdown-it/markdown-it) | Markdown parsing and rendering |
| [JsBarcode](https://github.com/lindell/JsBarcode) | Barcode generation |
| [qrcode](https://github.com/soldair/node-qrcode) | QR code generation |
| [signature_pad](https://github.com/szimek/signature_pad) | Signature capture |
| [SortableJS](https://sortablejs.github.io/Sortable/) | Drag-and-drop sorting |
| [SheetJS](https://sheetjs.com/) | Excel export/import |
| [frappe-gantt](https://github.com/frappe/gantt) | Gantt charts |

## Related Repositories

| Repo | Description |
|------|-------------|
| [go-json](https://github.com/bitcode-framework/go-json) | JSON/JSONC programming language engine |
| [go-json-runtimes](https://github.com/bitcode-framework/go-json-runtimes) | Script runtime engines for go-json (Goja, QuickJS, Yaegi, Node.js, Python) |
| [ui-stencil-web-components](https://github.com/bitcode-framework/ui-stencil-web-components) | This repository |
| [ui-tauri](https://github.com/bitcode-framework/ui-tauri) | Tauri 2.0 native shell for desktop and mobile |

## Documentation

Per-component documentation with props, events, methods, and examples lives in the [`docs/`](docs/README.md) folder.

| Guide | Description |
|-------|-------------|
| [Getting Started](docs/getting-started.md) | Installation and basic usage |
| [BcSetup](docs/bc-setup.md) | Global configuration: auth, headers, base URL, theme, validators |
| [Theming](docs/theming.md) | Light, dark, system-detect, and custom themes |
| [Data Fetching](docs/data-fetching.md) | 4-level data strategy |
| [Validation](docs/validation.md) | 3-level validation system |
| [Reactivity](docs/reactivity.md) | Dependent fields, cascading logic |
| [Component Reference](docs/README.md) | Full component catalog with links to per-component docs |

## License

[MIT](LICENSE)
