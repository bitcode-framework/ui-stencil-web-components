#!/usr/bin/env node
// generate-exports.mjs — Post-build: generates group barrel files, CSS barrels, full.js, and package.json exports
// Run AFTER stencil build: node scripts/generate-exports.mjs
// Or via: npm run generate (which runs after stencil build automatically)

import { readdir, writeFile, mkdir, readFile, stat, copyFile } from 'fs/promises';
import { join, dirname } from 'path';
import { fileURLToPath } from 'url';

const __dirname = dirname(fileURLToPath(import.meta.url));
const ROOT = join(__dirname, '..');
const DIST_DIR = join(ROOT, 'dist');
const DIST_COMPONENTS = join(DIST_DIR, 'components');
const DIST_COLLECTION = join(DIST_DIR, 'collection', 'components');
const SRC_COMPONENTS = join(ROOT, 'src', 'components');
const PKG_PATH = join(ROOT, 'package.json');

const GROUPS = ['charts', 'fields', 'dialogs', 'layout', 'views', 'datatable', 'media', 'print', 'search', 'social', 'widgets'];

function toClassName(tag) {
  return tag
    .split('-')
    .map(part => part.charAt(0).toUpperCase() + part.slice(1))
    .join('');
}

// Get components from dist output (compiled) — scan for .js files matching bc-*
async function getCompiledComponents(group) {
  try {
    const entries = await readdir(DIST_COMPONENTS);
    // Find all bc-*.js files, determine group from src directory structure
    const groupDir = join(SRC_COMPONENTS, group);
    const srcEntries = await readdir(groupDir, { withFileTypes: true });
    const componentDirs = srcEntries
      .filter(e => e.isDirectory() && e.name.startsWith('bc-'))
      .map(e => e.name)
      .sort();
    // Verify each exists in dist
    return componentDirs.filter(name => {
      try { return true; } catch { return false; }
    });
  } catch {
    return [];
  }
}

// Generate JS barrel for a group — references dist/components/bc-xxx.js
function generateGroupBarrel(group, components) {
  const lines = [
    '// Auto-generated barrel file — do not edit manually.',
    '// Run: node scripts/generate-exports.mjs',
    '',
  ];
  // Re-export component classes and defineCustomElement from compiled output
  for (const comp of components) {
    const className = toClassName(comp);
    lines.push(`export { ${className}, defineCustomElement as define${className} } from './${comp}.js';`);
  }
  lines.push('');
  // Group define function
  lines.push(`export async function define${toClassName(group)}(): Promise<void> {`);
  lines.push(`  const modules = await Promise.all([`);
  for (const comp of components) {
    lines.push(`    import('./${comp}.js'),`);
  }
  lines.push(`  ]);`);
  lines.push(`  modules.forEach(m => { if (m.defineCustomElement) m.defineCustomElement(); });`);
  lines.push('}');
  return lines.join('\n');
}

// Generate CSS barrel for a group
function generateGroupCssBarrel(group, components) {
  const lines = [
    '/* Auto-generated CSS barrel — do not edit manually. */',
    '/* Run: node scripts/generate-exports.mjs */',
    '',
  ];
  // CSS files live in collection output
  for (const comp of components) {
    lines.push(`@import './${comp}/${comp}.css';`);
  }
  return lines.join('\n');
}

// Generate full.js — defines all components
function generateFullJs(groupData) {
  const lines = [
    '// Auto-generated full bundle — do not edit manually.',
    '// Run: node scripts/generate-exports.mjs',
    '',
    '// Re-export Registry + types from main index',
    `export { Registry } from './index.js';`,
    `export type {} from './types/full.js';`,
    '',
  ];
  // Re-export group define functions
  for (const { group } of groupData) {
    const className = toClassName(group);
    lines.push(`export { define${className} } from './components/${group}/index.js';`);
  }
  lines.push('');
  lines.push('export async function defineAll(): Promise<void> {');
  lines.push('  await Promise.all([');
  for (const { group } of groupData) {
    const className = toClassName(group);
    lines.push(`    import('./components/${group}/index.js').then(m => m.define${className}()),`);
  }
  lines.push('  ]);');
  lines.push('}');
  return lines.join('\n');
}

// Generate full.d.ts
function generateFullDts(groupData) {
  const lines = [
    '// Auto-generated type declarations — do not edit manually.',
    '',
    "export { Registry } from './index.js';",
    "export type { ComponentMeta, RegistrySelector, RegistryResult } from './registry.js';",
    '',
  ];
  for (const { group } of groupData) {
    const className = toClassName(group);
    lines.push(`export declare function define${className}(): Promise<void>;`);
  }
  lines.push('');
  lines.push('export declare function defineAll(): Promise<void>;');
  return lines.join('\n');
}

async function main() {
  // Verify dist exists
  try {
    await stat(DIST_COMPONENTS);
  } catch {
    console.error('ERROR: dist/components/ not found. Run "stencil build" first.');
    process.exit(1);
  }

  console.log('Generating post-build barrel files in dist/...');

  const groupData = [];
  for (const group of GROUPS) {
    const components = await getCompiledComponents(group);
    groupData.push({ group, components });

    // Create group directory in dist/components/
    const groupDir = join(DIST_COMPONENTS, group);
    await mkdir(groupDir, { recursive: true });

    // JS barrel → dist/components/{group}/index.js
    const jsContent = generateGroupBarrel(group, components);
    await writeFile(join(groupDir, 'index.js'), jsContent, 'utf-8');

    // CSS barrel → dist/components/{group}/{group}.css (if collection output exists)
    const collGroupDir = join(DIST_COLLECTION, group);
    try {
      await mkdir(collGroupDir, { recursive: true });
      const cssContent = generateGroupCssBarrel(group, components);
      await writeFile(join(collGroupDir, `${group}.css`), cssContent, 'utf-8');
    } catch {
      // Collection dir may not exist — skip CSS barrel
    }

    console.log(`  ✓ ${group}/index.js (${components.length} components)`);
  }

  // Generate full.js → dist/full.js
  const fullJs = generateFullJs(groupData);
  await writeFile(join(DIST_DIR, 'full.js'), fullJs, 'utf-8');
  console.log('  ✓ full.js');

  // Generate full.d.ts → dist/types/full.d.ts
  const typesDir = join(DIST_DIR, 'types');
  await mkdir(typesDir, { recursive: true });
  const fullDts = generateFullDts(groupData);
  await writeFile(join(typesDir, 'full.d.ts'), fullDts, 'utf-8');
  console.log('  ✓ types/full.d.ts');

  // Copy theme CSS files → dist/themes/
  const themesDir = join(DIST_DIR, 'themes');
  await mkdir(themesDir, { recursive: true });
  const themesSrc = join(ROOT, 'src', 'global', 'themes');
  for (const theme of ['light.css', 'dark.css']) {
    const src = join(themesSrc, theme);
    const dest = join(themesDir, theme);
    try {
      await copyFile(src, dest);
      console.log(`  ✓ themes/${theme}`);
    } catch {
      console.warn(`  ⚠ themes/${theme} not found in src`);
    }
  }

  // Update package.json exports
  console.log('\nUpdating package.json exports...');
  const pkg = JSON.parse(await readFile(PKG_PATH, 'utf-8'));

  const exports = {
    '.': { 'import': './dist/index.js', 'require': './dist/index.cjs.js', 'types': './dist/types/index.d.ts' },
    './utils': { 'import': './dist/components/index.js', 'types': './dist/components/index.d.ts' },
    './loader': { 'import': './loader/index.js', 'require': './loader/index.cjs.js' },
    './registry': { 'import': './dist/collection/registry.js', 'types': './dist/types/registry.d.ts' },
    './full': { 'import': './dist/full.js', 'types': './dist/types/full.d.ts' },
  };

  for (const { group, components } of groupData) {
    // Per-group JS barrel
    exports[`./${group}`] = {
      'import': `./dist/components/${group}/index.js`,
    };
    // Per-group CSS
    exports[`./css/${group}`] = {
      'import': `./dist/collection/components/${group}/${group}.css`,
    };
    // Per-component CSS
    for (const comp of components) {
      exports[`./css/${comp}`] = {
        'import': `./dist/collection/components/${group}/${comp}/${comp}.css`,
      };
    }
  }

  // Themes
  exports['./themes/light'] = { 'import': './dist/themes/light.css' };
  exports['./themes/dark'] = { 'import': './dist/themes/dark.css' };

  // Sort exports
  const sortedExports = {};
  for (const key of Object.keys(exports).sort((a, b) => {
    const priority = (k) => {
      if (k === '.') return 0;
      if (k === './utils') return 1;
      if (k === './loader') return 2;
      if (k === './registry') return 3;
      if (k === './full') return 4;
      if (k.startsWith('./css/')) return 6;
      if (k.startsWith('./themes/')) return 7;
      return 5;
    };
    const pa = priority(a), pb = priority(b);
    if (pa !== pb) return pa - pb;
    return a.localeCompare(b);
  })) {
    sortedExports[key] = exports[key];
  }
  pkg.exports = sortedExports;

  await writeFile(PKG_PATH, JSON.stringify(pkg, null, 2) + '\n', 'utf-8');
  console.log(`  ✓ ${Object.keys(exports).length} export paths`);

  console.log('\nDone.');
}

main().catch(err => {
  console.error('Error:', err);
  process.exit(1);
});
