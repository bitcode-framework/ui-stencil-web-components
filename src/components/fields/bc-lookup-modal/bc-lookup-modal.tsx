import { Component, Prop, State, Event, EventEmitter, Element, Watch, h } from '@stencil/core';
import { getApiClient } from '../../../core/api-client';
import { fetchData } from '../../../core/data-fetcher';
import { BcSetup } from '../../../core/bc-setup';
import { DataFetcher } from '../../../core/types';
import { i18n } from '../../../core/i18n';

@Component({ tag: 'bc-lookup-modal', styleUrl: 'bc-lookup-modal.css', shadow: false })
export class BcLookupModal {
  @Element() el!: HTMLElement;
  @Prop({ mutable: true }) open: boolean = false;
  @Prop() model: string = '';
  @Prop() localData?: string;
  @Prop() displayField: string = 'name';
  @Prop() columns: string = '[]';
  @Prop() multiple: boolean = false;
  @Prop() apiUrl: string = '';
  @Prop() modalTitle: string = '';
  @Prop() dataSource: string = '';
  @Prop() fetchHeaders: string = '';
  @Prop() fetchOptions?: string;
  dataFetcher?: DataFetcher;

  @State() data: Array<Record<string, unknown>> = [];
  @State() total: number = 0;
  @State() page: number = 1;
  @State() query: string = '';
  @State() selected: Set<string> = new Set();
  @State() loading: boolean = false;

  @Event() lcLookupSelect!: EventEmitter<{ records: Array<Record<string, unknown>> }>;
  @Event() lcLookupClose!: EventEmitter<void>;
  @Event() lcError!: EventEmitter<{message: string}>;

  private getCols(): Array<{ field: string; label?: string }> {
    try { return JSON.parse(this.columns); } catch { return [{ field: this.displayField }]; }
  }

  componentWillRender() { this.el.dir = i18n.dir; }

  async componentDidLoad() { if (this.open) await this.fetchData(); }

  @Watch('open')
  onOpenChange() { if (this.open) this.fetchData(); }

  private async fetchData() {
    this.loading = true;
    try {
      if (this.dataFetcher) {
        const result = await this.dataFetcher({ page: this.page, pageSize: 10, search: this.query || undefined });
        this.data = result.data as Array<Record<string, unknown>>;
        this.total = result.total;
      } else if (this.dataSource || this.apiUrl) {
        const src = this.dataSource || this.apiUrl;
        const baseUrl = BcSetup.getBaseUrl();
        let url = src;
        if (url && !url.startsWith('http') && baseUrl) url = baseUrl + url;
        const headers = { ...BcSetup.getHeaders(), ...(this.fetchHeaders ? JSON.parse(this.fetchHeaders) : {}) };
        const sep = url.includes('?') ? '&' : '?';
        const q = this.query ? `&q=${encodeURIComponent(this.query)}` : '';
        const res = await fetch(`${url}${sep}page=${this.page}&pageSize=10${q}`, { headers });
        if (!res.ok) throw new Error(`HTTP ${res.status}`);
        const json = await res.json();
        this.data = json.data || json;
        this.total = json.total || this.data.length;
      } else if (this.model) {
        try {
          const result = await fetchData({ element: this.el, model: this.model, localData: this.localData, fetchOptions: this.fetchOptions ? JSON.parse(this.fetchOptions) : undefined, fetchHeaders: this.fetchHeaders, params: { page: this.page, pageSize: 10, search: this.query || undefined } });
          this.data = result.data as Array<Record<string, unknown>>;
          this.total = result.total;
        } catch {
          const api = getApiClient();
          const res = await api.list(this.model, { page: this.page, pageSize: 10, q: this.query || undefined });
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

  private async handleSearch(q: string) {
    this.query = q;
    this.page = 1;
    await this.fetchData();
  }

  private selectRow(row: Record<string, unknown>) {
    if (this.multiple) {
      const id = String(row['id'] || '');
      const s = new Set(this.selected);
      if (s.has(id)) s.delete(id); else s.add(id);
      this.selected = s;
    } else {
      this.lcLookupSelect.emit({ records: [row] });
      this.close();
    }
  }

  private confirmSelection() {
    const selectedRecords = this.data.filter(r => this.selected.has(String(r['id'] || '')));
    this.lcLookupSelect.emit({ records: selectedRecords });
    this.close();
  }

  private close() {
    this.open = false;
    this.query = '';
    this.selected = new Set();
    this.lcLookupClose.emit();
  }

  render() {
    if (!this.open) return null;
    const cols = this.getCols();
    const totalPages = Math.ceil(this.total / 10);
    const title = this.modalTitle || ('Select ' + (this.model || 'Record'));

    return (
      <div class="bc-lookup-overlay" onClick={() => this.close()}>
        <div class="bc-lookup-dialog" onClick={(e) => e.stopPropagation()}>
          <div class="bc-lookup-header">
            <h3>{title}</h3>
            <button type="button" class="bc-lookup-close" onClick={() => this.close()}>{'\u00D7'}</button>
          </div>
          <div class="bc-lookup-search">
            <input type="search" class="bc-lookup-search-input" placeholder={i18n.t('common.search')} value={this.query} onInput={(e) => this.handleSearch((e.target as HTMLInputElement).value)} autoFocus />
            <span class="bc-lookup-total">{i18n.plural('common.records', this.total)}</span>
          </div>
          <div class="bc-lookup-table-wrap">
            {this.loading && <div class="bc-lookup-loading">{i18n.t('common.loading')}</div>}
            <table class="bc-lookup-table">
              <thead><tr>
                {this.multiple && <th class="bc-lookup-check"></th>}
                {cols.map(c => <th>{c.label || c.field}</th>)}
              </tr></thead>
              <tbody>
                {this.data.map(row => {
                  const id = String(row['id'] || '');
                  const isSelected = this.selected.has(id);
                  return (
                    <tr class={'bc-lookup-row ' + (isSelected ? 'selected' : '')} onClick={() => this.selectRow(row)}>
                      {this.multiple && <td class="bc-lookup-check"><input type="checkbox" checked={isSelected} /></td>}
                      {cols.map(c => <td>{String(row[c.field] ?? '')}</td>)}
                    </tr>
                  );
                })}
                {this.data.length === 0 && !this.loading && <tr><td colSpan={cols.length + (this.multiple ? 1 : 0)} class="bc-lookup-empty">{i18n.t('common.noResults')}</td></tr>}
              </tbody>
            </table>
          </div>
          <div class="bc-lookup-footer">
            <div class="bc-lookup-pagination">
              <button type="button" disabled={this.page <= 1} onClick={() => { this.page--; this.fetchData(); }}>{'\u2039'}</button>
              <span>{this.page}/{totalPages || 1}</span>
              <button type="button" disabled={this.page >= totalPages} onClick={() => { this.page++; this.fetchData(); }}>{'\u203A'}</button>
            </div>
            {this.multiple && (
              <div class="bc-lookup-confirm">
                <span>{this.selected.size}</span>
                <button type="button" class="bc-lookup-confirm-btn" onClick={() => this.confirmSelection()} disabled={this.selected.size === 0}>{i18n.t('common.confirm')}</button>
              </div>
            )}
          </div>
        </div>
      </div>
    );
  }
}
