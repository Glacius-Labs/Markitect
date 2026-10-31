import MarkdownIt, { type Env, type StateInline } from 'markdown-it';

// A Definition's Markdown as the Reader shows it. Raw HTML stays text, links
// to other sites stay text with their address as a title, since a window of
// the app is no browser, and a link of the model, [[path|label]], becomes a
// link into the Reader, or a marked span when it resolves to nothing.

export interface LinkTarget {
  target: string; // a Definition ID or a Package path; '' when it resolves to nothing
  package?: boolean;
}

export interface RenderEnv {
  // resolve answers what a link's path names, as the compiler resolved it.
  resolve: (raw: string) => LinkTarget | undefined;
  // href is the Reader's address of a Definition or Package.
  href: (target: string) => string;
}

const md = new MarkdownIt({ html: false, linkify: false, typographer: false });

// splitLink takes a link's inner text apart: its path, the section after #
// and the label after |, which a table writes as \|.
export function splitLink(inner: string): { path: string; label: string } {
  const bar = inner.search(/\\?\|/);
  const target = bar < 0 ? inner : inner.slice(0, bar);
  const label = bar < 0 ? '' : inner.slice(bar).replace(/^\\?\|/, '');
  const path = target.split('#')[0].trim();
  return { path, label: label.trim() };
}

function wikilink(state: StateInline, silent: boolean): boolean {
  const start = state.pos;
  if (state.src.charCodeAt(start) !== 0x5b || state.src.charCodeAt(start + 1) !== 0x5b) {
    return false;
  }
  const end = state.src.indexOf(']]', start + 2);
  if (end < 0 || end > state.posMax) {
    return false;
  }
  // As in the compiler, the path stops at a line break, # or |; the label
  // and the section after it may span lines.
  const inner = state.src.slice(start + 2, end);
  const link = splitLink(inner);
  if (/[[\]]/.test(inner) || link.path === '' || link.path.includes('\n')) {
    return false;
  }
  if (!silent) {
    const token = state.push('wikilink', '', 0);
    token.meta = { path: link.path, label: link.label.replace(/\s+/g, ' ') };
  }
  state.pos = end + 2;
  return true;
}

md.inline.ruler.before('link', 'wikilink', wikilink);

md.renderer.rules.wikilink = (tokens, idx, _options, e) => {
  const env = e as unknown as RenderEnv;
  const { path, label } = tokens[idx].meta as { path: string; label: string };
  const text = md.utils.escapeHtml(label || path);
  const found = env.resolve(path);
  if (!found || !found.target) {
    return `<span class="wikilink" data-broken title="${md.utils.escapeHtml(`${path} resolves to nothing`)}">${text}</span>`;
  }
  return `<a class="wikilink" href="${md.utils.escapeHtml(env.href(found.target))}">${text}</a>`;
};

// Links to other sites keep their text and show their address.
md.renderer.rules.link_open = (tokens, idx) => {
  const href = String(tokens[idx].attrGet('href') ?? '');
  return `<span class="external" title="${md.utils.escapeHtml(href)}">`;
};
md.renderer.rules.link_close = () => '</span>';
md.renderer.rules.image = (tokens, idx) => {
  const t = tokens[idx];
  return `<span class="external" title="${md.utils.escapeHtml(String(t.attrGet('src') ?? ''))}">${md.utils.escapeHtml(t.content || 'image')}</span>`;
};
md.renderer.rules.table_open = () => '<div class="gl-table-wrap"><table class="gl-table" data-density="compact">';
md.renderer.rules.table_close = () => '</table></div>';

// body is a Definition's text without its front matter.
export function body(text: string): string {
  const m = /^---\r?\n[\s\S]*?\r?\n---\r?\n?/.exec(text);
  return m ? text.slice(m[0].length) : text;
}

// render turns a Definition's text into HTML for the Reader.
export function render(text: string, env: RenderEnv): string {
  return md.render(body(text), env as unknown as Env);
}
