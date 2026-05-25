/**
 * Build Demo — assembles shell.html + section files + framework files → www/demo/index.html
 *
 * Reads:
 *   src/demo/shell.html          (HTML skeleton with <!--SECTIONS--> placeholder)
 *   src/demo/sections/*.html      (component demo sections)
 *   src/demo/sections/*-framework.html (framework code examples)
 *   src/demo/demo-mock-api.js     (mock API interceptor)
 *
 * Writes:
 *   www/demo/index.html           (assembled single-file demo)
 *   www/demo/demo-mock-api.js     (copied)
 */

const fs = require('fs');
const path = require('path');

const ROOT = path.resolve(__dirname, '..', '..');
const SRC_DIR = path.join(__dirname);
const SECTIONS_DIR = path.join(SRC_DIR, 'sections');
const OUT_DIR = path.join(ROOT, 'www', 'demo');

// Section files in display order
const SECTION_ORDER = [
  'fields-text',
  'fields-number',
  'fields-datetime',
  'fields-choice',
  'fields-rich',
  'fields-special',
  'fields-relational',
  'charts',
  'datatable',
  'dialogs',
  'layout',
  'media',
  'views',
  'widgets',
  'print',
  'search',
  'social',
];

function build() {
  console.log('Building demo from modular sections...');

  // Read shell
  const shellPath = path.join(SRC_DIR, 'shell.html');
  if (!fs.existsSync(shellPath)) {
    console.error('ERROR: shell.html not found at', shellPath);
    process.exit(1);
  }
  let shell = fs.readFileSync(shellPath, 'utf8');

  // Assemble sections
  let sectionsHtml = '';

  for (const name of SECTION_ORDER) {
    const sectionFile = path.join(SECTIONS_DIR, `${name}.html`);
    const frameworkFile = path.join(SECTIONS_DIR, `${name}-framework.html`);

    if (!fs.existsSync(sectionFile)) {
      console.warn(`  SKIP: ${name}.html not found`);
      continue;
    }

    // Read section content
    const sectionContent = fs.readFileSync(sectionFile, 'utf8').trim();
    sectionsHtml += '\n  ' + sectionContent + '\n';

    // Read framework examples (optional)
    if (fs.existsSync(frameworkFile)) {
      const fwContent = fs.readFileSync(frameworkFile, 'utf8').trim();
      sectionsHtml += '\n  ' + fwContent + '\n';
    }
  }

  // Inject sections into shell
  const output = shell.replace('<!--SECTIONS-->', sectionsHtml);

  // Ensure output directory exists
  if (!fs.existsSync(OUT_DIR)) {
    fs.mkdirSync(OUT_DIR, { recursive: true });
  }

  // Write assembled index.html
  fs.writeFileSync(path.join(OUT_DIR, 'index.html'), output, 'utf8');
  console.log(`  ✓ www/demo/index.html assembled (${SECTION_ORDER.length} sections)`);

  // Copy mock API
  const mockApiSrc = path.join(SRC_DIR, 'demo-mock-api.js');
  if (fs.existsSync(mockApiSrc)) {
    fs.copyFileSync(mockApiSrc, path.join(OUT_DIR, 'demo-mock-api.js'));
    console.log('  ✓ demo-mock-api.js copied');
  }

  console.log('Demo build complete.');
}

build();
