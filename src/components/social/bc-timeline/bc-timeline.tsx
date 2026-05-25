import { Component, Prop, State, Element, Watch, Event, EventEmitter, Method, h } from '@stencil/core';
import { getApiClient } from '../../../core/api-client';
import { fetchData } from '../../../core/data-fetcher';
import { BcSetup } from '../../../core/bc-setup';
import { DataFetcher } from '../../../core/types';
import { i18n } from '../../../core/i18n';

interface AuditEntry { id: string; field: string; oldValue: string; newValue: string; user: string; date: string; }

@Component({ tag: 'bc-timeline', styleUrl: 'bc-timeline.css', shadow: false })
export class BcTimeline {
  @Element() el!: HTMLElement;
  @Prop() recordId: string = '';
  @Prop() model: string = '';
  @Prop() localData?: string;
  @Prop() dataSource: string = '';
  @Prop() fetchHeaders: string = '';
  @Prop() fetchOptions?: string;
  dataFetcher?: DataFetcher;
  @State() entries: AuditEntry[] = [];
  @State() loading: boolean = false;
  @Event() lcError!: EventEmitter<{message: string}>;

  componentWillRender() { this.el.dir = i18n.dir; }

  async componentDidLoad() {
    await this.fetchData();
  }

  @Watch('model') @Watch('dataSource')
  onSourceChange() { this.fetchData(); }

  private async fetchData() {
    if (!this.model && !this.dataSource && !this.dataFetcher) return;
    if (this.model && !this.recordId) return;
    this.loading = true;
    try {
      let rows: Array<Record<string, unknown>> = [];
      if (this.dataFetcher) {
        const result = await this.dataFetcher({ pageSize: 50, filters: { record_id: this.recordId } });
        rows = result.data as Array<Record<string, unknown>>;
      } else if (this.dataSource) {
        const baseUrl = BcSetup.getBaseUrl();
        let url = this.dataSource;
        if (url && !url.startsWith('http') && baseUrl) url = baseUrl + url;
        const headers = { ...BcSetup.getHeaders(), ...(this.fetchHeaders ? JSON.parse(this.fetchHeaders) : {}) };
        this.el.dispatchEvent(new CustomEvent('lcBeforeFetch', { detail: { url, headers, params: {} }, bubbles: true, cancelable: true }));
        const res = await fetch(url, { headers });
        if (!res.ok) throw new Error(`HTTP ${res.status}`);
        const json = await res.json();
        rows = json.data || json;
      } else if (this.model) {
        try {
          const result = await fetchData({ element: this.el, model: this.model + '_audit', fetchHeaders: this.fetchHeaders, params: { pageSize: 50 } });
          rows = result.data as Array<Record<string, unknown>>;
        } catch {
          const api = getApiClient();
          const res = await api.list(this.model + '_audit', { pageSize: 50 });
          rows = res.data;
        }
      }
      this.entries = rows.map(r => ({
        id: String(r['id'] || ''),
        field: String(r['field'] || ''),
        oldValue: String(r['old_value'] || ''),
        newValue: String(r['new_value'] || ''),
        user: String(r['user'] || 'System'),
        date: String(r['created_at'] || r['date'] || ''),
      }));
    } catch (err) {
      this.entries = [];
      this.lcError.emit({ message: String(err) });
    }
    this.loading = false;
  }

  @Method() async refresh(): Promise<void> { await this.fetchData(); }

  private formatDate(d: string): string {
    try { return i18n.tf.date(d, { day: 'numeric', month: 'short', hour: '2-digit', minute: '2-digit' }); }
    catch { return d; }
  }

  render() {
    return (
      <div class="bc-timeline">
        <div class="bc-tl-header"><h4>{i18n.t('timeline.title')}</h4></div>
        {this.loading && <div class="bc-tl-loading">{i18n.t('common.loading')}</div>}
        <div class="bc-tl-entries">
          {this.entries.map(e => (
            <div class="bc-tl-entry">
              <div class="bc-tl-dot"></div>
              <div class="bc-tl-content">
                <span class="bc-tl-user">{e.user}</span> changed <span class="bc-tl-field">{e.field}</span>
                {e.oldValue && <span class="bc-tl-old"> from "{e.oldValue}"</span>}
                <span class="bc-tl-new"> to "{e.newValue}"</span>
                <div class="bc-tl-date">{this.formatDate(e.date)}</div>
              </div>
            </div>
          ))}
          {this.entries.length === 0 && !this.loading && <div class="bc-tl-empty">{i18n.t('timeline.noChanges')}</div>}
        </div>
      </div>
    );
  }
}
