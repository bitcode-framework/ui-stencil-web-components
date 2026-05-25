import { Component, Prop, Element, Watch, Method, Event, EventEmitter, State, h } from '@stencil/core';
import { DataFetcher } from '../../../core/types';
import { fetchData } from '../../../core/data-fetcher';
import { getApiClient } from '../../../core/api-client';
import { BcSetup } from '../../../core/bc-setup';

// ============================================================================
// TYPES — full dhtmlxGantt API parity
// ============================================================================

export interface GanttTask {
  id: string | number;
  text?: string;
  start_date?: string | Date;
  end_date?: string | Date;
  duration?: number;
  progress?: number;
  parent?: string | number;
  type?: 'task' | 'project' | 'milestone';
  color?: string;
  textColor?: string;
  progressColor?: string;
  readonly?: boolean;
  open?: boolean;
  unscheduled?: boolean;
  deadline?: string | Date;
  constraint_type?: string;
  constraint_date?: string | Date;
  bar_height?: number;
  row_height?: number;
  [key: string]: unknown;
}

export interface GanttLink {
  id: string | number;
  source: string | number;
  target: string | number;
  type: '0' | '1' | '2' | '3';
  lag?: number;
  readonly?: boolean;
  color?: string;
  [key: string]: unknown;
}

export interface GanttColumn {
  name: string;
  label?: string;
  width?: number | string;
  tree?: boolean;
  align?: 'left' | 'center' | 'right';
  resize?: boolean;
  hide?: boolean;
  template?: (task: GanttTask) => string;
  sort?: boolean | string | ((a: GanttTask, b: GanttTask) => number);
}

export interface GanttScale {
  unit: 'minute' | 'hour' | 'day' | 'week' | 'month' | 'quarter' | 'year';
  step?: number;
  format?: string | ((date: Date) => string);
  css?: (date: Date) => string;
}

export interface GanttConfig {
  row_height?: number;
  bar_height?: number | 'full';
  bar_height_padding?: number;
  grid_width?: number;
  scale_height?: number;
  min_column_width?: number;
  duration_unit?: 'minute' | 'hour' | 'day' | 'week' | 'month' | 'year';
  duration_step?: number;
  date_format?: string;
  readonly?: boolean;
  drag_move?: boolean;
  drag_resize?: boolean;
  drag_progress?: boolean;
  drag_links?: boolean;
  drag_multiple?: boolean;
  show_progress?: boolean;
  show_links?: boolean;
  show_grid?: boolean;
  show_chart?: boolean;
  show_markers?: boolean;
  show_task_cells?: boolean;
  show_unscheduled?: boolean;
  fit_tasks?: boolean;
  auto_scheduling?: boolean;
  highlight_critical_path?: boolean;
  smart_rendering?: boolean;
  round_dnd_dates?: boolean;
  open_tree_initially?: boolean;
  rtl?: boolean;
  scroll_on_click?: boolean;
  preserve_scroll?: boolean;
  start_on_monday?: boolean;
  work_time?: boolean;
  skip_off_time?: boolean;
  touch?: boolean | string;
  multiselect?: boolean;
  keyboard_navigation?: boolean;
  undo?: boolean;
  redo?: boolean;
  undo_steps?: number;
  autosize?: boolean | 'x' | 'y' | 'xy';
  autoscroll?: boolean;
  autoscroll_speed?: number;
  static_background?: boolean;
  container_resize_timeout?: number;
  task_date?: string;
  time_picker?: string;
  end_date?: Date;
  start_date?: Date;
  min_duration?: number;
  root_id?: string | number;
  links?: {
    finish_to_start?: string;
    start_to_start?: string;
    finish_to_finish?: string;
    start_to_finish?: string;
  };
}

export interface GanttTemplates {
  task_class?: (start: Date, end: Date, task: GanttTask) => string;
  task_text?: (start: Date, end: Date, task: GanttTask) => string;
  grid_row_class?: (start: Date, end: Date, task: GanttTask) => string;
  grid_date_format?: (date: Date, column: string) => string;
  scale_cell_class?: (date: Date) => string;
  link_class?: (link: GanttLink) => string;
  tooltip_text?: (start: Date, end: Date, task: GanttTask) => string;
  tooltip_date_format?: (date: Date) => string;
  leftside_text?: (start: Date, end: Date, task: GanttTask) => string;
  rightside_text?: (start: Date, end: Date, task: GanttTask) => string;
  format_date?: (date: Date) => string;
}

export interface GanttMarker {
  id?: string | number;
  start_date: Date;
  end_date?: Date;
  css?: string;
  text?: string;
  title?: string;
}

export interface GanttTaskDragEvent {
  id: string | number;
  mode: 'move' | 'resize' | 'progress';
  task: GanttTask;
  original: GanttTask;
  nativeEvent: MouseEvent | TouchEvent;
}

export interface GanttTaskAddEvent { id: string | number; task: GanttTask; }
export interface GanttTaskUpdateEvent { id: string | number; task: GanttTask; oldTask?: GanttTask; }
export interface GanttTaskDeleteEvent { id: string | number; task: GanttTask; }
export interface GanttTaskSelectEvent { id: string | number; task: GanttTask; }
export interface GanttTaskMoveEvent { id: string | number; parent: string | number; index: number; }
export interface GanttLinkEvent { id: string | number; link: GanttLink; }
export interface GanttScaleClickEvent { date: Date; nativeEvent: MouseEvent; }
export interface GanttScrollEvent { left: number; top: number; }
export interface GanttColumnResizeEvent { index: number; column: GanttColumn; width: number; }
export interface GanttRowResizeEvent { id: string | number; task: GanttTask; oldHeight: number; newHeight: number; }
export interface GanttLinkDragEvent { from: string | number; fromStart: boolean; to: string | number | null; toStart: boolean; }
export interface GanttContextMenuEvent { taskId: string | number | null; linkId: string | number | null; nativeEvent: MouseEvent; }
export interface GanttErrorEvent { message: string; }

// ============================================================================
// INTERNAL
// ============================================================================

interface FlatTask extends GanttTask {
  _level: number;
  _index: number;
  _visible: boolean;
  _expanded: boolean;
  _x: number;
  _width: number;
  _y: number;
  _barY: number;
  _barHeight: number;
  _hasChildren: boolean;
}

const DURATION_UNITS: Record<string, number> = {
  minute: 60_000, hour: 3_600_000, day: 86_400_000,
  week: 604_800_000, month: 30 * 86_400_000, year: 365 * 86_400_000,
};

const DEFAULT_COLUMNS: GanttColumn[] = [
  { name: 'text', label: 'Task name', tree: true, width: 200 },
  { name: 'start_date', label: 'Start', align: 'center', width: 90 },
  { name: 'duration', label: 'Duration', align: 'center', width: 70 },
];

const DEFAULT_SCALES: GanttScale[] = [
  { unit: 'month', step: 1, format: '%F %Y' },
  { unit: 'day', step: 1, format: '%d' },
];

const SCALE_FMT: Record<string, (d: Date) => string> = {
  '%d': (d) => String(d.getDate()),
  '%D': (d) => ['Sun','Mon','Tue','Wed','Thu','Fri','Sat'][d.getDay()],
  '%m': (d) => String(d.getMonth() + 1).padStart(2, '0'),
  '%M': (d) => ['Jan','Feb','Mar','Apr','May','Jun','Jul','Aug','Sep','Oct','Nov','Dec'][d.getMonth()],
  '%F': (d) => ['January','February','March','April','May','June','July','August','September','October','November','December'][d.getMonth()],
  '%Y': (d) => String(d.getFullYear()),
  '%y': (d) => String(d.getFullYear()).slice(-2),
  '%H': (d) => String(d.getHours()).padStart(2, '0'),
  '%i': (d) => String(d.getMinutes()).padStart(2, '0'),
  '%Q': (d) => 'Q' + (Math.floor(d.getMonth() / 3) + 1),
};

function applyFormat(date: Date, fmt: string): string {
  let r = fmt;
  for (const [tok, fn] of Object.entries(SCALE_FMT)) {
    if (r.includes(tok)) r = r.replace(tok, fn(date));
  }
  return r;
}

function toDateString(v: string | Date | undefined): Date | null {
  if (!v) return null;
  if (v instanceof Date) return v;
  const d = new Date(v);
  return isNaN(d.getTime()) ? null : d;
}

@Component({ tag: 'bc-view-gantt', styleUrl: 'bc-view-gantt.css', shadow: false })
export class BcViewGantt {
  @Element() el!: HTMLElement;

  @Prop({ mutable: true }) tasks: string = '[]';
  @Prop({ mutable: true }) links: string = '[]';
  @Prop() columns: string = '';
  @Prop() scales: string = '';
  @Prop() markers: string = '[]';
  @Prop() config: string = '{}';
  @Prop() templates: string = '{}';
  @Prop() viewTitle: string = '';
  @Prop() height: string = '500px';
  @Prop() model: string = '';
  @Prop() localData?: string;
  @Prop() fields: string = '[]';
  @Prop() dataSource: string = '';
  @Prop() fetchHeaders: string = '';
  @Prop() fetchOptions?: string;
  @Prop() refreshInterval: number = 0;
  @Prop({ mutable: true }) loading: boolean = false;
  @Prop() readonly: boolean = false;
  @Prop() fieldMapping: string = '';
  dataFetcher?: DataFetcher;

  @State() _selectedId: string | number | null = null;

  private _pTasks: GanttTask[] = [];
  private _pLinks: GanttLink[] = [];
  private _pCols: GanttColumn[] = [];
  private _pScales: GanttScale[] = [];
  private _pMarkers: GanttMarker[] = [];
  private _pCfg: GanttConfig = {};
  private _pTpl: GanttTemplates = {};
  private _flat: FlatTask[] = [];
  private _tMap = new Map<string, GanttTask>();
  private _lMap = new Map<string, GanttLink>();
  private _collapsed = new Set<string>();
  private _tlStart!: Date;
  private _tlEnd!: Date;
  private _colW = 40;
  private _pxMs = 0;
  private _tlW = 0;
  private _ro: ResizeObserver | null = null;
  private _rt: ReturnType<typeof setInterval> | null = null;
  private _drag: {
    on: boolean; mode: 'move'|'rl'|'rr'|'prog'|null; tid: string|null;
    sx: number; orig: GanttTask|null;
  } | null = null;
  private _undo: Array<{ type: string; data: unknown }> = [];
  private _redo: Array<{ type: string; data: unknown }> = [];

  // ---- Events ----
  @Event() lcGanttTaskClick!: EventEmitter<{ id: string | number; task: GanttTask }>;
  @Event() lcGanttTaskDblClick!: EventEmitter<{ id: string | number; task: GanttTask }>;
  @Event() lcGanttTaskDrag!: EventEmitter<GanttTaskDragEvent>;
  @Event() lcGanttTaskAdd!: EventEmitter<GanttTaskAddEvent>;
  @Event() lcGanttTaskUpdate!: EventEmitter<GanttTaskUpdateEvent>;
  @Event() lcGanttTaskDelete!: EventEmitter<GanttTaskDeleteEvent>;
  @Event() lcGanttTaskSelect!: EventEmitter<GanttTaskSelectEvent>;
  @Event() lcGanttTaskUnselect!: EventEmitter<{ id: string | number }>;
  @Event() lcGanttTaskMove!: EventEmitter<GanttTaskMoveEvent>;
  @Event() lcGanttTaskRowClick!: EventEmitter<{ id: string | number; task: GanttTask }>;
  @Event() lcGanttLinkClick!: EventEmitter<{ id: string | number; link: GanttLink }>;
  @Event() lcGanttLinkDblClick!: EventEmitter<{ id: string | number; link: GanttLink }>;
  @Event() lcGanttLinkAdd!: EventEmitter<GanttLinkEvent>;
  @Event() lcGanttLinkUpdate!: EventEmitter<GanttLinkEvent>;
  @Event() lcGanttLinkDelete!: EventEmitter<GanttLinkEvent>;
  @Event() lcGanttLinkDrag!: EventEmitter<GanttLinkDragEvent>;
  @Event() lcGanttScaleClick!: EventEmitter<GanttScaleClickEvent>;
  @Event() lcGanttScroll!: EventEmitter<GanttScrollEvent>;
  @Event() lcGanttColumnResize!: EventEmitter<GanttColumnResizeEvent>;
  @Event() lcGanttRowResize!: EventEmitter<GanttRowResizeEvent>;
  @Event() lcGanttContextMenu!: EventEmitter<GanttContextMenuEvent>;
  @Event() lcGanttEmptyClick!: EventEmitter<{ nativeEvent: MouseEvent }>;
  @Event() lcGanttReady!: EventEmitter<void>;
  @Event() lcGanttRendered!: EventEmitter<void>;
  @Event() lcGanttParsed!: EventEmitter<void>;
  @Event() lcGanttError!: EventEmitter<GanttErrorEvent>;
  @Event() lcGanttTaskOpened!: EventEmitter<{ id: string | number }>;
  @Event() lcGanttTaskClosed!: EventEmitter<{ id: string | number }>;
  @Event() lcGanttBeforeTaskDrag!: EventEmitter<{ id: string | number; mode: string; nativeEvent: Event }>;
  @Event() lcGanttAfterTaskDrag!: EventEmitter<{ id: string | number; mode: string; nativeEvent: Event }>;
  @Event() lcGanttBeforeTaskAdd!: EventEmitter<{ id: string | number; task: GanttTask }>;
  @Event() lcGanttBeforeTaskDelete!: EventEmitter<{ id: string | number; task: GanttTask }>;
  @Event() lcGanttBeforeTaskUpdate!: EventEmitter<{ id: string | number; task: GanttTask }>;
  @Event() lcGanttBeforeTaskMove!: EventEmitter<{ id: string | number; parent: string | number; index: number }>;
  @Event() lcGanttBeforeLinkAdd!: EventEmitter<{ id: string | number; link: GanttLink }>;
  @Event() lcGanttBeforeLinkDelete!: EventEmitter<{ id: string | number; link: GanttLink }>;
  @Event() lcGanttBeforeLinkUpdate!: EventEmitter<{ id: string | number; link: GanttLink }>;

  // ========================================================================
  // LIFECYCLE
  // ========================================================================

  connectedCallback() { this._parseAll(); }

  componentDidLoad() {
    this._ro = new ResizeObserver(() => { this._recalc(); });
    this._ro.observe(this.el);
    if (this.model) this._fetchData();
    if (this.refreshInterval > 0) this._rt = setInterval(() => this._fetchData(), this.refreshInterval);
    this.lcGanttReady.emit();
  }

  disconnectedCallback() {
    this._ro?.disconnect();
    if (this._rt) clearInterval(this._rt);
  }

  @Watch('tasks') @Watch('links') @Watch('columns') @Watch('scales')
  @Watch('markers') @Watch('config') @Watch('templates')
  _onPropChange() { this._parseAll(); }

  @Watch('model') @Watch('dataSource')
  _onSourceChange() { this._fetchData(); }

  // ========================================================================
  // PUBLIC METHODS — dhtmlxGantt API
  // ========================================================================

  @Method() async setData(data: { tasks?: GanttTask[]; links?: GanttLink[] }): Promise<void> {
    if (data.tasks) this.tasks = JSON.stringify(data.tasks);
    if (data.links) this.links = JSON.stringify(data.links);
    this._parseAll();
    this.lcGanttParsed.emit();
  }

  @Method() async parseData(data: { tasks?: GanttTask[]; links?: GanttLink[] }): Promise<void> {
    return this.setData(data);
  }

  @Method() async getTask(id: string | number): Promise<GanttTask | undefined> {
    return this._tMap.get(String(id));
  }

  @Method() async getTasks(): Promise<GanttTask[]> { return [...this._pTasks]; }
  @Method() async getTaskCount(): Promise<number> { return this._pTasks.length; }
  @Method() async getVisibleTaskCount(): Promise<number> { return this._flat.filter(t => t._visible).length; }

  @Method() async addTask(task: GanttTask, parent?: string | number, index?: number): Promise<string | number> {
    const id = task.id ?? `t_${Date.now()}_${Math.random().toString(36).slice(2, 8)}`;
    const nt = { ...task, id, parent: parent ?? task.parent };
    this.lcGanttBeforeTaskAdd.emit({ id, task: nt });
    if (index !== undefined && parent !== undefined) {
      const sibs = this._pTasks.filter(t => String(t.parent) === String(parent));
      const at = Math.min(index, sibs.length);
      const gi = at < sibs.length ? this._pTasks.indexOf(sibs[at]) : this._pTasks.length;
      this._pTasks.splice(gi, 0, nt);
    } else { this._pTasks.push(nt); }
    this.tasks = JSON.stringify(this._pTasks);
    this._pushUndo('add', nt);
    this.lcGanttTaskAdd.emit({ id, task: nt });
    return id;
  }

  @Method() async updateTask(id: string | number, ns?: Partial<GanttTask>): Promise<void> {
    const t = this._tMap.get(String(id));
    if (!t) return;
    const old = { ...t };
    this.lcGanttBeforeTaskUpdate.emit({ id, task: { ...t, ...ns } });
    Object.assign(t, ns);
    this._pTasks = this._pTasks.map(x => String(x.id) === String(id) ? t : x);
    this.tasks = JSON.stringify(this._pTasks);
    this._pushUndo('update', { old, new: t });
    this.lcGanttTaskUpdate.emit({ id, task: t, oldTask: old });
  }

  @Method() async deleteTask(id: string | number): Promise<void> {
    const t = this._tMap.get(String(id));
    if (!t) return;
    this.lcGanttBeforeTaskDelete.emit({ id, task: t });
    const delIds = new Set<string>();
    delIds.add(String(id));
    this._pTasks.forEach(x => { if (delIds.has(String(x.parent))) delIds.add(String(x.id)); });
    this._pTasks = this._pTasks.filter(x => !delIds.has(String(x.id)));
    this._pLinks = this._pLinks.filter(l => String(l.source) !== String(id) && String(l.target) !== String(id));
    this.tasks = JSON.stringify(this._pTasks);
    this.links = JSON.stringify(this._pLinks);
    this._pushUndo('delete', t);
    this.lcGanttTaskDelete.emit({ id, task: t });
  }

  @Method() async moveTask(id: string | number, index: number, parent?: string | number): Promise<void> {
    const t = this._tMap.get(String(id));
    if (!t) return;
    const tp = parent ?? String(t.parent ?? '0');
    this.lcGanttBeforeTaskMove.emit({ id, parent: tp, index });
    this._pTasks = this._pTasks.filter(x => String(x.id) !== String(id));
    const sibs = this._pTasks.filter(x => String(x.parent) === String(tp));
    const at = Math.min(index, sibs.length);
    const gi = at < sibs.length ? this._pTasks.indexOf(sibs[at]) : this._pTasks.length;
    this._pTasks.splice(gi, 0, { ...t, parent: tp });
    this.tasks = JSON.stringify(this._pTasks);
    this.lcGanttTaskMove.emit({ id, parent: tp, index });
  }

  @Method() async getLink(id: string | number): Promise<GanttLink | undefined> { return this._lMap.get(String(id)); }
  @Method() async getLinks(): Promise<GanttLink[]> { return [...this._pLinks]; }
  @Method() async getLinkCount(): Promise<number> { return this._pLinks.length; }

  @Method() async addLink(link: GanttLink): Promise<string | number> {
    const id = link.id ?? `l_${Date.now()}_${Math.random().toString(36).slice(2, 8)}`;
    const nl = { ...link, id };
    this.lcGanttBeforeLinkAdd.emit({ id, link: nl });
    this._pLinks.push(nl);
    this.links = JSON.stringify(this._pLinks);
    this.lcGanttLinkAdd.emit({ id, link: nl });
    return id;
  }

  @Method() async updateLink(id: string | number, ns?: Partial<GanttLink>): Promise<void> {
    const l = this._lMap.get(String(id));
    if (!l) return;
    this.lcGanttBeforeLinkUpdate.emit({ id, link: { ...l, ...ns } });
    Object.assign(l, ns);
    this._pLinks = this._pLinks.map(x => String(x.id) === String(id) ? l : x);
    this.links = JSON.stringify(this._pLinks);
    this.lcGanttLinkUpdate.emit({ id, link: l });
  }

  @Method() async deleteLink(id: string | number): Promise<void> {
    const l = this._lMap.get(String(id));
    if (!l) return;
    this.lcGanttBeforeLinkDelete.emit({ id, link: l });
    this._pLinks = this._pLinks.filter(x => String(x.id) !== String(id));
    this.links = JSON.stringify(this._pLinks);
    this.lcGanttLinkDelete.emit({ id, link: l });
  }

  @Method() async selectTask(id: string | number): Promise<void> {
    if (this._selectedId === id) return;
    if (this._selectedId !== null) this.lcGanttTaskUnselect.emit({ id: this._selectedId });
    this._selectedId = id;
    const t = this._tMap.get(String(id));
    if (t) this.lcGanttTaskSelect.emit({ id, task: t });
  }

  @Method() async unselectTask(): Promise<void> {
    if (this._selectedId !== null) {
      this.lcGanttTaskUnselect.emit({ id: this._selectedId });
      this._selectedId = null;
    }
  }

  @Method() async getSelectedId(): Promise<string | number | null> { return this._selectedId; }
  @Method() async isSelected(id: string | number): Promise<boolean> { return this._selectedId === id; }
  @Method() async isTaskExists(id: string | number): Promise<boolean> { return this._tMap.has(String(id)); }
  @Method() async isTaskVisible(id: string | number): Promise<boolean> { return this._flat.some(t => String(t.id) === String(id) && t._visible); }
  @Method() async getChildren(id: string | number): Promise<GanttTask[]> { return this._pTasks.filter(t => String(t.parent) === String(id)); }
  @Method() async hasChild(id: string | number): Promise<number> { return this._pTasks.filter(t => String(t.parent) === String(id)).length; }
  @Method() async getParent(id: string | number): Promise<string | number | undefined> { return this._tMap.get(String(id))?.parent; }

  @Method() async getSiblings(id: string | number): Promise<GanttTask[]> {
    const t = this._tMap.get(String(id));
    if (!t) return [];
    return this._pTasks.filter(x => String(x.parent) === String(t.parent ?? '0'));
  }

  @Method() async getNext(id: string | number): Promise<string | number | null> {
    const i = this._flat.findIndex(t => String(t.id) === String(id));
    for (let j = i + 1; j < this._flat.length; j++) { if (this._flat[j]._visible) return this._flat[j].id; }
    return null;
  }

  @Method() async getPrev(id: string | number): Promise<string | number | null> {
    const i = this._flat.findIndex(t => String(t.id) === String(id));
    for (let j = i - 1; j >= 0; j--) { if (this._flat[j]._visible) return this._flat[j].id; }
    return null;
  }

  @Method() async open(id: string | number): Promise<void> { this._collapsed.delete(String(id)); this._recalc(); this.lcGanttTaskOpened.emit({ id }); }
  @Method() async close(id: string | number): Promise<void> { this._collapsed.add(String(id)); this._recalc(); this.lcGanttTaskClosed.emit({ id }); }

  @Method() async openAll(): Promise<void> {
    this._collapsed.clear();
    this._pTasks.forEach(t => { if (t.open !== false) this._collapsed.delete(String(t.id)); });
    this._recalc();
  }

  @Method() async closeAll(): Promise<void> {
    this._pTasks.forEach(t => { if (this._hasCh(t.id)) this._collapsed.add(String(t.id)); });
    this._recalc();
  }

  @Method() async scrollTo(x?: number, y?: number): Promise<void> {
    const tl = this.el.querySelector('.bc-gantt-tl-data') as HTMLElement;
    const gd = this.el.querySelector('.bc-gantt-grid-data') as HTMLElement;
    if (tl) { if (x != null) tl.scrollLeft = x; if (y != null) tl.scrollTop = y; }
    if (gd && y != null) gd.scrollTop = y;
  }

  @Method() async showTask(id: string | number): Promise<void> {
    const f = this._flat.find(t => String(t.id) === String(id));
    if (!f) return;
    const rH = this._pCfg.row_height || 32;
    const tl = this.el.querySelector('.bc-gantt-tl-data') as HTMLElement;
    if (tl) tl.scrollTop = f._index * rH - tl.clientHeight / 2;
  }

  @Method() async showDate(date: Date): Promise<void> {
    const x = this._d2x(date);
    const tl = this.el.querySelector('.bc-gantt-tl-data') as HTMLElement;
    if (tl) tl.scrollLeft = x - tl.clientWidth / 2;
  }

  @Method() async getScrollState(): Promise<GanttScrollEvent> {
    const tl = this.el.querySelector('.bc-gantt-tl-data') as HTMLElement;
    return { left: tl?.scrollLeft ?? 0, top: tl?.scrollTop ?? 0 };
  }

  @Method() async refresh(): Promise<void> {
    if (this.model || this.dataSource) await this._fetchData(); else this._parseAll();
  }

  @Method() async rerender(): Promise<void> { this._parseAll(); }

  @Method() async clearAll(): Promise<void> {
    this._pTasks = []; this._pLinks = []; this._pMarkers = [];
    this.tasks = '[]'; this.links = '[]'; this.markers = '[]';
    this._selectedId = null; this._collapsed.clear();
    this._undo = []; this._redo = [];
    this._parseAll();
  }

  @Method() async getState(): Promise<{ selectedIndex: string | number | null; scrollTop: number; scrollLeft: number }> {
    const s = await this.getScrollState();
    return { selectedIndex: this._selectedId, scrollTop: s.top, scrollLeft: s.left };
  }

  @Method() async addMarker(m: GanttMarker): Promise<string | number> {
    const id = m.id ?? `mk_${Date.now()}`;
    this._pMarkers.push({ ...m, id });
    this.markers = JSON.stringify(this._pMarkers);
    return id;
  }

  @Method() async deleteMarker(id: string | number): Promise<void> {
    this._pMarkers = this._pMarkers.filter(m => m.id !== id);
    this.markers = JSON.stringify(this._pMarkers);
  }

  @Method() async getMarker(id: string | number): Promise<GanttMarker | undefined> {
    return this._pMarkers.find(m => m.id === id);
  }

  @Method() async undo(): Promise<boolean> {
    if (!this._undo.length) return false;
    const a = this._undo.pop()!;
    this._redo.push(a);
    if (a.type === 'add') await this.deleteTask((a.data as GanttTask).id);
    else if (a.type === 'delete') { this._pTasks.push(a.data as GanttTask); this.tasks = JSON.stringify(this._pTasks); this._parseAll(); }
    else if (a.type === 'update') { const { old } = a.data as { old: GanttTask }; await this.updateTask(old.id, old); }
    return true;
  }

  @Method() async redo(): Promise<boolean> {
    if (!this._redo.length) return false;
    const a = this._redo.pop()!;
    this._undo.push(a);
    if (a.type === 'add') await this.addTask(a.data as GanttTask);
    else if (a.type === 'delete') await this.deleteTask((a.data as GanttTask).id);
    else if (a.type === 'update') { const { new: nt } = a.data as { new: GanttTask }; await this.updateTask(nt.id, nt); }
    return true;
  }

  @Method() async getUndoStack(): Promise<Array<{ type: string; data: unknown }>> { return [...this._undo]; }
  @Method() async getRedoStack(): Promise<Array<{ type: string; data: unknown }>> { return [...this._redo]; }
  @Method() async clearUndoStack(): Promise<void> { this._undo = []; }
  @Method() async clearRedoStack(): Promise<void> { this._redo = []; }
  @Method() async getTaskIndex(id: string | number): Promise<number> { return this._flat.findIndex(t => String(t.id) === String(id)); }
  @Method() async getGlobalTaskIndex(id: string | number): Promise<number> { return this._pTasks.findIndex(t => String(t.id) === String(id)); }

  @Method() async calculateDuration(cfg: { start_date: Date; end_date?: Date }): Promise<number> {
    const unit = DURATION_UNITS[this._pCfg.duration_unit || 'day'] || DURATION_UNITS.day;
    const step = this._pCfg.duration_step || 1;
    if (cfg.end_date) return Math.round((cfg.end_date.getTime() - cfg.start_date.getTime()) / (unit * step));
    return 0;
  }

  @Method() async calculateEndDate(cfg: { start_date: Date; duration: number }): Promise<Date> {
    const unit = DURATION_UNITS[this._pCfg.duration_unit || 'day'] || DURATION_UNITS.day;
    const step = this._pCfg.duration_step || 1;
    return new Date(cfg.start_date.getTime() + cfg.duration * unit * step);
  }

  @Method() async sortBy(field: string, desc = false): Promise<void> {
    this._pTasks.sort((a, b) => {
      const va = a[field], vb = b[field];
      let c = 0;
      if (typeof va === 'string' && typeof vb === 'string') c = va.localeCompare(vb);
      else c = ((va as number) ?? 0) - ((vb as number) ?? 0);
      return desc ? -c : c;
    });
    this.tasks = JSON.stringify(this._pTasks);
  }

  @Method() async eachTask(cb: (task: GanttTask) => void): Promise<void> { this._pTasks.forEach(cb); }

  @Method() async serialize(): Promise<{ tasks: GanttTask[]; links: GanttLink[] }> {
    return { tasks: [...this._pTasks], links: [...this._pLinks] };
  }

  @Method() async setSizes(): Promise<void> { this._parseAll(); }

  @Method() async expand(): Promise<void> {
    const g = this.el.querySelector('.bc-gantt') as HTMLElement;
    if (g?.requestFullscreen) await g.requestFullscreen();
  }

  @Method() async collapse(): Promise<void> {
    if (document.fullscreenElement) await document.exitFullscreen();
  }

  @Method() async dateFromPos(pos: number): Promise<Date | null> {
    if (!this._tlStart || !this._pxMs) return null;
    return new Date(this._tlStart.getTime() + pos / this._pxMs);
  }

  @Method() async posFromDate(date: Date): Promise<number> {
    if (!this._tlStart || !this._pxMs) return 0;
    return (date.getTime() - this._tlStart.getTime()) * this._pxMs;
  }

  // ========================================================================
  // INTERNALS
  // ========================================================================

  private _jsp<T>(json: string, fb: T): T {
    if (!json || json === '{}' || json === '[]') return fb;
    try { return JSON.parse(json); } catch { return fb; }
  }

  private _parseAll() {
    this._pTasks = this._jsp<GanttTask[]>(this.tasks, []);
    this._pLinks = this._jsp<GanttLink[]>(this.links, []);
    this._pCfg = { ...this._jsp<GanttConfig>(this.config, {}) };
    this._pTpl = { ...this._jsp<GanttTemplates>(this.templates, {}) };
    this._pMarkers = this._jsp<GanttMarker[]>(this.markers, []);
    this._pCols = this.columns ? this._jsp<GanttColumn[]>(this.columns, DEFAULT_COLUMNS) : [...DEFAULT_COLUMNS];
    this._pScales = this.scales ? this._jsp<GanttScale[]>(this.scales, DEFAULT_SCALES) : [...DEFAULT_SCALES];
    if (!this._pCols.length) this._pCols = [...DEFAULT_COLUMNS];
    if (!this._pScales.length) this._pScales = [...DEFAULT_SCALES];
    this._tMap.clear(); this._lMap.clear();
    this._pTasks.forEach(t => { this._tMap.set(String(t.id), t); });
    this._pLinks.forEach(l => { this._lMap.set(String(l.id), l); });
    if (this._pCfg.open_tree_initially) {
      this._pTasks.forEach(t => {
        if (t.open !== false && this._hasCh(t.id)) this._collapsed.delete(String(t.id));
      });
    }
    this._recalc();
  }

  private _hasCh(id: string | number): boolean {
    return this._pTasks.some(t => String(t.parent) === String(id));
  }

  private _recalc() {
    const cfg = this._pCfg;
    const rH = cfg.row_height || 32;
    const bH = cfg.bar_height === 'full' ? rH - 6 : (typeof cfg.bar_height === 'number' ? cfg.bar_height : 24);
    const bPad = cfg.bar_height === 'full' ? (cfg.bar_height_padding || 3) : Math.max(0, (rH - bH) / 2);
    const rootId = String(cfg.root_id ?? '0');

    const childMap = new Map<string, GanttTask[]>();
    this._pTasks.forEach(t => {
      const pid = String(t.parent ?? '0');
      if (!childMap.has(pid)) childMap.set(pid, []);
      childMap.get(pid)!.push(t);
    });

    const flat: FlatTask[] = [];
    let idx = 0;
    const walk = (pid: string, lv: number) => {
      for (const t of (childMap.get(pid) || [])) {
        const sid = String(t.id);
        const exp = !this._collapsed.has(sid);
        const hc = this._hasCh(t.id);
        flat.push({
          ...t, id: t.id, _level: lv, _index: idx, _visible: true,
          _expanded: exp, _x: 0, _width: 0, _y: idx * rH,
          _barY: idx * rH + bPad, _barHeight: bH, _hasChildren: hc,
        });
        idx++;
        if (hc && exp) walk(sid, lv + 1);
      }
    };
    walk(rootId, 0);
    this._flat = flat;

    // Timeline range
    const dates: number[] = [];
    this._pTasks.forEach(t => {
      const s = toDateString(t.start_date), e = toDateString(t.end_date);
      if (s) dates.push(s.getTime());
      if (e) dates.push(e.getTime());
    });
    if (!dates.length) {
      const now = new Date(); now.setHours(0, 0, 0, 0);
      dates.push(now.getTime(), now.getTime() + 30 * 86400000);
    }
    const range = Math.max(...dates) - Math.min(...dates);
    this._tlStart = cfg.start_date ?? new Date(Math.min(...dates) - range * 0.05);
    this._tlEnd = cfg.end_date ?? new Date(Math.max(...dates) + range * 0.1);

    const bs = this._pScales[this._pScales.length - 1];
    const step = bs.step || 1;
    const unitMs = DURATION_UNITS[bs.unit] || DURATION_UNITS.day;
    this._colW = Math.max(cfg.min_column_width || 30, 40);
    this._pxMs = this._colW / (unitMs * step);
    this._tlW = Math.ceil((this._tlEnd.getTime() - this._tlStart.getTime()) / (unitMs * step)) * this._colW;

    flat.forEach(t => {
      const s = toDateString(t.start_date), e = toDateString(t.end_date);
      if (s && e) {
        t._x = (s.getTime() - this._tlStart.getTime()) * this._pxMs;
        t._width = Math.max((e.getTime() - s.getTime()) * this._pxMs, 4);
      }
    });
    this.lcGanttRendered.emit();
  }

  private _d2x(d: Date): number { return (d.getTime() - this._tlStart.getTime()) * this._pxMs; }

  private _fmtLabel(d: Date, s: GanttScale): string {
    if (typeof s.format === 'function') return s.format(d);
    return applyFormat(d, s.format || '%d');
  }

  private _gridW(): number {
    if (this._pCfg.grid_width) return this._pCfg.grid_width;
    return this._pCols.reduce((s, c) => s + (typeof c.width === 'number' ? c.width : parseInt(String(c.width)) || 120), 0);
  }

  private _scaleH(): number { return (this._pCfg.scale_height || 40) * this._pScales.length; }

  private _pushUndo(type: string, data: unknown) {
    if (this._pCfg.undo === false) return;
    this._undo.push({ type, data });
    const mx = this._pCfg.undo_steps || 10;
    while (this._undo.length > mx) this._undo.shift();
    this._redo = [];
  }

  private _taskCls(t: FlatTask): string {
    const tpl = this._pTpl;
    const s = toDateString(t.start_date), e = toDateString(t.end_date);
    let extra = '';
    if (tpl.task_class && s && e) extra = tpl.task_class(s, e, t) || '';
    return `bc-gantt-bar bc-gantt-bar-${t.type || 'task'} ${extra}`;
  }

  private _taskTxt(t: FlatTask): string {
    const tpl = this._pTpl;
    const s = toDateString(t.start_date), e = toDateString(t.end_date);
    if (tpl.task_text && s && e) return tpl.task_text(s, e, t) || '';
    return String(t.text || '');
  }

  private _tipTxt(t: FlatTask): string {
    const tpl = this._pTpl;
    const s = toDateString(t.start_date), e = toDateString(t.end_date);
    if (tpl.tooltip_text && s && e) return tpl.tooltip_text(s, e, t) || '';
    const fmt = tpl.tooltip_date_format || ((d: Date) => d.toLocaleDateString());
    return `<b>${t.text || ''}</b><br/>${s ? fmt(s) : '?'} - ${e ? fmt(e) : '?'}<br/>Progress: ${Math.round((t.progress || 0) * 100)}%`;
  }

  private _cellVal(t: GanttTask, f: string): string {
    const v = t[f];
    if (f === 'start_date' || f === 'end_date') {
      const d = toDateString(v as string | Date);
      if (!d) return '';
      const tpl = this._pTpl;
      return tpl.grid_date_format ? tpl.grid_date_format(d, f) : d.toLocaleDateString();
    }
    if (f === 'duration' && (!v)) {
      const s = toDateString(t.start_date as string | Date), e = toDateString(t.end_date as string | Date);
      if (s && e) return String(Math.round((e.getTime() - s.getTime()) / 86400000));
      return '0';
    }
    if (f === 'progress') return Math.round((Number(v) || 0) * 100) + '%';
    return String(v ?? '');
  }

  // ---- Fetch ----

  private _fieldMap: Record<string, string> = {};

  private _resolveFieldMap(): Record<string, string> {
    if (this.fieldMapping) {
      try { return JSON.parse(this.fieldMapping); } catch { /* fall through */ }
    }
    if (this.fields) {
      try {
        const flds = JSON.parse(this.fields);
        if (Array.isArray(flds) && flds.length > 0) {
          const m: Record<string, string> = {};
          for (const f of flds) {
            if (typeof f === 'string') m[f] = f;
            else if (f && typeof f === 'object') {
              const o = f as Record<string, unknown>;
              if (o.gantt && o.name) m[String(o.gantt)] = String(o.name);
              else if (o.mapTo && o.field) m[String(o.mapTo)] = String(o.field);
            }
          }
          if (Object.keys(m).length > 0) return m;
        }
      } catch { /* fall through */ }
    }
    return {
      id: 'id',
      text: 'name',
      start_date: 'start_date',
      end_date: 'end_date',
      duration: 'duration',
      progress: 'progress',
      parent: 'parent',
      type: 'type',
      color: 'color',
    };
  }

  private _mapRecord(r: Record<string, unknown>): Record<string, unknown> {
    const fm = this._fieldMap;
    const get = (ganttField: string, ...fallbacks: string[]): unknown => {
      const mapped = fm[ganttField];
      if (mapped && r[mapped] !== undefined && r[mapped] !== null) return r[mapped];
      for (const fb of fallbacks) {
        if (r[fb] !== undefined && r[fb] !== null) return r[fb];
      }
      return undefined;
    };
    return {
      id: String(get('id', 'id') ?? ''),
      text: String(get('text', 'name', 'title', 'text') ?? ''),
      start_date: String(get('start_date', 'start_date', 'start') ?? ''),
      end_date: String(get('end_date', 'end_date', 'end') ?? ''),
      duration: Number(get('duration', 'duration') ?? 0) || undefined,
      progress: Number(get('progress', 'progress') ?? 0) || 0,
      parent: String(get('parent', 'parent_id', 'parent') ?? '0'),
      type: String(get('type', 'type') ?? 'task'),
      color: get('color', 'color') ? String(get('color', 'color')) : undefined,
    };
  }

  private async _fetchData() {
    this.loading = true;
    this._fieldMap = this._resolveFieldMap();
    const mapAndSet = (raw: unknown[]) => {
      if (raw.length > 0) this.tasks = JSON.stringify(raw.map((r) => this._mapRecord(r as Record<string, unknown>)));
    };

    // Layer 1: Custom data fetcher (standalone / framework integration)
    if (this.dataFetcher) {
      try {
        const result = await this.dataFetcher({ pageSize: 500 });
        mapAndSet(result.data);
      } catch (err) {
        this.lcGanttError.emit({ message: String(err) });
      }
      this.loading = false;
      return;
    }

    // Layer 2: Explicit data-source URL (standalone — manual fetch with BcSetup headers + intercept)
    if (this.dataSource) {
      try {
        const baseUrl = BcSetup.getBaseUrl();
        let url = this.dataSource;
        if (url && !url.startsWith('http') && baseUrl) url = baseUrl + url;
        const headers = { ...BcSetup.getHeaders(), ...(this.fetchHeaders ? JSON.parse(this.fetchHeaders) : {}) };

        const beforeEvent = new CustomEvent('lcBeforeFetch', { detail: { url, headers, params: {} }, bubbles: true, cancelable: true });
        this.el.dispatchEvent(beforeEvent);

        const res = await fetch(beforeEvent.detail.url, { headers: beforeEvent.detail.headers });
        if (!res.ok) throw new Error(`HTTP ${res.status}`);

        const json = await res.json();
        const afterEvent = new CustomEvent('lcAfterFetch', { detail: { response: json, data: null as unknown[] | null, total: 0 }, bubbles: true });
        this.el.dispatchEvent(afterEvent);

        if (afterEvent.detail.data) {
          mapAndSet(afterEvent.detail.data);
        } else {
          const arr = Array.isArray(json) ? json : json.tasks || json.data || [];
          mapAndSet(Array.isArray(arr) ? arr : []);
          if (json.links && Array.isArray(json.links)) this.links = JSON.stringify(json.links);
        }
      } catch (err) {
        this.lcGanttError.emit({ message: String(err) });
      }
      this.loading = false;
      return;
    }

    // Layer 3: Model name (BitCode engine — try shared data-fetcher, fallback to getApiClient)
    if (this.model) {
      try {
        const result = await fetchData({
          element: this.el,
          model: this.model, localData: this.localData, fetchOptions: this.fetchOptions ? JSON.parse(this.fetchOptions) : undefined,
          fetchHeaders: this.fetchHeaders,
          params: { pageSize: 500 },
        });
        mapAndSet(result.data);
      } catch {
        // Fallback: direct API client (BitCode engine guaranteed)
        try {
          const api = getApiClient();
          const res = await api.list(this.model, { pageSize: 500 });
          mapAndSet(res.data);
        } catch (err) {
          this.lcGanttError.emit({ message: String(err) });
        }
      }
      this.loading = false;
      return;
    }

    this.loading = false;
  }

  // ---- Drag ----

  private _onBarDown(ev: MouseEvent, tid: string | number, mode: 'move'|'rl'|'rr'|'prog') {
    const cfg = this._pCfg;
    if (this.readonly || cfg.readonly) return;
    const t = this._tMap.get(String(tid));
    if (!t || t.readonly) return;
    this.lcGanttBeforeTaskDrag.emit({ id: tid, mode, nativeEvent: ev });
    this._drag = { on: true, mode, tid: String(tid), sx: ev.clientX, orig: { ...t } };
    const onM = (e: MouseEvent) => this._onDragM(e);
    const onU = (e: MouseEvent) => { window.removeEventListener('mousemove', onM); window.removeEventListener('mouseup', onU); this._onDragU(e); };
    window.addEventListener('mousemove', onM);
    window.addEventListener('mouseup', onU);
    ev.preventDefault();
  }

  private _onDragM(e: MouseEvent) {
    if (!this._drag?.on || !this._drag.tid) return;
    const dx = e.clientX - this._drag.sx;
    const cfg = this._pCfg;
    const t = this._tMap.get(this._drag.tid);
    if (!t) return;
    const o = this._drag.orig!;
    const os = toDateString(o.start_date), oe = toDateString(o.end_date);
    if (!os || !oe) return;

    if (this._drag.mode === 'move' && cfg.drag_move !== false) {
      t.start_date = new Date(os.getTime() + dx / this._pxMs).toISOString();
      t.end_date = new Date(oe.getTime() + dx / this._pxMs).toISOString();
    } else if (this._drag.mode === 'rr' && cfg.drag_resize !== false) {
      t.end_date = new Date(oe.getTime() + dx / this._pxMs).toISOString();
    } else if (this._drag.mode === 'rl' && cfg.drag_resize !== false) {
      t.start_date = new Date(os.getTime() + dx / this._pxMs).toISOString();
    } else if (this._drag.mode === 'prog' && cfg.drag_progress !== false) {
      const bar = this.el.querySelector(`[data-task-id="${this._drag.tid}"]`) as HTMLElement;
      if (bar) { const r = bar.getBoundingClientRect(); t.progress = Math.max(0, Math.min(1, (e.clientX - r.left) / r.width)); }
    }
    this._pTasks = this._pTasks.map(x => String(x.id) === this._drag!.tid ? t : x);
    this._recalc();
  }

  private _onDragU(e: MouseEvent) {
    if (!this._drag?.on) return;
    const m = this._drag.mode;
    const tid = this._drag.tid!;
    const orig = this._drag.orig;
    const t = this._tMap.get(tid);
    this._drag = null;
    if (t && orig) {
      const ms = m === 'move' ? 'move' : m === 'prog' ? 'progress' : 'resize';
      this.lcGanttTaskDrag.emit({ id: tid, mode: ms as 'move'|'resize'|'progress', task: t, original: orig, nativeEvent: e });
      this.lcGanttAfterTaskDrag.emit({ id: tid, mode: ms, nativeEvent: e });
      this._pushUndo('update', { old: orig, new: t });
      this.lcGanttTaskUpdate.emit({ id: tid, task: t, oldTask: orig });
    }
  }

  // ---- Event handlers ----

  private _onTaskClick(id: string | number) {
    const t = this._tMap.get(String(id));
    if (!t) return;
    this.lcGanttTaskClick.emit({ id, task: t });
    this.selectTask(id);
  }

  private _onTaskDbl(id: string | number) {
    const t = this._tMap.get(String(id));
    if (t) this.lcGanttTaskDblClick.emit({ id, task: t });
  }

  private _onRowClick(id: string | number) {
    const t = this._tMap.get(String(id));
    if (t) this.lcGanttTaskRowClick.emit({ id, task: t });
  }

  private _onToggle(ev: MouseEvent, id: string | number) {
    ev.stopPropagation();
    if (this._collapsed.has(String(id))) this.open(id); else this.close(id);
  }

  private _onLinkClick(id: string | number) {
    const l = this._lMap.get(String(id));
    if (l) this.lcGanttLinkClick.emit({ id, link: l });
  }

  private _onLinkDbl(id: string | number) {
    const l = this._lMap.get(String(id));
    if (l) this.lcGanttLinkDblClick.emit({ id, link: l });
  }

  private _onScaleClick(ev: MouseEvent, date: Date) { this.lcGanttScaleClick.emit({ date, nativeEvent: ev }); }
  private _onEmptyClick(ev: MouseEvent) { this.lcGanttEmptyClick.emit({ nativeEvent: ev }); }

  private _onCtxMenu(ev: MouseEvent) {
    const bar = (ev.target as HTMLElement).closest('[data-task-id]') as HTMLElement;
    const lnk = (ev.target as HTMLElement).closest('[data-link-id]') as HTMLElement;
    this.lcGanttContextMenu.emit({ taskId: bar?.dataset?.taskId ?? null, linkId: lnk?.dataset?.linkId ?? null, nativeEvent: ev });
  }

  private _onTlScroll(ev: Event) {
    const tl = ev.target as HTMLElement;
    const gd = this.el.querySelector('.bc-gantt-grid-data') as HTMLElement;
    const th = this.el.querySelector('.bc-gantt-tl-head') as HTMLElement;
    if (gd) gd.scrollTop = tl.scrollTop;
    if (th) th.scrollLeft = tl.scrollLeft;
    this.lcGanttScroll.emit({ left: tl.scrollLeft, top: tl.scrollTop });
  }

  // ---- Scale cells ----

  private _scaleCells(s: GanttScale): Array<{ date: Date; label: string; left: number; width: number }> {
    const out: Array<{ date: Date; label: string; left: number; width: number }> = [];
    const unitMs = DURATION_UNITS[s.unit] || DURATION_UNITS.day;
    const stepMs = unitMs * (s.step || 1);
    let d = new Date(this._tlStart);
    while (d < this._tlEnd) {
      const nx = new Date(d.getTime() + stepMs);
      const l = this._d2x(d), r = this._d2x(nx);
      out.push({ date: new Date(d), label: this._fmtLabel(d, s), left: Math.max(0, l), width: Math.max(1, r - l) });
      d = nx;
    }
    return out;
  }

  private _isToday(d: Date): boolean {
    const n = new Date();
    return d.getFullYear() === n.getFullYear() && d.getMonth() === n.getMonth() && d.getDate() === n.getDate();
  }

  private _todayLabel(): string {
    const locale = (this.el as HTMLElement).lang || (document.documentElement && document.documentElement.lang) || 'en';
    try { return new Date().toLocaleDateString(locale, { month: 'short', day: 'numeric' }); } catch { return 'Today'; }
  }

  // ========================================================================
  // RENDER
  // ========================================================================

  render() {
    const cfg = this._pCfg;
    const rH = cfg.row_height || 32;
    const showProg = cfg.show_progress !== false;
    const showLinks = cfg.show_links !== false;
    const showCells = cfg.show_task_cells !== false;
    const cols = this._pCols.filter(c => !c.hide);
    const gridW = cfg.show_grid !== false ? this._gridW() : 0;
    const totalH = this._flat.length * rH;

    return (
      <div class={{ 'bc-gantt': true, 'bc-gantt-rtl': !!cfg.rtl }} style={{ height: this.height }}>

        {this.viewTitle && <div class="bc-gantt-head"><h2 class="bc-gantt-title">{this.viewTitle}</h2></div>}

        {this.loading && <div class="bc-gantt-loading"><span class="bc-gantt-spinner" /></div>}

        <div class="bc-gantt-body">

          {/* GRID */}
          {cfg.show_grid !== false && (
            <div class="bc-gantt-grid" style={{ width: gridW + 'px' }}>
              <div class="bc-gantt-grid-hd" style={{ height: this._scaleH() + 'px' }}>
                <div class="bc-gantt-grid-hd-row">
                  {cols.map(c => (
                    <div class="bc-gantt-grid-hd-cell" style={{ width: (c.width || 120) + 'px', textAlign: c.align || 'left' }}>
                      {c.label || c.name}
                    </div>
                  ))}
                </div>
              </div>
              <div class="bc-gantt-grid-data" style={{ maxHeight: `calc(${this.height} - ${this._scaleH()}px - 42px)` }}>
                <div style={{ height: totalH + 'px' }}>
                  {this._flat.map(t => (
                    <div
                      class={{
                        'bc-gantt-row': true,
                        'bc-gantt-row-odd': t._index % 2 === 1,
                        'bc-gantt-row-sel': String(this._selectedId) === String(t.id),
                      }}
                      style={{ height: rH + 'px' }}
                      onClick={() => this._onRowClick(t.id)}
                    >
                      {cols.map(c => (
                        <div class={{ 'bc-gantt-cell': true, 'bc-gantt-cell-tree': !!c.tree }} style={{ width: (c.width || 120) + 'px', textAlign: c.align || 'left' }}>
                          {c.tree && <span class="bc-gantt-indent" style={{ width: t._level * 20 + 'px' }} />}
                          {c.tree && t._hasChildren && (
                            <button class="bc-gantt-toggle" onClick={(ev) => this._onToggle(ev, t.id)}>
                              {t._expanded ? '▾' : '▸'}
                            </button>
                          )}
                          {c.tree && !t._hasChildren && <span class="bc-gantt-toggle-spacer" />}
                          <span class="bc-gantt-cell-val">{c.template ? c.template(t) : this._cellVal(t, c.name)}</span>
                        </div>
                      ))}
                    </div>
                  ))}
                </div>
              </div>
            </div>
          )}

          {/* TIMELINE */}
          {cfg.show_chart !== false && (
            <div class="bc-gantt-tl">
              <div class="bc-gantt-tl-head" style={{ height: this._scaleH() + 'px' }}>
                {this._pScales.map(sc => {
                  const cells = this._scaleCells(sc);
                  return (
                    <div class="bc-gantt-scale-row" style={{ height: (this._pCfg.scale_height || 40) + 'px' }}>
                      {cells.map(cell => (
                        <div class="bc-gantt-scale-cell" style={{ width: cell.width + 'px', left: cell.left + 'px' }} onClick={(ev) => this._onScaleClick(ev, cell.date)}>
                          <span class="bc-gantt-scale-lbl">{cell.label}</span>
                        </div>
                      ))}
                    </div>
                  );
                })}
              </div>

              <div class="bc-gantt-tl-data" onScroll={(ev) => this._onTlScroll(ev)} onClick={(ev) => this._onEmptyClick(ev)}>
                <div class="bc-gantt-tl-content" style={{ width: this._tlW + 'px', height: totalH + 'px' }}>

                  {/* Background cells */}
                  {showCells && this._flat.map(t => {
                    const bs = this._pScales[this._pScales.length - 1];
                    return this._scaleCells(bs).map(cell => (
                      <div
                        class={{ 'bc-gantt-bg': true, 'bc-gantt-bg-we': cell.date.getDay() === 0 || cell.date.getDay() === 6, 'bc-gantt-bg-today': this._isToday(cell.date) }}
                        style={{ left: cell.left + 'px', top: t._index * rH + 'px', width: cell.width + 'px', height: rH + 'px' }}
                      />
                    ));
                  })}

                  {/* Links — single SVG container */}
                  {showLinks && this._pLinks.length > 0 && (
                    <svg class="bc-gantt-link-svg" style={{ position: 'absolute', left: '0', top: '0', width: this._tlW + 'px', height: '100%', pointerEvents: 'none', overflow: 'visible' }}>
                      {this._pLinks.map(link => {
                        const src = this._flat.find(ft => String(ft.id) === String(link.source));
                        const tgt = this._flat.find(ft => String(ft.id) === String(link.target));
                        if (!src || !tgt) return null;
                        const se = src._x + src._width, sm = src._barY + src._barHeight / 2, tm = tgt._barY + tgt._barHeight / 2;
                        let x1: number, y1: number, x2: number, y2: number;
                        if (link.type === '0') { x1 = se; y1 = sm; x2 = tgt._x; y2 = tm; }
                        else if (link.type === '1') { x1 = src._x; y1 = sm; x2 = tgt._x; y2 = tm; }
                        else if (link.type === '2') { x1 = se; y1 = sm; x2 = tgt._x + tgt._width; y2 = tm; }
                        else { x1 = src._x; y1 = sm; x2 = tgt._x + tgt._width; y2 = tm; }
                        const mx = (x1 + x2) / 2;
                        const d = `M ${x1} ${y1} C ${mx} ${y1}, ${mx} ${y2}, ${x2} ${y2}`;
                        return (
                          <path
                            class="bc-gantt-link-path"
                            data-link-id={String(link.id)}
                            d={d} fill="none"
                            stroke={link.color || 'var(--bc-gantt-link)'}
                            stroke-width="2"
                            style={{ pointerEvents: 'stroke', cursor: 'pointer' }}
                            onClick={() => this._onLinkClick(link.id)}
                            onDblClick={() => this._onLinkDbl(link.id)}
                          />
                        );
                      })}
                    </svg>
                  )}

                  {/* Today marker */}
                  <div class="bc-gantt-marker bc-gantt-today" style={{ left: this._d2x(new Date()) + 'px' }}>
                    <div class="bc-gantt-marker-text">{this._todayLabel()}</div>
                  </div>

                  {/* Custom markers */}
                  {this._pMarkers.map(mk => (
                    <div class={`bc-gantt-marker ${mk.css || ''}`} style={{ left: this._d2x(mk.start_date) + 'px' }}>
                      {mk.text && <div class="bc-gantt-marker-text">{mk.text}</div>}
                    </div>
                  ))}

                  {/* Task bars */}
                  {this._flat.map(t => {
                    if (t.unscheduled && this._pCfg.show_unscheduled !== true) return null;

                    // Milestone
                    if (t.type === 'milestone') {
                      const sz = t._barHeight;
                      return (
                        <div
                          class="bc-gantt-bar bc-gantt-bar-milestone"
                          data-task-id={String(t.id)}
                          style={{ left: (t._x - sz / 2) + 'px', top: t._barY + 'px', width: sz + 'px', height: sz + 'px', ...(t.color ? { backgroundColor: t.color } : {}) }}
                          onClick={() => this._onTaskClick(t.id)}
                          onDblClick={() => this._onTaskDbl(t.id)}
                          onContextMenu={(ev) => this._onCtxMenu(ev)}
                        >
                          <div class="bc-gantt-diamond" />
                          <div class="bc-gantt-tip">{this._tipTxt(t)}</div>
                        </div>
                      );
                    }

                    // Regular / project bar
                    const prog = Math.max(0, Math.min(1, t.progress || 0));
                    return (
                      <div
                        class={this._taskCls(t)}
                        data-task-id={String(t.id)}
                        style={{
                          left: t._x + 'px', top: t._barY + 'px', width: t._width + 'px', height: t._barHeight + 'px',
                          ...(t.color ? { backgroundColor: t.color } : {}),
                          ...(t.textColor ? { color: t.textColor } : {}),
                        }}
                        onClick={() => this._onTaskClick(t.id)}
                        onDblClick={() => this._onTaskDbl(t.id)}
                        onContextMenu={(ev) => this._onCtxMenu(ev)}
                      >
                        {showProg && prog > 0 && (
                          <div class="bc-gantt-bar-prog" style={{ width: (prog * 100) + '%', ...(t.progressColor ? { backgroundColor: t.progressColor } : {}) }} />
                        )}
                        <div class="bc-gantt-bar-text">{this._taskTxt(t)}</div>
                        {!this.readonly && cfg.readonly !== true && cfg.drag_resize !== false && (
                          <div class="bc-gantt-handle bc-gantt-handle-l" onMouseDown={(ev) => this._onBarDown(ev, t.id, 'rl')} />
                        )}
                        {!this.readonly && cfg.readonly !== true && cfg.drag_resize !== false && (
                          <div class="bc-gantt-handle bc-gantt-handle-r" onMouseDown={(ev) => this._onBarDown(ev, t.id, 'rr')} />
                        )}
                        {!this.readonly && cfg.readonly !== true && showProg && cfg.drag_progress !== false && (
                          <div class="bc-gantt-handle bc-gantt-handle-prog" style={{ left: (prog * t._width) + 'px' }} onMouseDown={(ev) => this._onBarDown(ev, t.id, 'prog')} />
                        )}
                        {!this.readonly && cfg.readonly !== true && cfg.drag_move !== false && (
                          <div class="bc-gantt-drag-area" onMouseDown={(ev) => this._onBarDown(ev, t.id, 'move')} />
                        )}
                        <div class="bc-gantt-tip">{this._tipTxt(t)}</div>
                      </div>
                    );
                  })}
                </div>
              </div>
            </div>
          )}
        </div>
      </div>
    );
  }
}
