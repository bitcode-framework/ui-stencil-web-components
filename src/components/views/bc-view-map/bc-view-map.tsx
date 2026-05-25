import { Component, Method, Prop, State, Element, Watch, Event, EventEmitter, h } from '@stencil/core';
import { getApiClient } from '../../../core/api-client';
import { fetchData } from '../../../core/data-fetcher';
import { BcSetup } from '../../../core/bc-setup';
import { DataFetcher } from '../../../core/types';
import { i18n } from '../../../core/i18n';
import * as L from 'leaflet';

@Component({ tag: 'bc-view-map', styleUrl: 'bc-view-map.css', shadow: false })
export class BcViewMap {
  @Element() el!: HTMLElement;
  @Prop() model: string = '';
  @Prop() localData?: string;
  @Prop() viewTitle: string = '';
  @Prop() fields: string = '[]';
  @Prop() config: string = '{}';
  @Prop() geoField: string = 'location';
  @Prop() nameField: string = 'name';
  @Prop() dataSource: string = '';
  @Prop() fetchHeaders: string = '';
  @Prop() fetchOptions?: string;
  dataFetcher?: DataFetcher;
  @State() loading: boolean = false;
  @State() recordCount: number = 0;
  @Event() lcError!: EventEmitter<{message: string}>;
  private map: L.Map | null = null;

  componentWillRender() { this.el.dir = i18n.dir; }

  private static leafletCssLoaded = false;

  private ensureLeafletCSS() {
    if (BcViewMap.leafletCssLoaded) return;
    if (document.querySelector('link[data-leaflet-css]')) { BcViewMap.leafletCssLoaded = true; return; }
    const link = document.createElement('link');
    link.rel = 'stylesheet';
    link.href = 'https://unpkg.com/leaflet@1.9.4/dist/leaflet.css';
    link.setAttribute('data-leaflet-css', 'true');
    document.head.appendChild(link);
    BcViewMap.leafletCssLoaded = true;
  }

  async componentDidLoad() {
    this.ensureLeafletCSS();
    const container = this.el.querySelector('.bc-map-view-container') as HTMLElement;
    if (!container) return;
    this.map = L.map(container).setView([-6.2088, 106.8456], 10);
    L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', {
      attribution: '© OpenStreetMap contributors',
    }).addTo(this.map);
    await this.fetchMarkers();
  }

  disconnectedCallback() { this.map?.remove(); }

  @Watch('model') @Watch('dataSource')
  onSourceChange() { this.fetchMarkers(); }

  private async fetchMarkers() {
    if (!this.model && !this.dataSource && !this.dataFetcher) return;
    this.loading = true;
    this.recordCount = 0;
    try {
      let rows: Array<Record<string, unknown>> = [];
      if (this.dataFetcher) {
        const result = await this.dataFetcher({ pageSize: 500 });
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
          const result = await fetchData({ element: this.el, model: this.model, localData: this.localData, fetchOptions: this.fetchOptions ? JSON.parse(this.fetchOptions) : undefined, fetchHeaders: this.fetchHeaders, params: { pageSize: 500 } });
          rows = result.data as Array<Record<string, unknown>>;
        } catch {
          const api = getApiClient();
          const res = await api.list(this.model, { pageSize: 500 });
          rows = res.data;
        }
      }
      const bounds: L.LatLng[] = [];
      for (const row of rows) {
        const geo = row[this.geoField];
        if (geo && typeof geo === 'object') {
          const g = geo as Record<string, number>;
          if (g['lat'] && g['lng']) {
            const ll = L.latLng(g['lat'], g['lng']);
            bounds.push(ll);
            const name = String(row[this.nameField] || row['id'] || '');
            L.marker(ll).addTo(this.map!).bindPopup('<b>' + name + '</b>');
            this.recordCount++;
          }
        }
      }
      if (bounds.length > 0) this.map!.fitBounds(L.latLngBounds(bounds), { padding: [50, 50] });
    } catch (err) {
      this.lcError.emit({ message: String(err) });
    }
    this.loading = false;
  }

  @Method() async refresh(): Promise<void> { await this.fetchMarkers(); }

  render() {
    return (
      <div class="bc-view bc-view-map">
        <div class="bc-map-header">
          <h2>{this.viewTitle || i18n.t('map.title')}</h2>
          <span class="bc-map-count">{i18n.plural('map.locations', this.recordCount)}</span>
        </div>
        {this.loading && <div class="bc-map-loading">{i18n.t('map.loadingLocations')}</div>}
        <div class="bc-map-view-container"></div>
      </div>
    );
  }
}

