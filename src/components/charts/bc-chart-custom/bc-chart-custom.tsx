import { Component, Prop, Element, Watch, Method, Event, EventEmitter, h } from '@stencil/core';
import { ChartClickEvent, ChartHoverEvent, DataFetcher } from '../../../core/types';
import {
  echarts, ECharts,
  initChart, disposeChart, setupResizeObserver, fetchChartData,
} from '../../../core/chart-utils';
import {
  BarChart, LineChart, PieChart, ScatterChart, RadarChart, TreeChart, TreemapChart,
  GraphChart, GaugeChart, FunnelChart, ParallelChart as ParallelSeriesChart, SankeyChart,
  BoxplotChart, CandlestickChart, EffectScatterChart, HeatmapChart, PictorialBarChart,
  ThemeRiverChart, SunburstChart, CustomChart, LinesChart,
} from 'echarts/charts';
import {
  RadarComponent, PolarComponent, GeoComponent, SingleAxisComponent, ParallelComponent,
  CalendarComponent, VisualMapComponent, MarkPointComponent, MarkLineComponent,
  MarkAreaComponent, TimelineComponent, GraphicComponent, BrushComponent,
} from 'echarts/components';

echarts.use([
  BarChart, LineChart, PieChart, ScatterChart, RadarChart, TreeChart, TreemapChart,
  GraphChart, GaugeChart, FunnelChart, ParallelSeriesChart, SankeyChart,
  BoxplotChart, CandlestickChart, EffectScatterChart, HeatmapChart, PictorialBarChart,
  ThemeRiverChart, SunburstChart, CustomChart, LinesChart,
  RadarComponent, PolarComponent, GeoComponent, SingleAxisComponent, ParallelComponent,
  CalendarComponent, VisualMapComponent, MarkPointComponent, MarkLineComponent,
  MarkAreaComponent, TimelineComponent, GraphicComponent, BrushComponent,
]);

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
  @Prop() fetchOptions?: string;
  @Prop() refreshInterval: number = 0;
  @Prop() model: string = '';
  @Prop() localData?: string;
  @Prop() dataFetcher?: DataFetcher;

  private chart: ECharts | null = null;
  private _refreshTimer: ReturnType<typeof setInterval> | null = null;
  private _resizeObserver: ResizeObserver | null = null;

  @Event() lcChartClick!: EventEmitter<ChartClickEvent>;
  @Event() lcChartHover!: EventEmitter<ChartHoverEvent>;
  @Event() lcChartReady!: EventEmitter<void>;
  @Event() lcError!: EventEmitter<string>;

  componentDidLoad() {
    this.chart = initChart(this.el, this.theme, this.renderer, {
      lcChartClick: this.lcChartClick, lcChartHover: this.lcChartHover, lcChartReady: this.lcChartReady,
    });
    this._resizeObserver = setupResizeObserver(this.el, this.chart);
    this.renderChart();
    if (this.localData || this.dataSource || this.dataFetcher || this.model) this._fetchData();
    if (this.refreshInterval > 0) this._refreshTimer = setInterval(() => this._fetchData(), this.refreshInterval);
  }

  @Watch('option') onOptionChange() { this.renderChart(); }
  @Watch('theme') onThemeChange() { this._reinit(); }
  @Watch('renderer') onRendererChange() { this._reinit(); }
  @Watch('dataSource') onDataSourceChange() { if (this.dataSource || this.model) this._fetchData(); }
  @Watch('model') onModelChange() { if (this.dataSource || this.model) this._fetchData(); }

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
      const result = await fetchChartData({ dataFetcher: this.dataFetcher, el: this.el, dataSource: this.dataSource, fetchHeaders: this.fetchHeaders, model: this.model, localData: this.localData, fetchOptions: this.fetchOptions ? JSON.parse(this.fetchOptions) : undefined });
      this.option = JSON.stringify(result);
    } catch (err) { this.lcError.emit(String(err)); }
    this.loading = false;
  }

  private renderChart() {
    if (!this.chart) return;
    let opt: Record<string, unknown>;
    try {
      opt = JSON.parse(this.option);
    } catch {
      return;
    }
    this.chart.setOption(opt, true);
  }

  @Method() async setOption(opt: unknown): Promise<void> { this.option = typeof opt === 'string' ? opt : JSON.stringify(opt); }
  @Method() async updateData(newData: unknown): Promise<void> { this.option = typeof newData === 'string' ? newData : JSON.stringify(newData); }
  @Method() async setData(newData: unknown): Promise<void> { this.option = typeof newData === 'string' ? newData : JSON.stringify(newData); }
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
