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
import type { ECharts, EChartsCoreOption } from 'echarts/core';
import { ChartClickEvent, ChartHoverEvent, ChartLegendSelectEvent, ChartDataZoomEvent, DataFetcher } from './types';
import { fetchData } from './data-fetcher';
import { BcSetup } from './bc-setup';
import { applyChartTheme } from './chart-theme';
import type { EventEmitter } from '@stencil/core';

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

// Register BitCode dark theme matching our CSS variable palette
echarts.registerTheme('dark', {
  color: ['#818cf8', '#22d3ee', '#34d399', '#fbbf24', '#f87171', '#a78bfa', '#f472b6', '#2dd4bf', '#fb923c', '#818cf8'],
  backgroundColor: 'transparent',
  textStyle: { color: '#e2e8f0' },
  title: { textStyle: { color: '#f1f5f9' }, subtextStyle: { color: '#94a3b8' } },
  legend: { textStyle: { color: '#cbd5e1' } },
  tooltip: {
    backgroundColor: 'rgba(30, 41, 59, 0.95)',
    borderColor: '#334155',
    textStyle: { color: '#f1f5f9' },
  },
  categoryAxis: {
    axisLine: { lineStyle: { color: '#475569' } },
    axisTick: { lineStyle: { color: '#475569' } },
    axisLabel: { color: '#94a3b8' },
    splitLine: { lineStyle: { color: '#1e293b' } },
  },
  valueAxis: {
    axisLine: { lineStyle: { color: '#475569' } },
    axisTick: { lineStyle: { color: '#475569' } },
    axisLabel: { color: '#94a3b8' },
    splitLine: { lineStyle: { color: '#1e293b' } },
  },
  logAxis: {
    axisLine: { lineStyle: { color: '#475569' } },
    axisTick: { lineStyle: { color: '#475569' } },
    axisLabel: { color: '#94a3b8' },
    splitLine: { lineStyle: { color: '#1e293b' } },
  },
  timeAxis: {
    axisLine: { lineStyle: { color: '#475569' } },
    axisTick: { lineStyle: { color: '#475569' } },
    axisLabel: { color: '#94a3b8' },
    splitLine: { lineStyle: { color: '#1e293b' } },
  },
  radar: {
    name: { textStyle: { color: '#94a3b8' } },
    axisLine: { lineStyle: { color: '#475569' } },
    splitLine: { lineStyle: { color: '#334155' } },
    splitArea: { areaStyle: { color: ['rgba(51,65,85,0.3)', 'rgba(51,65,85,0.1)'] } },
  },
  toolbox: { iconStyle: { borderColor: '#94a3b8' }, emphasis: { iconStyle: { borderColor: '#818cf8' } } },
  dataZoom: {
    backgroundColor: 'rgba(30,41,59,0.6)',
    dataBackgroundColor: '#334155',
    fillerColor: 'rgba(129,140,248,0.15)',
    handleColor: '#818cf8',
    textStyle: { color: '#94a3b8' },
  },
  markLine: { lineStyle: { color: '#64748b' } },
});

export { echarts };
export type { ECharts, EChartsCoreOption as EChartsOption };

const chartThemeCleanup = new WeakMap<ECharts, () => void>();
const chartThemeObservers = new WeakMap<ECharts, MutationObserver>();

function applyChartThemePatch(chart: ECharts, theme: string): void {
  const option = chart.getOption() as Record<string, unknown>;
  if (!option || Object.keys(option).length === 0) return;
  chart.setOption(applyChartTheme(option, theme), true);
}

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
  data: Array<[number, number, number, number]>;
}

export interface ParallelData {
  dimensions: string[];
  data: number[][];
}

export interface RadarData {
  indicators: Array<{ name: string; max: number }>;
  series: Array<{ name: string; data: number[] }>;
}

export interface HeatmapStructuredData {
  xAxis: string[];
  yAxis: string[];
  data: number[][];
}

export type ChartData =
  | SingleSeriesItem[]
  | MultiSeriesData
  | HierarchicalData
  | GraphData
  | CandlestickData
  | ParallelData
  | RadarData
  | HeatmapStructuredData
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
  return data !== null && typeof data === 'object' && !Array.isArray(data) && 'name' in data && ('children' in data || ('value' in data && !('categories' in data)));
}

export function isGraphData(data: ChartData): data is GraphData {
  return data !== null && typeof data === 'object' && !Array.isArray(data) && 'nodes' in data && 'links' in data;
}

export function isCandlestickData(data: ChartData): data is CandlestickData {
  if (data === null || typeof data !== 'object' || Array.isArray(data)) return false;
  const d = data as Record<string, unknown>;
  return 'categories' in d && 'data' in d && Array.isArray(d.data) && d.data.length > 0 && Array.isArray(d.data[0]) && typeof d.data[0][0] === 'number';
}

export function isParallelData(data: ChartData): data is ParallelData {
  return data !== null && typeof data === 'object' && !Array.isArray(data) && 'dimensions' in data && 'data' in data;
}

export function isRadarData(data: ChartData): data is RadarData {
  return data !== null && typeof data === 'object' && !Array.isArray(data) && 'indicators' in data && 'series' in data;
}

export function isHeatmapStructured(data: ChartData): data is HeatmapStructuredData {
  return data !== null && typeof data === 'object' && !Array.isArray(data) && 'xAxis' in data && 'yAxis' in data && 'data' in data;
}

export function isHeatmapArray(data: ChartData): data is number[][] {
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
  } else {
    // Auto-detect from page theme when no explicit theme prop
    const pageTheme = document.documentElement.getAttribute('data-bc-theme');
    if (pageTheme === 'dark') {
      themeArg = 'dark';
    }
  }

  const chart = echarts.init(container as HTMLElement, themeArg as string | object | undefined, {
    renderer: (renderer === 'svg' ? 'svg' : 'canvas') as 'canvas' | 'svg',
  });
  const baseSetOption = chart.setOption.bind(chart) as (
    option: unknown,
    notMerge?: boolean,
    lazyUpdate?: boolean,
  ) => void;
  chart.setOption = ((option: unknown, notMerge?: boolean, lazyUpdate?: boolean) => {
    if (option && typeof option === 'object' && !Array.isArray(option)) {
      return baseSetOption(applyChartTheme(option as Record<string, unknown>, theme), notMerge, lazyUpdate);
    }
    return baseSetOption(option, notMerge, lazyUpdate);
  }) as typeof chart.setOption;

  const syncTheme = () => applyChartThemePatch(chart, theme);
  const unsubscribeTheme = BcSetup.onThemeChange(() => {
    if (!theme) syncTheme();
  });
  chartThemeCleanup.set(chart, unsubscribeTheme);

  if (typeof MutationObserver !== 'undefined' && typeof document !== 'undefined') {
    const observer = new MutationObserver(() => {
      if (!theme) syncTheme();
    });
    observer.observe(document.documentElement, { attributes: true, attributeFilter: ['data-bc-theme'] });
    chartThemeObservers.set(chart, observer);
  }

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
    chart.on('legendselectchanged', ((...args: unknown[]) => {
      const params = args[0] as Record<string, unknown>;
      legendEmitter.emit({
        name: params.name as string,
        selected: params.selected as Record<string, boolean>,
      });
    }) as (...args: unknown[]) => void);
  }

  if (events.lcChartDataZoom) {
    const zoomEmitter = events.lcChartDataZoom;
    chart.on('datazoom', ((...args: unknown[]) => {
      const params = args[0] as Record<string, unknown>;
      zoomEmitter.emit({
        start: params.start as number,
        end: params.end as number,
        startValue: params.startValue,
        endValue: params.endValue,
      });
    }) as (...args: unknown[]) => void);
  }

  if (events.lcChartReady) {
    events.lcChartReady.emit();
  }

  return chart;
}

export function disposeChart(chart: ECharts | null, timer: ReturnType<typeof setInterval> | null, observer: ResizeObserver | null): void {
  if (chart) {
    chartThemeCleanup.get(chart)?.();
    chartThemeCleanup.delete(chart);
    chartThemeObservers.get(chart)?.disconnect();
    chartThemeObservers.delete(chart);
  }
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
  model?: string;
  localData?: string | unknown[];
  fetchOptions?: RequestInit;
}): Promise<unknown[]> {
  const result = await fetchData({
    fetcher: opts.dataFetcher,
    element: opts.el,
    dataSource: opts.dataSource,
    fetchHeaders: opts.fetchHeaders,
    model: opts.model,
    localData: opts.localData,
    fetchOptions: opts.fetchOptions,
  });
  return result.data;
}
