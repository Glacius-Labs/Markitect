import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { duration } from './Settings';
import { atMostEvery } from './focus';
import { badge, nextStep, plural } from './status';

const check = { ok: true, errorCount: 0, errors: [], definitions: 12, packages: 3, files: 40, warnings: 0 };

describe('a refresh on focus', () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });
  afterEach(() => {
    vi.useRealTimers();
  });

  it('runs at most every few seconds, however often the focus comes back', () => {
    let runs = 0;
    const refresh = atMostEvery(5000, () => runs++);
    refresh.call();
    refresh.call();
    vi.advanceTimersByTime(4999);
    expect(runs).toBe(0);
    vi.advanceTimersByTime(1);
    expect(runs).toBe(1);
    refresh.call();
    refresh.call();
    vi.advanceTimersByTime(4999);
    expect(runs).toBe(1);
    vi.advanceTimersByTime(1);
    expect(runs).toBe(2);
  });

  it('runs at once after a quiet time, and not after it stopped', () => {
    let runs = 0;
    const refresh = atMostEvery(5000, () => runs++);
    vi.advanceTimersByTime(60000);
    refresh.call();
    vi.advanceTimersByTime(0);
    expect(runs).toBe(1);
    refresh.call();
    refresh.stop();
    vi.advanceTimersByTime(60000);
    expect(runs).toBe(1);
  });
});

describe('the project state', () => {
  it('counts in words', () => {
    expect(plural(1, 'error')).toBe('1 error');
    expect(plural(0, 'error')).toBe('0 errors');
  });

  it('asks to set Markitect up where it is not', () => {
    expect(badge({ initialized: false })).toEqual({ tone: 'neutral', label: 'Not set up' });
    expect(nextStep({ initialized: false })).toMatchObject({ tone: 'info', action: 'init' });
  });

  it('puts a problem first', () => {
    const s = { initialized: true, check, problem: 'gone' };
    expect(badge(s).tone).toBe('danger');
    expect(nextStep(s)).toMatchObject({ title: 'Markitect cannot read this project', body: 'gone' });
  });

  it('names the compile errors', () => {
    const s = { initialized: true, check: { ...check, ok: false, errorCount: 2 } };
    expect(badge(s)).toEqual({ tone: 'danger', label: '2 errors' });
    expect(nextStep(s).title).toBe('The model has 2 compile errors');
    expect(nextStep(s).action).toBeUndefined();
  });

  it('reports a compiling model with its size', () => {
    const s = { initialized: true, check };
    expect(badge(s).label).toBe('Model compiles');
    expect(nextStep(s).body).toBe('12 Definitions in 3 Packages, 40 files accounted for.');
  });
});

describe('what waits for the owner', () => {
  const s = { initialized: true, check };
  const quiet = { uncommitted: 0, landed: true, unlanded: 0 };
  const proposal = { chat: 'c1', title: 'Add a late fee' };

  it('asks for one thing at a time, the nearest first', () => {
    const all = { proposal, uncommitted: 2, landed: true, unlanded: 3 };
    expect(nextStep(s, all, 'C:/lib')).toMatchObject({ title: 'A proposal waits: Add a late fee', proposal: 'c1' });
    expect(nextStep(s, { ...all, proposal: undefined }, 'C:/lib').title).toBe('Commit the model change');
    expect(nextStep(s, { ...all, proposal: undefined, uncommitted: 0 }, 'C:/lib').title).toBe('Bring the code in line with the model');
  });

  it('puts what stops the model before what waits', () => {
    const broken = { initialized: true, check: { ...check, ok: false, errorCount: 1 } };
    expect(nextStep(broken, { ...quiet, proposal }).tone).toBe('danger');
    expect(nextStep({ initialized: true, check, problem: 'gone' }, { ...quiet, proposal }).title).toBe('Markitect cannot read this project');
  });

  it('names the files that differ from the last commit', () => {
    expect(nextStep(s, { ...quiet, uncommitted: 1 }).body).toContain('1 file in .markitect differs');
    expect(nextStep(s, { ...quiet, uncommitted: 2 }).body).toContain('2 files in .markitect differ');
  });

  it('gives the command of a Run, with the path in quotes', () => {
    const later = nextStep(s, { ...quiet, unlanded: 1 }, 'C:/my lib');
    expect(later.body).toContain('1 Definition changed since the last landing');
    expect(later.command).toBe('markitect apply --repo "C:/my lib"');
    const first = nextStep(s, { ...quiet, landed: false }, 'C:/my lib');
    expect(first.title).toBe('Start the first Run');
    expect(first.command).toBe('markitect apply --repo "C:/my lib" --full');
  });

  it('says when nothing waits', () => {
    const fine = nextStep(s, quiet);
    expect(fine).toMatchObject({ tone: 'success', title: 'The model compiles' });
    expect(fine.body).toContain('No Definition changed since the last landing.');
    expect(fine.command).toBeUndefined();
  });
});

describe("a required check's time limit", () => {
  it('reads like execution.yaml', () => {
    expect(duration(10 * 60e9)).toBe('10m');
    expect(duration(2 * 3600e9)).toBe('2h');
    expect(duration(90e9)).toBe('90s');
    expect(duration(1500e6)).toBe('1500ms');
    expect(duration(0)).toBe('0ms');
  });
});
