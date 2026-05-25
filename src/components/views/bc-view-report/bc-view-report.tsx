import { Component, Method, Prop, State, Element, Watch, Event, EventEmitter, h } from '@stencil/core';
import { getApiClient } from '../../../core/api-client';
import { fetchData } from '../../../core/data-fetcher';
import { BcSetup } from '../../../core/bc-setup';
import { DataFetcher } from '../../../core/types';
import { i18n } from '../../../core/i18n';

@Component({ tag: 'bc-view-report', styleUrl: 'bc-view-report.css', shadow: false })
export class BcViewReport {
  @Element() el!: HTMLElement;
  @Prop() model: string = '';
  @Prop() localData?: string;
  @Prop() viewTitle: string = '';
  @Prop() fields: string = '[]';
  @Prop() config: string = '{}';
  @Prop() dataSource: string = '';
  @Prop() fetchHeaders: string = '';
  @Prop() fetchOptions?: string;
  dataFetcher?: DataFetcher;
  @State() data: Array<Record<string, unknown>> = [];
  @State() loading: boolean = false;
  @State() visibleFields: string[] = [];
  @Event() lcError!: EventEmitter<{message: string}>;

  private getFields(): string[] { try { return JSON.parse(this.fields); } catch { return []; } }

  componentWillRender() {
    this.el.dir = i18n.dir;
  }

  async componentDidLoad() {
    this.visibleFields = this.getFields();
    await this.fetchData();
  }

  @Watch('model') @Watch('dataSource')
  onSourceChange() { this.fetchData(); }

  private async fetchData() {
    if (!this.model && !this.dataSource && !this.dataFetcher) return;
    this.loading = true;
    try {
      if (this.dataFetcher) {
        const result = await this.dataFetcher({ pageSize: 200 });
        this.data = result.data as Array<Record<string, unknown>>;
      } else if (this.dataSource) {
        const baseUrl = BcSetup.getBaseUrl();
        let url = this.dataSource;
        if (url && !url.startsWith('http') && baseUrl) url = baseUrl + url;
        const headers = { ...BcSetup.getHeaders(), ...(this.fetchHeaders ? JSON.parse(this.fetchHeaders) : {}) };
        this.el.dispatchEvent(new CustomEvent('lcBeforeFetch', { detail: { url, headers, params: {} }, bubbles: true, cancelable: true }));
        const res = await fetch(url, { headers });
        if (!res.ok) throw new Error(`HTTP ${res.status}`);
        const json = await res.json();
        this.data = json.data || json;
      } else if (this.model) {
        try {
          const result = await fetchData({ element: this.el, model: this.model, localData: this.localData, fetchOptions: this.fetchOptions ? JSON.parse(this.fetchOptions) : undefined, fetchHeaders: this.fetchHeaders, params: { pageSize: 200 } });
          this.data = result.data as Array<Record<string, unknown>>;
        } catch {
          const api = getApiClient();
          const res = await api.list(this.model, { pageSize: 200 });
          this.data = res.data;
        }
      }
    } catch (err) {
      this.data = [];
      this.lcError.emit({ message: String(err) });
    }
    this.loading = false;
  }

  private isNumeric(field: string): boolean {
    if (this.data.length === 0) return false;
    const val = this.data[0][field];
    return typeof val === 'number';
  }

  private computeTotal(field: string): number {
    return this.data.reduce((sum, row) => sum + (Number(row[field]) || 0), 0);
  }

  private computeAvg(field: string): number {
    if (this.data.length === 0) return 0;
    return this.computeTotal(field) / this.data.length;
  }

  @Method() async refresh(): Promise<void> { await this.fetchData(); }

  render() {
    return (
      <div class="bc-view bc-view-report">
        <div class="bc-rpt-header">
          <h2>{this.viewTitle || i18n.t('report.title')}</h2>
          <div class="bc-rpt-meta">
            <span class="bc-rpt-count">{i18n.plural('report.rows', this.data.length)}</span>
            <button type="button" class="bc-rpt-export" onClick={() => { console.log('Export CSV'); }}>{i18n.t('report.exportCsv')}</button>
          </div>
        </div>
        {this.loading && <div class="bc-rpt-loading">{i18n.t('common.loading')}</div>}
        <div class="bc-rpt-table-wrap">
          <table class="bc-rpt-table">
            <thead><tr>{this.visibleFields.map(f => <th>{f}</th>)}</tr></thead>
            <tbody>
              {this.data.map(row => (
                <tr>{this.visibleFields.map(f => <td class={{'numeric': this.isNumeric(f)}}>{this.isNumeric(f) ? i18n.tf.number(Number(row[f] || 0)) : String(row[f] ?? '')}</td>)}</tr>
              ))}
            </tbody>
            <tfoot>
              <tr class="bc-rpt-totals">
                {this.visibleFields.map((f, i) => (
                  <td class={{'numeric': this.isNumeric(f)}}>
                    {i === 0 ? i18n.t('common.total') : (this.isNumeric(f) ? i18n.tf.number(this.computeTotal(f)) : '')}
                  </td>
                ))}
              </tr>
              <tr class="bc-rpt-avg">
                {this.visibleFields.map((f, i) => (
                  <td class={{'numeric': this.isNumeric(f)}}>
                    {i === 0 ? i18n.t('report.average') : (this.isNumeric(f) ? i18n.tf.number(this.computeAvg(f), {maximumFractionDigits: 2}) : '')}
                  </td>
                ))}
              </tr>
            </tfoot>
          </table>
        </div>
      </div>
    );
  }
}

