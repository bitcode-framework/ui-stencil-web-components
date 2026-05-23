export interface ComponentMeta {
  tag: string;
  group: string;
  className: string;
  dependencies?: string[];
}

export interface RegistrySelector {
  includeGroups?: string[];
  excludeGroups?: string[];
  includeComponents?: string[];
  excludeComponents?: string[];
}

export interface RegistryResult {
  components: ComponentMeta[];
  tags: string[];
  define: () => void;
}

const COMPONENTS: ComponentMeta[] = [
  // charts (26)
  { tag: 'bc-chart-area', group: 'charts', className: 'BcChartArea' },
  { tag: 'bc-chart-bar', group: 'charts', className: 'BcChartBar' },
  { tag: 'bc-chart-boxplot', group: 'charts', className: 'BcChartBoxplot' },
  { tag: 'bc-chart-candlestick', group: 'charts', className: 'BcChartCandlestick' },
  { tag: 'bc-chart-custom', group: 'charts', className: 'BcChartCustom' },
  { tag: 'bc-chart-funnel', group: 'charts', className: 'BcChartFunnel' },
  { tag: 'bc-chart-gauge', group: 'charts', className: 'BcChartGauge' },
  { tag: 'bc-chart-graph', group: 'charts', className: 'BcChartGraph' },
  { tag: 'bc-chart-heatmap', group: 'charts', className: 'BcChartHeatmap' },
  { tag: 'bc-chart-kpi', group: 'charts', className: 'BcChartKpi' },
  { tag: 'bc-chart-line', group: 'charts', className: 'BcChartLine' },
  { tag: 'bc-chart-mixed', group: 'charts', className: 'BcChartMixed' },
  { tag: 'bc-chart-parallel', group: 'charts', className: 'BcChartParallel' },
  { tag: 'bc-chart-pictorialbar', group: 'charts', className: 'BcChartPictorialbar' },
  { tag: 'bc-chart-pie', group: 'charts', className: 'BcChartPie' },
  { tag: 'bc-chart-pivot', group: 'charts', className: 'BcChartPivot' },
  { tag: 'bc-chart-polar', group: 'charts', className: 'BcChartPolar' },
  { tag: 'bc-chart-progress', group: 'charts', className: 'BcChartProgress' },
  { tag: 'bc-chart-radar', group: 'charts', className: 'BcChartRadar' },
  { tag: 'bc-chart-sankey', group: 'charts', className: 'BcChartSankey' },
  { tag: 'bc-chart-scatter', group: 'charts', className: 'BcChartScatter' },
  { tag: 'bc-chart-scorecard', group: 'charts', className: 'BcChartScorecard' },
  { tag: 'bc-chart-sunburst', group: 'charts', className: 'BcChartSunburst' },
  { tag: 'bc-chart-themeriver', group: 'charts', className: 'BcChartThemeriver' },
  { tag: 'bc-chart-tree', group: 'charts', className: 'BcChartTree' },
  { tag: 'bc-chart-treemap', group: 'charts', className: 'BcChartTreemap' },

  // fields (36)
  { tag: 'bc-field-barcode', group: 'fields', className: 'BcFieldBarcode' },
  { tag: 'bc-field-checkbox', group: 'fields', className: 'BcFieldCheckbox' },
  { tag: 'bc-field-code', group: 'fields', className: 'BcFieldCode' },
  { tag: 'bc-field-color', group: 'fields', className: 'BcFieldColor' },
  { tag: 'bc-field-currency', group: 'fields', className: 'BcFieldCurrency' },
  { tag: 'bc-field-date', group: 'fields', className: 'BcFieldDate' },
  { tag: 'bc-field-datetime', group: 'fields', className: 'BcFieldDatetime' },
  { tag: 'bc-field-decimal', group: 'fields', className: 'BcFieldDecimal' },
  { tag: 'bc-field-duration', group: 'fields', className: 'BcFieldDuration' },
  { tag: 'bc-field-dynlink', group: 'fields', className: 'BcFieldDynlink' },
  { tag: 'bc-field-file', group: 'fields', className: 'BcFieldFile' },
  { tag: 'bc-field-float', group: 'fields', className: 'BcFieldFloat' },
  { tag: 'bc-field-geo', group: 'fields', className: 'BcFieldGeo' },
  { tag: 'bc-field-html', group: 'fields', className: 'BcFieldHtml' },
  { tag: 'bc-field-image', group: 'fields', className: 'BcFieldImage' },
  { tag: 'bc-field-integer', group: 'fields', className: 'BcFieldInteger' },
  { tag: 'bc-field-json', group: 'fields', className: 'BcFieldJson' },
  { tag: 'bc-field-link', group: 'fields', className: 'BcFieldLink' },
  { tag: 'bc-field-markdown', group: 'fields', className: 'BcFieldMarkdown' },
  { tag: 'bc-field-morph', group: 'fields', className: 'BcFieldMorph' },
  { tag: 'bc-field-multicheck', group: 'fields', className: 'BcFieldMulticheck' },
  { tag: 'bc-field-password', group: 'fields', className: 'BcFieldPassword' },
  { tag: 'bc-field-percent', group: 'fields', className: 'BcFieldPercent' },
  { tag: 'bc-field-radio', group: 'fields', className: 'BcFieldRadio' },
  { tag: 'bc-field-rating', group: 'fields', className: 'BcFieldRating' },
  { tag: 'bc-field-richtext', group: 'fields', className: 'BcFieldRichtext' },
  { tag: 'bc-field-select', group: 'fields', className: 'BcFieldSelect' },
  { tag: 'bc-field-signature', group: 'fields', className: 'BcFieldSignature' },
  { tag: 'bc-field-smalltext', group: 'fields', className: 'BcFieldSmalltext' },
  { tag: 'bc-field-string', group: 'fields', className: 'BcFieldString' },
  { tag: 'bc-field-tableselect', group: 'fields', className: 'BcFieldTableselect' },
  { tag: 'bc-field-tags', group: 'fields', className: 'BcFieldTags' },
  { tag: 'bc-field-text', group: 'fields', className: 'BcFieldText' },
  { tag: 'bc-field-time', group: 'fields', className: 'BcFieldTime' },
  { tag: 'bc-field-toggle', group: 'fields', className: 'BcFieldToggle' },
  { tag: 'bc-lookup-modal', group: 'fields', className: 'BcLookupModal' },

  // dialogs (7)
  { tag: 'bc-dialog-alert', group: 'dialogs', className: 'BcDialogAlert' },
  { tag: 'bc-dialog-confirm', group: 'dialogs', className: 'BcDialogConfirm' },
  { tag: 'bc-dialog-modal', group: 'dialogs', className: 'BcDialogModal' },
  { tag: 'bc-dialog-prompt', group: 'dialogs', className: 'BcDialogPrompt' },
  { tag: 'bc-dialog-quickentry', group: 'dialogs', className: 'BcDialogQuickentry', dependencies: ['bc-field-string', 'bc-field-select'] },
  { tag: 'bc-dialog-wizard', group: 'dialogs', className: 'BcDialogWizard', dependencies: ['bc-button-box'] },
  { tag: 'bc-toast', group: 'dialogs', className: 'BcToast' },

  // layout (10)
  { tag: 'bc-button-box', group: 'layout', className: 'BcButtonBox' },
  { tag: 'bc-column', group: 'layout', className: 'BcColumn' },
  { tag: 'bc-header', group: 'layout', className: 'BcHeader' },
  { tag: 'bc-html-block', group: 'layout', className: 'BcHtmlBlock' },
  { tag: 'bc-row', group: 'layout', className: 'BcRow' },
  { tag: 'bc-section', group: 'layout', className: 'BcSection' },
  { tag: 'bc-separator', group: 'layout', className: 'BcSeparator' },
  { tag: 'bc-sheet', group: 'layout', className: 'BcSheet' },
  { tag: 'bc-tab', group: 'layout', className: 'BcTab' },
  { tag: 'bc-tabs', group: 'layout', className: 'BcTabs' },

  // views (10)
  { tag: 'bc-view-activity', group: 'views', className: 'BcViewActivity', dependencies: ['bc-timeline'] },
  { tag: 'bc-view-calendar', group: 'views', className: 'BcViewCalendar', dependencies: ['bc-dialog-modal'] },
  { tag: 'bc-view-editor', group: 'views', className: 'BcViewEditor' },
  { tag: 'bc-view-form', group: 'views', className: 'BcViewForm', dependencies: ['bc-field-string', 'bc-field-select', 'bc-field-date', 'bc-field-text', 'bc-field-integer', 'bc-field-float', 'bc-field-checkbox', 'bc-field-toggle', 'bc-field-radio', 'bc-field-tags', 'bc-field-file', 'bc-field-image', 'bc-field-html', 'bc-field-richtext', 'bc-field-json', 'bc-field-code', 'bc-field-color', 'bc-field-currency', 'bc-field-decimal', 'bc-field-duration', 'bc-field-percent', 'bc-field-time', 'bc-field-datetime', 'bc-field-markdown', 'bc-field-geo', 'bc-field-barcode', 'bc-field-signature', 'bc-field-multicheck', 'bc-field-rating', 'bc-field-link', 'bc-field-dynlink', 'bc-field-tableselect', 'bc-field-smalltext', 'bc-field-password', 'bc-field-morph', 'bc-lookup-modal', 'bc-button-box', 'bc-tabs', 'bc-tab'] },
  { tag: 'bc-view-gantt', group: 'views', className: 'BcViewGantt', dependencies: ['bc-progress'] },
  { tag: 'bc-view-kanban', group: 'views', className: 'BcViewKanban', dependencies: ['bc-badge'] },
  { tag: 'bc-view-list', group: 'views', className: 'BcViewList' },
  { tag: 'bc-view-map', group: 'views', className: 'BcViewMap' },
  { tag: 'bc-view-report', group: 'views', className: 'BcViewReport' },
  { tag: 'bc-view-tree', group: 'views', className: 'BcViewTree' },

  // datatable (5)
  { tag: 'bc-child-table', group: 'datatable', className: 'BcChildTable', dependencies: ['bc-field-string'] },
  { tag: 'bc-datatable', group: 'datatable', className: 'BcDatatable', dependencies: ['bc-field-string', 'bc-field-select', 'bc-filter-bar', 'bc-filter-panel'] },
  { tag: 'bc-filter-bar', group: 'datatable', className: 'BcFilterBar' },
  { tag: 'bc-filter-builder', group: 'datatable', className: 'BcFilterBuilder' },
  { tag: 'bc-filter-panel', group: 'datatable', className: 'BcFilterPanel' },

  // media (8)
  { tag: 'bc-viewer-audio', group: 'media', className: 'BcViewerAudio' },
  { tag: 'bc-viewer-document', group: 'media', className: 'BcViewerDocument' },
  { tag: 'bc-viewer-image', group: 'media', className: 'BcViewerImage' },
  { tag: 'bc-viewer-instagram', group: 'media', className: 'BcViewerInstagram' },
  { tag: 'bc-viewer-pdf', group: 'media', className: 'BcViewerPdf' },
  { tag: 'bc-viewer-tiktok', group: 'media', className: 'BcViewerTiktok' },
  { tag: 'bc-viewer-video', group: 'media', className: 'BcViewerVideo' },
  { tag: 'bc-viewer-youtube', group: 'media', className: 'BcViewerYoutube' },

  // print (3)
  { tag: 'bc-export', group: 'print', className: 'BcExport' },
  { tag: 'bc-print', group: 'print', className: 'BcPrint' },
  { tag: 'bc-report-link', group: 'print', className: 'BcReportLink' },

  // search (2)
  { tag: 'bc-favorites', group: 'search', className: 'BcFavorites' },
  { tag: 'bc-search', group: 'search', className: 'BcSearch', dependencies: ['bc-favorites'] },

  // social (3)
  { tag: 'bc-activity', group: 'social', className: 'BcActivity' },
  { tag: 'bc-chatter', group: 'social', className: 'BcChatter', dependencies: ['bc-timeline'] },
  { tag: 'bc-timeline', group: 'social', className: 'BcTimeline' },

  // widgets (12)
  { tag: 'bc-badge', group: 'widgets', className: 'BcWidgetBadge' },
  { tag: 'bc-copy', group: 'widgets', className: 'BcCopy' },
  { tag: 'bc-domain', group: 'widgets', className: 'BcDomain' },
  { tag: 'bc-email', group: 'widgets', className: 'BcEmail' },
  { tag: 'bc-handle', group: 'widgets', className: 'BcHandle' },
  { tag: 'bc-phone', group: 'widgets', className: 'BcPhone' },
  { tag: 'bc-placeholder', group: 'widgets', className: 'BcPlaceholder' },
  { tag: 'bc-priority', group: 'widgets', className: 'BcPriority' },
  { tag: 'bc-progress', group: 'widgets', className: 'BcProgress' },
  { tag: 'bc-statusbar', group: 'widgets', className: 'BcStatusbar' },
  { tag: 'bc-sync-status', group: 'widgets', className: 'BcSyncStatus' },
  { tag: 'bc-url', group: 'widgets', className: 'BcUrl' },
];

const GROUPS: string[] = ['charts', 'fields', 'dialogs', 'layout', 'views', 'datatable', 'media', 'print', 'search', 'social', 'widgets'];

const BY_TAG = new Map<string, ComponentMeta>();
const BY_GROUP = new Map<string, ComponentMeta[]>();

for (const c of COMPONENTS) {
  BY_TAG.set(c.tag, c);
  const list = BY_GROUP.get(c.group);
  if (list) list.push(c);
  else BY_GROUP.set(c.group, [c]);
}

function resolveDependencies(tags: Set<string>, visited: Set<string>): void {
  for (const tag of Array.from(tags)) {
    if (visited.has(tag)) continue;
    visited.add(tag);
    const meta = BY_TAG.get(tag);
    if (meta?.dependencies) {
      for (const dep of meta.dependencies) {
        tags.add(dep);
      }
    }
  }
}

export class Registry {
  static getAll(): ComponentMeta[] {
    return [...COMPONENTS];
  }

  static getByGroup(group: string): ComponentMeta[] {
    return BY_GROUP.get(group)?.slice() || [];
  }

  static getGroups(): string[] {
    return [...GROUPS];
  }

  static find(tag: string): ComponentMeta | undefined {
    return BY_TAG.get(tag);
  }

  static getCount(): { total: number; perGroup: Record<string, number> } {
    const perGroup: Record<string, number> = {};
    for (const g of GROUPS) {
      perGroup[g] = BY_GROUP.get(g)?.length || 0;
    }
    return { total: COMPONENTS.length, perGroup };
  }

  static select(selector: RegistrySelector): RegistryResult {
    const {
      includeGroups,
      excludeGroups,
      includeComponents,
      excludeComponents,
    } = selector;

    const selectedTags = new Set<string>();

    // Step 1: Start from includeGroups or all
    if (includeGroups && includeGroups.length > 0) {
      for (const group of includeGroups) {
        const components = BY_GROUP.get(group);
        if (components) {
          for (const c of components) selectedTags.add(c.tag);
        }
      }
    } else {
      for (const c of COMPONENTS) selectedTags.add(c.tag);
    }

    // Step 2: Add explicitly included components (union)
    if (includeComponents) {
      for (const tag of includeComponents) selectedTags.add(tag);
    }

    // Step 3: Resolve dependencies (auto-include)
    resolveDependencies(selectedTags, new Set());

    // Step 4: Remove excluded groups
    if (excludeGroups) {
      for (const group of excludeGroups) {
        const components = BY_GROUP.get(group);
        if (components) {
          for (const c of components) selectedTags.delete(c.tag);
        }
      }
    }

    // Step 5: Remove explicitly excluded components (exclude wins)
    if (excludeComponents) {
      for (const tag of excludeComponents) selectedTags.delete(tag);
    }

    const components = Array.from(selectedTags)
      .map(tag => BY_TAG.get(tag))
      .filter((c): c is ComponentMeta => c !== undefined);

    return {
      components,
      tags: Array.from(selectedTags),
      define: () => {
        if (typeof customElements === 'undefined') return;
        for (const tag of selectedTags) {
          if (!customElements.get(tag)) {
            console.warn(`[BitCode Registry] Component "${tag}" is selected but not loaded. Import it first.`);
          }
        }
      },
    };
  }
}
