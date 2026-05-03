# bc-chart-radar

> Radar chart (ECharts)

## Quick Start

```html
<bc-chart-radar data='{"indicators":[{"name":"Sales","max":100},{"name":"Admin","max":100},{"name":"Tech","max":100}],"series":[{"name":"Budget","data":[80,60,90]}]}' />
```

## Props

| Prop | Type | Default | Description |
|------|------|---------|-------------|
| data | string (JSON) | '[]' | Chart data — `{indicators, series}` |
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

### Type-Specific Props

| Prop | Type | Default | Description |
|------|------|---------|-------------|
| shape | string | 'polygon' | Radar shape ('polygon' or 'circle') |
| filled | boolean | false | Fill area under radar lines |

## Data Format

```json
{
  "indicators": [
    {"name":"Sales","max":100},
    {"name":"Admin","max":100},
    {"name":"Tech","max":100},
    {"name":"Support","max":100},
    {"name":"Marketing","max":100}
  ],
  "series": [
    {"name":"Budget","data":[80,60,90,70,50]},
    {"name":"Actual","data":[70,80,75,85,60]}
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
