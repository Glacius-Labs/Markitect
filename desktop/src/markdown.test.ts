import { describe, expect, it } from 'vitest';
import { diffLines } from './diff';
import { body, render, splitLink, type RenderEnv } from './markdown';

const env: RenderEnv = {
  resolve: (raw) => ({ 'members/member': { target: 'members/member' }, members: { target: 'members', package: true } })[raw],
  href: (target) => `#/reader/${target}`,
};

describe('the Reader', () => {
  it('drops the front matter', () => {
    expect(body('---\nkind: std/Rule\n---\n# Limit\n')).toBe('# Limit\n');
    expect(body('# No front matter\n')).toBe('# No front matter\n');
  });

  it('takes a link apart as the compiler does', () => {
    expect(splitLink('members/member')).toEqual({ path: 'members/member', label: '' });
    expect(splitLink('members/member#Status|member')).toEqual({ path: 'members/member', label: 'member' });
    expect(splitLink('members/member\\|member')).toEqual({ path: 'members/member', label: 'member' });
  });

  it('links the model into the Reader and marks what resolves to nothing', () => {
    const html = render('---\nkind: std/Feature\n---\n# Borrow\n\nA [[members/member|member]] of [[members]] borrows a [[books/book]].\n', env);
    expect(html).toContain('<h1>Borrow</h1>');
    expect(html).toContain('<a class="wikilink" href="#/reader/members/member">member</a>');
    expect(html).toContain('<a class="wikilink" href="#/reader/members">members</a>');
    expect(html).toContain('<span class="wikilink" data-broken title="books/book resolves to nothing">books/book</span>');
  });

  it('leaves links in code as text', () => {
    const html = render('Write `[[members/member]]` to link.\n\n```\n[[members]]\n```\n', env);
    expect(html).not.toContain('wikilink');
    expect(html).toContain('<code>[[members/member]]</code>');
  });

  it('reads a link with an escaped bar in a table', () => {
    const html = render('| File | Realizes |\n|---|---|\n| a.go | [[members/member\\|member]] |\n', env);
    expect(html).toContain('<table class="gl-table"');
    expect(html).toContain('<a class="wikilink" href="#/reader/members/member">member</a>');
  });

  it('keeps raw HTML and links to other sites as text', () => {
    const html = render('<script>alert(1)</script>\n\nSee [the site](https://example.com) and ![a chart](chart.png).\n', env);
    expect(html).not.toContain('<script>');
    expect(html).not.toContain('<a ');
    expect(html).not.toContain('<img');
    expect(html).toContain('<span class="external" title="https://example.com">the site</span>');
  });
});

describe('the line diff', () => {
  it('keeps what both share and marks the rest', () => {
    expect(diffLines('a\nb\nc\n', 'a\nx\nc\nd\n')).toEqual([
      { op: 'same', text: 'a' },
      { op: 'del', text: 'b' },
      { op: 'add', text: 'x' },
      { op: 'same', text: 'c' },
      { op: 'add', text: 'd' },
    ]);
  });

  it('shows a new text as added and a removed one as deleted', () => {
    expect(diffLines('', 'a\n')).toEqual([{ op: 'add', text: 'a' }]);
    expect(diffLines('a\r\nb\r\n', '')).toEqual([
      { op: 'del', text: 'a' },
      { op: 'del', text: 'b' },
    ]);
  });
});
