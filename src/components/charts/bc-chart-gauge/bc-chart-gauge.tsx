import { Component, Prop, Element, Watch, Method, Event, EventEmitter, h } from '@stencil/core';
import { ChartClickEvent, ChartHoverEvent, ChartLegendSelectEvent, ChartDataZoomEvent } from '../../../core/types';
import {
  echarts, ECharts,
  initChart, disposeChart, setupResizeObserver,
  buildTitleOption,
} from '../../../core/chart-utils';
import { GaugeChart } from 'echarts/charts';

echarts.use([GaugeChart]);

@Component({ tag: 'bc-chart-gauge', styleUrl: 'bc-chart-gauge.css', shadow: false })
export class BcChartGauge {
  @Element() el!: HTMLElement;

  @Prop({ mutable: true }) value: string = '0';
  @Prop() max: string = '100';
  @Prop() min: string = '0';
  @Prop() chartTitle: string = '';
  @Prop() colors: string = '';
  @Prop() height: string = '300px';
  @Prop() width: string = '100%';
  @Prop() loading: boolean = false;
  @Prop() animate: boolean = true;
  @Prop() locale: string = '';

  @Prop() theme: string = '';
  @Prop() renderer: string = 'canvas';
  @Prop() toolbox: boolean = false;
  @Prop() segments: string = '';

  private chart: ECharts | null = null;
  private _resizeObserver: ResizeObserver | null = null;

  @Event() lcChartClick!: EventEmitter<ChartClickEvent>;
  @Event() lcChartHover!: EventEmitter<ChartHoverEvent>;
  @Event() lcChartLegendSelect!: EventEmitter<ChartLegendSelectEvent>;
  @Event() lcChartDataZoom!: EventEmitter<ChartDataZoomEvent>;
  @Event() lcChartReady!: EventEmitter<void>;

  componentDidLoad() {
    this.chart = initChart(this.el, this.theme, this.renderer, {
      lcChartClick: this.lcChartClick, lcChartHover: this.lcChartHover,
      lcChartLegendSelect: this.lcChartLegendSelect, lcChartDataZoom: this.lcChartDataZoom, lcChartReady: this.lcChartReady,
    });
    this._resizeObserver = setupResizeObserver(this.el, this.chart);
    this.renderChart();
  }

  @Watch('value') onValueChange() { this.renderChart(); }
  @Watch('theme') onThemeChange() { this._reinit(); }
  @Watch('renderer') onRendererChange() { this._reinit(); }

  disconnectedCallback() { disposeChart(this.chart, null, this._resizeObserver); }

  private _reinit() {
    disposeChart(this.chart, null, this._resizeObserver);
    this.chart = initChart(this.el, this.theme, this.renderer, {
      lcChartClick: this.lcChartClick, lcChartHover: this.lcChartHover,
      lcChartLegendSelect: this.lcChartLegendSelect, lcChartDataZoom: this.lcChartDataZoom, lcChartReady: this.lcChartReady,
    });
    this._resizeObserver = setupResizeObserver(this.el, this.chart);
    this.renderChart();
  }

  private renderChart() {
    if (!this.chart) return;
    const series: Record<string, unknown> = {
      type: 'gauge',
      min: Number(this.min),
      max: Number(this.max),
      data: [{ value: Number(this.value), name: this.chartTitle }],
      detail: { formatter: '{value}%' },
    };

    if (this.segments) {
      try {
        const parsed = JSON.parse(this.segments);
        if (Array.isArray(parsed)) series.axisLine = { lineStyle: { color: parsed } };
      } catch { /* ignore invalid segments */ }
    }

    this.chart.setOption({
      title: buildTitleOption(this.chartTitle),
      animation: this.animate,
      series: [series],
    }, true);
  }

  @Method() async updateData(newData: unknown): Promise<void> { this.value = String(newData); }
  @Method() async setData(newData: unknown): Promise<void> { this.value = String(newData); }
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
