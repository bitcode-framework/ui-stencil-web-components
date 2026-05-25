import { Component, Prop, State, Event, EventEmitter, Element, Listen, Method, Watch, h } from '@stencil/core';
import { getApiClient } from '../../../core/api-client';
import { fetchData } from '../../../core/data-fetcher';
import { BcSetup } from '../../../core/bc-setup';
import { DataFetcher } from '../../../core/types';
import { i18n } from '../../../core/i18n';

interface Permissions {
  can_select?: boolean;
  can_read?: boolean;
  can_write?: boolean;
  can_create?: boolean;
  can_delete?: boolean;
  can_print?: boolean;
  can_email?: boolean;
  can_report?: boolean;
  can_export?: boolean;
  can_import?: boolean;
  can_mask?: boolean;
  can_clone?: boolean;
}

@Component({
  tag: 'bc-view-form',
  styleUrl: 'bc-view-form.css',
  shadow: false,
})
export class BcViewForm {
  @Element() el!: HTMLElement;
  @Prop() model: string = '';
  @Prop() localData?: string;
  @Prop() viewTitle: string = '';
  @Prop() recordId: string = '';
  @Prop() fields: string = '[]';
  @Prop() config: string = '{}';
  @Prop() permissions: string = '{}';
  @Prop() moduleName: string = '';
  @Prop() dataSource: string = '';
  @Prop() fetchHeaders: string = '';
  @Prop() fetchOptions?: string;
  dataFetcher?: DataFetcher;

  @State() data: Record<string, unknown> = {};
  @State() loading: boolean = false;
  @State() dirty: boolean = false;
  @State() perms: Permissions = {};

  @Event() lcFormSubmit!: EventEmitter<{model: string; data: Record<string, unknown>; id?: string}>;
  @Event() lcError!: EventEmitter<{message: string}>;

  private can(op: string): boolean {
    const key = `can_${op}` as keyof Permissions;
    return this.perms[key] !== false;
  }

  componentWillRender() {
    this.el.dir = i18n.dir;
  }

  componentWillLoad() {
    try { this.perms = JSON.parse(this.permissions); } catch { this.perms = {}; }
  }

  async componentDidLoad() {
    await this.fetchRecord();
  }

  @Watch('model') @Watch('recordId') @Watch('dataSource')
  onSourceChange() { this.fetchRecord(); }

  private async fetchRecord() {
    if (!this.recordId) return;
    if (!this.model && !this.dataSource && !this.dataFetcher) return;
    this.loading = true;
    try {
      if (this.dataFetcher) {
        const result = await this.dataFetcher({ filters: { id: this.recordId } });
        this.data = (result.data?.[0] as Record<string, unknown>) || {};
      } else if (this.dataSource) {
        const baseUrl = BcSetup.getBaseUrl();
        let url = this.dataSource;
        if (url && !url.startsWith('http') && baseUrl) url = baseUrl + url;
        const headers = { ...BcSetup.getHeaders(), ...(this.fetchHeaders ? JSON.parse(this.fetchHeaders) : {}) };
        this.el.dispatchEvent(new CustomEvent('lcBeforeFetch', { detail: { url: `${url}/${this.recordId}`, headers }, bubbles: true, cancelable: true }));
        const res = await fetch(`${url}/${this.recordId}`, { headers });
        if (!res.ok) throw new Error(`HTTP ${res.status}`);
        const json = await res.json();
        this.data = json.data || json;
      } else if (this.model) {
        try {
          const result = await fetchData({ element: this.el, model: this.model, localData: this.localData, fetchOptions: this.fetchOptions ? JSON.parse(this.fetchOptions) : undefined, fetchHeaders: this.fetchHeaders, params: { filters: { id: this.recordId } } });
          this.data = (result.data?.[0] as Record<string, unknown>) || {};
        } catch {
          const api = getApiClient();
          const res = await api.read(this.model, this.recordId);
          this.data = (res as any).data || res;
        }
      }
    } catch (err) {
      this.data = {};
      this.lcError.emit({ message: String(err) });
    }
    this.loading = false;
  }

  @Listen('lcFieldChange')
  handleFieldChange(e: CustomEvent) {
    const { name, value } = e.detail;
    this.data = { ...this.data, [name]: value };
    this.dirty = true;
  }

  private async handleSave() {
    try {
      const api = getApiClient();
      if (this.recordId) { await api.update(this.model, this.recordId, this.data); }
      else { const r = await api.create(this.model, this.data); this.data = r; }
      this.dirty = false;
      this.lcFormSubmit.emit({ model: this.model, data: this.data, id: this.recordId });
    } catch (err) { this.lcError.emit({ message: String(err) }); }
  }

  private async handleDelete() {
    if (!this.recordId || !confirm(i18n.t('confirm.message'))) return;
    try {
      const api = getApiClient();
      await api.remove(this.model, this.recordId);
      window.history.back();
    } catch (err) { this.lcError.emit({ message: String(err) }); }
  }

  private async handleClone() {
    if (!this.recordId) return;
    try {
      const baseUrl = BcSetup.getBaseUrl();
      const url = baseUrl ? `${baseUrl}/api/${this.model}s/${this.recordId}/clone` : `/api/${this.model}s/${this.recordId}/clone`;
      const headers = { ...BcSetup.getHeaders(), 'Content-Type': 'application/json' };
      await fetch(url, { method: 'POST', headers });
      await this.fetchRecord();
    } catch (err) { this.lcError.emit({ message: String(err) }); }
  }

  @Method() async refresh(): Promise<void> { await this.fetchRecord(); }

  render() {
    const isNew = !this.recordId;
    const canSave = isNew ? this.can('create') : this.can('write');

    return (
      <div class="bc-view bc-view-form">
        {this.loading && <div class="bc-form-loading">{i18n.t('common.loading')}</div>}
        <div class="bc-form-body"><slot></slot></div>
        <div class="bc-form-footer">
          {canSave && (
            <button type="button" class="bc-btn bc-btn-primary" onClick={() => this.handleSave()} disabled={!this.dirty}>
              {isNew ? i18n.t('common.create') : i18n.t('common.save')}
            </button>
          )}
          {!isNew && this.can('delete') && (
            <button type="button" class="bc-btn bc-btn-danger" onClick={() => this.handleDelete()}>
              {i18n.t('common.delete') || 'Delete'}
            </button>
          )}
          {!isNew && this.can('clone') && (
            <button type="button" class="bc-btn bc-btn-secondary" onClick={() => this.handleClone()}>
              {i18n.t('datatable.clone') || 'Clone'}
            </button>
          )}
          {!isNew && this.can('print') && (
            <button type="button" class="bc-btn bc-btn-secondary" onClick={() => window.print()}>
              {i18n.t('common.print') || 'Print'}
            </button>
          )}
        </div>
      </div>
    );
  }
}

