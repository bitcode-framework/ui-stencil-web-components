import { Component, Prop, State, Event, EventEmitter, Element, Watch, Method, h } from '@stencil/core';
import Sortable from 'sortablejs';
import { fetchData } from '../../../core/data-fetcher';
import { DataFetcher } from '../../../core/types';
import { i18n } from '../../../core/i18n';
import { kanbanFetch, kanbanCreate, kanbanUpdate, kanbanRemove } from '../core/kanban-data-fetcher';
import {
  KanbanCard, KanbanColumnConfig, KanbanColumnData, KanbanUser, KanbanLabel,
  KanbanCardMoveEvent, KanbanColumnReorderEvent,
  KanbanCardCreateEvent, KanbanCardUpdateEvent, KanbanCardDeleteEvent,
  KanbanColumnAddEvent, KanbanColumnRenameEvent, KanbanColumnDeleteEvent,
} from '../core/kanban-types';

@Component({ tag: 'bc-kanban-board', styleUrl: 'bc-kanban-board.css', shadow: false })
export class BcKanbanBoard {
  @Element() el!: HTMLElement;

  // ─── 4-Layer Data Fetch: Board (cards) ───
  @Prop() model: string = '';
  @Prop() localData?: string;
  @Prop() dataFetcher?: DataFetcher;
  @Prop() dataSource: string = '';
  @Prop() fetchHeaders: string = '';
  @Prop() fetchOptions?: string;

  // ─── Board Config ───
  @Prop() boardTitle: string = '';
  @Prop() groupBy: string = 'stage';
  @Prop() cardTitleField: string = 'name';
  @Prop() cardDescriptionField: string = 'description';
  @Prop() cardCoverField: string = '';
  @Prop() cardAssigneesField: string = 'assignees';
  @Prop() cardDueDateField: string = 'due_date';
  @Prop() cardStartDateField: string = 'start_date';
  @Prop() cardPriorityField: string = 'priority';
  @Prop() cardLabelsField: string = 'labels';
  @Prop() cardPositionField: string = 'position';
  @Prop() columnsConfig?: string;
  @Prop() columnModel: string = '';
  @Prop() columnDataSource: string = '';
  @Prop() columnLocalData?: string;
  @Prop() allowAddColumn: boolean = true;
  @Prop() allowAddCard: boolean = true;
  @Prop() allowRenameColumn: boolean = true;
  @Prop() allowDeleteColumn: boolean = false;
  @Prop() allowReorderColumns: boolean = true;
  @Prop() cardDetailModel: string = '';
  @Prop() cardDetailDataSource: string = '';

  // ─── Sub-component 4-Layer: Comments ───
  @Prop() commentModel: string = '';
  @Prop() commentDataSource: string = '';
  @Prop() commentLocalData?: string;

  // ─── Sub-component 4-Layer: Subtasks ───
  @Prop() subtaskModel: string = '';
  @Prop() subtaskDataSource: string = '';
  @Prop() subtaskLocalData?: string;

  // ─── Sub-component 4-Layer: Attachments ───
  @Prop() attachmentModel: string = '';
  @Prop() attachmentDataSource: string = '';
  @Prop() attachmentLocalData?: string;

  // ─── Sub-component 4-Layer: Activity ───
  @Prop() activityModel: string = '';
  @Prop() activityDataSource: string = '';
  @Prop() activityLocalData?: string;

  // ─── Sub-component 4-Layer: @Mention Users ───
  @Prop() mentionModel: string = '';
  @Prop() mentionDataSource: string = '';
  @Prop() mentionLocalData?: string;

  // ─── State ───
  @State() columns: Map<string, KanbanColumnData> = new Map();
  @State() loading = false;
  @State() error?: string;
  @State() selectedCardId?: string;
  @State() addingToColumn?: string;
  @State() newCardTitle = '';
  @State() dialogOpen = false;
  @State() dialogMode: 'add-col' | 'rename-col' | 'delete-col' = 'add-col';
  @State() dialogValue = '';
  @State() dialogTargetColId = '';
  @State() dialogTargetColName = '';
  @State() editingTitle = false;
  @State() editingTitleValue = '';
  @State() editingDescription = false;
  @State() editingPriority = false;
  @State() editingStartDate = false;
  @State() editingDueDate = false;
  @State() editingAssignees = false;
  @State() editingLabels = false;
  @State() assigneeQuery = '';
  @State() labelNewName = '';
  @State() labelNewColor = '#8b5cf6';
  @State() mentionUsers: KanbanUser[] = [];
  @State() draftAssignees: KanbanUser[] = [];
  @State() draftLabels: KanbanLabel[] = [];
  @State() draftDescription = '';

  private cardSortables: Sortable[] = [];
  private columnSortable?: Sortable;
  private boardRef?: HTMLDivElement;

  // ─── Events ───
  @Event() kanbanCardMove!: EventEmitter<KanbanCardMoveEvent>;
  @Event() kanbanColumnReorder!: EventEmitter<KanbanColumnReorderEvent>;
  @Event() kanbanCardCreate!: EventEmitter<KanbanCardCreateEvent>;
  @Event() kanbanCardUpdate!: EventEmitter<KanbanCardUpdateEvent>;
  @Event() kanbanCardDelete!: EventEmitter<KanbanCardDeleteEvent>;
  @Event() kanbanColumnAdd!: EventEmitter<KanbanColumnAddEvent>;
  @Event() kanbanColumnRename!: EventEmitter<KanbanColumnRenameEvent>;
  @Event() kanbanColumnDelete!: EventEmitter<KanbanColumnDeleteEvent>;
  @Event() kanbanError!: EventEmitter<{ message: string }>;

  componentWillRender() { this.el.dir = i18n.dir; }

  async componentDidLoad() {
    await this.loadColumns();
    await this.loadCards();
    this.initSortable();
    if (this.mentionModel || this.mentionDataSource || this.mentionLocalData) {
      await this.loadMentionUsers();
    }
  }

  disconnectedCallback() {
    for (const s of this.cardSortables) s.destroy();
    this.columnSortable?.destroy();
  }

  @Watch('model') @Watch('dataSource') @Watch('localData')
  async onSourceChange() {
    await this.loadColumns();
    await this.loadCards();
    this.initSortable();
  }

  @Method() async refresh(): Promise<void> {
    await this.loadColumns();
    await this.loadCards();
    this.initSortable();
  }

  // ═══════════════════════════════════════════
  // DATA LOADING
  // ═══════════════════════════════════════════

  private getColumnsConfig(): KanbanColumnConfig[] {
    if (this.columnsConfig) {
      try { return JSON.parse(this.columnsConfig); } catch { return []; }
    }
    return [];
  }

  private async loadColumns(): Promise<void> {
    const explicitCols = this.getColumnsConfig();

    if (this.columnLocalData) {
      try {
        const cols: KanbanColumnConfig[] = JSON.parse(this.columnLocalData);
        this.mergeColumns(cols);
        return;
      } catch { /* fall through */ }
    }

    if (this.columnDataSource) {
      try {
        const result = await kanbanFetch({
          dataSource: this.columnDataSource,
          element: this.el,
          fetchHeaders: this.fetchHeaders,
        });
        const cols = result.data as KanbanColumnConfig[];
        this.mergeColumns(cols.length > 0 ? cols : explicitCols);
        return;
      } catch { /* fall through */ }
    }

    if (this.columnModel) {
      try {
        const result = await kanbanFetch({
          model: this.columnModel,
          element: this.el,
          fetchHeaders: this.fetchHeaders,
        });
        const cols = result.data as KanbanColumnConfig[];
        this.mergeColumns(cols.length > 0 ? cols : explicitCols);
        return;
      } catch { /* fall through */ }
    }

    this.mergeColumns(explicitCols);
  }

  private mergeColumns(configs: KanbanColumnConfig[]): void {
    const newCols = new Map<string, KanbanColumnData>();
    for (const cfg of configs) {
      const key = cfg.id || cfg.name;
      const existing = this.columns.get(key);
      newCols.set(key, {
        config: cfg,
        cards: existing?.cards || [],
      });
    }
    this.columns = newCols;
  }

  private async loadCards(): Promise<void> {
    this.loading = true;
    this.error = undefined;
    try {
      let rows: Record<string, unknown>[] = [];

      if (this.localData) {
        const data = JSON.parse(this.localData);
        rows = Array.isArray(data) ? data : [];
      } else if (this.dataFetcher) {
        const result = await this.dataFetcher({ pageSize: 500 });
        rows = result.data as Record<string, unknown>[];
      } else if (this.dataSource) {
        const result = await kanbanFetch({
          dataSource: this.dataSource,
          element: this.el,
          fetchHeaders: this.fetchHeaders,
          fetchOptions: this.fetchOptions,
          params: { pageSize: 500 },
        });
        rows = result.data as Record<string, unknown>[];
      } else if (this.model) {
        const result = await fetchData({
          element: this.el,
          model: this.model,
          localData: this.localData,
          fetchHeaders: this.fetchHeaders,
          fetchOptions: this.fetchOptions ? JSON.parse(this.fetchOptions || '{}') : undefined,
          params: { pageSize: 500 },
        });
        rows = result.data as Record<string, unknown>[];
      }

      const newCols = new Map(this.columns);
      for (const [key, colData] of newCols.entries()) {
        newCols.set(key, { ...colData, cards: [] });
      }

      for (const row of rows) {
        const card = this.mapRowToCard(row);
        const colKey = String(row[this.groupBy] || 'Other');
        const colData = newCols.get(colKey);
        if (colData) {
          colData.cards.push(card);
        } else {
          const newCol: KanbanColumnData = {
            config: { id: colKey, name: colKey },
            cards: [card],
          };
          newCols.set(colKey, newCol);
        }
      }

      for (const [, colData] of newCols.entries()) {
        colData.cards.sort((a, b) => (a.position ?? 0) - (b.position ?? 0));
      }

      this.columns = newCols;
    } catch (err) {
      this.error = String(err);
      this.kanbanError.emit({ message: String(err) });
    }
    this.loading = false;
  }

  private mapRowToCard(row: Record<string, unknown>): KanbanCard {
    return {
      id: String(row.id || ''),
      title: String(row[this.cardTitleField] || ''),
      description: row[this.cardDescriptionField] ? String(row[this.cardDescriptionField]) : undefined,
      column: String(row[this.groupBy] || 'Other'),
      position: typeof row[this.cardPositionField] === 'number' ? row[this.cardPositionField] as number : undefined,
      cover_image: row[this.cardCoverField] ? String(row[this.cardCoverField]) : undefined,
      priority: row[this.cardPriorityField] as KanbanCard['priority'],
      due_date: row[this.cardDueDateField] ? String(row[this.cardDueDateField]) : undefined,
      start_date: row[this.cardStartDateField] ? String(row[this.cardStartDateField]) : undefined,
      labels: row[this.cardLabelsField] as KanbanCard['labels'],
      assignees: row[this.cardAssigneesField] as KanbanCard['assignees'],
      comments_count: typeof row.comments_count === 'number' ? row.comments_count as number : undefined,
      attachments_count: typeof row.attachments_count === 'number' ? row.attachments_count as number : undefined,
      subtasks: row.subtasks as KanbanCard['subtasks'],
      ...row,
    };
  }

  private async loadMentionUsers(): Promise<void> {
    try {
      let rows: Record<string, unknown>[] = [];
      if (this.mentionLocalData) {
        rows = JSON.parse(this.mentionLocalData);
      } else if (this.mentionDataSource) {
        const result = await kanbanFetch({
          dataSource: this.mentionDataSource,
          element: this.el,
          fetchHeaders: this.fetchHeaders,
          fetchOptions: this.fetchOptions,
          params: { pageSize: 500 },
        });
        rows = result.data as Record<string, unknown>[];
      } else if (this.mentionModel) {
        const result = await fetchData({
          element: this.el,
          model: this.mentionModel,
          localData: '',
          fetchHeaders: this.fetchHeaders,
          fetchOptions: this.fetchOptions ? JSON.parse(this.fetchOptions || '{}') : undefined,
          params: { pageSize: 500 },
        });
        rows = result.data as Record<string, unknown>[];
      }
      this.mentionUsers = rows.map(u => ({
        id: String(u.id || ''),
        name: String(u.name || u.username || ''),
        username: String(u.username || u.name || ''),
        avatar: u.avatar ? String(u.avatar) : undefined,
      }));
    } catch {
      this.mentionUsers = [];
    }
  }

  // ═══════════════════════════════════════════
  // SORTABLE
  // ═══════════════════════════════════════════

  private initSortable() {
    for (const s of this.cardSortables) s.destroy();
    this.cardSortables = [];
    this.columnSortable?.destroy();

    setTimeout(() => {
      if (!this.boardRef) return;

      if (this.allowReorderColumns) {
        this.columnSortable = Sortable.create(this.boardRef, {
          animation: 150,
          handle: '.kb-col-header',
          draggable: '.kb-column',
          ghostClass: 'kb-col-ghost',
          onEnd: () => {
            const cols = Array.from(this.boardRef!.querySelectorAll('.kb-column'));
            const names = cols.map(c => c.getAttribute('data-col-id') || '').filter(Boolean);
            this.kanbanColumnReorder.emit({ columns: names });
          },
        });
      }

      this.el.querySelectorAll('.kb-cards').forEach(list => {
        const s = Sortable.create(list as HTMLElement, {
          group: 'kb-cards',
          animation: 150,
          ghostClass: 'kb-card-ghost',
          onEnd: (evt) => {
            const cardId = evt.item.getAttribute('data-card-id') || '';
            const from = evt.from.getAttribute('data-col-id') || '';
            const to = evt.to.getAttribute('data-col-id') || '';
            if (from !== to) {
              this.kanbanCardMove.emit({ cardId, fromColumn: from, toColumn: to, toPosition: evt.newIndex });
              this.moveCardLocally(cardId, from, to, evt.newIndex);
            }
          },
        });
        this.cardSortables.push(s);
      });
    }, 100);
  }

  private moveCardLocally(cardId: string, from: string, to: string, newIndex?: number) {
    const newCols = new Map(this.columns);
    const fromCol = newCols.get(from);
    const toCol = newCols.get(to);
    if (!fromCol || !toCol) return;

    const cardIdx = fromCol.cards.findIndex(c => c.id === cardId);
    if (cardIdx === -1) return;

    const [card] = fromCol.cards.splice(cardIdx, 1);
    card.column = to;
    const insertAt = newIndex !== undefined ? Math.min(newIndex, toCol.cards.length) : toCol.cards.length;
    toCol.cards.splice(insertAt, 0, card);
    this.columns = newCols;
  }

  // ═══════════════════════════════════════════
  // ACTIONS
  // ═══════════════════════════════════════════

  private async addCard(columnId: string) {
    if (!this.newCardTitle.trim()) return;
    const title = this.newCardTitle.trim();
    this.newCardTitle = '';
    this.addingToColumn = undefined;

    this.kanbanCardCreate.emit({ column: columnId, title });

    try {
      const result = await kanbanCreate(
        this.model || undefined,
        this.dataSource || undefined,
        { [this.cardTitleField]: title, [this.groupBy]: columnId },
      );
      const newCard: KanbanCard = {
        id: result?.id ? String(result.id) : `temp-${Date.now()}`,
        title,
        column: columnId,
      };
      const newCols = new Map(this.columns);
      const col = newCols.get(columnId);
      if (col) {
        col.cards.push(newCard);
        this.columns = newCols;
      }
    } catch (err) {
      this.kanbanError.emit({ message: String(err) });
    }
  }

  private async addColumn() {
    this.dialogMode = 'add-col';
    this.dialogValue = '';
    this.dialogTargetColId = '';
    this.dialogTargetColName = '';
    this.dialogOpen = true;
  }

  private async renameColumn(columnId: string, currentName: string) {
    this.dialogMode = 'rename-col';
    this.dialogValue = currentName;
    this.dialogTargetColId = columnId;
    this.dialogTargetColName = currentName;
    this.dialogOpen = true;
  }

  private async deleteColumn(columnId: string) {
    const col = this.columns.get(columnId);
    if (!col) return;
    this.dialogMode = 'delete-col';
    this.dialogTargetColId = columnId;
    this.dialogTargetColName = col.config.name;
    this.dialogValue = String(col.cards.length);
    this.dialogOpen = true;
  }

  private async onDialogConfirm() {
    this.dialogOpen = false;
    const val = this.dialogValue;

    if (this.dialogMode === 'add-col') {
      if (!val.trim()) return;
      this.kanbanColumnAdd.emit({ name: val.trim() });
      try {
        const result = await kanbanCreate(
          this.columnModel || undefined,
          this.columnDataSource || undefined,
          { name: val.trim() },
        );
        const colId = result?.id ? String(result.id) : val.trim();
        const newCols = new Map(this.columns);
        newCols.set(colId, {
          config: { id: colId, name: val.trim() },
          cards: [],
        });
        this.columns = newCols;
      } catch (err) {
        this.kanbanError.emit({ message: String(err) });
      }
    } else if (this.dialogMode === 'rename-col') {
      if (!val.trim() || val.trim() === this.dialogTargetColName) return;
      this.kanbanColumnRename.emit({ columnId: this.dialogTargetColId, name: val.trim() });
      try {
        await kanbanUpdate(
          this.columnModel || undefined,
          this.columnDataSource || undefined,
          this.dialogTargetColId,
          { name: val.trim() },
        );
        const newCols = new Map(this.columns);
        const col = newCols.get(this.dialogTargetColId);
        if (col) {
          newCols.set(this.dialogTargetColId, { ...col, config: { ...col.config, name: val.trim() } });
          this.columns = newCols;
        }
      } catch (err) {
        this.kanbanError.emit({ message: String(err) });
      }
    } else if (this.dialogMode === 'delete-col') {
      const columnId = this.dialogTargetColId;
      this.kanbanColumnDelete.emit({ columnId });
      try {
        await kanbanRemove(
          this.columnModel || undefined,
          this.columnDataSource || undefined,
          columnId,
        );
        const newCols = new Map(this.columns);
        newCols.delete(columnId);
        this.columns = newCols;
      } catch (err) {
        this.kanbanError.emit({ message: String(err) });
      }
    }
  }

  private onDialogCancel() {
    this.dialogOpen = false;
  }

  private openCardDetail(cardId: string) {
    this.selectedCardId = cardId;
    this.editingTitle = false;
    this.editingDescription = false;
    this.editingPriority = false;
    this.editingStartDate = false;
    this.editingDueDate = false;
    this.editingAssignees = false;
    this.editingLabels = false;
  }

  private closeCardDetail() {
    this.selectedCardId = undefined;
    this.editingTitle = false;
    this.editingDescription = false;
    this.editingPriority = false;
    this.editingStartDate = false;
    this.editingDueDate = false;
    this.editingAssignees = false;
    this.editingLabels = false;
  }

  private getSelectedCard(): KanbanCard | undefined {
    if (!this.selectedCardId) return undefined;
    for (const [, col] of this.columns) {
      const card = col.cards.find(c => c.id === this.selectedCardId);
      if (card) return card;
    }
    return undefined;
  }

  // ═══════════════════════════════════════════
  // RENDER
  // ═══════════════════════════════════════════

  render() {
    return (
      <div class="kb-board-container">
        <div class="kb-board-header">
          <h2 class="kb-board-title">{this.boardTitle || this.model}</h2>
          {this.allowAddColumn && (
            <button type="button" class="kb-btn kb-btn-ghost kb-add-col-btn" onClick={() => this.addColumn()}>
              <span class="kb-icon">+</span> {i18n.t('kanban.add_column')}
            </button>
          )}
        </div>

        {this.loading && <div class="kb-loading">{i18n.t('kanban.loading')}</div>}
        {this.error && <div class="kb-error">{this.error}</div>}

        <div class="kb-board" ref={el => { this.boardRef = el; }}>
          {Array.from(this.columns.entries()).map(([colId, colData]) => this.renderColumn(colId, colData))}
        </div>

        {this.selectedCardId && this.renderCardDetail()}

        {this.dialogOpen && this.renderDialog()}
      </div>
    );
  }

  private renderDialog() {
    if (this.dialogMode === 'delete-col') {
      const count = parseInt(this.dialogValue, 10);
      const msg = count > 0
        ? i18n.t('kanban.delete_column_confirm', { name: this.dialogTargetColName, count })
        : i18n.t('kanban.delete_column_empty_confirm', { name: this.dialogTargetColName });
      return (
        <div class="kb-overlay" onClick={() => this.onDialogCancel()}>
          <div class="kb-dialog kb-dialog-sm" onClick={(e) => e.stopPropagation()} role="alertdialog" aria-modal="true">
            <div class="kb-dialog-header">
              <h3>{i18n.t('kanban.delete_column')}</h3>
              <button type="button" class="kb-close" onClick={() => this.onDialogCancel()}>&times;</button>
            </div>
            <div class="kb-dialog-body">
              <p class="kb-dialog-message">{msg}</p>
            </div>
            <div class="kb-dialog-footer">
              <button type="button" class="kb-btn" onClick={() => this.onDialogCancel()}>{i18n.t('common.cancel')}</button>
              <button type="button" class="kb-btn kb-btn-danger" onClick={() => this.onDialogConfirm()}>{i18n.t('common.delete')}</button>
            </div>
          </div>
        </div>
      );
    }

    const isRename = this.dialogMode === 'rename-col';
    const title = isRename ? i18n.t('kanban.rename') : i18n.t('kanban.add_column');
    const placeholder = isRename ? i18n.t('kanban.new_name') : i18n.t('kanban.column_name');
    return (
      <div class="kb-overlay" onClick={() => this.onDialogCancel()}>
        <div class="kb-dialog kb-dialog-sm" onClick={(e) => e.stopPropagation()} role="dialog" aria-modal="true">
          <div class="kb-dialog-header">
            <h3>{title}</h3>
            <button type="button" class="kb-close" onClick={() => this.onDialogCancel()}>&times;</button>
          </div>
          <div class="kb-dialog-body">
            <input type="text" class="kb-dialog-input" value={this.dialogValue}
              placeholder={placeholder}
              onInput={(e) => { this.dialogValue = (e.target as HTMLInputElement).value; }}
              onKeyDown={(e) => { if (e.key === 'Enter') this.onDialogConfirm(); if (e.key === 'Escape') this.onDialogCancel(); }}
            />
          </div>
          <div class="kb-dialog-footer">
            <button type="button" class="kb-btn" onClick={() => this.onDialogCancel()}>{i18n.t('common.cancel')}</button>
            <button type="button" class="kb-btn kb-btn-primary" onClick={() => this.onDialogConfirm()}>{i18n.t('common.ok')}</button>
          </div>
        </div>
      </div>
    );
  }

  private renderColumn(colId: string, colData: KanbanColumnData) {
    const colColor = colData.config.color;
    const wipLimit = colData.config.wip_limit;
    const isOverWip = wipLimit && colData.cards.length >= wipLimit;
    const isAdding = this.addingToColumn === colId;

    return (
      <div class="kb-column" data-col-id={colId}>
        <div class="kb-col-header" style={colColor ? { borderTopColor: colColor } : {}}>
          <div class="kb-col-header-left">
            <span class="kb-col-name">{colData.config.name}</span>
            <span class={`kb-col-count${isOverWip ? ' kb-wip-over' : ''}`}>{colData.cards.length}{wipLimit ? `/${wipLimit}` : ''}</span>
          </div>
          <div class="kb-col-header-right">
            {this.allowRenameColumn && (
              <button type="button" class="kb-btn-icon" title={i18n.t('kanban.rename')} onClick={() => this.renameColumn(colId, colData.config.name)}>✏️</button>
            )}
            {this.allowDeleteColumn && (
              <button type="button" class="kb-btn-icon" title={i18n.t('kanban.delete')} onClick={() => this.deleteColumn(colId)}>🗑️</button>
            )}
            <button type="button" class="kb-btn-icon kb-col-menu" title={i18n.t('kanban.menu')}>⋯</button>
          </div>
        </div>

        <div class="kb-cards" data-col-id={colId}>
          {colData.cards.map(card => (
            <div class="kb-card" data-card-id={card.id} onClick={() => this.openCardDetail(card.id)}>
              {card.cover_image && (
                <div class="kb-card-cover">
                  <img src={card.cover_image} alt="" loading="lazy" />
                </div>
              )}
              {card.labels && card.labels.length > 0 && (
                <div class="kb-card-labels">
                  {card.labels.map(l => (
                    <span class="kb-card-label" style={{ backgroundColor: l.color }} title={l.name}></span>
                  ))}
                </div>
              )}
              <div class="kb-card-title">{card.title}</div>
              {card.due_date && (
                <div class={`kb-card-due${this.isOverdue(card.due_date) ? ' kb-overdue' : ''}${card.due_date_complete ? ' kb-complete' : ''}`}>
                  📅 {this.formatDate(card.due_date)}
                </div>
              )}
              <div class="kb-card-footer">
                <div class="kb-card-indicators">
                  {card.description && <span class="kb-indicator" title={i18n.t('kanban.has_description')}>📄</span>}
                  {(card.comments_count != null && card.comments_count > 0) && <span class="kb-indicator" title={i18n.t('kanban.comments')}>💬 {card.comments_count}</span>}
                  {(card.attachments_count != null && card.attachments_count > 0) && <span class="kb-indicator" title={i18n.t('kanban.attachments')}>📎 {card.attachments_count}</span>}
                  {card.subtasks && card.subtasks.length > 0 && (
                    <span class={`kb-indicator${this.subtaskProgress(card.subtasks) === 100 ? ' kb-complete' : ''}`} title={i18n.t('kanban.subtasks')}>
                      ✅ {card.subtasks.filter(s => s.done).length}/{card.subtasks.length}
                    </span>
                  )}
                </div>
                {card.priority && <span class={`kb-priority kb-priority-${card.priority}`}>{card.priority}</span>}
              </div>
              {card.assignees && card.assignees.length > 0 && (
                <div class="kb-card-assignees">
                  {card.assignees.map(a => (
                    <div class="kb-avatar" title={a.name} style={a.avatar ? { backgroundImage: `url(${a.avatar})` } : {}}>
                      {!a.avatar && a.name.charAt(0).toUpperCase()}
                    </div>
                  ))}
                </div>
              )}
            </div>
          ))}
        </div>

        {isAdding ? (
          <div class="kb-add-card-form">
            <textarea
              class="kb-add-card-input"
              placeholder={i18n.t('kanban.card_title')}
              value={this.newCardTitle}
              onInput={(e) => { this.newCardTitle = (e.target as HTMLTextAreaElement).value; }}
              onKeyDown={(e) => { if (e.key === 'Enter' && !e.shiftKey) { e.preventDefault(); this.addCard(colId); } }}
            ></textarea>
            <div class="kb-add-card-actions">
              <button type="button" class="kb-btn kb-btn-primary" onClick={() => this.addCard(colId)}>
                {i18n.t('kanban.add_card')}
              </button>
              <button type="button" class="kb-btn kb-btn-ghost" onClick={() => { this.addingToColumn = undefined; this.newCardTitle = ''; }}>
                ✕
              </button>
            </div>
          </div>
        ) : (
          this.allowAddCard && (
            <button type="button" class="kb-add-card-btn" onClick={() => { this.addingToColumn = colId; this.newCardTitle = ''; }}>
              + {i18n.t('kanban.add_a_card')}
            </button>
          )
        )}
      </div>
    );
  }

  private renderCardDetail() {
    const card = this.getSelectedCard();
    if (!card) return null;
    const priorities: Array<KanbanCard['priority']> = ['low', 'medium', 'high', 'critical'];
    const allLabelColors = ['#ef4444', '#f59e0b', '#10b981', '#3b82f6', '#8b5cf6', '#ec4899', '#06b6d4', '#6b7280'];

    return (
      <div class="kb-detail-overlay" onClick={() => this.closeCardDetail()}>
        <div class="kb-detail-panel" onClick={(e) => e.stopPropagation()}>
          <div class="kb-detail-header">
            {this.editingTitle ? (
              <div class="kb-edit-inline">
                <input type="text" class="kb-detail-title-input" value={this.editingTitleValue}
                  onInput={(e) => { this.editingTitleValue = (e.target as HTMLInputElement).value; }}
                  onKeyDown={(e) => { if (e.key === 'Enter') this.saveTitle(); if (e.key === 'Escape') this.editingTitle = false; }}
                />
                <button type="button" class="kb-btn kb-btn-ghost kb-btn-sm" onClick={() => this.editingTitle = false}>{i18n.t('common.cancel')}</button>
                <button type="button" class="kb-btn kb-btn-primary kb-btn-sm" onClick={() => this.saveTitle()}>{i18n.t('common.save')}</button>
              </div>
            ) : (
              <h3 class="kb-detail-title-editable" onClick={() => { this.editingTitleValue = card.title; this.editingTitle = true; }}>{card.title}</h3>
            )}
            <button type="button" class="kb-btn-icon kb-detail-close" onClick={() => this.closeCardDetail()}>✕</button>
          </div>

          <div class="kb-detail-body">
            <div class="kb-detail-meta">
              {/* PRIORITY */}
              <div class="kb-detail-field">
                <span class="kb-detail-label">{i18n.t('kanban.priority')}</span>
                {this.editingPriority ? (
                  <div class="kb-dropdown" onClick={(e) => e.stopPropagation()}>
                    <div class="kb-dropdown-header">
                      <span>{i18n.t('kanban.priority')}</span>
                      <button type="button" class="kb-btn-icon kb-dropdown-close" onClick={() => this.editingPriority = false}>✕</button>
                    </div>
                    {priorities.map(p => (
                      <div class={`kb-dropdown-item${card.priority === p ? ' kb-dropdown-active' : ''}`}
                        onClick={() => { this.updateField('priority', p); this.editingPriority = false; }}>
                        <span class={`kb-priority kb-priority-${p}`}>{p}</span>
                      </div>
                    ))}
                    <div class="kb-dropdown-item" onClick={() => { this.updateField('priority', undefined); this.editingPriority = false; }}>
                      <span class="kb-detail-empty">{i18n.t('kanban.no_priority')}</span>
                    </div>
                  </div>
                ) : (
                  <div class="kb-field-clickable" onClick={() => this.editingPriority = true}>
                    {card.priority
                      ? <span class={`kb-priority kb-priority-${card.priority}`}>{card.priority}</span>
                      : <span class="kb-detail-empty">—</span>}
                  </div>
                )}
              </div>

              {/* START DATE */}
              <div class="kb-detail-field">
                <span class="kb-detail-label">{i18n.t('kanban.start_date')}</span>
                {this.editingStartDate ? (
                  <div class="kb-inline-date" onClick={(e) => e.stopPropagation()}>
                    <input type="date" class="kb-date-input" value={card.start_date || ''}
                      onChange={(e) => { this.updateField('start_date', (e.target as HTMLInputElement).value || undefined); this.editingStartDate = false; }}
                      onKeyDown={(e) => { if (e.key === 'Escape') this.editingStartDate = false; }}
                    />
                    <button type="button" class="kb-date-clear" onClick={() => { this.updateField('start_date', undefined); this.editingStartDate = false; }}>✕</button>
                    <button type="button" class="kb-btn-icon kb-dropdown-close" onClick={() => this.editingStartDate = false}>✕</button>
                  </div>
                ) : (
                  <div class="kb-field-clickable" onClick={() => this.editingStartDate = true}>
                    {card.start_date
                      ? <span>{this.formatDate(card.start_date)}</span>
                      : <span class="kb-detail-empty">—</span>}
                  </div>
                )}
              </div>

              {/* DUE DATE */}
              <div class="kb-detail-field">
                <span class="kb-detail-label">{i18n.t('kanban.due_date')}</span>
                {this.editingDueDate ? (
                  <div class="kb-inline-date" onClick={(e) => e.stopPropagation()}>
                    <input type="date" class="kb-date-input" value={card.due_date || ''}
                      onChange={(e) => { this.updateField('due_date', (e.target as HTMLInputElement).value || undefined); this.editingDueDate = false; }}
                      onKeyDown={(e) => { if (e.key === 'Escape') this.editingDueDate = false; }}
                    />
                    <button type="button" class="kb-date-clear" onClick={() => { this.updateField('due_date', undefined); this.editingDueDate = false; }}>✕</button>
                    <button type="button" class="kb-btn-icon kb-dropdown-close" onClick={() => this.editingDueDate = false}>✕</button>
                  </div>
                ) : (
                  <div class="kb-field-clickable" onClick={() => this.editingDueDate = true}>
                    {card.due_date
                      ? <span class={this.isOverdue(card.due_date) ? 'kb-overdue' : ''}>{this.formatDate(card.due_date)}</span>
                      : <span class="kb-detail-empty">—</span>}
                  </div>
                )}
              </div>

              {/* ASSIGNEES — batch mode */}
              <div class="kb-detail-field">
                <span class="kb-detail-label">{i18n.t('kanban.assignees')}</span>
                {this.editingAssignees ? (
                  <div class="kb-dropdown kb-dropdown-wide" onClick={(e) => e.stopPropagation()}>
                    <div class="kb-dropdown-header">
                      <span>{i18n.t('kanban.assignees')}</span>
                      <div class="kb-dropdown-header-actions">
                        <button type="button" class="kb-btn kb-btn-ghost kb-btn-sm" onClick={() => this.cancelAssignees()}>{i18n.t('common.cancel')}</button>
                        <button type="button" class="kb-btn kb-btn-primary kb-btn-sm" onClick={() => this.saveAssignees(card)}>{i18n.t('kanban.done')}</button>
                      </div>
                    </div>
                    {this.mentionUsers.map(u => {
                      const isAssigned = this.draftAssignees.some(a => a.id === u.id);
                      return (
                        <div class={`kb-dropdown-item${isAssigned ? ' kb-dropdown-active' : ''}`}
                          onClick={() => this.toggleDraftAssignee(u)}>
                          <div class="kb-mention-avatar">{u.name.charAt(0).toUpperCase()}</div>
                          <span class="kb-mention-name">{u.name}</span>
                          {isAssigned && <span class="kb-check">✓</span>}
                        </div>
                      );
                    })}
                  </div>
                ) : (
                  <div class="kb-field-clickable" onClick={() => this.openAssignees(card)}>
                    <div class="kb-detail-assignees">
                      {card.assignees && card.assignees.length > 0
                        ? card.assignees.map(a => (
                          <div class="kb-avatar" title={a.name} style={a.avatar ? { backgroundImage: `url(${a.avatar})` } : {}}>
                            {!a.avatar && a.name.charAt(0).toUpperCase()}
                          </div>
                        ))
                        : <span class="kb-detail-empty">—</span>
                      }
                    </div>
                  </div>
                )}
              </div>

              {/* LABELS — batch mode */}
              <div class="kb-detail-field">
                <span class="kb-detail-label">{i18n.t('kanban.labels')}</span>
                {this.editingLabels ? (
                  <div class="kb-dropdown kb-dropdown-wide" onClick={(e) => e.stopPropagation()}>
                    <div class="kb-dropdown-header">
                      <span>{i18n.t('kanban.labels')}</span>
                      <div class="kb-dropdown-header-actions">
                        <button type="button" class="kb-btn kb-btn-ghost kb-btn-sm" onClick={() => this.cancelLabels()}>{i18n.t('common.cancel')}</button>
                        <button type="button" class="kb-btn kb-btn-primary kb-btn-sm" onClick={() => this.saveLabels(card)}>{i18n.t('kanban.done')}</button>
                      </div>
                    </div>
                    {this.draftLabels.map(l => (
                      <div class="kb-dropdown-item kb-dropdown-active"
                        onClick={() => this.toggleDraftLabel(l)}>
                        <span class="kb-color-dot" style={{ backgroundColor: l.color }}></span>
                        <span>{l.name}</span>
                        <span class="kb-check">✓</span>
                      </div>
                    ))}
                    <div class="kb-dropdown-divider"></div>
                    {allLabelColors.map(c => (
                      <div key={c} class="kb-dropdown-item" onClick={() => this.addDraftLabel(c)}>
                        <span class="kb-color-dot" style={{ backgroundColor: c }}></span>
                        <span>{i18n.t('kanban.create_label')}</span>
                      </div>
                    ))}
                  </div>
                ) : (
                  <div class="kb-field-clickable" onClick={() => this.openLabels(card)}>
                    <div class="kb-detail-labels">
                      {card.labels && card.labels.length > 0
                        ? card.labels.map(l => (
                          <span class="kb-detail-label-chip" style={{ backgroundColor: l.color }}>{l.name}</span>
                        ))
                        : <span class="kb-detail-empty">—</span>
                      }
                    </div>
                  </div>
                )}
              </div>
            </div>

            {/* DESCRIPTION */}
            <div class="kb-detail-section">
              <div class="kb-detail-section-header">
                <h4>{i18n.t('kanban.description')}</h4>
                {!this.editingDescription && (
                  <button type="button" class="kb-btn kb-btn-ghost" onClick={() => { this.draftDescription = card.description || ''; this.editingDescription = true; }}>{i18n.t('common.edit')}</button>
                )}
              </div>
              {this.editingDescription ? (
                <div class="kb-desc-editor-wrap">
                  <div class="kb-desc-editor">
                    <bc-field-richtext
                      name="description"
                      value={this.draftDescription}
                      toolbar="basic"
                      onLcFieldChange={(e: CustomEvent) => { this.draftDescription = (e.detail as { value: string }).value; }}
                    />
                  </div>
                  <div class="kb-desc-actions">
                    <button type="button" class="kb-btn kb-btn-ghost" onClick={() => this.cancelDescription()}>{i18n.t('common.cancel')}</button>
                    <button type="button" class="kb-btn kb-btn-primary" onClick={() => this.saveDescription(card)}>{i18n.t('common.save')}</button>
                  </div>
                </div>
              ) : (
                <div class="kb-detail-description">{card.description || <span class="kb-detail-empty">{i18n.t('kanban.no_description')}</span>}</div>
              )}
            </div>

            <bc-kanban-card-subtasks
              card-id={card.id}
              local-data={this.subtaskLocalData}
              data-source={this.subtaskDataSource}
              model={this.subtaskModel}
              fetch-headers={this.fetchHeaders}
            />

            <bc-kanban-card-attachments
              card-id={card.id}
              local-data={this.attachmentLocalData}
              data-source={this.attachmentDataSource}
              model={this.attachmentModel}
              fetch-headers={this.fetchHeaders}
            />

            <bc-kanban-card-comments
              card-id={card.id}
              local-data={this.commentLocalData}
              data-source={this.commentDataSource}
              model={this.commentModel}
              fetch-headers={this.fetchHeaders}
              mention-model={this.mentionModel}
              mention-data-source={this.mentionDataSource}
              mention-local-data={this.mentionLocalData}
            />

            <bc-kanban-card-activity
              card-id={card.id}
              local-data={this.activityLocalData}
              data-source={this.activityDataSource}
              model={this.activityModel}
              fetch-headers={this.fetchHeaders}
            />
          </div>
        </div>
      </div>
    );
  }

  // ─── EDIT HELPERS ───

  private async saveTitle() {
    const card = this.getSelectedCard();
    if (!card || !this.editingTitleValue.trim() || this.editingTitleValue.trim() === card.title) {
      this.editingTitle = false;
      return;
    }
    const newTitle = this.editingTitleValue.trim();
    this.updateCardInState(card.id, { title: newTitle });
    this.editingTitle = false;
    this.kanbanCardUpdate.emit({ cardId: card.id, data: { title: newTitle } });
    try { await kanbanUpdate(this.model || undefined, this.dataSource || undefined, card.id, { [this.cardTitleField]: newTitle }); } catch { /* optimistic */ }
  }

  // ─── Description: explicit save/cancel ───
  private saveDescription(card: KanbanCard) {
    this.updateCardInState(card.id, { description: this.draftDescription });
    this.editingDescription = false;
    this.kanbanCardUpdate.emit({ cardId: card.id, data: { description: this.draftDescription } });
    kanbanUpdate(this.model || undefined, this.dataSource || undefined, card.id, { [this.cardDescriptionField]: this.draftDescription }).catch(() => {});
  }

  private cancelDescription() {
    this.editingDescription = false;
  }

  // ─── Assignees: batch mode ───
  private openAssignees(card: KanbanCard) {
    this.draftAssignees = [...(card.assignees || [])];
    this.editingAssignees = true;
  }

  private toggleDraftAssignee(user: KanbanUser) {
    const isAssigned = this.draftAssignees.some(a => a.id === user.id);
    this.draftAssignees = isAssigned
      ? this.draftAssignees.filter(a => a.id !== user.id)
      : [...this.draftAssignees, user];
  }

  private cancelAssignees() {
    this.editingAssignees = false;
  }

  private async saveAssignees(card: KanbanCard) {
    this.updateCardInState(card.id, { assignees: this.draftAssignees });
    this.editingAssignees = false;
    this.kanbanCardUpdate.emit({ cardId: card.id, data: { assignees: this.draftAssignees } });
    try { await kanbanUpdate(this.model || undefined, this.dataSource || undefined, card.id, { [this.cardAssigneesField]: this.draftAssignees }); } catch { /* optimistic */ }
  }

  // ─── Labels: batch mode ───
  private openLabels(card: KanbanCard) {
    this.draftLabels = [...(card.labels || [])];
    this.editingLabels = true;
  }

  private toggleDraftLabel(label: KanbanLabel) {
    const exists = this.draftLabels.some(l => l.id === label.id);
    this.draftLabels = exists
      ? this.draftLabels.filter(l => l.id !== label.id)
      : [...this.draftLabels, label];
  }

  private addDraftLabel(color: string) {
    const name = prompt(i18n.t('kanban.label_name'));
    if (!name?.trim()) return;
    this.draftLabels = [...this.draftLabels, { id: `lbl-${Date.now()}`, name: name.trim(), color }];
  }

  private cancelLabels() {
    this.editingLabels = false;
  }

  private async saveLabels(card: KanbanCard) {
    this.updateCardInState(card.id, { labels: this.draftLabels });
    this.editingLabels = false;
    this.kanbanCardUpdate.emit({ cardId: card.id, data: { labels: this.draftLabels } });
    try { await kanbanUpdate(this.model || undefined, this.dataSource || undefined, card.id, { [this.cardLabelsField]: this.draftLabels }); } catch { /* optimistic */ }
  }

  private async updateField(field: string, value: unknown) {
    const card = this.getSelectedCard();
    if (!card) return;
    this.updateCardInState(card.id, { [field]: value });
    this.kanbanCardUpdate.emit({ cardId: card.id, data: { [field]: value } });
    const fieldMap: Record<string, string> = {
      priority: this.cardPriorityField,
      start_date: this.cardStartDateField,
      due_date: this.cardDueDateField,
    };
    const modelField = fieldMap[field] || field;
    try { await kanbanUpdate(this.model || undefined, this.dataSource || undefined, card.id, { [modelField]: value }); } catch { /* optimistic */ }
  }

  private updateCardInState(cardId: string, changes: Partial<KanbanCard>) {
    const newCols = new Map(this.columns);
    for (const [, col] of newCols) {
      const idx = col.cards.findIndex(c => c.id === cardId);
      if (idx !== -1) {
        col.cards[idx] = { ...col.cards[idx], ...changes };
        this.columns = newCols;
        return;
      }
    }
  }

  // ═══════════════════════════════════════════
  // HELPERS
  // ═══════════════════════════════════════════

  private formatDate(dateStr: string): string {
    try {
      const d = new Date(dateStr);
      return d.toLocaleDateString(undefined, { month: 'short', day: 'numeric' });
    } catch {
      return dateStr;
    }
  }

  private isOverdue(dateStr: string): boolean {
    try {
      return new Date(dateStr) < new Date();
    } catch {
      return false;
    }
  }

  private subtaskProgress(subtasks: KanbanCard['subtasks']): number {
    if (!subtasks || subtasks.length === 0) return 0;
    return Math.round((subtasks.filter(s => s.done).length / subtasks.length) * 100);
  }
}
