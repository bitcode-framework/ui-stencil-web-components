import { Component, Method, Prop, State, Event, EventEmitter, Element, Watch, h } from '@stencil/core';
import { getApiClient } from '../../../core/api-client';
import { fetchData } from '../../../core/data-fetcher';
import { BcSetup } from '../../../core/bc-setup';
import { DataFetcher } from '../../../core/types';
import { i18n } from '../../../core/i18n';

@Component({
  tag: 'bc-view-list',
  styleUrl: 'bc-view-list.css',
  shadow: false,
})
export class BcViewList {
  @Element() el!: HTMLElement;
  @Prop() model: string = '';
  @Prop() localData?: string;
  @Prop() viewTitle: string = '';
  @Prop() fields: string = '[]';
  @Prop() config: string = '{}';
  @Prop() dataSource: string = '';
  @Prop() fetchHeaders: string = '';
  @Prop() fetchOptions?: string;
  @Prop() refreshInterval: number = 0;
  dataFetcher?: DataFetcher;

  @State() data: Array<Record<string, unknown>> = [];
  @State() total: number = 0;
  @State() page: number = 1;
  @State() pageSize: number = 20;
  @State() sortField: string = '';
  @State() sortOrder: 'asc' | 'desc' = 'asc';
  @State() loading: boolean = false;
  @State() selected: Set<string> = new Set();
  @State() searchQuery: string = '';

  @Event() lcRowSelect!: EventEmitter<{ids: string[]}>;
  @Event() lcError!: EventEmitter<{message: string}>;

  private _rt: ReturnType<typeof setInterval> | null = null;

  private getFields(): string[] { try { return JSON.parse(this.fields); } catch { return []; } }

  componentWillRender() {
    this.el.dir = i18n.dir;
  }

  async componentDidLoad() {
    await this.fetchData();
    if (this.refreshInterval > 0) this._rt = setInterval(() => this.fetchData(), this.refreshInterval);
  }

  disconnectedCallback() {
    if (this._rt) clearInterval(this._rt);
  }

  @Watch('model') @Watch('dataSource')
  onSourceChange() { this.fetchData(); }

  private async fetchData() {
    if (!this.model && !this.dataSource && !this.dataFetcher) return;
    this.loading = true;
    try {
      if (this.dataFetcher) {
        const result = await this.dataFetcher({ page: this.page, pageSize: this.pageSize, sort: this.sortField ? [{ field: this.sortField, direction: this.sortOrder }] : undefined, search: this.searchQuery || undefined });
        this.data = result.data as Array<Record<string, unknown>>;
        this.total = result.total;
      } else if (this.dataSource) {
        const baseUrl = BcSetup.getBaseUrl();
        let url = this.dataSource;
        if (url && !url.startsWith('http') && baseUrl) url = baseUrl + url;
        const headers = { ...BcSetup.getHeaders(), ...(this.fetchHeaders ? JSON.parse(this.fetchHeaders) : {}) };
        const beforeEvent = new CustomEvent('lcBeforeFetch', { detail: { url, headers, params: {} }, bubbles: true, cancelable: true });
        this.el.dispatchEvent(beforeEvent);
        const res = await fetch(beforeEvent.detail.url, { headers: beforeEvent.detail.headers });
        if (!res.ok) throw new Error(`HTTP ${res.status}`);
        const json = await res.json();
        const afterEvent = new CustomEvent('lcAfterFetch', { detail: { response: json, data: null as unknown[] | null, total: 0 }, bubbles: true });
        this.el.dispatchEvent(afterEvent);
        this.data = (afterEvent.detail.data || json.data || json) as Array<Record<string, unknown>>;
        this.total = afterEvent.detail.total || json.total || this.data.length;
      } else if (this.model) {
        try {
          const result = await fetchData({ element: this.el, model: this.model, localData: this.localData, fetchOptions: this.fetchOptions ? JSON.parse(this.fetchOptions) : undefined, fetchHeaders: this.fetchHeaders, params: { page: this.page, pageSize: this.pageSize, sort: this.sortField ? [{ field: this.sortField, direction: this.sortOrder }] : undefined, search: this.searchQuery || undefined } });
          this.data = result.data as Array<Record<string, unknown>>;
          this.total = result.total;
        } catch {
          const api = getApiClient();
          const res = await api.list(this.model, { page: this.page, pageSize: this.pageSize, sort: this.sortField || undefined, order: this.sortField ? this.sortOrder : undefined, q: this.searchQuery || undefined });
          this.data = res.data;
          this.total = res.total;
        }
      }
    } catch (err) {
      this.data = []; this.total = 0;
      this.lcError.emit({ message: String(err) });
    }
    this.loading = false;
  }

  private async handleSort(field: string) {
    if (this.sortField === field) { this.sortOrder = this.sortOrder === 'asc' ? 'desc' : 'asc'; }
    else { this.sortField = field; this.sortOrder = 'asc'; }
    await this.fetchData();
  }

  private async handlePage(p: number) { this.page = p; await this.fetchData(); }

  private toggleSelect(id: string) {
    const s = new Set(this.selected);
    if (s.has(id)) s.delete(id); else s.add(id);
    this.selected = s;
    this.lcRowSelect.emit({ ids: Array.from(s) });
  }

  private toggleSelectAll() {
    if (this.selected.size === this.data.length) { this.selected = new Set(); }
    else { this.selected = new Set(this.data.map(r => String(r['id'] || ''))); }
    this.lcRowSelect.emit({ ids: Array.from(this.selected) });
  }

  private async handleSearch(q: string) { this.searchQuery = q; this.page = 1; await this.fetchData(); }

  @Method() async refresh(): Promise<void> { await this.fetchData(); }

  render() {
    const fields = this.getFields();
    const totalPages = Math.ceil(this.total / this.pageSize);
    return (
      <div class="bc-view bc-view-list">
        <div class="bc-list-header">
          <h2>{this.viewTitle || this.model}</h2>
          <div class="bc-list-actions">
            <input type="search" class="bc-list-search" placeholder={i18n.t('common.search')} value={this.searchQuery} onInput={(e: Event) => this.handleSearch((e.target as HTMLInputElement).value)} />
            <span class="bc-list-count">{i18n.plural('common.records', this.total)}</span>
          </div>
        </div>
        {this.loading && <div class="bc-list-loading">{i18n.t('common.loading')}</div>}
        <div class="bc-list-table-wrap">
          <table class="bc-list-table">
            <thead><tr>
              <th class="bc-list-check"><input type="checkbox" checked={this.selected.size === this.data.length && this.data.length > 0} onChange={() => this.toggleSelectAll()} /></th>
              {fields.map(f => (
                <th class={{'sortable': true, 'sorted': this.sortField === f}} onClick={() => this.handleSort(f)}>
                  {f}{this.sortField === f && <span class="sort-icon">{this.sortOrder === 'asc' ? ' \u25B2' : ' \u25BC'}</span>}
                </th>
              ))}
            </tr></thead>
            <tbody>
              {this.data.map(row => {
                const id = String(row['id'] || '');
                return (<tr class={{'selected': this.selected.has(id)}}>
                  <td class="bc-list-check"><input type="checkbox" checked={this.selected.has(id)} onChange={() => this.toggleSelect(id)} /></td>
                  {fields.map(f => <td>{String(row[f] ?? '')}</td>)}
                </tr>);
              })}
              {this.data.length === 0 && !this.loading && <tr><td colSpan={fields.length + 1} class="bc-list-empty">{i18n.t('datatable.noRecords')}</td></tr>}
            </tbody>
          </table>
        </div>
        {totalPages > 1 && (
          <div class="bc-list-pagination">
            <button type="button" disabled={this.page <= 1} onClick={() => this.handlePage(this.page - 1)}>{'\u2190'} {i18n.t('common.prev')}</button>
            <span>{i18n.t('common.page')} {this.page} {i18n.t('common.of')} {totalPages}</span>
            <button type="button" disabled={this.page >= totalPages} onClick={() => this.handlePage(this.page + 1)}>{i18n.t('common.next')} {'\u2192'}</button>
          </div>
        )}
      </div>
    );
  }
}


