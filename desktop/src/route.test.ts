import { describe, expect, it } from 'vitest';
import { modelHref, parse } from './route';

describe('the places in the app', () => {
  it('reads the projects, a hub and the Model area', () => {
    expect(parse('#/')).toEqual({});
    expect(parse('#/p/ab12')).toEqual({ project: 'ab12' });
    expect(parse('#/p/ab12/model')).toEqual({ project: 'ab12', area: 'model', view: 'reader' });
    expect(parse('#/p/ab12/model/map')).toEqual({ project: 'ab12', area: 'model', view: 'map' });
  });

  it('reads an item, Root included', () => {
    expect(parse('#/p/ab12/model/reader/members/member')).toMatchObject({ view: 'reader', item: 'members/member' });
    expect(parse('#/p/ab12/model/organization/.')).toMatchObject({ view: 'organization', item: '' });
  });

  it('writes what it reads', () => {
    for (const [view, item] of [
      ['reader', 'members/member'],
      ['map', ''],
      ['quality', undefined],
    ] as const) {
      expect(parse(modelHref('ab12', view, item))).toEqual({ project: 'ab12', area: 'model', view, ...(item === undefined ? {} : { item }) });
    }
  });

  it('falls back to the Reader for an unknown view', () => {
    expect(parse('#/p/ab12/model/nowhere')).toMatchObject({ view: 'reader' });
  });

  it('reads a broken escape as no item', () => {
    expect(parse('#/p/ab12/model/reader/%')).toEqual({ project: 'ab12', area: 'model', view: 'reader' });
  });
});
