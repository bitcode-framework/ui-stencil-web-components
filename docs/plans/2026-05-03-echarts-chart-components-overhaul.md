# ECharts Chart Components Overhaul — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Refactor all 8 existing ECharts chart components to use a shared utility with modular imports (tree-shaking), multi-series support, toolbox, dataZoom, theme switching, and more events. Add 15 new chart components covering all major ECharts chart types. Total: 26 ECharts + 3 pure CSS = 29 chart components.

**Architecture:** Shared `chart-utils.ts` utility handles ECharts lifecycle (init/dispose/resize/fetch/events). Each component imports only the ECharts modules it needs via `echarts/core` + `echarts/charts` + `echarts/components` (tree-shakeable). Components define their chart-specific `buildOption()` and type-specific props. Data shape is backward-compatible: old `[{name,value}]` still works, new multi-series `{categories,series}` format added.

**Tech Stack:** Stencil.js 4.x, Apache ECharts 6.x (modular imports), TypeScript

---

## Conventions (READ FIRST)

### File Naming
- Component folder: `bc-chart-{type}/` inside `packages/components/src/components/charts/`
- TSX file: `bc-chart-{type}.tsx`
- CSS file: `bc-chart-{type}.css`
- Doc file: `packages/components/docs/charts/bc-chart-{type}.md`

### CSS Pattern (all ECharts components share this)
```css
.bc-chart-wrap{position:relative;padding:16px;border:1px solid var(--bc-border-color);border-radius:var(--bc-radius-lg);background:var(--bc-bg)}.bc-echart{width:100%;height:300px}.bc-chart-loading{opacity:0.6}.bc-chart-loading-overlay{position:absolute;inset:0;display:flex;align-items:center;justify-content:center;z-index:1}.bc-chart-empty{display:flex;align-items:center;justify-content:center;height:200px;color:var(--bc-text-secondary);font-size:var(--bc-font-size-sm)}
```

### Component Pattern (all ECharts components follow this)
Every ECharts chart component:
1. Imports from `echarts/core`, `echarts/charts`, `echarts/components`, `echarts/renderers`
2. Calls `echarts.use([...])` at module level (idempotent, safe for multiple components)
3. Uses shared helpers from `chart-utils.ts` for lifecycle
4. Defines `buildOption()` that converts props → ECharts option
5. Supports both single-series `[{name,value}]` and multi-series `{categories,series}` data shapes
6. Emits events: `lcChartClick`, `lcChartHover`, `lcChartLegendSelect`, `lcChartDataZoom`, `lcChartReady`
7. Exposes methods: `updateData()`, `setData()`, `refresh()`, `resize()`, `exportImage()`, `getChartInstance()`

### Data Shape Convention
```typescript
// Single series (backward compatible) — auto-detected when data is an array
[{name: "A", value: 10}, {name: "B", value: 20}]

// Multi-series — auto-detected when data is an object with categories+series
{categories: ["Jan","Feb","Mar"], series: [{name: "Sales", data: [10,20,30]}, {name: "Cost", data: [5,10,15]}]}

// Heatmap-specific
[[0, 0, 5], [0, 1, 10], [1, 0, 15]]

// Tree/Sunburst/Treemap hierarchical
{name: "root", children: [{name: "A", value: 10, children: [...]}, ...]}

// Graph-specific
{nodes: [{name: "A"}, {name: "B"}], links: [{source: "A", target: "B", value: 1}]}

// Sankey-specific
{nodes: [{name: "A"}, {name: "B"}], links: [{source: "A", target: "B", value: 100}]}

// Boxplot-specific
[[850, 900, 950, 1000, 1050], [800, 850, 900, 950, 1000]]

// Candlestick-specific (OHLC)
{categories: ["2024-01","2024-02"], data: [[open, close, low, high], ...]}

// Parallel-specific
{dimensions: ["cost","revenue","profit"], data: [[100, 200, 100], [150, 300, 150]]}
```

### Shared Props (all ECharts components)
```typescript
@Prop({ mutable: true }) data: string = '[]';
@Prop() chartTitle: string = '';
@Prop() colors: string = '';
@Prop() legend: boolean = true;
@Prop() tooltipEnabled: boolean = true;
@Prop() animate: boolean = true;
@Prop() height: string = '300px';
@Prop() width: string = '100%';
@Prop({ mutable: true }) loading: boolean = false;
@Prop() dataSource: string = '';
@Prop() fetchHeaders: string = '';
@Prop() refreshInterval: number = 0;
@Prop() theme: string = '';           // 'dark' | custom theme JSON
@Prop() renderer: string = 'canvas';  // 'canvas' | 'svg'
@Prop() toolbox: boolean = false;
@Prop() dataZoom: boolean = false;
@Prop() locale: string = '';
```

### Shared Events
```typescript
@Event() lcChartClick!: EventEmitter<ChartClickEvent>;
@Event() lcChartHover!: EventEmitter<ChartHoverEvent>;
@Event() lcChartLegendSelect!: EventEmitter<ChartLegendSelectEvent>;
@Event() lcChartDataZoom!: EventEmitter<ChartDataZoomEvent>;
@Event() lcChartReady!: EventEmitter<void>;
```

### Shared Methods
```typescript
@Method() async updateData(newData: unknown): Promise<void>;
@Method() async setData(newData: unknown): Promise<void>;
@Method() async refresh(): Promise<void>;
@Method() async resize(): Promise<void>;
@Method() async exportImage(format?: string): Promise<string>;
@Method() async getChartInstance(): Promise<unknown>;
```

---

## Task 1: Add Chart Event Types to core/types.ts

**Files:**
- Modify: `packages/components/src/core/types.ts`

**Step 1: Add new chart event interfaces**

Add after the existing `ChartHoverEvent` interface (around line 187):

```typescript
export interface ChartLegendSelectEvent {
  name: string;
  selected: Record<string, boolean>;
}

export interface ChartDataZoomEvent {
  start: number;
  end: number;
  startValue?: unknown;
  endValue?: unknown;
}

export interface ChartBrushEvent {
  areas: Array<{
    brushType: string;
    coordRange: number[];
  }>;
}
```

**Step 2: Verify no type errors**

Run: `cd packages/components && npx stencil build --dev 2>&1 | head -20`
Expected: No type errors

---

## Task 2: Create chart-utils.ts Shared Utility

**Files:**
- Create: `packages/components/src/core/chart-utils.ts`

**Step 1: Create the shared chart utility**

```typescript
import * as echarts from 'echarts/core';
import { CanvasRenderer, SVGRenderer } from 'echarts/renderers';
import { LabelLayout, UniversalTransition } from 'echarts/features';
import {
  TitleComponent,
  TooltipComponent,
  LegendComponent,
  ToolboxComponent,
  DataZoomComponent,
  DataZoomInsideComponent,
  DataZoomSliderComponent,
  GridComponent,
  DatasetComponent,
  TransformComponent,
  AriaComponent,
} from 'echarts/components';
import type { ECharts, EChartsOption } from 'echarts/core';
import { ChartClickEvent, ChartHoverEvent, ChartLegendSelectEvent, ChartDataZoomEvent, DataFetcher } from './types';
import { fetchData } from './data-fetcher';
import { EventEmitter } from '@stencil/core';

// Register common components once (idempotent)
echarts.use([
  CanvasRenderer,
  SVGRenderer,
  LabelLayout,
  UniversalTransition,
  TitleComponent,
  TooltipComponent,
  LegendComponent,
  ToolboxComponent,
  DataZoomComponent,
  DataZoomInsideComponent,
  DataZoomSliderComponent,
  GridComponent,
  DatasetComponent,
  TransformComponent,
  AriaComponent,
]);

export { echarts };
export type { ECharts, EChartsOption };

// ============================================================================
// DATA PARSING
// ============================================================================

export interface SingleSeriesItem {
  name: string;
  value: number;
  [key: string]: unknown;
}

export interface MultiSeriesData {
  categories: string[];
  series: Array<{
    name: string;
    data: number[];
    type?: string;
    [key: string]: unknown;
  }>;
}

export interface HierarchicalData {
  name: string;
  value?: number;
  children?: HierarchicalData[];
  [key: string]: unknown;
}

export interface GraphData {
  nodes: Array<{ name: string; value?: number; category?: number; [key: string]: unknown }>;
  links: Array<{ source: string; target: string; value?: number; [key: string]: unknown }>;
  categories?: Array<{ name: string }>;
}

export interface CandlestickData {
  categories: string[];
  data: Array<[number, number, number, number]>; // [open, close, low, high]
}

export interface ParallelData {
  dimensions: string[];
  data: number[][];
}

export type ChartData =
  | SingleSeriesItem[]
  | MultiSeriesData
  | HierarchicalData
  | GraphData
  | CandlestickData
  | ParallelData
  | number[][]
  | unknown;

export function parseChartData(raw: string): ChartData {
  try {
    return JSON.parse(raw);
  } catch {
    return [];
  }
}

export function isSingleSeries(data: ChartData): data is SingleSeriesItem[] {
  return Array.isArray(data) && data.length > 0 && typeof data[0] === 'object' && data[0] !== null && 'name' in data[0] && 'value' in data[0];
}

export function isMultiSeries(data: ChartData): data is MultiSeriesData {
  return data !== null && typeof data === 'object' && !Array.isArray(data) && 'categories' in data && 'series' in data;
}

export function isHierarchical(data: ChartData): data is HierarchicalData {
  return data !== null && typeof data === 'object' && !Array.isArray(data) && 'name' in data && ('children' in data || 'value' in data);
}

export function isGraphData(data: ChartData): data is GraphData {
  return data !== null && typeof data === 'object' && !Array.isArray(data) && 'nodes' in data && 'links' in data;
}

export function isCandlestickData(data: ChartData): data is CandlestickData {
  return data !== null && typeof data === 'object' && !Array.isArray(data) && 'categories' in data && 'data' in data && Array.isArray((data as CandlestickData).data) && Array.isArray((data as CandlestickData).data[0]);
}

export function isParallelData(data: ChartData): data is ParallelData {
  return data !== null && typeof data === 'object' && !Array.isArray(data) && 'dimensions' in data && 'data' in data;
}

export function isHeatmapData(data: ChartData): data is number[][] {
  return Array.isArray(data) && data.length > 0 && Array.isArray(data[0]) && typeof data[0][0] === 'number';
}

// ============================================================================
// COLOR PARSING
// ============================================================================

const DEFAULT_COLORS = ['#4f46e5', '#06b6d4', '#10b981', '#f59e0b', '#ef4444', '#8b5cf6', '#ec4899', '#14b8a6', '#f97316', '#6366f1'];

export function parseColors(colorsStr: string): string[] | undefined {
  if (!colorsStr) return undefined;
  try {
    const parsed = JSON.parse(colorsStr);
    return Array.isArray(parsed) ? parsed : undefined;
  } catch {
    return undefined;
  }
}

export function getColorList(colorsStr: string): string[] {
  return parseColors(colorsStr) || DEFAULT_COLORS;
}

// ============================================================================
// CHART LIFECYCLE
// ============================================================================

export interface ChartHostElement extends HTMLElement {
  querySelector(selector: string): HTMLElement | null;
}

export interface ChartEvents {
  lcChartClick: EventEmitter<ChartClickEvent>;
  lcChartHover?: EventEmitter<ChartHoverEvent>;
  lcChartLegendSelect?: EventEmitter<ChartLegendSelectEvent>;
  lcChartDataZoom?: EventEmitter<ChartDataZoomEvent>;
  lcChartReady?: EventEmitter<void>;
}

export function initChart(
  el: ChartHostElement,
  theme: string,
  renderer: string,
  events: ChartEvents,
): ECharts | null {
  const container = el.querySelector('.bc-echart');
  if (!container) return null;

  let themeArg: string | object | undefined = undefined;
  if (theme === 'dark') {
    themeArg = 'dark';
  } else if (theme) {
    try { themeArg = JSON.parse(theme); } catch { themeArg = theme; }
  }

  const chart = echarts.init(container as HTMLElement, themeArg, {
    renderer: (renderer === 'svg' ? 'svg' : 'canvas') as 'canvas' | 'svg',
  });

  chart.on('click', (params: Record<string, unknown>) => {
    events.lcChartClick.emit({
      name: params.name as string,
      value: params.value as unknown,
      dataIndex: params.dataIndex as number,
    });
  });

  if (events.lcChartHover) {
    const hoverEmitter = events.lcChartHover;
    chart.on('mouseover', (params: Record<string, unknown>) => {
      hoverEmitter.emit({
        name: params.name as string,
        value: params.value as unknown,
        dataIndex: params.dataIndex as number,
      });
    });
  }

  if (events.lcChartLegendSelect) {
    const legendEmitter = events.lcChartLegendSelect;
    chart.on('legendselectchanged', (params: Record<string, unknown>) => {
      legendEmitter.emit({
        name: params.name as string,
        selected: params.selected as Record<string, boolean>,
      });
    });
  }

  if (events.lcChartDataZoom) {
    const zoomEmitter = events.lcChartDataZoom;
    chart.on('datazoom', (params: Record<string, unknown>) => {
      zoomEmitter.emit({
        start: params.start as number,
        end: params.end as number,
        startValue: params.startValue,
        endValue: params.endValue,
      });
    });
  }

  if (events.lcChartReady) {
    events.lcChartReady.emit();
  }

  return chart;
}

export function disposeChart(chart: ECharts | null, timer: ReturnType<typeof setInterval> | null, observer: ResizeObserver | null): void {
  chart?.dispose();
  if (timer) clearInterval(timer);
  if (observer) observer.disconnect();
}

export function setupResizeObserver(el: ChartHostElement, chart: ECharts | null): ResizeObserver | null {
  if (!chart) return null;
  const container = el.querySelector('.bc-echart');
  if (!container) return null;
  const observer = new ResizeObserver(() => {
    chart.resize();
  });
  observer.observe(container);
  return observer;
}

// ============================================================================
// COMMON OPTION BUILDERS
// ============================================================================

export function buildTitleOption(chartTitle: string): Record<string, unknown> | undefined {
  if (!chartTitle) return undefined;
  return { text: chartTitle, left: 'center', textStyle: { fontSize: 14 } };
}

export function buildTooltipOption(enabled: boolean, trigger: 'item' | 'axis' = 'axis'): Record<string, unknown> | undefined {
  if (!enabled) return undefined;
  return { trigger };
}

export function buildLegendOption(enabled: boolean): Record<string, unknown> | undefined {
  if (!enabled) return undefined;
  return { bottom: 0 };
}

export function buildToolboxOption(enabled: boolean): Record<string, unknown> | undefined {
  if (!enabled) return undefined;
  return {
    feature: {
      saveAsImage: { title: 'Save' },
      dataView: { title: 'Data', readOnly: true },
      restore: { title: 'Restore' },
      dataZoom: { title: { zoom: 'Zoom', back: 'Reset' } },
    },
    right: 10,
    top: 0,
  };
}

export function buildDataZoomOption(enabled: boolean): Array<Record<string, unknown>> | undefined {
  if (!enabled) return undefined;
  return [
    { type: 'inside' },
    { type: 'slider', bottom: 24 },
  ];
}

export function buildGridOption(hasDataZoom: boolean, hasLegend: boolean): Record<string, unknown> {
  return {
    left: '3%',
    right: '4%',
    bottom: hasDataZoom ? 60 : (hasLegend ? 40 : '3%'),
    top: 40,
    containLabel: true,
  };
}

// ============================================================================
// DATA FETCHING HELPER
// ============================================================================

export async function fetchChartData(opts: {
  dataFetcher?: DataFetcher;
  el: ChartHostElement;
  dataSource: string;
  fetchHeaders: string;
}): Promise<unknown[]> {
  const result = await fetchData({
    fetcher: opts.dataFetcher,
    element: opts.el,
    dataSource: opts.dataSource,
    fetchHeaders: opts.fetchHeaders,
  });
  return result.data;
}
```

**Step 2: Verify the utility compiles**

Run: `cd packages/components && npx stencil build --dev 2>&1 | head -30`
Expected: No errors related to chart-utils.ts

---

## Task 3: Register Chart-Specific ECharts Modules

Each chart component needs to register its specific chart type. This is done at module level (top of each .tsx file). The shared utility already registers common components (Grid, Tooltip, Legend, Toolbox, DataZoom, Title, Dataset, Transform, Aria, Renderers).

**Per-component registrations:**

| Component | Additional Registration |
|-----------|----------------------|
| bc-chart-bar | `BarChart` from `echarts/charts` |
| bc-chart-line | `LineChart` from `echarts/charts` |
| bc-chart-area | `LineChart` from `echarts/charts` |
| bc-chart-pie | `PieChart` from `echarts/charts` |
| bc-chart-funnel | `FunnelChart` from `echarts/charts` |
| bc-chart-heatmap | `HeatmapChart` from `echarts/charts`, `VisualMapComponent` from `echarts/components` |
| bc-chart-gauge | `GaugeChart` from `echarts/charts` |
| bc-chart-scorecard | `GaugeChart` from `echarts/charts` |
| bc-chart-scatter | `ScatterChart`, `EffectScatterChart` from `echarts/charts` |
| bc-chart-radar | `RadarChart` from `echarts/charts`, `RadarComponent` from `echarts/components` |
| bc-chart-treemap | `TreemapChart` from `echarts/charts`, `VisualMapComponent` from `echarts/components` |
| bc-chart-candlestick | `CandlestickChart` from `echarts/charts`, `MarkLineComponent`, `MarkPointComponent` from `echarts/components` |
| bc-chart-mixed | `BarChart`, `LineChart`, `ScatterChart` from `echarts/charts` |
| bc-chart-sankey | `SankeyChart` from `echarts/charts` |
| bc-chart-sunburst | `SunburstChart` from `echarts/charts` |
| bc-chart-boxplot | `BoxplotChart` from `echarts/charts` |
| bc-chart-graph | `GraphChart` from `echarts/charts` |
| bc-chart-tree | `TreeChart` from `echarts/charts` |
| bc-chart-polar | `BarChart`, `LineChart` from `echarts/charts`, `PolarComponent` from `echarts/components` |
| bc-chart-parallel | `ParallelChart` from `echarts/charts`, `ParallelComponent` from `echarts/components` |
| bc-chart-themeriver | `ThemeRiverChart` from `echarts/charts`, `SingleAxisComponent` from `echarts/components` |
| bc-chart-pictorialbar | `PictorialBarChart` from `echarts/charts` |
| bc-chart-custom | ALL chart types + ALL components (escape hatch) |

---

## Task 4: Refactor bc-chart-bar

**Files:**
- Modify: `packages/components/src/components/charts/bc-chart-bar/bc-chart-bar.tsx`
- Modify: `packages/components/src/components/charts/bc-chart-bar/bc-chart-bar.css`

**Step 1: Rewrite bc-chart-bar.tsx**

```typescript
import { Component, Prop, Element, Watch, Method, Event, EventEmitter, h } from '@stencil/core';
import { ChartClickEvent, ChartHoverEvent, ChartLegendSelectEvent, ChartDataZoomEvent, DataFetcher } from '../../../core/types';
import {
  echarts, ECharts,
  parseChartData, isSingleSeries, isMultiSeries, getColorList,
  initChart, disposeChart, setupResizeObserver, fetchChartData,
  buildTitleOption, buildTooltipOption, buildLegendOption, buildToolboxOption, buildDataZoomOption, buildGridOption,
  SingleSeriesItem, MultiSeriesData,
} from '../../../core/chart-utils';
import { BarChart } from 'echarts/charts';

echarts.use([BarChart]);

@Component({ tag: 'bc-chart-bar', styleUrl: 'bc-chart-bar.css', shadow: false })
export class BcChartBar {
  @Element() el!: HTMLElement;

  // Data
  @Prop({ mutable: true }) data: string = '[]';
  @Prop() chartTitle: string = '';
  @Prop() colors: string = '';

  // Display
  @Prop() legend: boolean = true;
  @Prop() tooltipEnabled: boolean = true;
  @Prop() animate: boolean = true;
  @Prop() height: string = '300px';
  @Prop() width: string = '100%';
  @Prop({ mutable: true }) loading: boolean = false;

  // Data fetching
  @Prop() dataSource: string = '';
  @Prop() fetchHeaders: string = '';
  @Prop() refreshInterval: number = 0;
  dataFetcher?: DataFetcher;

  // Enhanced
  @Prop() theme: string = '';
  @Prop() renderer: string = 'canvas';
  @Prop() toolbox: boolean = false;
  @Prop() dataZoom: boolean = false;
  @Prop() stacked: boolean = false;
  @Prop() horizontal: boolean = false;
  @Prop() locale: string = '';

  // Internal
  private chart: ECharts | null = null;
  private _refreshTimer: ReturnType<typeof setInterval> | null = null;
  private _resizeObserver: ResizeObserver | null = null;

  // Events
  @Event() lcChartClick!: EventEmitter<ChartClickEvent>;
  @Event() lcChartHover!: EventEmitter<ChartHoverEvent>;
  @Event() lcChartLegendSelect!: EventEmitter<ChartLegendSelectEvent>;
  @Event() lcChartDataZoom!: EventEmitter<ChartDataZoomEvent>;
  @Event() lcChartReady!: EventEmitter<void>;

  componentDidLoad() {
    this.chart = initChart(this.el, this.theme, this.renderer, {
      lcChartClick: this.lcChartClick,
      lcChartHover: this.lcChartHover,
      lcChartLegendSelect: this.lcChartLegendSelect,
      lcChartDataZoom: this.lcChartDataZoom,
      lcChartReady: this.lcChartReady,
    });
    this._resizeObserver = setupResizeObserver(this.el, this.chart);
    this.renderChart();
    if (this.dataSource || this.dataFetcher) this._fetchData();
    if (this.refreshInterval > 0) this._refreshTimer = setInterval(() => this._fetchData(), this.refreshInterval);
  }

  @Watch('data') onDataChange() { this.renderChart(); }
  @Watch('theme') onThemeChange() { this._reinit(); }
  @Watch('renderer') onRendererChange() { this._reinit(); }

  disconnectedCallback() {
    disposeChart(this.chart, this._refreshTimer, this._resizeObserver);
  }

  private _reinit() {
    disposeChart(this.chart, null, this._resizeObserver);
    this.chart = initChart(this.el, this.theme, this.renderer, {
      lcChartClick: this.lcChartClick,
      lcChartHover: this.lcChartHover,
      lcChartLegendSelect: this.lcChartLegendSelect,
      lcChartDataZoom: this.lcChartDataZoom,
      lcChartReady: this.lcChartReady,
    });
    this._resizeObserver = setupResizeObserver(this.el, this.chart);
    this.renderChart();
  }

  private async _fetchData() {
    this.loading = true;
    try {
      const result = await fetchChartData({ dataFetcher: this.dataFetcher, el: this.el, dataSource: this.dataSource, fetchHeaders: this.fetchHeaders });
      this.data = JSON.stringify(result);
    } catch { /* keep existing data */ }
    this.loading = false;
  }

  private renderChart() {
    if (!this.chart) return;
    const parsed = parseChartData(this.data);
    const colorList = getColorList(this.colors);
    const isHz = this.horizontal;

    let categoryData: string[] = [];
    let seriesList: Array<Record<string, unknown>> = [];

    if (isSingleSeries(parsed)) {
      categoryData = (parsed as SingleSeriesItem[]).map(d => d.name);
      seriesList = [{
        type: 'bar',
        data: (parsed as SingleSeriesItem[]).map(d => d.value),
        ...(this.stacked ? { stack: 'total' } : {}),
      }];
    } else if (isMultiSeries(parsed)) {
      const ms = parsed as MultiSeriesData;
      categoryData = ms.categories;
      seriesList = ms.series.map(s => ({
        type: 'bar',
        name: s.name,
        data: s.data,
        ...(this.stacked ? { stack: 'total' } : {}),
        ...s,
      }));
    }

    this.chart.setOption({
      title: buildTitleOption(this.chartTitle),
      tooltip: buildTooltipOption(this.tooltipEnabled, 'axis'),
      legend: buildLegendOption(this.legend),
      toolbox: buildToolboxOption(this.toolbox),
      dataZoom: buildDataZoomOption(this.dataZoom),
      grid: buildGridOption(this.dataZoom, this.legend),
      color: colorList,
      animation: this.animate,
      xAxis: { type: isHz ? 'value' : 'category', ...(isHz ? {} : { data: categoryData }) },
      yAxis: { type: isHz ? 'category' : 'value', ...(isHz ? { data: categoryData } : {}) },
      series: seriesList,
    }, true);
  }

  @Method() async updateData(newData: unknown): Promise<void> { this.data = typeof newData === 'string' ? newData : JSON.stringify(newData); }
  @Method() async setData(newData: unknown): Promise<void> { this.data = typeof newData === 'string' ? newData : JSON.stringify(newData); }
  @Method() async refresh(): Promise<void> { if (this.dataSource || this.dataFetcher) await this._fetchData(); else this.renderChart(); }
  @Method() async resize(): Promise<void> { this.chart?.resize(); }
  @Method() async exportImage(format: string = 'png'): Promise<string> { return this.chart?.getDataURL({ type: format as 'png' | 'jpeg' | 'svg', pixelRatio: 2 }) || ''; }
  @Method() async getChartInstance(): Promise<unknown> { return this.chart; }

  render() {
    return (
      <div class={{ 'bc-chart-wrap': true, 'bc-chart-loading': this.loading }}>
        {this.loading && <div class="bc-chart-loading-overlay"><span class="bc-field-loading-indicator" /></div>}
        <div class="bc-echart" style={{ height: this.height, width: this.width }}></div>
      </div>
    );
  }
}
```

**Step 2: Update CSS**

```css
.bc-chart-wrap{position:relative;padding:16px;border:1px solid var(--bc-border-color);border-radius:var(--bc-radius-lg);background:var(--bc-bg)}.bc-echart{width:100%;height:300px}.bc-chart-loading{opacity:0.6}.bc-chart-loading-overlay{position:absolute;inset:0;display:flex;align-items:center;justify-content:center;z-index:1}
```

**Step 3: Verify build**

Run: `cd packages/components && npx stencil build --dev 2>&1 | head -30`
Expected: No errors

---

## Task 5: Refactor bc-chart-line

**Files:**
- Modify: `packages/components/src/components/charts/bc-chart-line/bc-chart-line.tsx`
- Modify: `packages/components/src/components/charts/bc-chart-line/bc-chart-line.css`

Same pattern as Task 4 but with:
- `LineChart` registration
- `type: 'line'` in series
- `smooth: true` by default
- Additional prop: `@Prop() smooth: boolean = true;`
- Additional prop: `@Prop() stacked: boolean = false;`

---

## Task 6: Refactor bc-chart-area

**Files:**
- Modify: `packages/components/src/components/charts/bc-chart-area/bc-chart-area.tsx`
- Modify: `packages/components/src/components/charts/bc-chart-area/bc-chart-area.css`

Same pattern as Task 5 (line) but with:
- `areaStyle: {}` added to each series
- `smooth: true` by default
- `boundaryGap: false` on xAxis

---

## Task 7: Refactor bc-chart-pie

**Files:**
- Modify: `packages/components/src/components/charts/bc-chart-pie/bc-chart-pie.tsx`
- Modify: `packages/components/src/components/charts/bc-chart-pie/bc-chart-pie.css`

Same shared pattern but with:
- `PieChart` registration
- No xAxis/yAxis/grid
- Tooltip trigger: `'item'`
- Additional props: `@Prop() donut: boolean = true;` (radius ['40%','70%'] vs ['0%','70%'])
- Additional prop: `@Prop() roseType: string = '';` ('radius' | 'area' | '')
- Data shape: only single series `[{name, value}]` (pie doesn't support multi-series in the same way)

---

## Task 8: Refactor bc-chart-funnel

**Files:**
- Modify: `packages/components/src/components/charts/bc-chart-funnel/bc-chart-funnel.tsx`
- Modify: `packages/components/src/components/charts/bc-chart-funnel/bc-chart-funnel.css`

Same shared pattern but with:
- `FunnelChart` registration
- No xAxis/yAxis/grid
- Tooltip trigger: `'item'`
- Additional prop: `@Prop() sortOrder: string = 'descending';` ('descending' | 'ascending' | 'none')

---

## Task 9: Refactor bc-chart-heatmap

**Files:**
- Modify: `packages/components/src/components/charts/bc-chart-heatmap/bc-chart-heatmap.tsx`
- Modify: `packages/components/src/components/charts/bc-chart-heatmap/bc-chart-heatmap.css`

Same shared pattern but with:
- `HeatmapChart` registration + `VisualMapComponent`
- Data shape: `number[][]` or `{xAxis: string[], yAxis: string[], data: number[][]}`
- Additional props: `@Prop() visualMapMin: number = 0;`, `@Prop() visualMapMax: number = 100;`
- VisualMap component in option

---

## Task 10: Refactor bc-chart-gauge

**Files:**
- Modify: `packages/components/src/components/charts/bc-chart-gauge/bc-chart-gauge.tsx`
- Modify: `packages/components/src/components/charts/bc-chart-gauge/bc-chart-gauge.css`

Same shared pattern but with:
- `GaugeChart` registration
- No xAxis/yAxis/grid
- Props: `value`, `max`, `min` (string, parsed to number)
- Additional props: `@Prop() segments: string = '';` (JSON array of color stops)

---

## Task 11: Refactor bc-chart-scorecard

**Files:**
- Modify: `packages/components/src/components/charts/bc-chart-scorecard/bc-chart-scorecard.tsx`
- Modify: `packages/components/src/components/charts/bc-chart-scorecard/bc-chart-scorecard.css`

Same shared pattern but with:
- `GaugeChart` registration
- Half-gauge (startAngle: 180, endAngle: 0)
- Props: `value`, `target`, `label`

---

## Task 12: New bc-chart-scatter

**Files:**
- Create: `packages/components/src/components/charts/bc-chart-scatter/bc-chart-scatter.tsx`
- Create: `packages/components/src/components/charts/bc-chart-scatter/bc-chart-scatter.css`

Registration: `ScatterChart`, `EffectScatterChart`

Data shape:
- Single: `[{name, value: [x, y]}]` or `[[x, y], [x, y]]`
- Multi: `{series: [{name, data: [[x,y], [x,y]]}]}`

Additional props:
- `@Prop() bubble: boolean = false;` — if true, data is `[[x, y, size]]` and symbolSize maps to 3rd dimension
- `@Prop() effectScatter: boolean = false;` — ripple effect on points

---

## Task 13: New bc-chart-radar

**Files:**
- Create: `packages/components/src/components/charts/bc-chart-radar/bc-chart-radar.tsx`
- Create: `packages/components/src/components/charts/bc-chart-radar/bc-chart-radar.css`

Registration: `RadarChart`, `RadarComponent`

Data shape:
```json
{
  "indicators": [{"name": "Sales", "max": 100}, {"name": "Admin", "max": 100}],
  "series": [{"name": "Budget", "data": [80, 60]}, {"name": "Actual", "data": [90, 50]}]
}
```

Additional props:
- `@Prop() shape: string = 'polygon';` ('polygon' | 'circle')
- `@Prop() filled: boolean = false;` — areaStyle

---

## Task 14: New bc-chart-treemap

**Files:**
- Create: `packages/components/src/components/charts/bc-chart-treemap/bc-chart-treemap.tsx`
- Create: `packages/components/src/components/charts/bc-chart-treemap/bc-chart-treemap.css`

Registration: `TreemapChart`, `VisualMapComponent`

Data shape: hierarchical `{name, value, children: [...]}`

Additional props:
- `@Prop() leafDepth: number = 0;` — drill-down depth (0 = show all)
- `@Prop() roam: boolean = true;`

---

## Task 15: New bc-chart-candlestick

**Files:**
- Create: `packages/components/src/components/charts/bc-chart-candlestick/bc-chart-candlestick.tsx`
- Create: `packages/components/src/components/charts/bc-chart-candlestick/bc-chart-candlestick.css`

Registration: `CandlestickChart`, `MarkLineComponent`, `MarkPointComponent`

Data shape: `{categories: ["2024-01",...], data: [[open, close, low, high], ...]}`

Additional props:
- `@Prop() showMA: boolean = false;` — show moving average line
- `@Prop() maPeriod: number = 5;`

---

## Task 16: New bc-chart-mixed

**Files:**
- Create: `packages/components/src/components/charts/bc-chart-mixed/bc-chart-mixed.tsx`
- Create: `packages/components/src/components/charts/bc-chart-mixed/bc-chart-mixed.css`

Registration: `BarChart`, `LineChart`, `ScatterChart`

The "combo chart" — each series specifies its own `type`.

Data shape:
```json
{
  "categories": ["Jan","Feb","Mar"],
  "series": [
    {"name": "Revenue", "type": "bar", "data": [100, 200, 300]},
    {"name": "Trend", "type": "line", "data": [150, 180, 250]},
    {"name": "Target", "type": "scatter", "data": [120, 220, 280]}
  ]
}
```

Additional props:
- `@Prop() dualAxis: boolean = false;` — second yAxis for the second series type
- `@Prop() stacked: string = '';` — JSON map of stack groups `{"Revenue":"group1","Cost":"group1"}`

---

## Task 17: New bc-chart-sankey

**Files:**
- Create: `packages/components/src/components/charts/bc-chart-sankey/bc-chart-sankey.tsx`
- Create: `packages/components/src/components/charts/bc-chart-sankey/bc-chart-sankey.css`

Registration: `SankeyChart`

Data shape: `{nodes: [{name}], links: [{source, target, value}]}`

Additional props:
- `@Prop() orient: string = 'horizontal';` ('horizontal' | 'vertical')
- `@Prop() draggable: boolean = true;`

---

## Task 18: New bc-chart-sunburst

**Files:**
- Create: `packages/components/src/components/charts/bc-chart-sunburst/bc-chart-sunburst.tsx`
- Create: `packages/components/src/components/charts/bc-chart-sunburst/bc-chart-sunburst.css`

Registration: `SunburstChart`

Data shape: hierarchical `{name, value, children: [...]}`

Additional props:
- `@Prop() innerRadius: string = '0%';`
- `@Prop() outerRadius: string = '90%';`

---

## Task 19: New bc-chart-boxplot

**Files:**
- Create: `packages/components/src/components/charts/bc-chart-boxplot/bc-chart-boxplot.tsx`
- Create: `packages/components/src/components/charts/bc-chart-boxplot/bc-chart-boxplot.css`

Registration: `BoxplotChart`

Data shape:
```json
{
  "categories": ["Group A", "Group B"],
  "data": [[850, 900, 950, 1000, 1050], [800, 850, 900, 950, 1000]]
}
```
Each inner array: [min, Q1, median, Q3, max]

---

## Task 20: New bc-chart-graph

**Files:**
- Create: `packages/components/src/components/charts/bc-chart-graph/bc-chart-graph.tsx`
- Create: `packages/components/src/components/charts/bc-chart-graph/bc-chart-graph.css`

Registration: `GraphChart`

Data shape: `{nodes: [{name, value?, category?}], links: [{source, target, value?}], categories?: [{name}]}`

Additional props:
- `@Prop() layout: string = 'force';` ('force' | 'circular' | 'none')
- `@Prop() roam: boolean = true;`
- `@Prop() draggable: boolean = true;`

---

## Task 21: New bc-chart-tree

**Files:**
- Create: `packages/components/src/components/charts/bc-chart-tree/bc-chart-tree.tsx`
- Create: `packages/components/src/components/charts/bc-chart-tree/bc-chart-tree.css`

Registration: `TreeChart`

Data shape: hierarchical `{name, value?, children: [...]}`

Additional props:
- `@Prop() layout: string = 'orthogonal';` ('orthogonal' | 'radial')
- `@Prop() orient: string = 'LR';` ('LR' | 'RL' | 'TB' | 'BT')

---

## Task 22: New bc-chart-polar

**Files:**
- Create: `packages/components/src/components/charts/bc-chart-polar/bc-chart-polar.tsx`
- Create: `packages/components/src/components/charts/bc-chart-polar/bc-chart-polar.css`

Registration: `BarChart`, `LineChart`, `PolarComponent`

Data shape: same as bar/line but rendered in polar coordinates.

Additional props:
- `@Prop() polarType: string = 'bar';` ('bar' | 'line')
- `@Prop() stacked: boolean = false;`

---

## Task 23: New bc-chart-parallel

**Files:**
- Create: `packages/components/src/components/charts/bc-chart-parallel/bc-chart-parallel.tsx`
- Create: `packages/components/src/components/charts/bc-chart-parallel/bc-chart-parallel.css`

Registration: `ParallelChart`, `ParallelComponent`

Data shape: `{dimensions: ["cost","revenue","profit"], data: [[100, 200, 100], ...]}`

---

## Task 24: New bc-chart-themeriver

**Files:**
- Create: `packages/components/src/components/charts/bc-chart-themeriver/bc-chart-themeriver.tsx`
- Create: `packages/components/src/components/charts/bc-chart-themeriver/bc-chart-themeriver.css`

Registration: `ThemeRiverChart`, `SingleAxisComponent`

Data shape: `[[date, value, name], ...]` e.g. `[["2024-01", 100, "Sales"], ["2024-01", 50, "Cost"]]`

---

## Task 25: New bc-chart-pictorialbar

**Files:**
- Create: `packages/components/src/components/charts/bc-chart-pictorialbar/bc-chart-pictorialbar.tsx`
- Create: `packages/components/src/components/charts/bc-chart-pictorialbar/bc-chart-pictorialbar.css`

Registration: `PictorialBarChart`

Data shape: same as bar `[{name, value}]`

Additional props:
- `@Prop() symbol: string = 'roundRect';` — ECharts symbol type or SVG path

---

## Task 26: New bc-chart-custom

**Files:**
- Create: `packages/components/src/components/charts/bc-chart-custom/bc-chart-custom.tsx`
- Create: `packages/components/src/components/charts/bc-chart-custom/bc-chart-custom.css`

**The escape hatch component.** Accepts raw ECharts option JSON.

Registration: ALL chart types + ALL components (this is the full-bundle component for maximum flexibility).

```typescript
import { Component, Prop, Element, Watch, Method, Event, EventEmitter, h } from '@stencil/core';
import { ChartClickEvent, ChartHoverEvent, DataFetcher } from '../../../core/types';
import { echarts, ECharts, initChart, disposeChart, setupResizeObserver, fetchChartData } from '../../../core/chart-utils';

// Register ALL chart types for maximum flexibility
import { BarChart, LineChart, PieChart, ScatterChart, RadarChart, TreeChart, TreemapChart, GraphChart, GaugeChart, FunnelChart, ParallelChart, SankeyChart, BoxplotChart, CandlestickChart, EffectScatterChart, HeatmapChart, PictorialBarChart, ThemeRiverChart, SunburstChart, CustomChart, LinesChart, MapChart } from 'echarts/charts';
import { RadarComponent, PolarComponent, GeoComponent, SingleAxisComponent, ParallelComponent as ParallelComp, CalendarComponent, VisualMapComponent, MarkPointComponent, MarkLineComponent, MarkAreaComponent, TimelineComponent, GraphicComponent, BrushComponent } from 'echarts/components';

echarts.use([BarChart, LineChart, PieChart, ScatterChart, RadarChart, TreeChart, TreemapChart, GraphChart, GaugeChart, FunnelChart, ParallelChart, SankeyChart, BoxplotChart, CandlestickChart, EffectScatterChart, HeatmapChart, PictorialBarChart, ThemeRiverChart, SunburstChart, CustomChart, LinesChart, MapChart, RadarComponent, PolarComponent, GeoComponent, SingleAxisComponent, ParallelComp, CalendarComponent, VisualMapComponent, MarkPointComponent, MarkLineComponent, MarkAreaComponent, TimelineComponent, GraphicComponent, BrushComponent]);

@Component({ tag: 'bc-chart-custom', styleUrl: 'bc-chart-custom.css', shadow: false })
export class BcChartCustom {
  @Element() el!: HTMLElement;
  @Prop({ mutable: true }) option: string = '{}';
  @Prop() height: string = '400px';
  @Prop() width: string = '100%';
  @Prop({ mutable: true }) loading: boolean = false;
  @Prop() theme: string = '';
  @Prop() renderer: string = 'canvas';
  @Prop() dataSource: string = '';
  @Prop() fetchHeaders: string = '';
  @Prop() refreshInterval: number = 0;
  dataFetcher?: DataFetcher;

  private chart: ECharts | null = null;
  private _refreshTimer: ReturnType<typeof setInterval> | null = null;
  private _resizeObserver: ResizeObserver | null = null;

  @Event() lcChartClick!: EventEmitter<ChartClickEvent>;
  @Event() lcChartHover!: EventEmitter<ChartHoverEvent>;
  @Event() lcChartReady!: EventEmitter<void>;

  componentDidLoad() {
    this.chart = initChart(this.el, this.theme, this.renderer, {
      lcChartClick: this.lcChartClick,
      lcChartHover: this.lcChartHover,
      lcChartReady: this.lcChartReady,
    });
    this._resizeObserver = setupResizeObserver(this.el, this.chart);
    this.renderChart();
    if (this.refreshInterval > 0) this._refreshTimer = setInterval(() => this.refresh(), this.refreshInterval);
  }

  @Watch('option') onOptionChange() { this.renderChart(); }
  disconnectedCallback() { disposeChart(this.chart, this._refreshTimer, this._resizeObserver); }

  private renderChart() {
    if (!this.chart) return;
    try {
      const opt = JSON.parse(this.option);
      this.chart.setOption(opt, true);
    } catch { /* invalid option JSON */ }
  }

  @Method() async updateData(newData: unknown): Promise<void> { this.option = typeof newData === 'string' ? newData : JSON.stringify(newData); }
  @Method() async setData(newData: unknown): Promise<void> { this.option = typeof newData === 'string' ? newData : JSON.stringify(newData); }
  @Method() async setOption(opt: unknown): Promise<void> { this.option = typeof opt === 'string' ? opt : JSON.stringify(opt); }
  @Method() async refresh(): Promise<void> { this.renderChart(); }
  @Method() async resize(): Promise<void> { this.chart?.resize(); }
  @Method() async exportImage(format: string = 'png'): Promise<string> { return this.chart?.getDataURL({ type: format as 'png' | 'jpeg' | 'svg', pixelRatio: 2 }) || ''; }
  @Method() async getChartInstance(): Promise<unknown> { return this.chart; }

  render() {
    return (
      <div class={{ 'bc-chart-wrap': true, 'bc-chart-loading': this.loading }}>
        {this.loading && <div class="bc-chart-loading-overlay"><span class="bc-field-loading-indicator" /></div>}
        <div class="bc-echart" style={{ height: this.height, width: this.width }}></div>
      </div>
    );
  }
}
```

---

## Task 27: Build Verification

**Step 1: Full build**

Run: `cd packages/components && npx stencil build 2>&1`
Expected: Build succeeds with 0 errors

**Step 2: Verify component count**

Run: `ls packages/components/src/components/charts/ | wc -l`
Expected: 29 directories (26 ECharts + 3 pure CSS)

---

## Task 28: Documentation — Update All Chart Docs

**Files:**
- Modify: 11 existing docs in `packages/components/docs/charts/`
- Create: 15 new docs in `packages/components/docs/charts/`
- Modify: `packages/components/docs/README.md` — update Charts section count and table

Each doc follows the existing pattern (see `bc-chart-bar.md`):

```markdown
# bc-chart-{type}

> {Description} (ECharts)

## Quick Start

```html
<bc-chart-{type} data='...' />
```

## Props

| Prop | Type | Default | Description |
|------|------|---------|-------------|
| data | string (JSON) | '[]' | Chart data |
| chart-title | string | '' | Chart title |
| ... (all shared + type-specific props)

## Data Format

{type-specific data shape examples}

## Events

| Event | Payload |
|-------|---------|
| lcChartClick | {name, value, dataIndex} |
| lcChartHover | {name, value, dataIndex} |
| lcChartLegendSelect | {name, selected} |
| lcChartDataZoom | {start, end} |
| lcChartReady | void |

## Methods

| Method | Returns |
|--------|---------|
| updateData(data) | Promise<void> |
| setData(data) | Promise<void> |
| refresh() | Promise<void> |
| resize() | Promise<void> |
| exportImage(format?) | Promise<string> |
| getChartInstance() | Promise<ECharts> |

See [theming](../theming.md), [data-fetching](../data-fetching.md).
```

New docs to create:
1. `bc-chart-scatter.md`
2. `bc-chart-radar.md`
3. `bc-chart-treemap.md`
4. `bc-chart-candlestick.md`
5. `bc-chart-mixed.md`
6. `bc-chart-sankey.md`
7. `bc-chart-sunburst.md`
8. `bc-chart-boxplot.md`
9. `bc-chart-graph.md`
10. `bc-chart-tree.md`
11. `bc-chart-polar.md`
12. `bc-chart-parallel.md`
13. `bc-chart-themeriver.md`
14. `bc-chart-pictorialbar.md`
15. `bc-chart-custom.md`

---

## Task 29: Update Project Documentation

**Files:**
- Modify: `packages/components/docs/README.md` — Charts section: 11 → 29
- Modify: `docs/codebase.md` — add new chart component entries
- Modify: `docs/features.md` — update chart feature status
- Modify: `README.md` — update component count (103 → 121)

---

## Task 30: Final Build + Verification

**Step 1: Full build**
Run: `cd packages/components && npx stencil build 2>&1`
Expected: 0 errors

**Step 2: Test**
Run: `cd packages/components && npx stencil test --spec 2>&1`
Expected: All tests pass

**Step 3: Verify component count**
Count all component directories in `src/components/` to confirm total is 121 (was 103 + 18 new chart components).

---

## Execution Order Summary

| Phase | Tasks | Description |
|-------|-------|-------------|
| **Phase 1: Foundation** | 1-3 | Types, chart-utils.ts, registration map |
| **Phase 2: Refactor Existing** | 4-11 | Refactor 8 existing ECharts components |
| **Phase 3: P0 New Charts** | 12-16 | scatter, radar, treemap, candlestick, mixed |
| **Phase 4: P1 New Charts** | 17-22 | sankey, sunburst, boxplot, graph, tree, polar |
| **Phase 5: P2 New Charts** | 23-26 | parallel, themeriver, pictorialbar, custom |
| **Phase 6: Verification** | 27 | Build verification |
| **Phase 7: Documentation** | 28-29 | All docs updated |
| **Phase 8: Final** | 30 | Final build + test |
