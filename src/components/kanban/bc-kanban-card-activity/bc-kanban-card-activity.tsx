import { Component, Prop, State, Element, Watch, Method, h } from '@stencil/core';
import { DataFetcher } from '../../../core/types';
import { i18n } from '../../../core/i18n';
import { kanbanFetch } from '../core/kanban-data-fetcher';
import { KanbanActivity } from '../core/kanban-types';

@Component({ tag: 'bc-kanban-card-activity', styleUrl: 'bc-kanban-card-activity.css', shadow: false })
export class BcKanbanCardActivity {
  @Element() el!: HTMLElement;
  @Prop() cardId: string = '';
  @Prop() localData?: string;
  @Prop() dataFetcher?: DataFetcher;
  @Prop() dataSource: string = '';
  @Prop() model: string = '';
  @Prop() fetchHeaders: string = '';
  @Prop() filterBy: string = 'card_id';

  @State() activities: KanbanActivity[] = [];
  @State() loading = false;

  async componentDidLoad() { await this.loadActivities(); }
  @Watch('cardId') async onCardChange() { await this.loadActivities(); }
  @Method() async refresh(): Promise<void> { await this.loadActivities(); }

  private async loadActivities() {
    if (!this.cardId && !this.localData) return;
    this.loading = true;
    try {
      const result = await kanbanFetch({
        localData: this.localData, dataFetcher: this.dataFetcher,
        dataSource: this.dataSource, model: this.model,
        fetchHeaders: this.fetchHeaders, filterBy: this.filterBy, filterValue: this.cardId,
        element: this.el, params: { pageSize: 50 },
      });
      this.activities = (result.data as KanbanActivity[]).sort((a, b) =>
        new Date(b.created_at).getTime() - new Date(a.created_at).getTime()
      );
    } catch { this.activities = []; }
    this.loading = false;
  }

  private formatTime(dateStr: string): string {
    const diff = (Date.now() - new Date(dateStr).getTime()) / 1000;
    if (diff < 60) return i18n.t('kanban.just_now');
    if (diff < 3600) return i18n.t('kanban.minutes_ago', { count: Math.floor(diff / 60) });
    if (diff < 86400) return i18n.t('kanban.hours_ago', { count: Math.floor(diff / 3600) });
    return new Date(dateStr).toLocaleDateString(undefined, { month: 'short', day: 'numeric' });
  }

  render() {
    return (
      <div class="kb-activity">
        <h4>{i18n.t('kanban.activity')}</h4>
        {this.loading && <div class="kb-activity-loading">{i18n.t('kanban.loading')}</div>}
        <div class="kb-activity-list">
          {this.activities.map(a => (
            <div class="kb-activity-item">
              <div class="kb-activity-dot"></div>
              <div class="kb-activity-content">
                <span class="kb-activity-user">{a.user?.name || 'Unknown'}</span>
                <span class="kb-activity-action">{a.action}</span>
                {a.detail && <span class="kb-activity-detail">{a.detail}</span>}
                <span class="kb-activity-time">{this.formatTime(a.created_at)}</span>
              </div>
            </div>
          ))}
        </div>
      </div>
    );
  }
}
