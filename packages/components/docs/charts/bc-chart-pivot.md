# bc-chart-pivot

> Pivot table (pure CSS, no ECharts)

## Quick Start

```html
<bc-chart-pivot data='[{"region":"North","product":"A","sales":100},{"region":"South","product":"B","sales":200}]' rows="region" cols="product" value-field="sales" />
```

## Props

| Prop | Type | Default | Description |
|------|------|---------|-------------|
| data | string (JSON) | '[]' | Chart data — array of objects |
| chart-title | string | '' | Table title |
| colors | string (JSON) | '' | Color palette |
| height | string | '300px' | Table height |
| width | string | '100%' | Table width |
| loading | boolean | false | Show loading overlay |
| data-source | string | '' | Remote data URL |
| fetch-headers | string | '' | Custom fetch headers (JSON) |
| refresh-interval | number | 0 | Auto-refresh interval (ms) |

Note: Pure CSS component — no ECharts dependency. Props like `theme`, `renderer`, `toolbox`, `data-zoom` are not applicable.

## Data Format

```json
[
  {"region":"North","product":"Widget A","sales":100},
  {"region":"North","product":"Widget B","sales":150},
  {"region":"South","product":"Widget A","sales":200}
]
```

## Events

| Event | Payload |
|-------|---------|
| lcChartClick | `{name, value, dataIndex}` |

## Methods

| Method | Returns | Description |
|--------|---------|-------------|
| updateData(data) | Promise<void> | Update table data |
| setData(data) | Promise<void> | Alias for updateData |
| refresh() | Promise<void> | Re-fetch or re-render |

See [theming](../theming.md), [data-fetching](../data-fetching.md).
