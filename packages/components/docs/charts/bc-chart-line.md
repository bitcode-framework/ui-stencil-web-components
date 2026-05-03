# bc-chart-line

> Line chart (ECharts)

## Quick Start

```html
<bc-chart-line data='[{"name":"A","value":10},{"name":"B","value":20}]' />
```

## Props

| Prop | Type | Default | Description |
|------|------|---------|-------------|
| data | string (JSON) | '[]' | Chart data — single `[{name,value}]` or multi `{categories,series}` |
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
| data-zoom | boolean | false | Show data zoom slider |
| locale | string | '' | ECharts locale |

### Type-Specific Props

| Prop | Type | Default | Description |
|------|------|---------|-------------|
| smooth | boolean | true | Smooth curved lines |
| stacked | boolean | false | Stack lines (filled area between) |

## Data Format

**Single series:**

```json
[{"name":"Jan","value":100},{"name":"Feb","value":200}]
```

**Multi-series:**

```json
{
  "categories": ["Jan","Feb","Mar"],
  "series": [
    {"name":"2024","data":[100,200,150]},
    {"name":"2025","data":[120,180,220]}
  ]
}
```

## Events

| Event | Payload |
|-------|---------|
| lcChartClick | `{name, value, dataIndex}` |
| lcChartHover | `{name, value, dataIndex}` |
| lcChartLegendSelect | `{name, selected}` |
| lcChartDataZoom | `{start, end}` |
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
