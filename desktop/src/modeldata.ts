import type { Change, Checklist, Definition, Diagnostic, ModelView, Package } from './api';
import type { Tone } from './status';

// Indexes over the Model area's document that the views share.
export interface Index {
  m: ModelView;
  defs: Map<string, Definition>;
  pkgs: Map<string, Package>;
  // The Packages depth first from Root, siblings in name order.
  tree: Package[];
  checklists: Map<string, Checklist>;
  working: Set<string>;
}

export function index(m: ModelView): Index {
  const defs = new Map(m.definitions.map((d) => [d.id, d]));
  const pkgs = new Map(m.packages.map((p) => [p.path, p]));
  const tree: Package[] = [];
  const walk = (path: string) => {
    const p = pkgs.get(path);
    if (!p) {
      return;
    }
    tree.push(p);
    [...p.children].sort().forEach(walk);
  };
  walk('');
  const checklists = new Map((m.impact?.checklists ?? []).map((c) => [c.package, c]));
  return { m, defs, pkgs, tree, checklists, working: new Set(m.impact?.packages ?? []) };
}

export function packageName(path: string): string {
  return path === '' ? 'Root' : path;
}

// depth is how far below Root a Package lies.
export function depth(path: string): number {
  return path === '' ? 0 : path.split('/').length;
}

// leaf is a Package's own name, the last part of its path.
export function leaf(path: string): string {
  return path === '' ? 'Root' : path.slice(path.lastIndexOf('/') + 1);
}

export const changeTone: Record<Change, Tone> = { added: 'success', modified: 'warning', removed: 'danger' };
export const changeLabel: Record<Change, string> = { added: 'New', modified: 'Changed', removed: 'Removed' };

export function byPackage(ds: Diagnostic[]): Map<string, Diagnostic[]> {
  const out = new Map<string, Diagnostic[]>();
  for (const d of ds) {
    out.set(d.package, [...(out.get(d.package) ?? []), d]);
  }
  return out;
}

// items counts a checklist's items.
export function items(c: Checklist | undefined): number {
  if (!c) {
    return 0;
  }
  return (c.own?.length ?? 0) + (c.inherited?.length ?? 0) + (c.findings?.length ?? 0) + (c.stale?.length ?? 0);
}

// firstSentence is a Definition's first sentence of prose, for lists.
export function firstSentence(text: string): string {
  const body = text.replace(/^---\r?\n[\s\S]*?\r?\n---\r?\n?/, '');
  for (const block of body.split(/\r?\n\s*\r?\n/)) {
    const t = block.trim();
    if (t === '' || /^(#|```|\||[-*+] |\d+\. |>)/.test(t)) {
      continue;
    }
    const plain = t
      .replace(/\[\[([^\]|#]+)(?:#[^\]|]*)?(?:\\?\|([^\]]*))?\]\]/g, (_, path: string, label?: string) => label || path)
      .replace(/`([^`]*)`/g, '$1')
      .replace(/\s+/g, ' ');
    const m = /^(.+?[.!?])(\s|$)/.exec(plain);
    return m ? m[1] : plain;
  }
  return '';
}

// The lint codes in a few words, as docs/guides/model-language.md explains
// them.
export const lintTitle: Record<string, string> = {
  'lint.coupling': 'Depends on many Packages',
  'lint.unstable_dependency': 'Depends on a less stable Package',
  'lint.definition_size': 'Long Definition',
  'lint.package_size': 'Large Package',
  'lint.unused': 'Nothing uses it',
  'lint.reach': 'A change reaches many Managers',
  'lint.thin_package': 'Thin Package',
  'lint.no_paths': 'Owns no file',
  'lint.stale': 'Stale realization',
};
