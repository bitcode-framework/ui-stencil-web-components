import { Component, Prop, State, Event, EventEmitter, Element, Watch, Method, h } from '@stencil/core';
import { DataFetcher } from '../../../core/types';
import { i18n } from '../../../core/i18n';
import { kanbanFetch, kanbanCreate, kanbanRemove, kanbanUpload } from '../core/kanban-data-fetcher';
import { KanbanComment, KanbanUser, KanbanCommentCreateEvent, KanbanCommentDeleteEvent } from '../core/kanban-types';
import { highlightMentions } from '../core/mention-parser';

@Component({ tag: 'bc-kanban-card-comments', styleUrl: 'bc-kanban-card-comments.css', shadow: false })
export class BcKanbanCardComments {
  @Element() el!: HTMLElement;
  @Prop() cardId: string = '';
  @Prop() localData?: string;
  @Prop() dataFetcher?: DataFetcher;
  @Prop() dataSource: string = '';
  @Prop() model: string = '';
  @Prop() fetchHeaders: string = '';
  @Prop() filterBy: string = 'card_id';
  @Prop() mentionModel: string = '';
  @Prop() mentionDataSource: string = '';
  @Prop() mentionLocalData?: string;

  @State() comments: KanbanComment[] = [];
  @State() loading = false;
  @State() newBody = '';
  @State() mentionUsers: KanbanUser[] = [];
  @State() mentionQuery = '';
  @State() mentionActive = false;
  @State() mentionIndex = 0;
  @State() mentionStart = -1;
  @State() pendingFiles: File[] = [];
  @State() uploading = false;

  private textareaRef?: HTMLTextAreaElement;

  @Event() kanbanCommentCreate!: EventEmitter<KanbanCommentCreateEvent>;
  @Event() kanbanCommentDelete!: EventEmitter<KanbanCommentDeleteEvent>;

  async componentWillLoad() {
    if (this.mentionLocalData || this.mentionModel || this.mentionDataSource) {
      await this.loadMentionUsers();
    }
  }

  async componentDidLoad() { await this.loadComments(); }
  @Watch('cardId') async onCardChange() { await this.loadComments(); }
  @Method() async refresh(): Promise<void> { await this.loadComments(); }

  private async loadMentionUsers() {
    try {
      const result = await kanbanFetch({
        localData: this.mentionLocalData,
        dataSource: this.mentionDataSource,
        model: this.mentionModel,
        fetchHeaders: this.fetchHeaders,
        element: this.el,
        params: { pageSize: 200 },
      });
      this.mentionUsers = (result.data as KanbanUser[]).filter(u => u.name);
    } catch { this.mentionUsers = []; }
  }

  private async loadComments() {
    if (!this.cardId && !this.localData) return;
    this.loading = true;
    try {
      const result = await kanbanFetch({
        localData: this.localData, dataFetcher: this.dataFetcher,
        dataSource: this.dataSource, model: this.model,
        fetchHeaders: this.fetchHeaders, filterBy: this.filterBy, filterValue: this.cardId,
        element: this.el, params: { pageSize: 200 },
      });
      this.comments = (result.data as KanbanComment[]).sort((a, b) =>
        new Date(b.created_at).getTime() - new Date(a.created_at).getTime()
      );
    } catch { this.comments = []; }
    this.loading = false;
  }

  private async submitComment() {
    if (!this.newBody.trim() && this.pendingFiles.length === 0) return;
    const body = this.newBody.trim();
    const files = [...this.pendingFiles];
    this.kanbanCommentCreate.emit({ cardId: this.cardId, body, attachments: files });
    const optimistic: KanbanComment = {
      id: `temp-${Date.now()}`, body, user: { id: 'me', name: 'You' },
      created_at: new Date().toISOString(),
      attachments: files.map((f, i) => ({
        id: `temp-att-${Date.now()}-${i}`, name: f.name, url: '', type: f.type.startsWith('image/') ? 'image' as const : 'file' as const,
        size: f.size, created_at: new Date().toISOString(),
      })),
    };
    this.comments = [optimistic, ...this.comments];
    this.newBody = '';
    this.pendingFiles = [];
    this.mentionActive = false;
    try {
      await kanbanCreate(this.model || undefined, this.dataSource || undefined, { body, [this.filterBy]: this.cardId });
      if (files.length > 0 && this.dataSource) {
        await kanbanUpload(this.dataSource, files);
      }
    } catch { /* optimistic */ }
  }

  private async deleteComment(id: string) {
    this.kanbanCommentDelete.emit({ cardId: this.cardId, commentId: id });
    this.comments = this.comments.filter(c => c.id !== id);
    try { await kanbanRemove(this.model || undefined, this.dataSource || undefined, id); } catch { /* optimistic */ }
  }

  private removePendingFile(index: number) {
    this.pendingFiles = this.pendingFiles.filter((_, i) => i !== index);
  }

  // ─── MENTION DETECTION ───

  private handleInput(e: Event) {
    const ta = e.target as HTMLTextAreaElement;
    this.newBody = ta.value;
    this.detectMention(ta);
  }

  private detectMention(ta: HTMLTextAreaElement) {
    const text = ta.value;
    const pos = ta.selectionStart;
    const before = text.substring(0, pos);
    const atIdx = before.lastIndexOf('@');
    if (atIdx === -1) { this.mentionActive = false; return; }
    const between = before.substring(atIdx + 1);
    if (between.includes(' ') || between.includes('\n')) { this.mentionActive = false; return; }
    this.mentionQuery = between.toLowerCase();
    this.mentionStart = atIdx;
    this.mentionActive = true;
    this.mentionIndex = 0;
  }

  private handleKeyDown(e: KeyboardEvent) {
    if (this.mentionActive) {
      const filtered = this.filteredUsers();
      if (e.key === 'ArrowDown') {
        e.preventDefault();
        this.mentionIndex = Math.min(this.mentionIndex + 1, filtered.length - 1);
        return;
      }
      if (e.key === 'ArrowUp') {
        e.preventDefault();
        this.mentionIndex = Math.max(this.mentionIndex - 1, 0);
        return;
      }
      if (e.key === 'Enter' || e.key === 'Tab') {
        e.preventDefault();
        if (filtered.length > 0) this.insertMention(filtered[this.mentionIndex]);
        return;
      }
      if (e.key === 'Escape') {
        e.preventDefault();
        this.mentionActive = false;
        return;
      }
    }
    if (e.key === 'Enter' && (e.ctrlKey || e.metaKey)) {
      this.submitComment();
    }
  }

  private filteredUsers(): KanbanUser[] {
    if (!this.mentionQuery) return this.mentionUsers.slice(0, 8);
    return this.mentionUsers.filter(u =>
      u.name.toLowerCase().includes(this.mentionQuery) ||
      (u.email && u.email.toLowerCase().includes(this.mentionQuery))
    ).slice(0, 8);
  }

  private insertMention(user: KanbanUser) {
    const ta = this.textareaRef;
    if (!ta) return;
    const text = this.newBody;
    const before = text.substring(0, this.mentionStart);
    const after = text.substring(ta.selectionStart);
    const insert = `@${user.name.replace(/\s+/g, '.')} `;
    this.newBody = before + insert + after;
    this.mentionActive = false;
    setTimeout(() => {
      ta.value = this.newBody;
      const newPos = before.length + insert.length;
      ta.setSelectionRange(newPos, newPos);
      ta.focus();
    }, 0);
  }

  private onFileSelect(e: Event) {
    const input = e.target as HTMLInputElement;
    if (input.files) {
      this.pendingFiles = [...this.pendingFiles, ...Array.from(input.files)];
    }
    input.value = '';
  }

  private formatTime(dateStr: string): string {
    const diff = (Date.now() - new Date(dateStr).getTime()) / 1000;
    if (diff < 60) return i18n.t('kanban.just_now');
    if (diff < 3600) return i18n.t('kanban.minutes_ago', { count: Math.floor(diff / 60) });
    if (diff < 86400) return i18n.t('kanban.hours_ago', { count: Math.floor(diff / 3600) });
    return new Date(dateStr).toLocaleDateString(undefined, { month: 'short', day: 'numeric' });
  }

  private formatSize(bytes: number): string {
    if (bytes < 1024) return `${bytes} B`;
    if (bytes < 1048576) return `${(bytes / 1024).toFixed(1)} KB`;
    return `${(bytes / 1048576).toFixed(1)} MB`;
  }

  render() {
    const filtered = this.mentionActive ? this.filteredUsers() : [];
    return (
      <div class="kb-comments">
        <h4>{i18n.t('kanban.comments')} ({this.comments.length})</h4>
        <div class="kb-comment-list">
          {this.comments.map(c => (
            <div class="kb-comment">
              <div class="kb-comment-avatar">{c.user?.name?.charAt(0) || '?'}</div>
              <div class="kb-comment-body">
                <div class="kb-comment-header">
                  <span class="kb-comment-user">{c.user?.name || 'Unknown'}</span>
                  <span class="kb-comment-time">{this.formatTime(c.created_at)}</span>
                </div>
                <div class="kb-comment-text" innerHTML={highlightMentions(c.body)}></div>
                {c.attachments && c.attachments.length > 0 && (
                  <div class="kb-comment-attachments">
                    {c.attachments.map(a => (
                      <div class="kb-comment-att">
                        {a.type === 'image' ? (
                          <span class="kb-comment-att-icon">🖼</span>
                        ) : (
                          <span class="kb-comment-att-icon">📎</span>
                        )}
                        <span class="kb-comment-att-name">{a.name}</span>
                        {a.size > 0 && <span class="kb-comment-att-size">{this.formatSize(a.size)}</span>}
                      </div>
                    ))}
                  </div>
                )}
                <button type="button" class="kb-comment-delete" onClick={() => this.deleteComment(c.id)}>{i18n.t('common.delete')}</button>
              </div>
            </div>
          ))}
        </div>
        <div class="kb-comment-input-area">
          {this.pendingFiles.length > 0 && (
            <div class="kb-pending-files">
              {this.pendingFiles.map((f, i) => (
                <div class="kb-pending-file">
                  <span class="kb-pending-file-name">📎 {f.name}</span>
                  <button type="button" class="kb-pending-file-remove" onClick={() => this.removePendingFile(i)}>✕</button>
                </div>
              ))}
            </div>
          )}
          <div class="kb-comment-input-row">
            <textarea ref={el => { this.textareaRef = el; }} value={this.newBody}
              onInput={(e) => this.handleInput(e)}
              onKeyDown={(e) => this.handleKeyDown(e)}
              placeholder={`${i18n.t('kanban.write_comment')} ${i18n.t('kanban.ctrl_enter_send')}`}
            ></textarea>
          </div>
          <div class="kb-comment-actions">
            <label class="kb-attach-btn">
              <input type="file" multiple onChange={(e) => this.onFileSelect(e)} style={{ display: 'none' }} />
              📎
            </label>
            <button type="button" class="kb-btn kb-btn-primary kb-send-btn" onClick={() => this.submitComment()} disabled={!this.newBody.trim() && this.pendingFiles.length === 0}>
              {i18n.t('kanban.send')}
            </button>
          </div>
          {this.mentionActive && filtered.length > 0 && (
            <div class="kb-mention-dropdown">
              <div class="kb-mention-header">{i18n.t('kanban.mention_users')}</div>
              {filtered.map((u, i) => (
                <div class={`kb-mention-item${i === this.mentionIndex ? ' kb-mention-active' : ''}`}
                  onClick={() => this.insertMention(u)}
                  onMouseEnter={() => { this.mentionIndex = i; }}
                >
                  <div class="kb-mention-avatar">{u.name.charAt(0).toUpperCase()}</div>
                  <div class="kb-mention-info">
                    <span class="kb-mention-name">{u.name}</span>
                    {u.email && <span class="kb-mention-email">{u.email}</span>}
                  </div>
                </div>
              ))}
            </div>
          )}
          {this.mentionActive && filtered.length === 0 && (
            <div class="kb-mention-dropdown">
              <div class="kb-mention-empty">{i18n.t('kanban.no_users')}</div>
            </div>
          )}
        </div>
      </div>
    );
  }
}
