import { describe, expect, it } from 'vitest';
import { badge, nextStep, plural } from './status';

const check = { ok: true, errorCount: 0, errors: [], definitions: 12, packages: 3, files: 40, warnings: 0 };

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
