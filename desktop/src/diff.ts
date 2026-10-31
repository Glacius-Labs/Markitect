// A line diff for the Reader's diff toggle: the longest common subsequence of
// two texts' lines. Definitions are short, so the quadratic table is cheap.

export interface Line {
  op: 'same' | 'add' | 'del';
  text: string;
}

export function diffLines(before: string, after: string): Line[] {
  const a = split(before);
  const b = split(after);
  const n = a.length;
  const m = b.length;
  // lcs[i][j] is the length of the common subsequence of a[i:] and b[j:].
  const lcs: number[][] = Array.from({ length: n + 1 }, () => new Array<number>(m + 1).fill(0));
  for (let i = n - 1; i >= 0; i--) {
    for (let j = m - 1; j >= 0; j--) {
      lcs[i][j] = a[i] === b[j] ? lcs[i + 1][j + 1] + 1 : Math.max(lcs[i + 1][j], lcs[i][j + 1]);
    }
  }
  const out: Line[] = [];
  let i = 0;
  let j = 0;
  while (i < n && j < m) {
    if (a[i] === b[j]) {
      out.push({ op: 'same', text: a[i] });
      i++;
      j++;
    } else if (lcs[i + 1][j] >= lcs[i][j + 1]) {
      out.push({ op: 'del', text: a[i++] });
    } else {
      out.push({ op: 'add', text: b[j++] });
    }
  }
  while (i < n) {
    out.push({ op: 'del', text: a[i++] });
  }
  while (j < m) {
    out.push({ op: 'add', text: b[j++] });
  }
  return out;
}

function split(text: string): string[] {
  if (text === '') {
    return [];
  }
  return text.replace(/\r\n/g, '\n').replace(/\n$/, '').split('\n');
}
