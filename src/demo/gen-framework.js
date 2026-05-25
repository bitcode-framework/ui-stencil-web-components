/**
 * Generate 17 framework example HTML files.
 * Each file contains <details> with tabs for Vanilla JS, React, Vue, Angular.
 * Run: node src/demo/gen-framework.js
 */
const fs = require('fs');
const path = require('path');

const SECTIONS_DIR = path.join(__dirname, 'sections');

// Component registry: tag → example code per framework
// Each entry has a "pick" (canonical usage example) showing key props
const registry = {
  // === FIELDS — TEXT ===
  'bc-field-string': { pick: '<bc-field-string name="username" label="Username" placeholder="Enter..." required clearable></bc-field-string>' },
  'bc-field-password': { pick: '<bc-field-password name="pw" label="Password" required min-length="8" show-reveal></bc-field-password>' },
  'bc-field-text': { pick: '<bc-field-text name="bio" label="Bio" rows="4" max-length="500" show-count></bc-field-text>' },
  'bc-field-smalltext': { pick: '<bc-field-smalltext name="note" label="Note" rows="2"></bc-field-smalltext>' },

  // === FIELDS — NUMBER ===
  'bc-field-integer': { pick: '<bc-field-integer name="qty" label="Quantity" min="0" max="999" step="1"></bc-field-integer>' },
  'bc-field-float': { pick: '<bc-field-float name="weight" label="Weight" precision="2" suffix-text="kg"></bc-field-float>' },
  'bc-field-decimal': { pick: '<bc-field-decimal name="price" label="Price" precision="2" prefix-text="$"></bc-field-decimal>' },
  'bc-field-currency': { pick: '<bc-field-currency name="amount" label="Amount" currency="USD"></bc-field-currency>' },
  'bc-field-percent': { pick: '<bc-field-percent name="discount" label="Discount" min="0" max="100"></bc-field-percent>' },

  // === FIELDS — DATETIME ===
  'bc-field-date': { pick: '<bc-field-date name="birthday" label="Birthday"></bc-field-date>' },
  'bc-field-time': { pick: '<bc-field-time name="start" label="Start Time"></bc-field-time>' },
  'bc-field-datetime': { pick: '<bc-field-datetime name="created" label="Created At"></bc-field-datetime>' },
  'bc-field-timerange': { pick: '<bc-field-timerange name="shift" label="Shift" start-label="From" end-label="To"></bc-field-timerange>' },
  'bc-field-daterange': { pick: '<bc-field-daterange name="period" label="Period" start-label="From" end-label="To"></bc-field-daterange>' },

  // === FIELDS — CHOICE ===
  'bc-field-boolean': { pick: '<bc-field-boolean name="active" label="Active"></bc-field-boolean>' },
  'bc-field-select': { pick: '<bc-field-select name="status" label="Status" options=\'[{"label":"Active","value":"active"},{"label":"Inactive","value":"inactive"}]\'></bc-field-select>' },
  'bc-field-radio': { pick: '<bc-field-radio name="gender" label="Gender" options=\'[{"label":"Male","value":"M"},{"label":"Female","value":"F"}]\'></bc-field-radio>' },
  'bc-field-checkbox': { pick: '<bc-field-checkbox name="colors" label="Colors" options=\'[{"label":"Red","value":"red"},{"label":"Blue","value":"blue"}]\'></bc-field-checkbox>' },
  'bc-field-rating': { pick: '<bc-field-rating name="stars" label="Rating" max="5"></bc-field-rating>' },
  'bc-field-color': { pick: '<bc-field-color name="theme" label="Theme Color" value="#4f46e5"></bc-field-color>' },
  'bc-field-slider': { pick: '<bc-field-slider name="volume" label="Volume" min="0" max="100" step="1"></bc-field-slider>' },
  'bc-field-toggle': { pick: '<bc-field-toggle name="dark" label="Dark Mode"></bc-field-toggle>' },

  // === FIELDS — RICH ===
  'bc-field-richtext': { pick: '<bc-field-richtext name="content" label="Content"></bc-field-richtext>' },
  'bc-field-code': { pick: '<bc-field-code name="script" label="Script" language="javascript"></bc-field-code>' },
  'bc-field-markdown': { pick: '<bc-field-markdown name="readme" label="README"></bc-field-markdown>' },
  'bc-field-json': { pick: '<bc-field-json name="config" label="Config"></bc-field-json>' },

  // === FIELDS — SPECIAL ===
  'bc-field-email': { pick: '<bc-field-email name="email" label="Email" required></bc-field-email>' },
  'bc-field-phone': { pick: '<bc-field-phone name="phone" label="Phone"></bc-field-phone>' },
  'bc-field-url': { pick: '<bc-field-url name="website" label="Website"></bc-field-url>' },
  'bc-field-file': { pick: '<bc-field-file name="avatar" label="Avatar" accept="image/*"></bc-field-file>' },
  'bc-field-image': { pick: '<bc-field-image name="photo" label="Photo"></bc-field-image>' },
  'bc-field-signature': { pick: '<bc-field-signature name="sig" label="Signature"></bc-field-signature>' },

  // === FIELDS — RELATIONAL ===
  'bc-field-many2one': { pick: '<bc-field-many2one name="partner" label="Partner" model="contact"></bc-field-many2one>' },
  'bc-field-one2many': { pick: '<bc-field-one2many name="lines" label="Order Lines" model="sale_line"></bc-field-one2many>' },
  'bc-field-many2many': { pick: '<bc-field-many2many name="tags" label="Tags" model="tag"></bc-field-many2many>' },
  'bc-field-link': { pick: '<bc-field-link name="ref" label="Reference" href="https://example.com" text="Open"></bc-field-link>' },
  'bc-field-dynlink': { pick: '<bc-field-dynlink name="doc" label="Document" pattern="/docs/{id}"></bc-field-dynlink>' },
  'bc-field-tableselect': { pick: '<bc-field-tableselect name="product" label="Product" model="product" columns=\'[{"field":"name","label":"Name"}]\'></bc-field-tableselect>' },
  'bc-field-lookup': { pick: '<bc-field-lookup name="city" label="City" model="city" display-field="name"></bc-field-lookup>' },

  // === CHARTS ===
  'bc-chart-bar': { pick: '<bc-chart-bar chart-title="Sales" data=\'[{"name":"Q1","value":120}]\'></bc-chart-bar>' },
  'bc-chart-line': { pick: '<bc-chart-line chart-title="Trend" data=\'[{"name":"Jan","value":100}]\' smooth></bc-chart-line>' },
  'bc-chart-area': { pick: '<bc-chart-area chart-title="Traffic" data=\'[{"name":"Mon","value":820}]\'></bc-chart-area>' },
  'bc-chart-pie': { pick: '<bc-chart-pie chart-title="Share" data=\'[{"name":"A","value":60}]\' donut></bc-chart-pie>' },
  'bc-chart-scatter': { pick: '<bc-chart-scatter chart-title="Correlation" data=\'[[10,8],[8,7]]\'></bc-chart-scatter>' },
  'bc-chart-radar': { pick: '<bc-chart-radar chart-title="Skills" data=\'[{"name":"Speed","value":80}]\'></bc-chart-radar>' },
  'bc-chart-funnel': { pick: '<bc-chart-funnel chart-title="Pipeline" data=\'[{"name":"Leads","value":100}]\'></bc-chart-funnel>' },
  'bc-chart-heatmap': { pick: '<bc-chart-heatmap chart-title="Activity" data=\'[[0,0,5],[0,1,8]]\'></bc-chart-heatmap>' },
  'bc-chart-treemap': { pick: '<bc-chart-treemap chart-title="Budget" data=\'[{"name":"Eng","value":50}]\'></bc-chart-treemap>' },
  'bc-chart-gauge': { pick: '<bc-chart-gauge chart-title="CPU" value="72" max="100"></bc-chart-gauge>' },
  'bc-chart-boxplot': { pick: '<bc-chart-boxplot chart-title="Dist" data=\'[[1,2,3,4,5]]\'></bc-chart-boxplot>' },
  'bc-chart-candlestick': { pick: '<bc-chart-candlestick chart-title="Stock" data=\'[["2024-01",100,120,90,110]]\'></bc-chart-candlestick>' },
  'bc-chart-sankey': { pick: '<bc-chart-sankey chart-title="Flow" data=\'...\'></bc-chart-sankey>' },
  'bc-chart-themeriver': { pick: '<bc-chart-themeriver chart-title="Trend" data=\'...\'></bc-chart-themeriver>' },
  'bc-chart-parallel': { pick: '<bc-chart-parallel chart-title="Multi" data=\'...\'></bc-chart-parallel>' },
  'bc-chart-graph': { pick: '<bc-chart-graph chart-title="Network" data=\'...\'></bc-chart-graph>' },
  'bc-chart-wordcloud': { pick: '<bc-chart-wordcloud chart-title="Tags" data=\'[{"name":"ERP","value":50}]\'></bc-chart-wordcloud>' },
  'bc-chart-combo': { pick: '<bc-chart-combo chart-title="Overview" data=\'...\'></bc-chart-combo>' },
  'bc-chart-funnel3d': { pick: '<bc-chart-funnel3d chart-title="3D Pipeline" data=\'...\'></bc-chart-funnel3d>' },
  'bc-chart-geo': { pick: '<bc-chart-geo chart-title="Regions" data=\'...\'></bc-chart-geo>' },
  'bc-chart-liquidfill': { pick: '<bc-chart-liquidfill chart-title="Progress" value="0.72"></bc-chart-liquidfill>' },

  // === DATATABLE ===
  'bc-datatable': { pick: '<bc-datatable model="contact" columns=\'[{"field":"name","label":"Name"},{"field":"email","label":"Email"}]\'></bc-datatable>' },

  // === DIALOGS ===
  'bc-dialog-modal': { pick: '<bc-dialog-modal dialog-title="Confirm" open>Content here</bc-dialog-modal>' },
  'bc-dialog-confirm': { pick: '<bc-dialog-confirm dialog-title="Delete?" message="Are you sure?"></bc-dialog-confirm>' },
  'bc-dialog-alert': { pick: '<bc-dialog-alert dialog-title="Info" message="Saved!"></bc-dialog-alert>' },
  'bc-dialog-prompt': { pick: '<bc-dialog-prompt dialog-title="Name" message="Enter name:" value=""></bc-dialog-prompt>' },
  'bc-dialog-toast': { pick: '<bc-dialog-toast variant="success" message="Done!"></bc-dialog-toast>' },
  'bc-dialog-quickentry': { pick: '<bc-dialog-quickentry dialog-title="New" model="contact" fields=\'["name","email"]\'></bc-dialog-quickentry>' },
  'bc-dialog-form': { pick: '<bc-dialog-form dialog-title="Edit" model="contact" fields=\'["name","email"]\'></bc-dialog-form>' },
  'bc-dialog-wizard': { pick: '<bc-dialog-wizard dialog-title="Setup" steps=\'[{"title":"Step 1"},{"title":"Step 2"}]\'></bc-dialog-wizard>' },

  // === LAYOUT ===
  'bc-layout-tabs': { pick: '<bc-layout-tabs tabs=\'[{"label":"Tab 1","id":"t1"},{"label":"Tab 2","id":"t2"}]\'></bc-layout-tabs>' },
  'bc-layout-accordion': { pick: '<bc-layout-accordion items=\'[{"title":"Section 1","content":"..."}]\'></bc-layout-accordion>' },
  'bc-layout-card': { pick: '<bc-layout-card card-title="Stats">Content</bc-layout-card>' },
  'bc-layout-panel': { pick: '<bc-layout-panel panel-title="Details">Content</bc-layout-panel>' },
  'bc-layout-grid': { pick: '<bc-layout-grid cols="3"><div>1</div><div>2</div><div>3</div></bc-layout-grid>' },
  'bc-layout-split': { pick: '<bc-layout-split direction="horizontal"><div>Left</div><div>Right</div></bc-layout-split>' },
  'bc-layout-sidebar': { pick: '<bc-layout-sidebar side="left" width="250px">Nav</bc-layout-sidebar>' },
  'bc-layout-toolbar': { pick: '<bc-layout-toolbar>Actions</bc-layout-toolbar>' },
  'bc-layout-stepper': { pick: '<bc-layout-stepper steps=\'[{"label":"Cart"},{"label":"Pay"},{"label":"Done"}]\' current="1"></bc-layout-stepper>' },
  'bc-layout-breadcrumb': { pick: '<bc-layout-breadcrumb items=\'[{"label":"Home","href":"/"},{"label":"Products"}]\'></bc-layout-breadcrumb>' },
  'bc-layout-drawer': { pick: '<bc-layout-drawer side="right" open>Drawer content</bc-layout-drawer>' },
  'bc-layout-stack': { pick: '<bc-layout-stack gap="1rem"><div>A</div><div>B</div></bc-layout-stack>' },

  // === MEDIA ===
  'bc-viewer-pdf': { pick: '<bc-viewer-pdf src="/files/doc.pdf"></bc-viewer-pdf>' },
  'bc-viewer-image': { pick: '<bc-viewer-image src="/files/photo.jpg" thumbnail></bc-viewer-image>' },
  'bc-viewer-document': { pick: '<bc-viewer-document src="/files/report.docx"></bc-viewer-document>' },
  'bc-viewer-youtube': { pick: '<bc-viewer-youtube video-id="dQw4w9WgXcQ"></bc-viewer-youtube>' },
  'bc-viewer-instagram': { pick: '<bc-viewer-instagram post-url="https://instagram.com/p/xxx"></bc-viewer-instagram>' },
  'bc-viewer-tiktok': { pick: '<bc-viewer-tiktok video-url="https://tiktok.com/@x/video/123"></bc-viewer-tiktok>' },
  'bc-viewer-video': { pick: '<bc-viewer-video src="/files/clip.mp4"></bc-viewer-video>' },
  'bc-viewer-audio': { pick: '<bc-viewer-audio src="/files/podcast.mp3"></bc-viewer-audio>' },

  // === VIEWS ===
  'bc-view-list': { pick: '<bc-view-list model="contact" fields=\'["name","email"]\'></bc-view-list>' },
  'bc-view-form': { pick: '<bc-view-form model="contact" fields=\'["name","email","phone"]\'></bc-view-form>' },
  'bc-view-kanban': { pick: '<bc-view-kanban model="lead" fields=\'["name","stage"]\'></bc-view-kanban>' },
  'bc-kanban-board': { pick: '<bc-kanban-board model="kb_card" group-by="stage" board-title="Project Board"></bc-kanban-board>' },
  'bc-view-calendar': { pick: '<bc-view-calendar model="event" fields=\'["name","start","end"]\'></bc-view-calendar>' },
  'bc-view-gantt': { pick: '<bc-view-gantt tasks=\'[{"id":"1","text":"Task 1","start":"2024-01-01","duration":5}]\'></bc-view-gantt>' },
  'bc-view-map': { pick: '<bc-view-map model="branch" fields=\'["name","lat","lng"]\'></bc-view-map>' },
  'bc-view-tree': { pick: '<bc-view-tree model="category" fields=\'["name","parent"]\'></bc-view-tree>' },
  'bc-view-report': { pick: '<bc-view-report model="sale" fields=\'["name","amount"]\'></bc-view-report>' },
  'bc-view-editor': { pick: '<bc-view-editor model-fields=\'[{"name":"name","type":"string"}]\'></bc-view-editor>' },
  'bc-view-activity': { pick: '<bc-view-activity model="contact" fields=\'["name","desc"]\'></bc-view-activity>' },

  // === WIDGETS (actual tags used in demo) ===
  'bc-badge': { pick: '<bc-badge value="Active" variant="success"></bc-badge>' },
  'bc-progress': { pick: '<bc-progress value="65" max="100"></bc-progress>' },
  'bc-priority': { pick: '<bc-priority value="3" label="High"></bc-priority>' },
  'bc-handle': { pick: '<bc-handle></bc-handle>' },
  'bc-copy': { pick: '<bc-copy value="https://example.com"></bc-copy>' },
  'bc-url': { pick: '<bc-url value="https://example.com"></bc-url>' },
  'bc-email': { pick: '<bc-email value="hello@example.com"></bc-email>' },
  'bc-phone': { pick: '<bc-phone value="+6281234567890"></bc-phone>' },
  'bc-domain': { pick: '<bc-domain value=\'[["status","=","active"]]\'></bc-domain>' },
  'bc-statusbar': { pick: '<bc-statusbar states=\'[{"label":"Draft","value":"draft"}]\' value="draft"></bc-statusbar>' },
  'bc-placeholder': { pick: '<bc-placeholder text="No data"></bc-placeholder>' },
  'bc-sync-status': { pick: '<bc-sync-status compact></bc-sync-status>' },
  'bc-toast': { pick: '<bc-toast variant="success" message="Saved!"></bc-toast>' },
  'bc-child-table': { pick: '<bc-child-table model="contact" fields=\'["name","email"]\'></bc-child-table>' },
  'bc-filter-bar': { pick: '<bc-filter-bar fields=\'[{"key":"name","label":"Name","type":"string"}]\'></bc-filter-bar>' },
  'bc-filter-panel': { pick: '<bc-filter-panel fields=\'[{"key":"name","label":"Name","type":"string"}]\'></bc-filter-panel>' },

  // === PRINT ===
  'bc-print-page': { pick: '<bc-print-page><h1>Invoice</h1><p>Content</p></bc-print-page>' },
  'bc-print-barcode': { pick: '<bc-print-barcode value="SKU-12345" format="CODE128"></bc-print-barcode>' },
  'bc-print-qrcode': { pick: '<bc-print-qrcode value="https://example.com" size="128"></bc-print-qrcode>' },

  // === SEARCH ===
  'bc-search-global': { pick: '<bc-search-global placeholder="Search..."></bc-search-global>' },
  'bc-search-filters': { pick: '<bc-search-filters fields=\'[{"key":"name","label":"Name","type":"string"}]\'></bc-search-filters>' },

  // === SOCIAL ===
  'bc-social-chatter': { pick: '<bc-social-chatter model="contact" record-id="1"></bc-social-chatter>' },
  'bc-social-timeline': { pick: '<bc-social-timeline model="contact" record-id="1"></bc-social-timeline>' },
  'bc-social-activity': { pick: '<bc-social-activity model="contact" record-id="1"></bc-social-activity>' },
  'bc-chatter': { pick: '<bc-chatter model="contact" record-id="1"></bc-chatter>' },
  'bc-timeline': { pick: '<bc-timeline model="contact" record-id="1"></bc-timeline>' },
  'bc-activity': { pick: '<bc-activity model="contact" record-id="1"></bc-activity>' },

  // === LAYOUT (actual tags used in demo) ===
  'bc-section': { pick: '<bc-section title="Section Title">Content</bc-section>' },
  'bc-tabs': { pick: '<bc-tabs><bc-tab label="Tab 1">Content 1</bc-tab><bc-tab label="Tab 2">Content 2</bc-tab></bc-tabs>' },
  'bc-tab': { pick: '<bc-tab label="Tab Content">Panel content here</bc-tab>' },
  'bc-row': { pick: '<bc-row gap="1rem"><div>Col 1</div><div>Col 2</div></bc-row>' },
  'bc-column': { pick: '<bc-column width="50%">Half-width column</bc-column>' },
  'bc-header': { pick: '<bc-header title="Page Title" subtitle="Description"></bc-header>' },
  'bc-sheet': { pick: '<bc-sheet title="Card Title">Card content</bc-sheet>' },
  'bc-separator': { pick: '<bc-separator></bc-separator>' },
  'bc-html-block': { pick: '<bc-html-block content="<h2>Title</h2><p>Text</p>"></bc-html-block>' },
  'bc-button-box': { pick: '<bc-button-box><button>Action</button></bc-button-box>' },

  // === PRINT (actual tags) ===
  'bc-print': { pick: '<bc-print><h1>Invoice</h1><p>Content to print</p></bc-print>' },
  'bc-export': { pick: '<bc-export model="contact" format="csv"></bc-export>' },
  'bc-report-link': { pick: '<bc-report-link report="sales-summary" title="Sales Report"></bc-report-link>' },

  // === SEARCH (actual tags) ===
  'bc-search': { pick: '<bc-search placeholder="Search..." model="contact"></bc-search>' },
  'bc-favorites': { pick: '<bc-favorites model="contact"></bc-favorites>' },

  // === EXTRA WIDGETS ===
  'bc-filter-builder': { pick: '<bc-filter-builder fields=\'[{"key":"name","label":"Name","type":"string"}]\'></bc-filter-builder>' },
};

// Extract component tags from a section file
function extractTags(html) {
  const re = /<([a-z][a-z0-9-]+)/g;
  const tags = new Set();
  let m;
  while ((m = re.exec(html)) !== null) {
    const tag = m[1];
    if (tag.startsWith('bc-') && registry[tag]) {
      tags.add(tag);
    }
  }
  return [...tags];
}

// Convert HTML pick to JSX (React)
function htmlToJsx(html) {
  return html
    .replace(/\/>/g, '/>')  // already self-closing
    .replace(/ ([a-z-]+)=/, (m, attr) => {
      // Convert kebab to camelCase for React
      const parts = attr.split('-');
      if (parts.length > 1) {
        return ' ' + parts[0] + parts.slice(1).map(p => p.charAt(0).toUpperCase() + p.slice(1)).join('') + '=';
      }
      return m;
    })
    .replace(/='([^']*)'/g, (m, v) => {
      // Try parse as JSON for arrays/objects
      try { JSON.parse(v); return '={' + v + '}'; } catch { return '="' + v + '"'; }
    })
    .replace(/ ([a-z]+)>/g, (m, attr) => ' ' + attr + '={true}>');
}

// Escape for embedding code inside <pre><code> in HTML
function esc(s) {
  return s
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;');
}

// Framework code generators for a set of components
function genVanilla(tags) {
  return tags.map(tag => {
    const pick = registry[tag].pick;
    return `<!-- ${tag} -->\n${pick}`;
  }).join('\n\n');
}

function genReact(tags) {
  return `import '${esc(tags.map(t => '@bitcode-framework/ui-components/' + t).join(', '))}';
// All bc-* tags work as JSX — they're standard Web Components
// Set props via attributes or ref

export default function Demo() {
  return (
    <>
${tags.map(tag => {
      const pick = registry[tag].pick;
      const jsx = htmlToJsx(pick);
      return `      {/* ${tag} */}\n      ${jsx}`;
    }).join('\n\n')}
    </>
  );
}`;
}

function genVue(tags) {
  return `<template>
${tags.map(tag => {
    const pick = registry[tag].pick;
    return `  <!-- ${tag} -->\n  ${pick}`;
  }).join('\n\n')}
</template>

<script setup>
// No import needed — bc-* components are registered globally
// via '@bitcode-framework/ui-components' loader in main.ts
</script>`;
}

function genAngular(tags) {
  return `<!-- app-demo.component.html -->
<!-- Add CUSTOM_ELEMENTS_SCHEMA to your module -->
${tags.map(tag => {
    const pick = registry[tag].pick;
    return `<!-- ${tag} -->\n${pick}`;
  }).join('\n\n')}`;
}

// Build one framework file per section
const sectionFiles = fs.readdirSync(SECTIONS_DIR).filter(f => f.endsWith('.html') && !f.includes('-framework'));

let written = 0;
for (const file of sectionFiles) {
  const sectionHtml = fs.readFileSync(path.join(SECTIONS_DIR, file), 'utf8');
  const tags = extractTags(sectionHtml);
  if (tags.length === 0) {
    console.log('SKIP:', file, '- no registered bc-* components');
    continue;
  }

  const vanilla = genVanilla(tags);
  const react = genReact(tags);
  const vue = genVue(tags);
  const angular = genAngular(tags);

  const fwHtml = `<div class="framework-examples">
  <details class="fw-details">
    <summary class="fw-summary">📦 Framework Examples (${tags.length} components)</summary>
    <div class="fw-tabs">
      <button class="fw-tab active" onclick="showFwTab(this, 'vanilla')">Vanilla JS</button>
      <button class="fw-tab" onclick="showFwTab(this, 'react')">React</button>
      <button class="fw-tab" onclick="showFwTab(this, 'vue')">Vue</button>
      <button class="fw-tab" onclick="showFwTab(this, 'angular')">Angular</button>
    </div>
    <div class="fw-content">
      <div class="fw-panel active" data-fw="vanilla">
        <pre><code>&lt;!-- ${tags.length === 1 ? tags[0] : tags.length + ' components'} — Vanilla HTML --&gt;
${esc(vanilla)}</code></pre>
      </div>
      <div class="fw-panel" data-fw="react">
        <pre><code>${esc(react)}</code></pre>
      </div>
      <div class="fw-panel" data-fw="vue">
        <pre><code>${esc(vue)}</code></pre>
      </div>
      <div class="fw-panel" data-fw="angular">
        <pre><code>${esc(angular)}</code></pre>
      </div>
    </div>
  </details>
</div>
`;

  const baseName = file.replace('.html', '');
  const fwPath = path.join(SECTIONS_DIR, baseName + '-framework.html');
  fs.writeFileSync(fwPath, fwHtml, 'utf8');
  console.log('OK:', baseName + '-framework.html', '(' + tags.length + ' components:', tags.slice(0, 3).join(', ') + (tags.length > 3 ? ', ...' : '') + ')');
  written++;
}

console.log('\nTotal:', written, 'framework files written');
