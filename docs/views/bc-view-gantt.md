# bc-view-gantt

> Enterprise Gantt chart — tree hierarchy, drag-and-drop, dependencies, multi-scale timeline, undo/redo, markers, light/dark theme. Standalone or integrated with BitCode engine via shared data-fetcher.

## Quick Start

**Inline data (standalone):**

```html
<bc-view-gantt
  tasks='[{"id":"1","text":"Planning","start_date":"2026-01-01","duration":5,"progress":0.4}]'
  scales='[{"unit":"month","step":1},{"unit":"day","step":1}]'
/>
```

**Model-based (BitCode engine integration):**

```html
<bc-view-gantt model="project_task" field-mapping='{"text":"name","start_date":"planned_start","end_date":"planned_end"}' />
```

**Custom data fetcher (framework integration):**

```js
const gantt = document.querySelector('bc-view-gantt');
gantt.dataFetcher = async (params) => {
  const res = await fetch('/api/tasks');
  const data = await res.json();
  return { data: data.items, total: data.total };
};
```

## Props

| Prop | Type | Default | Description |
|------|------|---------|-------------|
| tasks | string (JSON) | '[]' | Task data — `GanttTask[]` |
| links | string (JSON) | '[]' | Dependency links — `GanttLink[]` |
| scales | string (JSON) | '[{"unit":"day","step":1}]' | Timeline scale levels — `GanttScale[]` |
| columns | string (JSON) | '[]' | Grid columns — `GanttColumn[]` |
| config | string (JSON) | '{}' | GanttConfig — behavior flags |
| templates | string (JSON) | '{}' | Template overrides — `GanttTemplates` |
| markers | string (JSON) | '[]' | Custom markers — `GanttMarker[]` |
| model | string | '' | Model name for BitCode engine data-fetch |
| fields | string (JSON) | '[]' | Field definitions for model mapping |
| data-source | string | '' | Remote data URL |
| fetch-headers | string | '' | Custom fetch headers (JSON) |
| refresh-interval | number | 0 | Auto-refresh interval in ms |
| height | string | '500px' | Chart height |
| readonly | boolean | false | Disable all drag interactions (component-level) |
| field-mapping | string (JSON) | '' | Custom field mapping: gantt field → model field |
| loading | boolean | false | Show loading overlay |
| view-title | string | '' | Chart title |
| dataFetcher | DataFetcher | undefined | Custom data fetcher function (JS property, not HTML attribute) |

### Field Mapping

When using `model` or `data-source`, the component maps API response fields to Gantt task fields. Three ways to configure:

**1. `field-mapping` prop (recommended):**

```html
<bc-view-gantt
  model="task"
  field-mapping='{"text":"title","start_date":"planned_start","end_date":"planned_end"}'
/>
```

Maps gantt field names to your model field names.

**2. `fields` prop (structured):**

```html
<bc-view-gantt
  model="task"
  fields='[{"gantt":"text","name":"title"},{"gantt":"start_date","name":"planned_start"}]'
/>
```

**3. Default mapping (fallback):**

| Gantt Field | Tries |
|-------------|-------|
| id | `id` |
| text | `name` → `title` → `text` |
| start_date | `start_date` → `start` |
| end_date | `end_date` → `end` |
| duration | `duration` |
| progress | `progress` |
| parent | `parent` → `parent_id` |
| type | `type` |
| color | `color` |

### Data Fetching (3-Layer Strategy)

Follows the same multi-layer pattern as `bc-datatable` — standalone-capable with seamless BitCode engine integration:

| Priority | Trigger | Method | Standalone | BitCode |
|----------|---------|--------|------------|---------|
| 1 | `dataFetcher` property set | Custom function (you control everything) | ✅ | ✅ |
| 2 | `data-source` URL provided | Raw `fetch()` + `BcSetup.getHeaders()` + `BcSetup.getBaseUrl()` + `lcBeforeFetch`/`lcAfterFetch` events | ✅ | ✅ |
| 3 | `model` name provided | Shared `fetchData()` (offline store + API client) → fallback direct `getApiClient()` | ❌ | ✅ |
| — | `tasks` prop only | No fetch, inline JSON data | ✅ | ✅ |

**Layer 1 — Custom dataFetcher (standalone + framework):**

```js
const gantt = document.querySelector('bc-view-gantt');
gantt.dataFetcher = async (params) => {
  const res = await fetch('/api/tasks?page=' + (params.page || 1));
  return { data: await res.json(), total: 100 };
};
```

**Layer 2 — dataSource URL (standalone + BitCode):**

```html
<bc-view-gantt
  data-source="/api/gantt-data"
  fetch-headers='{"X-Custom":"value"}'
/>
```

- Merges `BcSetup.getHeaders()` with custom `fetch-headers`
- Resolves relative URLs via `BcSetup.getBaseUrl()`
- Dispatches `lcBeforeFetch` (cancelable, can modify URL/headers) and `lcAfterFetch` (can transform response)
- Parses multiple response formats: raw array, `{tasks:[...]}`, `{data:[...]}`

**Layer 3 — model name (BitCode engine only):**

```html
<bc-view-gantt model="project_task" field-mapping='{"text":"name","start_date":"planned_start"}' />
```

- First tries shared `fetchData()` from `core/data-fetcher` (supports offline via `OfflineStore`, `normalizeResponse`, BcSetup integration)
- On failure, falls back to direct `getApiClient().list(model, {pageSize:500})` — guaranteed BitCode engine path
- Field mapping via `field-mapping` prop (see above)

### GanttTask

| Field | Type | Description |
|-------|------|-------------|
| id | string/number | Unique task ID |
| text | string | Task label |
| start_date | string | Start date (ISO or parseable) |
| duration | number | Duration in scale units (days default) |
| end_date | string | End date (alternative to duration) |
| progress | number | Progress 0–1 |
| parent | string/number | Parent task ID (tree hierarchy) |
| open | boolean | Expanded/collapsed state |
| type | string | Task type: 'task', 'milestone', 'project' |
| color | string | Bar color override |
| readonly | boolean | Prevent drag on this task |

### GanttLink

| Field | Type | Description |
|-------|------|-------------|
| id | string/number | Unique link ID |
| source | string/number | Source task ID |
| target | string/number | Target task ID |
| type | string | '0'=FS, '1'=SS, '2'=FF, '3'=SF |

### GanttScale

| Field | Type | Description |
|-------|------|-------------|
| unit | string | 'year', 'month', 'quarter', 'week', 'day', 'hour', 'minute' |
| step | number | Step size |
| format | string | Date format string (`%Y`, `%M`, `%d`, `%D`, `%H`, `%i`, `%Q`) |

### GanttColumn

| Field | Type | Description |
|-------|------|-------------|
| name | string | Field name |
| label | string | Column header |
| width | number/string | Column width (px) |
| tree | boolean | Show tree expand/collapse toggle |
| align | string | 'left', 'center', 'right' |
| hide | boolean | Hide column |

### GanttConfig

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| row_height | number | 32 | Row height in px |
| bar_height | number/'full' | 24 | Bar height in px |
| grid_width | number | — | Left grid panel width |
| scale_height | number | 40 | Scale header height |
| duration_unit | string | 'day' | 'minute','hour','day','week','month','year' |
| readonly | boolean | false | Disable drag (also available as component prop) |
| drag_move | boolean | true | Allow drag to move tasks |
| drag_resize | boolean | true | Allow drag to resize tasks |
| drag_progress | boolean | true | Allow drag to change progress |
| drag_links | boolean | true | Allow creating links by drag |
| show_progress | boolean | true | Show progress bars |
| show_links | boolean | true | Show dependency links |
| show_grid | boolean | true | Show left grid panel |
| show_chart | boolean | true | Show timeline area |
| show_markers | boolean | true | Show markers |
| show_task_cells | boolean | true | Show background cells |
| open_tree_initially | boolean | false | Expand all tree nodes on load |
| rtl | boolean | false | Right-to-left mode |
| undo | boolean | — | Enable undo stack |
| undo_steps | number | — | Max undo steps |

## Events

| Event | Payload | Description |
|-------|---------|-------------|
| lcGanttReady | void | Component initialized |
| lcGanttRendered | void | Chart rendered |
| lcGanttParsed | void | Data parsed |
| lcGanttTaskClick | `{id, task}` | Task clicked |
| lcGanttTaskDblClick | `{id, task}` | Task double-clicked |
| lcGanttTaskRowClick | `{id, task}` | Grid row clicked |
| lcGanttTaskDrag | `{id, mode, task, original, nativeEvent}` | Task being dragged |
| lcGanttBeforeTaskDrag | `{id, mode, nativeEvent}` | Before drag starts |
| lcGanttAfterTaskDrag | `{id, mode, nativeEvent}` | After drag completes |
| lcGanttBeforeTaskAdd | `{id, task}` | Before task added |
| lcGanttTaskAdd | `{id, task}` | After task added |
| lcGanttBeforeTaskUpdate | `{id, task}` | Before task updated |
| lcGanttTaskUpdate | `{id, task, oldTask}` | After task updated |
| lcGanttBeforeTaskDelete | `{id, task}` | Before task deleted |
| lcGanttTaskDelete | `{id, task}` | After task deleted |
| lcGanttTaskSelect | `{id, task}` | Task selected |
| lcGanttTaskUnselect | `{id}` | Task unselected |
| lcGanttTaskMove | `{id, parent, index}` | Task moved in tree |
| lcGanttTaskOpened | `{id}` | Task expanded |
| lcGanttTaskClosed | `{id}` | Task collapsed |
| lcGanttLinkClick | `{id, link}` | Link clicked |
| lcGanttLinkDblClick | `{id, link}` | Link double-clicked |
| lcGanttBeforeLinkAdd | `{id, link}` | Before link added |
| lcGanttLinkAdd | `{id, link}` | After link added |
| lcGanttBeforeLinkUpdate | `{id, link}` | Before link updated |
| lcGanttLinkUpdate | `{id, link}` | After link updated |
| lcGanttBeforeLinkDelete | `{id, link}` | Before link deleted |
| lcGanttLinkDelete | `{id, link}` | After link deleted |
| lcGanttLinkDrag | `{from, fromStart, to, toStart}` | Link being created by drag |
| lcGanttScaleClick | `{date, nativeEvent}` | Scale cell clicked |
| lcGanttScroll | `{left, top}` | Timeline scrolled |
| lcGanttColumnResize | `{index, column, width}` | Column resized |
| lcGanttRowResize | `{id, task, oldHeight, newHeight}` | Row resized |
| lcGanttContextMenu | `{taskId, linkId, nativeEvent}` | Right-click |
| lcGanttEmptyClick | `{nativeEvent}` | Click on empty area |
| lcGanttError | `{message}` | Error occurred |

## Methods

| Method | Returns | Description |
|--------|---------|-------------|
| setData({tasks, links}) | Promise\<void\> | Set tasks and links |
| parseData({tasks, links}) | Promise\<void\> | Alias for setData |
| addTask(task, parent?, index?) | Promise\<string\|number\> | Add task, returns ID |
| updateTask(id, data) | Promise\<void\> | Update task fields |
| deleteTask(id) | Promise\<void\> | Delete task + children |
| moveTask(id, index, parent?) | Promise\<void\> | Move task in tree |
| getTask(id) | Promise\<GanttTask\> | Get task by ID |
| getTasks() | Promise\<GanttTask[]\> | Get all tasks |
| getTaskCount() | Promise\<number\> | Total task count |
| getVisibleTaskCount() | Promise\<number\> | Visible task count |
| getLink(id) | Promise\<GanttLink\> | Get link by ID |
| getLinks() | Promise\<GanttLink[]\> | Get all links |
| getLinkCount() | Promise\<number\> | Total link count |
| addLink(link) | Promise\<string\|number\> | Add dependency link |
| updateLink(id, data) | Promise\<void\> | Update link fields |
| deleteLink(id) | Promise\<void\> | Delete link |
| selectTask(id) | Promise\<void\> | Select a task |
| unselectTask() | Promise\<void\> | Clear selection |
| getSelectedId() | Promise\<string\|number\|null\> | Get selected task ID |
| isSelected(id) | Promise\<boolean\> | Check if task is selected |
| open(id) | Promise\<void\> | Expand task children |
| close(id) | Promise\<void\> | Collapse task children |
| openAll() | Promise\<void\> | Expand all |
| closeAll() | Promise\<void\> | Collapse all |
| scrollTo(x?, y?) | Promise\<void\> | Scroll to position |
| showTask(id) | Promise\<void\> | Scroll task into view |
| showDate(date) | Promise\<void\> | Scroll to date |
| getScrollState() | Promise\<{left, top}\> | Get current scroll |
| refresh() | Promise\<void\> | Re-fetch or re-render |
| rerender() | Promise\<void\> | Force full re-render |
| clearAll() | Promise\<void\> | Remove all tasks, links, markers |
| getState() | Promise\<{selectedIndex, scrollTop, scrollLeft}\> | Get component state |
| addMarker(marker) | Promise\<string\|number\> | Add custom marker |
| deleteMarker(id) | Promise\<void\> | Delete marker |
| getMarker(id) | Promise\<GanttMarker\> | Get marker by ID |
| undo() | Promise\<boolean\> | Undo last action |
| redo() | Promise\<boolean\> | Redo last undone action |
| getUndoStack() | Promise\<Array\> | Get undo history |
| getRedoStack() | Promise\<Array\> | Get redo history |
| clearUndoStack() | Promise\<void\> | Clear undo history |
| clearRedoStack() | Promise\<void\> | Clear redo history |
| sortBy(field, desc?) | Promise\<void\> | Sort tasks by field |
| eachTask(callback) | Promise\<void\> | Iterate all tasks |
| serialize() | Promise\<{tasks, links}\> | Export all data |
| isTaskExists(id) | Promise\<boolean\> | Check if task exists |
| isTaskVisible(id) | Promise\<boolean\> | Check if task is visible |
| getChildren(id) | Promise\<GanttTask[]\> | Get child tasks |
| hasChild(id) | Promise\<number\> | Count children |
| getParent(id) | Promise\<string\|number\> | Get parent ID |
| getSiblings(id) | Promise\<GanttTask[]\> | Get sibling tasks |
| getNext(id) | Promise\<string\|number\|null\> | Next visible task |
| getPrev(id) | Promise\<string\|number\|null\> | Previous visible task |
| getTaskIndex(id) | Promise\<number\> | Flat index |
| getGlobalTaskIndex(id) | Promise\<number\> | Global index |
| calculateDuration({start_date, end_date}) | Promise\<number\> | Duration between dates |
| calculateEndDate({start_date, duration}) | Promise\<Date\> | End date from duration |
| setSizes() | Promise\<void\> | Force recalculate |
| expand() | Promise\<void\> | Fullscreen |
| collapse() | Promise\<void\> | Exit fullscreen |
| dateFromPos(pos) | Promise\<Date\|null\> | Date from pixel position |
| posFromDate(date) | Promise\<number\> | Pixel position from date |

## Data Format

```json
{
  "tasks": [
    {"id":"1","text":"Phase 1","start_date":"2026-01-01","duration":10,"progress":0.6,"open":true,"type":"project"},
    {"id":"2","text":"Task A","start_date":"2026-01-01","duration":5,"progress":0.8,"parent":"1"},
    {"id":"3","text":"Task B","start_date":"2026-01-06","duration":5,"progress":0.3,"parent":"1"},
    {"id":"4","text":"Milestone","start_date":"2026-01-11","duration":0,"type":"milestone"}
  ],
  "links": [
    {"id":"l1","source":"2","target":"3","type":"0"}
  ]
}
```

## Drag Modes

| Mode | Behavior |
|------|----------|
| move | Drag task bar to move dates |
| resize-left | Drag left edge to change start |
| resize-right | Drag right edge to change end |
| progress | Drag progress handle |

## Dependency Types

| Type | Code | Description |
|------|------|-------------|
| Finish-to-Start | '0' | Default — predecessor finishes before successor starts |
| Start-to-Start | '1' | Both start together |
| Finish-to-Finish | '2' | Both finish together |
| Start-to-Finish | '3' | Successor finishes when predecessor starts |

## Standalone vs Integrated

| Feature | Standalone | BitCode Engine |
|---------|-----------|----------------|
| **Data source** | Inline `tasks` prop or `data-source` URL | `model` name or `data-source` URL |
| **Custom fetch** | `dataFetcher` property | `dataFetcher` property |
| **Auth headers** | Manual via `fetch-headers` | Auto via `BcSetup.getHeaders()` |
| **Base URL** | Full URL in `data-source` | Auto via `BcSetup.getBaseUrl()` |
| **Offline mode** | No | Auto via `OfflineStore` (model only) |
| **API fallback** | N/A | `getApiClient()` direct fallback (model only) |
| **Field mapping** | Not needed | `field-mapping` or `fields` prop |
| **Response transform** | Manual | `BcSetup.responseTransformer` |
| **Fetch intercept** | `lcBeforeFetch`/`lcAfterFetch` events | Same events |
| **Error handling** | `lcGanttError` event | `lcGanttError` event + engine fallback |

## Theming

All colors use CSS custom properties. Supports `[data-bc-theme="dark"]` for dark mode via `BcSetup.setTheme('dark')`.

Key variables:
- `--bc-gantt-bg` — chart background
- `--bc-gantt-grid-bg` — grid panel background
- `--bc-gantt-task-bar-bg` — default task bar color
- `--bc-gantt-today-color` — today marker color
- `--bc-gantt-weekend-bg` — weekend column highlight
- `--bc-gantt-link` — dependency link color

See [theming](../theming.md), [data-fetching](../data-fetching.md).
