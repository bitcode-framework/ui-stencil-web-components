# bc-chart-custom

> Custom chart — raw ECharts option escape hatch (ECharts)

## Quick Start

```html
<bc-chart-custom option='{"xAxis":{"type":"category","data":["A","B","C"]},"yAxis":{"type":"value"},"series":[{"type":"bar","data":[10,20,30]}]}' />
```

## Props

| Prop | Type | Default | Description |
|------|------|---------|-------------|
| option | string (JSON) | '{}' | Raw ECharts option object |
| height | string | '400px' | Chart height |
| width | string | '100%' | Chart width |
| loading | boolean | false | Show loading overlay |
| theme | string | '' | ECharts theme ('dark' or custom JSON) |
| renderer | string | 'canvas' | Renderer ('canvas' or 'svg') |
| data-source | string | '' | Remote data URL (fetched result used as option) |
| fetch-headers | string | '' | Custom fetch headers (JSON) |
| refresh-interval | number | 0 | Auto-refresh interval (ms) |

Note: This component accepts a full ECharts option object. All chart types and features available in ECharts are supported. No `data` prop — use `option` instead.

## Usage

Pass any valid ECharts option as JSON:

```html
<bc-chart-custom option='{
  "title": {"text": "Custom Chart"},
  "tooltip": {},
  "xAxis": {"type": "category", "data": ["Mon","Tue","Wed"]},
  "yAxis": {"type": "value"},
  "series": [{"type": "bar", "data": [120, 200, 150]}]
}' />
```

All ECharts chart types are registered (bar, line, pie, scatter, radar, tree, treemap, graph, gauge, funnel, parallel, sankey, boxplot, candlestick, effectScatter, heatmap, pictorialBar, themeRiver, sunburst, custom, lines) along with all components (polar, geo, calendar, visualMap, markPoint, markLine, markArea, timeline, graphic, brush).

## Events

| Event | Payload |
|-------|---------|
| lcChartClick | `{name, value, dataIndex}` |
| lcChartHover | `{name, value, dataIndex}` |
| lcChartReady | void |

## Methods

| Method | Returns | Description |
|--------|---------|-------------|
| setOption(option) | Promise<void> | Set full ECharts option |
| updateData(data) | Promise<void> | Alias for setOption |
| setData(data) | Promise<void> | Alias for setOption |
| refresh() | Promise<void> | Re-fetch or re-render |
| resize() | Promise<void> | Resize chart |
| exportImage(format?) | Promise<string> | Export as image (png/jpeg/svg) |
| getChartInstance() | Promise<ECharts> | Get raw ECharts instance |

See [theming](../theming.md), [data-fetching](../data-fetching.md).
