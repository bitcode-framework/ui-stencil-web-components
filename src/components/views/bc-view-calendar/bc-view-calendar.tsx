import { Component, Method, Prop, State, Element, Watch, Event, EventEmitter, h } from '@stencil/core';
import { i18n } from '../../../core/i18n';
import { Calendar } from '@fullcalendar/core';
import dayGridPlugin from '@fullcalendar/daygrid';
import interactionPlugin from '@fullcalendar/interaction';
import { getApiClient } from '../../../core/api-client';
import { fetchData } from '../../../core/data-fetcher';
import { BcSetup } from '../../../core/bc-setup';
import { DataFetcher } from '../../../core/types';

@Component({ tag: 'bc-view-calendar', styleUrl: 'bc-view-calendar.css', shadow: false })
export class BcViewCalendar {
  @Element() el!: HTMLElement;
  @Prop() model: string = '';
  @Prop() localData?: string;
  @Prop() viewTitle: string = '';
  @Prop() fields: string = '[]';
  @Prop() config: string = '{}';
  @Prop() dateField: string = 'date';
  @Prop() titleField: string = 'name';
  @Prop() dataSource: string = '';
  @Prop() fetchHeaders: string = '';
  @Prop() fetchOptions?: string;
  dataFetcher?: DataFetcher;
  @State() loading: boolean = false;
  @Event() lcError!: EventEmitter<{message: string}>;
  private calendar: Calendar | null = null;

  componentWillRender() { this.el.dir = i18n.dir; }

  async componentDidLoad() {
    await this.fetchAndRender();
  }

  disconnectedCallback() { this.calendar?.destroy(); }

  @Watch('model') @Watch('dataSource')
  onSourceChange() { this.fetchAndRender(); }

  private async fetchAndRender() {
    const container = this.el.querySelector('.bc-cal-container') as HTMLElement;
    if (!container) return;
    let events: Array<{title: string; start: string; id: string}> = [];
    if (this.model || this.dataSource || this.dataFetcher) {
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
            const res = await api.list(this.model, { pageSize: 200 });
            rows = res.data;
          }
        }
        events = rows.map(r => ({
          id: String(r['id'] || ''),
          title: String(r[this.titleField] || r['name'] || ''),
          start: String(r[this.dateField] || ''),
        }));
      } catch (err) {
        this.lcError.emit({ message: String(err) });
      }
      this.loading = false;
    }
    if (this.calendar) { this.calendar.destroy(); this.calendar = null; }
    this.calendar = new Calendar(container, {
      plugins: [dayGridPlugin, interactionPlugin],
      initialView: 'dayGridMonth',
      events,
      headerToolbar: { left: 'prev,next today', center: 'title', right: 'dayGridMonth,dayGridWeek' },
      editable: true,
      selectable: true,
      height: 'auto',
      eventClick: (info) => { console.log('Event clicked:', info.event.id); },
      dateClick: (info) => { console.log('Date clicked:', info.dateStr); },
    });
    this.calendar.render();
  }

  @Method() async refresh(): Promise<void> { await this.fetchAndRender(); }

  render() {
    return (
      <div class="bc-view bc-view-calendar">
        {this.loading && <div class="bc-cal-loading">{i18n.t('calendar.loadingEvents')}</div>}
        <div class="bc-cal-container"></div>
      </div>
    );
  }
}

