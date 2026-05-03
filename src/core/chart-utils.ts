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

export { echarts };
export type { ECharts, EChartsCoreOption as EChartsOption };

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
  }

  const chart = echarts.init(container as HTMLElement, themeArg as string | object | undefined, {
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
