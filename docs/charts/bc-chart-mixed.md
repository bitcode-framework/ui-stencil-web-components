# bc-chart-mixed

> Mixed/combo chart (ECharts)

## Quick Start

```html
<bc-chart-mixed data='{"categories":["Jan","Feb","Mar"],"series":[{"name":"Revenue","type":"bar","data":[100,200,150]},{"name":"Trend","type":"line","data":[90,180,160]}]}' />
```

## Props

| Prop | Type | Default | Description |
|------|------|---------|-------------|
| data | string (JSON) | '[]' | Chart data — `{categories, series: [{name, type, data}]}` |
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
| dual-axis | boolean | false | Enable dual Y-axis (second type gets right axis) |
| stacked | string (JSON) | '' | Stack config `{"Series1":"group1","Series2":"group1"}` |

## Data Format

Each series specifies its own `type` (bar, line, scatter):

```json
{
  "categories": ["Jan","Feb","Mar","Apr"],
  "series": [
    {"name":"Revenue","type":"bar","data":[100,200,150,180]},
    {"name":"Profit","type":"bar","data":[30,60,45,55]},
    {"name":"Trend","type":"line","data":[90,180,160,200]}
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
