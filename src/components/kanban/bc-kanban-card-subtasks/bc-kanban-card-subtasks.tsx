import { Component, Prop, State, Event, EventEmitter, Element, Watch, Method, h } from '@stencil/core';
import { DataFetcher } from '../../../core/types';
import { i18n } from '../../../core/i18n';
import { kanbanFetch, kanbanCreate, kanbanRemove } from '../core/kanban-data-fetcher';
import { KanbanSubtask, KanbanSubtaskToggleEvent, KanbanSubtaskCreateEvent, KanbanSubtaskDeleteEvent } from '../core/kanban-types';

@Component({ tag: 'bc-kanban-card-subtasks', styleUrl: 'bc-kanban-card-subtasks.css', shadow: false })
export class BcKanbanCardSubtasks {
  @Element() el!: HTMLElement;
  @Prop() cardId: string = '';
  @Prop() localData?: string;
  @Prop() dataFetcher?: DataFetcher;
  @Prop() dataSource: string = '';
  @Prop() model: string = '';
  @Prop() fetchHeaders: string = '';
  @Prop() filterBy: string = 'card_id';

  @State() subtasks: KanbanSubtask[] = [];
  @State() loading = false;
  @State() addingNew = false;
  @State() newTitle = '';

  @Event() kanbanSubtaskToggle!: EventEmitter<KanbanSubtaskToggleEvent>;
  @Event() kanbanSubtaskCreate!: EventEmitter<KanbanSubtaskCreateEvent>;
  @Event() kanbanSubtaskDelete!: EventEmitter<KanbanSubtaskDeleteEvent>;

  async componentDidLoad() { await this.loadSubtasks(); }

  @Watch('cardId') async onCardChange() { await this.loadSubtasks(); }

  @Method() async refresh(): Promise<void> { await this.loadSubtasks(); }

  private async loadSubtasks() {
    if (!this.cardId && !this.localData) return;
    this.loading = true;
    try {
      const result = await kanbanFetch({
        localData: this.localData, dataFetcher: this.dataFetcher,
        dataSource: this.dataSource, model: this.model,
        fetchHeaders: this.fetchHeaders, filterBy: this.filterBy, filterValue: this.cardId,
        element: this.el, params: { pageSize: 200 },
      });
      this.subtasks = result.data as KanbanSubtask[];
    } catch { this.subtasks = []; }
    this.loading = false;
  }

  private toggleSubtask(st: KanbanSubtask) {
    this.kanbanSubtaskToggle.emit({ cardId: this.cardId, subtaskId: st.id, done: !st.done });
    this.subtasks = this.subtasks.map(s => s.id === st.id ? { ...s, done: !s.done } : s);
  }

  private async addSubtask() {
    if (!this.newTitle.trim()) return;
    const title = this.newTitle.trim();
    this.kanbanSubtaskCreate.emit({ cardId: this.cardId, title });
    const optimistic: KanbanSubtask = { id: `temp-${Date.now()}`, title, done: false };
    this.subtasks = [...this.subtasks, optimistic];
    this.newTitle = '';
    this.addingNew = false;
    try {
      await kanbanCreate(this.model || undefined, this.dataSource || undefined, { title, [this.filterBy]: this.cardId, done: false });
    } catch { /* optimistic already added */ }
  }

  private async deleteSubtask(id: string) {
    this.kanbanSubtaskDelete.emit({ cardId: this.cardId, subtaskId: id });
    this.subtasks = this.subtasks.filter(s => s.id !== id);
    try { await kanbanRemove(this.model || undefined, this.dataSource || undefined, id); } catch { /* optimistic */ }
  }

  render() {
    const done = this.subtasks.filter(s => s.done).length;
    const total = this.subtasks.length;
    const progress = total > 0 ? Math.round((done / total) * 100) : 0;

    return (
      <div class="kb-subtasks">
        <div class="kb-section-header">
          <h4>{i18n.t('kanban.checklist')}</h4>
          {total > 0 && <span class="kb-progress">{done}/{total}</span>}
        </div>
        {total > 0 && (
          <div class="kb-progress-bar"><div class="kb-progress-fill" style={{ width: `${progress}%` }}></div></div>
        )}
        {this.subtasks.map(st => (
          <div class="kb-subtask">
            <input type="checkbox" checked={st.done} onChange={() => this.toggleSubtask(st)} />
            <span class={st.done ? 'kb-subtask-done' : ''}>{st.title}</span>
            <button type="button" class="kb-subtask-delete" onClick={() => this.deleteSubtask(st.id)}>✕</button>
          </div>
        ))}
        {this.addingNew ? (
          <div class="kb-subtask-add">
            <input type="text" value={this.newTitle}
              onInput={(e) => { this.newTitle = (e.target as HTMLInputElement).value; }}
              onKeyDown={(e) => { if (e.key === 'Enter') this.addSubtask(); }}
              placeholder={i18n.t('kanban.add_item_placeholder')}
            />
            <button type="button" class="kb-btn kb-btn-primary" onClick={() => this.addSubtask()}>{i18n.t('common.create')}</button>
            <button type="button" class="kb-btn kb-btn-ghost" onClick={() => { this.addingNew = false; this.newTitle = ''; }}>{i18n.t('common.cancel')}</button>
          </div>
        ) : (
          <button type="button" class="kb-add-item" onClick={() => { this.addingNew = true; this.newTitle = ''; }}>+ {i18n.t('kanban.add_item')}</button>
        )}
      </div>
    );
  }
}
