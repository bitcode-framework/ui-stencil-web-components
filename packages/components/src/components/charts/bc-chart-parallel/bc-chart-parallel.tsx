import { Component, Prop, Element, Watch, Method, Event, EventEmitter, h } from '@stencil/core';
import { ChartClickEvent, ChartHoverEvent, DataFetcher } from '../../../core/types';
import {
  echarts, ECharts,
  parseChartData, isParallelData, getColorList,
  initChart, disposeChart, setupResizeObserver, fetchChartData,
  buildTitleOption, buildTooltipOption, buildToolboxOption,
  ParallelData,
} from '../../../core/chart-utils';
import { ParallelChart } from 'echarts/charts';
import { ParallelComponent } from 'echarts/components';

echarts.use([ParallelChart, ParallelComponent]);

@Component({ tag: 'bc-chart-parallel', styleUrl: 'bc-chart-parallel.css', shadow: false })
export class BcChartParallel {
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
  @Prop() refreshInterval: number = 0;
  dataFetcher?: DataFetcher;

  @Prop() theme: string = '';
  @Prop() renderer: string = 'canvas';
  @Prop() toolbox: boolean = false;
  @Prop() locale: string = '';

  private chart: ECharts | null = null;
  private _refreshTimer: ReturnType<typeof setInterval> | null = null;
  private _resizeObserver: ResizeObserver | null = null;

  @Event() lcChartClick!: EventEmitter<ChartClickEvent>;
  @Event() lcChartHover!: EventEmitter<ChartHoverEvent>;
  @Event() lcChartReady!: EventEmitter<void>;

  componentDidLoad() {
    this.chart = initChart(this.el, this.theme, this.renderer, {
      lcChartClick: this.lcChartClick, lcChartHover: this.lcChartHover, lcChartReady: this.lcChartReady,
    });
    this._resizeObserver = setupResizeObserver(this.el, this.chart);
    this.renderChart();
    if (this.dataSource || this.dataFetcher) this._fetchData();
    if (this.refreshInterval > 0) this._refreshTimer = setInterval(() => this._fetchData(), this.refreshInterval);
  }

  @Watch('data') onDataChange() { this.renderChart(); }
  @Watch('theme') onThemeChange() { this._reinit(); }
  @Watch('renderer') onRendererChange() { this._reinit(); }

  disconnectedCallback() { disposeChart(this.chart, this._refreshTimer, this._resizeObserver); }

  private _reinit() {
    disposeChart(this.chart, null, this._resizeObserver);
    this.chart = initChart(this.el, this.theme, this.renderer, {
      lcChartClick: this.lcChartClick, lcChartHover: this.lcChartHover, lcChartReady: this.lcChartReady,
    });
    this._resizeObserver = setupResizeObserver(this.el, this.chart);
    this.renderChart();
  }

  private async _fetchData() {
    this.loading = true;
    try {
      const result = await fetchChartData({ dataFetcher: this.dataFetcher, el: this.el, dataSource: this.dataSource, fetchHeaders: this.fetchHeaders });
      this.data = JSON.stringify(result);
    } catch { /* keep existing */ }
    this.loading = false;
  }

  private renderChart() {
    if (!this.chart) return;
    const parsed = parseChartData(this.data);
    const colorList = getColorList(this.colors);

    if (!isParallelData(parsed)) return;
    const pd = parsed as ParallelData;

    const option: Record<string, unknown> = {
      title: buildTitleOption(this.chartTitle),
      tooltip: buildTooltipOption(this.tooltipEnabled, 'item'),
      toolbox: buildToolboxOption(this.toolbox),
      animation: this.animate,
      color: colorList,
      parallelAxis: pd.dimensions.map((dim, i) => ({ dim: i, name: dim })),
      series: [{
        type: 'parallel',
        data: pd.data,
        lineStyle: { width: 2, opacity: 0.5 },
      }],
    };
    this.chart.setOption(option, true);
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
