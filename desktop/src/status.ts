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
  if (s.check.notDrawn) {
    return { tone: 'neutral', label: 'No model yet' };
  }
  return { tone: 'danger', label: plural(s.check.errorCount, 'error') };
}

export interface Step {
  tone: Exclude<Tone, 'neutral'>;
  title: string;
  body: string;
  action?: 'init' | 'draw';
  // The chat whose proposal waits.
  proposal?: string;
  // The chat that draws the model.
  chat?: string;
  // What to start in a terminal, until Runs are in the app.
  command?: string;
}

const inATerminal = "Start the Run from a terminal in the project's folder, as Runs are not in the app yet. It starts from the branch's tip on its remote, so pull and push first.";

// nextStep is the one thing the hub asks of the owner now.
export function nextStep(s: Status, w?: Waiting): Step {
  if (s.problem) {
    return { tone: 'danger', title: 'Markitect cannot read this project', body: s.problem };
  }
  if (!s.initialized || !s.check) {
    return {
      tone: 'info',
      title: 'Set up Markitect here',
      body: 'markitect init adds the .markitect folder with a first model page and installs the modeling skills. If the project has code, an agent can then draw the model from it. Its output lists every file it writes.',
      action: 'init',
    };
  }
  const c = s.check;
  if (c.notDrawn && !w?.proposal) {
    if (w?.extract) {
      return {
        tone: 'info',
        title: w.extract.working ? 'The model is being drawn' : 'The chat that draws the model waits for you',
        body: w.extract.working
          ? 'An agent is reading the code. Follow it in the chat, where it asks about what the code leaves open.'
          : 'See where it stands in the chat: it may have a question, or it may not have started. Nothing reaches your files until you accept its proposal.',
        chat: w.extract.chat,
        action: w.extract.working ? undefined : 'draw',
      };
    }
    return {
      tone: 'info',
      title: 'Draw the model from the code',
      body: `Markitect is set up, and ${plural(c.errorCount, 'file')} of this project ${c.errorCount === 1 ? 'has' : 'have'} no place in a model yet. An agent reads the code in a container of its own, asks you what the code leaves open, and proposes a model. Nothing reaches your files until you accept it.`,
      action: 'draw',
    };
  }
  if (!c.ok && !c.notDrawn) {
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
  if (w && !w.landed) {
    return {
      tone: 'info',
      title: 'Start the first Run',
      body: `No Run has landed here, so the first one says where to start: --full has every Manager work, and --base COMMIT takes what changed since that commit. ${inATerminal}`,
      command: 'markitect apply --full',
    };
  }
  if (w && w.unlanded > 0) {
    return {
      tone: 'info',
      title: 'Bring the code in line with the model',
      body: `${plural(w.unlanded, 'Definition')} changed since the last landing. ${inATerminal}`,
      command: 'markitect apply',
    };
  }
  const size = `${plural(c.definitions, 'Definition')} in ${plural(c.packages, 'Package')}, ${plural(c.files, 'file')} accounted for.`;
  return {
    tone: 'success',
    title: 'The model compiles',
    body: w ? `${size} No Definition changed since the last landing.` : size,
  };
}
