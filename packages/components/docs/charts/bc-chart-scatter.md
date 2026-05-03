# bc-chart-scatter

> Scatter/bubble chart (ECharts)

## Quick Start

```html
<bc-chart-scatter data='[[10,20],[30,40],[50,60]]' />
```

## Props

| Prop | Type | Default | Description |
|------|------|---------|-------------|
| data | string (JSON) | '[]' | Chart data — `[[x,y]]` or `{series:[{name,data}]}` |
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
| bubble | boolean | false | Enable bubble mode (3rd value = size) |
| effect-scatter | boolean | false | Use ripple effect scatter points |

## Data Format

**Simple (array of points):**

```json
[[10,20],[30,40],[50,60],[70,80]]
```

**Bubble (3rd value = size):**

```json
[[10,20,5],[30,40,15],[50,60,25]]
```

**Multi-series:**

```json
{
  "series": [
    {"name":"Group A","data":[[10,20],[30,40]]},
    {"name":"Group B","data":[[15,25],[35,45]]}
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
