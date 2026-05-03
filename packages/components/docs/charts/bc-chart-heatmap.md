# bc-chart-heatmap

> Heatmap (ECharts)

## Quick Start

```html
<bc-chart-heatmap data='{"xAxis":["Mon","Tue","Wed"],"yAxis":["Morning","Afternoon"],"data":[[0,0,5],[1,0,10],[0,1,8],[1,1,3],[2,0,7],[2,1,12]]}' />
```

## Props

| Prop | Type | Default | Description |
|------|------|---------|-------------|
| data | string (JSON) | '[]' | Chart data — structured `{xAxis, yAxis, data}` |
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
| visual-map-min | number | 0 | Visual map minimum value |
| visual-map-max | number | 100 | Visual map maximum value |

## Data Format

```json
{
  "xAxis": ["Mon","Tue","Wed","Thu","Fri"],
  "yAxis": ["Morning","Afternoon","Evening"],
  "data": [
    [0, 0, 5], [1, 0, 10], [2, 0, 7],
    [0, 1, 8], [1, 1, 3], [2, 1, 12],
    [0, 2, 2], [1, 2, 6], [2, 2, 9]
  ]
}
```

Each data item is `[xIndex, yIndex, value]`.

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
