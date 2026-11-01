import type { Status, Waiting } from './api';

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
  // The chat whose proposal waits.
  proposal?: string;
  // What to start in a terminal, until Runs are in the app.
  command?: string;
}

const inATerminal = 'Start the Run from a terminal, as Runs are not in the app yet. It works from the tip of the branch, so push first.';

// nextStep is the one thing the hub asks of the owner now.
export function nextStep(s: Status, w?: Waiting, path = ''): Step {
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
  if (w?.proposal) {
    return {
      tone: 'info',
      title: `A proposal waits: ${w.proposal.title}`,
      body: 'Read it in the Model views, then accept or reject it. Nothing reaches your working tree before that.',
      proposal: w.proposal.chat,
    };
  }
  if (w && w.uncommitted > 0) {
    return {
      tone: 'info',
      title: 'Commit the model change',
      body: `${plural(w.uncommitted, 'file')} in .markitect ${w.uncommitted === 1 ? 'differs' : 'differ'} from the last commit. A Run starts from the tip of the branch, never from the working tree, so commit and push first.`,
    };
  }
  const apply = `markitect apply --repo "${path}"`;
  if (w && !w.landed) {
    return {
      tone: 'info',
      title: 'Start the first Run',
      body: `No Run has landed here, so the first one says where to start: --full has every Manager work, and --base COMMIT takes what changed since that commit. ${inATerminal}`,
      command: `${apply} --full`,
    };
  }
  if (w && w.unlanded > 0) {
    return {
      tone: 'info',
      title: 'Bring the code in line with the model',
      body: `${plural(w.unlanded, 'Definition')} changed since the last landing. ${inATerminal}`,
      command: apply,
    };
  }
  const size = `${plural(c.definitions, 'Definition')} in ${plural(c.packages, 'Package')}, ${plural(c.files, 'file')} accounted for.`;
  return {
    tone: 'success',
    title: 'The model compiles',
    body: w ? `${size} No Definition changed since the last landing.` : size,
  };
}
