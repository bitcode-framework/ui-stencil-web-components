/**
 * Split monolithic index.html into modular section files.
 * Run once: node src/demo/split-sections.js
 */
const fs = require('fs');
const path = require('path');

const htmlFile = path.join(__dirname, 'index.html');
const sectionsDir = path.join(__dirname, 'sections');
const html = fs.readFileSync(htmlFile, 'utf8');

if (!fs.existsSync(sectionsDir)) fs.mkdirSync(sectionsDir, { recursive: true });

const groups = [
  { name: 'fields-text', start: '<!-- FIELDS \u2014 TEXT -->', end: '<!-- FIELDS \u2014 NUMBER -->' },
  { name: 'fields-number', start: '<!-- FIELDS \u2014 NUMBER -->', end: '<!-- FIELDS \u2014 DATE/TIME -->' },
  { name: 'fields-datetime', start: '<!-- FIELDS \u2014 DATE/TIME -->', end: '<!-- FIELDS \u2014 CHOICE -->' },
  { name: 'fields-choice', start: '<!-- FIELDS \u2014 CHOICE -->', end: '<!-- FIELDS \u2014 RICH -->' },
  { name: 'fields-rich', start: '<!-- FIELDS \u2014 RICH -->', end: '<!-- FIELDS \u2014 SPECIAL -->' },
  { name: 'fields-special', start: '<!-- FIELDS \u2014 SPECIAL -->', end: '<!-- FIELDS \u2014 RELATIONAL -->' },
  { name: 'fields-relational', start: '<!-- FIELDS \u2014 RELATIONAL -->', end: '<div class="group-heading">Charts</div>' },
  { name: 'charts', start: '<div class="group-heading">Charts</div>', end: '<div class="group-heading">Datatable</div>' },
  { name: 'datatable', start: '<div class="group-heading">Datatable</div>', end: '<div class="group-heading">Dialogs</div>' },
  { name: 'dialogs', start: '<div class="group-heading">Dialogs</div>', end: '<div class="group-heading">Layout</div>' },
  { name: 'layout', start: '<div class="group-heading">Layout</div>', end: '<div class="group-heading">Media Viewers</div>' },
  { name: 'media', start: '<div class="group-heading">Media Viewers</div>', end: '<div class="group-heading">Views</div>' },
  { name: 'views', start: '<div class="group-heading">Views</div>', end: '<div class="group-heading">Widgets</div>' },
  { name: 'widgets', start: '<div class="group-heading">Widgets</div>', end: '<div class="group-heading">Print</div>' },
  { name: 'print', start: '<div class="group-heading">Print</div>', end: '<div class="group-heading">Search</div>' },
  { name: 'search', start: '<div class="group-heading">Search</div>', end: '<div class="group-heading">Social</div>' },
  { name: 'social', start: '<div class="group-heading">Social</div>', end: null },
];

let written = 0;
for (const g of groups) {
  const si = html.indexOf(g.start);
  if (si === -1) {
    console.log('SKIP:', g.name, '- start not found:', g.start.substring(0, 60));
    continue;
  }
  let ei;
  if (g.end) {
    ei = html.indexOf(g.end, si + g.start.length);
    if (ei === -1) {
      console.log('SKIP:', g.name, '- end not found:', g.end.substring(0, 60));
      continue;
    }
  } else {
    ei = html.indexOf('</main>', si);
    if (ei === -1) ei = html.length;
  }
  const section = html.substring(si, ei).trim();
  const fp = path.join(sectionsDir, g.name + '.html');
  fs.writeFileSync(fp, section + '\n', 'utf8');
  console.log('OK:', g.name, '(' + section.split('\n').length + ' lines)');
  written++;
}
console.log('\nTotal:', written, 'section files written');
