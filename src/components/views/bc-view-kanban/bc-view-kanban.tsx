import { Component, Method, Prop, State, Event, EventEmitter, Element, Watch, h } from '@stencil/core';
import { getApiClient } from '../../../core/api-client';
import { fetchData } from '../../../core/data-fetcher';
import { BcSetup } from '../../../core/bc-setup';
import { DataFetcher } from '../../../core/types';
import { i18n } from '../../../core/i18n';
import Sortable from 'sortablejs';

@Component({ tag: 'bc-view-kanban', styleUrl: 'bc-view-kanban.css', shadow: false })
export class BcViewKanban {
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
  @State() columns: Map<string, Array<Record<string, unknown>>> = new Map();
  @State() loading: boolean = false;
  @Event() lcKanbanMove!: EventEmitter<{id: string; from: string; to: string}>;
  @Event() lcKanbanColumnReorder!: EventEmitter<{columns: string[]}>;
  @Event() lcError!: EventEmitter<{message: string}>;

  private sortables: Sortable[] = [];
  private columnSortable?: Sortable;

  private getConfig(): Record<string, unknown> { try { return JSON.parse(this.config); } catch { return {}; } }
  private getFields(): string[] { try { return JSON.parse(this.fields); } catch { return []; } }

  componentWillRender() { this.el.dir = i18n.dir; }

  async componentDidLoad() {
    await this.fetchData();
    this.initSortable();
  }

  disconnectedCallback() {
    this.sortables.forEach(s => { s.destroy(); });
    this.columnSortable?.destroy();
  }

  @Watch('model') @Watch('dataSource')
  onSourceChange() { this.fetchData().then(() => this.initSortable()); }

  private async fetchData() {
    if (!this.model && !this.dataSource && !this.dataFetcher) return;
    const cfg = this.getConfig();
    const groupBy = String(cfg['group_by'] || 'status');
    this.loading = true;
    try {
      let rows: Array<Record<string, unknown>> = [];
      if (this.dataFetcher) {
        const result = await this.dataFetcher({ pageSize: 200 });
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
          const result = await fetchData({ element: this.el, model: this.model, localData: this.localData, fetchOptions: this.fetchOptions ? JSON.parse(this.fetchOptions) : undefined, fetchHeaders: this.fetchHeaders, params: { pageSize: 200 } });
          rows = result.data as Array<Record<string, unknown>>;
        } catch {
          const api = getApiClient();
          const res = await api.list(this.model, { pageSize: 100 });
          rows = res.data;
        }
      }
      const cols = new Map<string, Array<Record<string, unknown>>>();
      for (const row of rows) {
        const col = String(row[groupBy] || 'Other');
        if (!cols.has(col)) cols.set(col, []);
        cols.get(col)!.push(row);
      }
      this.columns = cols;
    } catch (err) {
      this.columns = new Map();
      this.lcError.emit({ message: String(err) });
    }
    this.loading = false;
  }

  private initSortable() {
    this.sortables.forEach(s => { s.destroy(); });
    this.sortables = [];
    this.columnSortable?.destroy();
    setTimeout(() => {
      const board = this.el.querySelector('.bc-kanban-board') as HTMLElement;
      if (board) {
        this.columnSortable = Sortable.create(board, {
          animation: 150,
          handle: '.bc-kanban-col-header',
          draggable: '.bc-kanban-column',
          ghostClass: 'bc-kanban-col-ghost',
          onEnd: () => {
            const cols = Array.from(board.querySelectorAll('.bc-kanban-column'));
            const names = cols.map(c => {
              const cards = c.querySelector('.bc-kanban-cards');
              return cards?.getAttribute('data-column') || '';
            }).filter(Boolean);
            this.lcKanbanColumnReorder.emit({ columns: names });
          },
        });
      }
      this.el.querySelectorAll('.bc-kanban-cards').forEach(list => {
        const s = Sortable.create(list as HTMLElement, {
          group: 'kanban', animation: 150, ghostClass: 'bc-kanban-ghost',
          onEnd: (evt) => {
            const id = evt.item.getAttribute('data-id') || '';
            const from = evt.from.getAttribute('data-column') || '';
            const to = evt.to.getAttribute('data-column') || '';
            if (from !== to) this.lcKanbanMove.emit({ id, from, to });
          },
        });
        this.sortables.push(s);
      });
    }, 100);
  }

  @Method() async refresh(): Promise<void> { await this.fetchData(); this.initSortable(); }

  render() {
    const fields = this.getFields();
    return (
      <div class="bc-view bc-view-kanban">
        <div class="bc-kanban-header"><h2>{this.viewTitle || this.model}</h2></div>
        {this.loading && <div class="bc-kanban-loading">{i18n.t('common.loading')}</div>}
        <div class="bc-kanban-board">
          {Array.from(this.columns.entries()).map(([colName, cards]) => (
            <div class="bc-kanban-column">
              <div class="bc-kanban-col-header">
                <span>{colName}</span><span class="bc-kanban-col-count">{cards.length}</span>
              </div>
              <div class="bc-kanban-cards" data-column={colName}>
                {cards.map(card => (
                  <div class="bc-kanban-card" data-id={String(card['id'] || '')}>
                    {fields.slice(0, 3).map(f => (
                      <div class="bc-kanban-card-field">
                        <span class="bc-kf-label">{f}</span>
                        <span class="bc-kf-value">{String(card[f] ?? '')}</span>
                      </div>
                    ))}
                  </div>
                ))}
              </div>
            </div>
          ))}
        </div>
      </div>
    );
  }
}

