# bc-chart-graph

> Network/relationship graph (ECharts)

## Quick Start

```html
<bc-chart-graph data='{"nodes":[{"name":"A","value":10},{"name":"B","value":20}],"links":[{"source":"A","target":"B"}]}' />
```

## Props

| Prop | Type | Default | Description |
|------|------|---------|-------------|
| data | string (JSON) | '[]' | Chart data — `{nodes, links, categories?}` |
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
| layout | string | 'force' | Layout algorithm ('force', 'circular', 'none') |
| roam | boolean | true | Enable zoom and pan |
| draggable | boolean | true | Allow dragging nodes |

## Data Format

```json
{
  "nodes": [
    {"name":"Node A","value":10,"category":0},
    {"name":"Node B","value":20,"category":0},
    {"name":"Node C","value":15,"category":1}
  ],
  "links": [
    {"source":"Node A","target":"Node B"},
    {"source":"Node B","target":"Node C"},
    {"source":"Node A","target":"Node C"}
  ],
  "categories": [
    {"name":"Group 1"},
    {"name":"Group 2"}
  ]
}
```

Node `value` determines symbol size. `categories` are optional for color grouping.

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
