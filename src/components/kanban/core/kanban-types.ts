import { DataFetcher } from '../../../core/types';

// ═══════════════════════════════════════════
// DATA MODELS
// ═══════════════════════════════════════════

export interface KanbanLabel {
  id: string;
  name: string;
  color: string;
}

export interface KanbanUser {
  id: string;
  name: string;
  avatar?: string;
  email?: string;
}

export interface KanbanAttachment {
  id: string;
  name: string;
  url: string;
  type: 'image' | 'file';
  size: number;
  mime_type?: string;
  thumbnail_url?: string;
  created_at: string;
  user?: KanbanUser;
}

export interface KanbanSubtask {
  id: string;
  title: string;
  done: boolean;
  position?: number;
}

export interface KanbanComment {
  id: string;
  body: string;
  user: KanbanUser;
  created_at: string;
  updated_at?: string;
  attachments?: KanbanAttachment[];
  reactions?: KanbanReaction[];
}

export interface KanbanReaction {
  emoji: string;
  user: KanbanUser;
}

export interface KanbanActivity {
  id: string;
  action: string;
  user: KanbanUser;
  created_at: string;
  detail?: string;
}

export interface KanbanCard {
  id: string;
  title: string;
  description?: string;
  column: string;
  position?: number;
  labels?: KanbanLabel[];
  assignees?: KanbanUser[];
  due_date?: string;
  start_date?: string;
  due_date_complete?: boolean;
  priority?: 'low' | 'medium' | 'high' | 'critical';
  cover_image?: string;
  subtasks?: KanbanSubtask[];
  comments_count?: number;
  attachments_count?: number;
  [key: string]: unknown;
}

export interface KanbanColumnConfig {
  id: string;
  name: string;
  color?: string;
  wip_limit?: number;
  position?: number;
  collapsible?: boolean;
}

// ═══════════════════════════════════════════
// 4-LAYER DATA FETCH CONFIG
// Each sub-component has its own 4-layer config
// ═══════════════════════════════════════════

export interface KanbanFetchConfig {
  localData?: string;
  dataFetcher?: DataFetcher;
  dataSource?: string;
  model?: string;
  fetchHeaders?: string;
  fetchOptions?: string;
}

export interface KanbanBoardConfig extends KanbanFetchConfig {
  columns?: string;
  groupBy?: string;
  fields?: string;
  cardTitle?: string;
  cardDescription?: string;
  cardCover?: string;
  cardAssignees?: string;
  cardDueDate?: string;
  cardPriority?: string;
  cardLabels?: string;
  cardPosition?: string;
  columnModel?: string;
  columnDataSource?: string;
  columnLocalData?: string;
  mentionModel?: string;
  mentionDataSource?: string;
  mentionLocalData?: string;
}

export interface KanbanSubFetchConfig extends KanbanFetchConfig {
  filterBy?: string;
  filterValue?: string;
}

// ═══════════════════════════════════════════
// EVENTS
// ═══════════════════════════════════════════

export interface KanbanCardMoveEvent {
  cardId: string;
  fromColumn: string;
  toColumn: string;
  toPosition?: number;
}

export interface KanbanColumnReorderEvent {
  columns: string[];
}

export interface KanbanCardCreateEvent {
  column: string;
  title: string;
  position?: number;
}

export interface KanbanCardUpdateEvent {
  cardId: string;
  data: Partial<KanbanCard>;
}

export interface KanbanCardDeleteEvent {
  cardId: string;
  column: string;
}

export interface KanbanColumnAddEvent {
  name: string;
  position?: number;
}

export interface KanbanColumnRenameEvent {
  columnId: string;
  name: string;
}

export interface KanbanColumnDeleteEvent {
  columnId: string;
}

export interface KanbanCommentCreateEvent {
  cardId: string;
  body: string;
  attachments?: File[];
}

export interface KanbanCommentDeleteEvent {
  cardId: string;
  commentId: string;
}

export interface KanbanSubtaskToggleEvent {
  cardId: string;
  subtaskId: string;
  done: boolean;
}

export interface KanbanSubtaskCreateEvent {
  cardId: string;
  title: string;
}

export interface KanbanSubtaskDeleteEvent {
  cardId: string;
  subtaskId: string;
}

export interface KanbanAttachmentUploadEvent {
  cardId: string;
  files: File[];
}

export interface KanbanAttachmentDeleteEvent {
  cardId: string;
  attachmentId: string;
}

// ═══════════════════════════════════════════
// INTERNAL STATE
// ═══════════════════════════════════════════

export interface KanbanBoardState {
  columns: Map<string, KanbanColumnData>;
  loading: boolean;
  error?: string;
}

export interface KanbanColumnData {
  config: KanbanColumnConfig;
  cards: KanbanCard[];
}

export type KanbanViewMode = 'board' | 'list';

export interface KanbanCardDetailTab {
  id: string;
  label: string;
  icon?: string;
}
