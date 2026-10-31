import type { Status } from './api';

export type Tone = 'neutral' | 'info' | 'success' | 'warning' | 'danger';

export function plural(n: number, word: string): string {
  return `${n} ${word}${n === 1 ? '' : 's'}`;
}

// badge is a project's state in a few words, for its card.
export function badge(s: Status): { tone: Tone; label: string } {
  if (s.problem) {
    return { tone: 'danger', label: 'Cannot be read' };
  }
  if (!s.initialized || !s.check) {
    return { tone: 'neutral', label: 'Not set up' };
  }
  if (s.check.ok) {
    return { tone: 'success', label: 'Model compiles' };
  }
  return { tone: 'danger', label: plural(s.check.errorCount, 'error') };
}

export interface Step {
  tone: Exclude<Tone, 'neutral'>;
  title: string;
  body: string;
  action?: 'init';
}

// nextStep is the one thing the hub asks of the owner now.
export function nextStep(s: Status): Step {
  if (s.problem) {
    return { tone: 'danger', title: 'Markitect cannot read this project', body: s.problem };
  }
  if (!s.initialized || !s.check) {
    return {
      tone: 'info',
      title: 'Set up Markitect here',
      body: 'markitect init adds the .markitect folder with a first model page and installs the modeling skills. Its output lists every file it writes.',
      action: 'init',
    };
  }
  const c = s.check;
  if (!c.ok) {
    return {
      tone: 'danger',
      title: `The model has ${plural(c.errorCount, 'compile error')}`,
      body: 'Nothing runs on a model that does not compile. The errors are listed below.',
    };
  }
  return {
    tone: 'success',
    title: 'The model compiles',
    body: `${plural(c.definitions, 'Definition')} in ${plural(c.packages, 'Package')}, ${plural(c.files, 'file')} accounted for.`,
  };
}
