import { Component, Prop, Element, Watch, Method, Event, EventEmitter, h } from '@stencil/core';
import { ChartClickEvent, ChartHoverEvent, ChartLegendSelectEvent, ChartDataZoomEvent } from '../../../core/types';
import {
  echarts, ECharts,
  initChart, disposeChart, setupResizeObserver,
} from '../../../core/chart-utils';
import { GaugeChart } from 'echarts/charts';

echarts.use([GaugeChart]);

@Component({ tag: 'bc-chart-scorecard', styleUrl: 'bc-chart-scorecard.css', shadow: false })
export class BcChartScorecard {
  @Element() el!: HTMLElement;

  @Prop({ mutable: true }) value: string = '0';
  @Prop() target: string = '100';
  @Prop() label: string = '';
  @Prop() height: string = '200px';
  @Prop() width: string = '100%';
  @Prop() colors: string = '';
  @Prop() animate: boolean = true;
  @Prop() locale: string = '';

  @Prop() theme: string = '';
  @Prop() renderer: string = 'canvas';
  @Prop() toolbox: boolean = false;

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
    const val = Number(this.value);
    const tgt = Number(this.target);
    const pct = tgt > 0 ? Math.round((val / tgt) * 100) : 0;

    this.chart.setOption({
      animation: this.animate,
      series: [{
        type: 'gauge', startAngle: 180, endAngle: 0,
        min: 0, max: tgt,
        data: [{ value: val, name: this.label }],
        detail: { formatter: pct + '%', fontSize: 20 },
        title: { fontSize: 12 },
      }],
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
      <div class={{ 'bc-chart-wrap': true }}>
        <div class="bc-echart" style={{ height: this.height, width: this.width }}></div>
      </div>
    );
  }
}
