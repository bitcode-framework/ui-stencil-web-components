import { BcSetup } from './bc-setup';

export type TemplateResolverFn = (key: string) => string | null;

interface ResolverEntry {
  fn: TemplateResolverFn;
  builtIn: boolean;
}

const resolvers = new Map<string, ResolverEntry>();

function resolveDomSelector(expr: string): string | null {
  if (expr.startsWith('/')) {
    try {
      const result = document.evaluate(
        expr,
        document,
        null,
        XPathResult.STRING_TYPE,
        null,
      );
      return result.stringValue || null;
    } catch {
      return null;
    }
  }

  let attrSuffix: string | undefined;
  let selector = expr;

  const atIdx = expr.lastIndexOf('@');
  if (atIdx > 0) {
    const candidateSel = expr.substring(0, atIdx);
    const candidateAttr = expr.substring(atIdx + 1);
    if (/^[\w-]+$/.test(candidateAttr)) {
      const openBrackets = (candidateSel.match(/\[/g) || []).length;
      const closeBrackets = (candidateSel.match(/\]/g) || []).length;
      if (openBrackets === closeBrackets) {
        selector = candidateSel;
        attrSuffix = candidateAttr;
      }
    }
  }

  try {
    const el = document.querySelector(selector);
    if (!el) return null;

    if (attrSuffix) {
      return el.getAttribute(attrSuffix) ?? null;
    }

    if (
      el instanceof HTMLInputElement ||
      el instanceof HTMLSelectElement ||
      el instanceof HTMLTextAreaElement
    ) {
      return el.value;
    }
    if (el instanceof HTMLMetaElement) {
      return el.content;
    }
    return el.textContent?.trim() ?? null;
  } catch {
    return null;
  }
}

function registerBuiltIn(prefix: string, fn: TemplateResolverFn): void {
  resolvers.set(prefix, { fn, builtIn: true });
}

registerBuiltIn('ls', (key) => {
  try { return localStorage.getItem(key); } catch { return null; }
});
registerBuiltIn('ss', (key) => {
  try { return sessionStorage.getItem(key); } catch { return null; }
});
registerBuiltIn('dom', (key) => resolveDomSelector(key));
registerBuiltIn('meta', (name) => {
  try {
    const el = document.querySelector(`meta[name="${CSS.escape(name)}"]`);
    return el?.getAttribute('content') ?? null;
  } catch { return null; }
});
registerBuiltIn('bc', (key) => {
  const cfg = BcSetup.getConfig() as Record<string, unknown>;
  return (cfg[key] as string) ?? null;
});

const TEMPLATE_RE = /\$\{(\w+):([^}]+)\}/g;

export function resolveTemplate(template: string): string {
  if (typeof template !== 'string' || !template.includes('${')) return template;

  return template.replace(TEMPLATE_RE, (_match, prefix: string, key: string) => {
    const entry = resolvers.get(prefix);
    if (!entry) return '';
    try { return entry.fn(key) ?? ''; }
    catch { return ''; }
  });
}

export function registerResolver(prefix: string, fn: TemplateResolverFn): void {
  if (!/^\w+$/.test(prefix)) {
    throw new Error(`[resolveTemplate] Invalid prefix "${prefix}". Use alphanumeric only.`);
  }
  const existing = resolvers.get(prefix);
  if (existing?.builtIn) {
    throw new Error(`[resolveTemplate] Cannot override built-in resolver "${prefix}".`);
  }
  resolvers.set(prefix, { fn, builtIn: false });
}

export function getRegisteredPrefixes(): string[] {
  return Array.from(resolvers.keys());
}
