import { Component, Prop, State, Event, EventEmitter, Element, Watch, Method, h } from '@stencil/core';
import { DataFetcher } from '../../../core/types';
import { i18n } from '../../../core/i18n';
import { kanbanFetch, kanbanRemove, kanbanUpload } from '../core/kanban-data-fetcher';
import { KanbanAttachment, KanbanAttachmentUploadEvent, KanbanAttachmentDeleteEvent } from '../core/kanban-types';

@Component({ tag: 'bc-kanban-card-attachments', styleUrl: 'bc-kanban-card-attachments.css', shadow: false })
export class BcKanbanCardAttachments {
  @Element() el!: HTMLElement;
  @Prop() cardId: string = '';
  @Prop() localData?: string;
  @Prop() dataFetcher?: DataFetcher;
  @Prop() dataSource: string = '';
  @Prop() model: string = '';
  @Prop() fetchHeaders: string = '';
  @Prop() filterBy: string = 'card_id';

  @State() attachments: KanbanAttachment[] = [];
  @State() loading = false;
  @State() uploading = false;

  @Event() kanbanAttachmentUpload!: EventEmitter<KanbanAttachmentUploadEvent>;
  @Event() kanbanAttachmentDelete!: EventEmitter<KanbanAttachmentDeleteEvent>;

  async componentDidLoad() { await this.loadAttachments(); }
  @Watch('cardId') async onCardChange() { await this.loadAttachments(); }
  @Method() async refresh(): Promise<void> { await this.loadAttachments(); }

  private async loadAttachments() {
    if (!this.cardId && !this.localData) return;
    this.loading = true;
    try {
      const result = await kanbanFetch({
        localData: this.localData, dataFetcher: this.dataFetcher,
        dataSource: this.dataSource, model: this.model,
        fetchHeaders: this.fetchHeaders, filterBy: this.filterBy, filterValue: this.cardId,
        element: this.el, params: { pageSize: 200 },
      });
      this.attachments = result.data as KanbanAttachment[];
    } catch { this.attachments = []; }
    this.loading = false;
  }

  private async onFileUpload(e: Event) {
    const input = e.target as HTMLInputElement;
    const files = input.files ? Array.from(input.files) : [];
    if (files.length === 0) return;
    this.uploading = true;
    this.kanbanAttachmentUpload.emit({ cardId: this.cardId, files });
    try {
      const results = await kanbanUpload(this.dataSource || undefined, files);
      const newAtts = (results as KanbanAttachment[]).map((r, i) => ({
        id: r.id || `temp-${Date.now()}-${i}`,
        name: r.name || files[i]?.name || 'file',
        url: r.url || '',
        type: (r.type || (files[i]?.type?.startsWith('image/') ? 'image' : 'file')) as 'image' | 'file',
        size: r.size || files[i]?.size || 0,
        created_at: new Date().toISOString(),
      }));
      this.attachments = [...this.attachments, ...newAtts];
    } catch { /* silent */ }
    this.uploading = false;
    input.value = '';
  }

  private async deleteAttachment(id: string) {
    this.kanbanAttachmentDelete.emit({ cardId: this.cardId, attachmentId: id });
    this.attachments = this.attachments.filter(a => a.id !== id);
    try { await kanbanRemove(this.model || undefined, this.dataSource || undefined, id); } catch { /* optimistic */ }
  }

  private formatSize(bytes: number): string {
    if (bytes < 1024) return `${bytes} B`;
    if (bytes < 1048576) return `${(bytes / 1024).toFixed(1)} KB`;
    return `${(bytes / 1048576).toFixed(1)} MB`;
  }

  render() {
    return (
      <div class="kb-attachments">
        <h4>{i18n.t('kanban.attachments')} ({this.attachments.length})</h4>
        {this.attachments.map(a => (
          <div class="kb-attachment">
            {a.type === 'image' ? (
              <div class="kb-attachment-thumb"><img src={a.thumbnail_url || a.url} alt={a.name} loading="lazy" /></div>
            ) : (
              <div class="kb-attachment-icon">📄</div>
            )}
            <div class="kb-attachment-info">
              <a class="kb-attachment-name" href={a.url} target="_blank" rel="noopener">{a.name}</a>
              <span class="kb-attachment-size">{this.formatSize(a.size)}</span>
            </div>
            <button type="button" class="kb-attachment-delete" onClick={() => this.deleteAttachment(a.id)}>✕</button>
          </div>
        ))}
        <div class="kb-attachment-upload">
          <label class="kb-upload-label">
            <input type="file" multiple onChange={(e) => this.onFileUpload(e)} style={{ display: 'none' }} />
            {this.uploading ? i18n.t('kanban.uploading') : `+ ${i18n.t('kanban.add_attachment')}`}
          </label>
        </div>
      </div>
    );
  }
}
