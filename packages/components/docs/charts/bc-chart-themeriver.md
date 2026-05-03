# bc-chart-themeriver

> Theme river / stream graph (ECharts)

## Quick Start

```html
<bc-chart-themeriver data='[["2024-01","10","Product A"],["2024-02","20","Product A"],["2024-01","15","Product B"],["2024-02","25","Product B"]]' />
```

## Props

| Prop | Type | Default | Description |
|------|------|---------|-------------|
| data | string (JSON) | '[]' | Chart data — `[[date, value, name], ...]` |
| chart-title | string | '' | Chart title |
| colors | string (JSON) | '' | Color palette `["#4f46e5","#06b6d4"]` |
| legend | boolean | true | Show legend |
| tooltip-enabled | boolean | true | Show tooltip |
| animate | boolean | true | Enable animation |
| height | string | '300px' | Chart height |
| width | string | '100%' | Chart width |
| loading | boolean | false | Show loading overlay |
| data-source | string | '' | Remote data URL |
| fetch-headers | string | '' | Custom fetch headers (JSON) |
| refresh-interval | number | 0 | Auto-refresh interval (ms) |
| theme | string | '' | ECharts theme ('dark' or custom JSON) |
| renderer | string | 'canvas' | Renderer ('canvas' or 'svg') |
| toolbox | boolean | false | Show toolbox (save, zoom, restore) |
| locale | string | '' | ECharts locale |

## Data Format

```json
[
  ["2024-01-01", 10, "Product A"],
  ["2024-01-02", 20, "Product A"],
  ["2024-01-03", 15, "Product A"],
  ["2024-01-01", 15, "Product B"],
  ["2024-01-02", 25, "Product B"],
  ["2024-01-03", 30, "Product B"]
]
```

Each item is `[date/time, value, category_name]`.

## Events

| Event | Payload |
|-------|---------|
| lcChartClick | `{name, value, dataIndex}` |
| lcChartHover | `{name, value, dataIndex}` |
| lcChartLegendSelect | `{name, selected}` |
| lcChartReady | void |

## Methods

| Method | Returns | Description |
|--------|---------|-------------|
| updateData(data) | Promise<void> | Update chart data |
| setData(data) | Promise<void> | Alias for updateData |
| refresh() | Promise<void> | Re-fetch or re-render |
| resize() | Promise<void> | Resize chart |
| exportImage(format?) | Promise<string> | Export as image (png/jpeg/svg) |
| getChartInstance() | Promise<ECharts> | Get raw ECharts instance |

See [theming](../theming.md), [data-fetching](../data-fetching.md).
