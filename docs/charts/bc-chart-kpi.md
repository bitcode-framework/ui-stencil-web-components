# bc-chart-kpi

> KPI card (pure CSS, no ECharts)

## Quick Start

```html
<bc-chart-kpi data='[{"name":"Revenue","value":125000}]' chart-title="Monthly Revenue" />
```

## Props

| Prop | Type | Default | Description |
|------|------|---------|-------------|
| data | string (JSON) | '[]' | Chart data — `[{name,value}]` |
| chart-title | string | '' | Card title |
| colors | string (JSON) | '' | Color palette |
| height | string | '300px' | Card height |
| width | string | '100%' | Card width |
| loading | boolean | false | Show loading overlay |
| data-source | string | '' | Remote data URL |
| fetch-headers | string | '' | Custom fetch headers (JSON) |
| refresh-interval | number | 0 | Auto-refresh interval (ms) |

Note: Pure CSS component — no ECharts dependency. Props like `theme`, `renderer`, `toolbox`, `data-zoom` are not applicable.

## Data Format

```json
[{"name":"Revenue","value":125000}]
```

## Events

| Event | Payload |
|-------|---------|
| lcChartClick | `{name, value, dataIndex}` |

## Methods

| Method | Returns | Description |
|--------|---------|-------------|
| updateData(data) | Promise<void> | Update card data |
| setData(data) | Promise<void> | Alias for updateData |
| refresh() | Promise<void> | Re-fetch or re-render |

See [theming](../theming.md), [data-fetching](../data-fetching.md).
