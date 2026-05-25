import { Component, Prop, Element, Watch, Method, Event, EventEmitter, h } from '@stencil/core';
import { ChartClickEvent, ChartHoverEvent, ChartLegendSelectEvent, ChartDataZoomEvent, DataFetcher } from '../../../core/types';
import {
  echarts, ECharts,
  parseChartData, getColorList,
  initChart, disposeChart, setupResizeObserver, fetchChartData,
  buildTitleOption, buildTooltipOption, buildLegendOption, buildToolboxOption, buildDataZoomOption, buildGridOption,
  ChartData,
} from '../../../core/chart-utils';
import { ScatterChart, EffectScatterChart } from 'echarts/charts';

echarts.use([ScatterChart, EffectScatterChart]);

interface ScatterSeriesData {
  series: Array<{ name: string; data: number[][] }>;
}

function isScatterMultiSeries(data: ChartData): data is ScatterSeriesData {
  return data !== null && typeof data === 'object' && !Array.isArray(data) && 'series' in data;
}

@Component({ tag: 'bc-chart-scatter', styleUrl: 'bc-chart-scatter.css', shadow: false })
export class BcChartScatter {
  @Element() el!: HTMLElement;

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
  @Prop() fetchOptions?: string;
  @Prop() refreshInterval: number = 0;
  @Prop() model: string = '';
  @Prop() localData?: string;
  @Prop() dataFetcher?: DataFetcher;

  @Prop() theme: string = '';
  @Prop() renderer: string = 'canvas';
  @Prop() toolbox: boolean = false;
  @Prop() dataZoom: boolean = false;
  @Prop() locale: string = '';
  @Prop() bubble: boolean = false;
  @Prop() effectScatter: boolean = false;

  private chart: ECharts | null = null;
  private _refreshTimer: ReturnType<typeof setInterval> | null = null;
  private _resizeObserver: ResizeObserver | null = null;

  @Event() lcChartClick!: EventEmitter<ChartClickEvent>;
  @Event() lcChartHover!: EventEmitter<ChartHoverEvent>;
  @Event() lcChartLegendSelect!: EventEmitter<ChartLegendSelectEvent>;
  @Event() lcChartDataZoom!: EventEmitter<ChartDataZoomEvent>;
  @Event() lcChartReady!: EventEmitter<void>;
  @Event() lcError!: EventEmitter<string>;

  componentDidLoad() {
    this.chart = initChart(this.el, this.theme, this.renderer, {
      lcChartClick: this.lcChartClick, lcChartHover: this.lcChartHover,
      lcChartLegendSelect: this.lcChartLegendSelect, lcChartDataZoom: this.lcChartDataZoom, lcChartReady: this.lcChartReady,
    });
    this._resizeObserver = setupResizeObserver(this.el, this.chart);
    this.renderChart();
    if (this.localData || this.dataSource || this.dataFetcher || this.model) this._fetchData();
    if (this.refreshInterval > 0) this._refreshTimer = setInterval(() => this._fetchData(), this.refreshInterval);
  }

  @Watch('data') onDataChange() { this.renderChart(); }
  @Watch('theme') onThemeChange() { this._reinit(); }
  @Watch('renderer') onRendererChange() { this._reinit(); }
  @Watch('dataSource') onDataSourceChange() { if (this.dataSource || this.model) this._fetchData(); }
  @Watch('model') onModelChange() { if (this.dataSource || this.model) this._fetchData(); }

  disconnectedCallback() { disposeChart(this.chart, this._refreshTimer, this._resizeObserver); }

  private _reinit() {
    disposeChart(this.chart, null, this._resizeObserver);
    this.chart = initChart(this.el, this.theme, this.renderer, {
      lcChartClick: this.lcChartClick, lcChartHover: this.lcChartHover,
      lcChartLegendSelect: this.lcChartLegendSelect, lcChartDataZoom: this.lcChartDataZoom, lcChartReady: this.lcChartReady,
    });
    this._resizeObserver = setupResizeObserver(this.el, this.chart);
    this.renderChart();
  }

  private async _fetchData() {
    this.loading = true;
    try {
      const result = await fetchChartData({ dataFetcher: this.dataFetcher, el: this.el, dataSource: this.dataSource, fetchHeaders: this.fetchHeaders, model: this.model, localData: this.localData, fetchOptions: this.fetchOptions ? JSON.parse(this.fetchOptions) : undefined });
      this.data = JSON.stringify(result);
    } catch (err) { this.lcError.emit(String(err)); }
    this.loading = false;
  }

  private renderChart() {
    if (!this.chart) return;
    const parsed = parseChartData(this.data);
    const colorList = getColorList(this.colors);
    const chartType = this.effectScatter ? 'effectScatter' : 'scatter';
    const seriesList: Array<Record<string, unknown>> = [];

    if (isScatterMultiSeries(parsed)) {
      parsed.series.forEach((s, i) => {
        seriesList.push({
          type: chartType,
          name: s.name,
          data: s.data,
          ...(this.bubble ? { symbolSize: (val: number[]) => val[2] || 10 } : {}),
          itemStyle: { color: colorList[i % colorList.length] },
        });
      });
    } else if (Array.isArray(parsed)) {
      const items = parsed as unknown[];
      const rawData: unknown[] = [];
      for (const item of items) {
        if (Array.isArray(item)) {
          rawData.push(item);
        } else if (item !== null && typeof item === 'object' && 'value' in item) {
          rawData.push((item as Record<string, unknown>).value);
        }
      }
      seriesList.push({
        type: chartType,
        data: rawData,
        ...(this.bubble ? { symbolSize: (val: number[]) => val[2] || 10 } : {}),
      });
    }

    const option: Record<string, unknown> = {
      title: buildTitleOption(this.chartTitle),
      tooltip: buildTooltipOption(this.tooltipEnabled, 'item'),
      legend: buildLegendOption(this.legend),
      toolbox: buildToolboxOption(this.toolbox),
      dataZoom: buildDataZoomOption(this.dataZoom),
      grid: buildGridOption(this.dataZoom, this.legend),
      animation: this.animate,
      color: colorList,
      xAxis: { type: 'value' },
      yAxis: { type: 'value' },
      series: seriesList,
    };
    this.chart.setOption(option, true);
  }

  @Method() async updateData(newData: unknown): Promise<void> { this.data = typeof newData === 'string' ? newData : JSON.stringify(newData); }
  @Method() async setData(newData: unknown): Promise<void> { this.data = typeof newData === 'string' ? newData : JSON.stringify(newData); }
  @Method() async refresh(): Promise<void> { if (this.localData || this.dataSource || this.dataFetcher || this.model) await this._fetchData(); else this.renderChart(); }
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
