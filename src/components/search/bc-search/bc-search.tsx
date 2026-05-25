import { Component, Prop, State, Event, EventEmitter, Element, h } from '@stencil/core';
import { getApiClient } from '../../../core/api-client';
import { fetchData } from '../../../core/data-fetcher';
import { BcSetup } from '../../../core/bc-setup';
import { DataFetcher } from '../../../core/types';
import { i18n } from '../../../core/i18n';

@Component({ tag: 'bc-search', styleUrl: 'bc-search.css', shadow: false })
export class BcSearch {
  @Element() el!: HTMLElement;
  @Prop({ mutable: true }) value: string = '';
  @Prop() placeholder: string = '';
  @Prop() model: string = '';
  @Prop() localData?: string;
  @Prop() dataSource: string = '';
  @Prop() fetchHeaders: string = '';
  @Prop() fetchOptions?: string;
  dataFetcher?: DataFetcher;
  @State() suggestions: Array<Record<string, unknown>> = [];
  @State() showSuggestions: boolean = false;
  @State() loading: boolean = false;
  @Event() lcSearch!: EventEmitter<{query: string}>;
  @Event() lcError!: EventEmitter<{message: string}>;
  private debounceTimer: ReturnType<typeof setTimeout> | null = null;

  componentWillRender() { this.el.dir = i18n.dir; }

  private async doSearch(q: string) {
    this.loading = true;
    try {
      if (this.dataFetcher) {
        const result = await this.dataFetcher({ search: q, pageSize: 10 });
        this.suggestions = result.data as Array<Record<string, unknown>>;
      } else if (this.dataSource) {
        const baseUrl = BcSetup.getBaseUrl();
        let url = this.dataSource;
        if (url && !url.startsWith('http') && baseUrl) url = baseUrl + url;
        const headers = { ...BcSetup.getHeaders(), ...(this.fetchHeaders ? JSON.parse(this.fetchHeaders) : {}) };
        const sep = url.includes('?') ? '&' : '?';
        const res = await fetch(`${url}${sep}q=${encodeURIComponent(q)}&pageSize=10`, { headers });
        if (!res.ok) throw new Error(`HTTP ${res.status}`);
        const json = await res.json();
        this.suggestions = json.data || json;
      } else if (this.model) {
        try {
          const result = await fetchData({ element: this.el, model: this.model, localData: this.localData, fetchOptions: this.fetchOptions ? JSON.parse(this.fetchOptions) : undefined, fetchHeaders: this.fetchHeaders, params: { search: q, pageSize: 10 } });
          this.suggestions = result.data as Array<Record<string, unknown>>;
        } catch {
          const api = getApiClient();
          this.suggestions = await api.search(this.model, q);
        }
      } else {
        this.suggestions = [];
      }
      this.showSuggestions = this.suggestions.length > 0;
    } catch (err) {
      this.suggestions = [];
      this.showSuggestions = false;
      this.lcError.emit({ message: String(err) });
    }
    this.loading = false;
  }

  private handleInput(q: string) {
    this.value = q;
    this.lcSearch.emit({ query: q });
    if (this.debounceTimer) clearTimeout(this.debounceTimer);
    if (q.length < 2) { this.suggestions = []; this.showSuggestions = false; return; }
    if (!this.model && !this.dataSource && !this.dataFetcher) return;
    this.debounceTimer = setTimeout(() => this.doSearch(q), 300);
  }

  private selectSuggestion(item: Record<string, unknown>) {
    this.value = String(item['name'] || item['id'] || '');
    this.showSuggestions = false;
    this.lcSearch.emit({ query: this.value });
  }

  render() {
    return (
      <div class="bc-search-wrapper">
        <div class="bc-search-input-wrap">
          <span class="bc-search-icon">{'\uD83D\uDD0D'}</span>
          <input type="search" class="bc-search-input" value={this.value} placeholder={this.placeholder || i18n.t('common.search')} onInput={(e: Event) => this.handleInput((e.target as HTMLInputElement).value)} onFocus={() => { if (this.suggestions.length > 0) this.showSuggestions = true; }} onBlur={() => setTimeout(() => { this.showSuggestions = false; }, 200)} />
          {this.loading && <span class="bc-search-spinner">...</span>}
        </div>
        {this.showSuggestions && (
          <div class="bc-search-dropdown">
            {this.suggestions.map(item => (
              <div class="bc-search-option" onMouseDown={() => this.selectSuggestion(item)}>
                <span class="bc-search-opt-name">{String(item['name'] || item['id'] || '')}</span>
                {item['description'] && <span class="bc-search-opt-desc">{String(item['description'])}</span>}
              </div>
            ))}
          </div>
        )}
        <slot></slot>
      </div>
    );
  }
}
