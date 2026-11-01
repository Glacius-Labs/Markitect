import { useCallback, useEffect, useMemo, useRef, useState, type KeyboardEvent as ReactKeyboardEvent } from 'react';
import { api, ApiError, type ChatSummary, type Conversation, type Proposal, type Skill } from './api';
import { renderMessage } from './markdown';
import { changeLabel, changeTone } from './modeldata';
import { modelHref } from './route';
import { chatChanged, showChat, useChatChanged, useShownChat } from './shown';
import { plural } from './status';
import { Alert, Badge, Icon } from './ui';

// The chat panel: conversations with an agent that works on a copy of the
// project and proposes model changes, which the owner reads in the Model
// views and accepts into the working tree or rejects.

const currentKey = (pid: string) => `markitect.chat.current.${pid}`;

// openChat opens the panel on a conversation, from anywhere in the window.
const opened = 'markitect-chat-opened';

export function openChat(pid: string, id: string) {
  writeCurrent(pid, id);
  window.dispatchEvent(new CustomEvent<string>(opened, { detail: id }));
}

export function useChatOpened(react: (id: string) => void) {
  useEffect(() => {
    const listen = (e: Event) => react((e as CustomEvent<string>).detail);
    window.addEventListener(opened, listen);
    return () => window.removeEventListener(opened, listen);
  }, [react]);
}

function readCurrent(pid: string): string | null {
  try {
    return sessionStorage.getItem(currentKey(pid));
  } catch {
    return null;
  }
}

function writeCurrent(pid: string, id: string | null) {
  try {
    if (id === null) {
      sessionStorage.removeItem(currentKey(pid));
    } else {
      sessionStorage.setItem(currentKey(pid), id);
    }
  } catch {
    // No storage: the panel opens on a new conversation.
  }
}

// While a turn runs, the panel asks for the conversation every second.
const followEvery = 1000;

const skills: { skill: Skill; title: string; body: string; placeholder: string }[] = [
  {
    skill: 'modeling',
    title: 'Change the model',
    body: 'Talk about what the project should do. The chat asks where something is open and proposes the model change.',
    placeholder: 'What should change in the model?',
  },
  {
    skill: 'extract',
    title: 'Extract from the code',
    body: 'Draw a first model from the existing code. The chat proposes the Packages first, then the whole model.',
    placeholder: 'Anything the chat should know about the code first?',
  },
];

export function ChatPanel(props: { pid: string; onClose: () => void }) {
  const { pid, onClose } = props;
  const [list, setList] = useState<{ chats: ChatSummary[]; available: boolean } | null>(null);
  const [current, setCurrent] = useState<string | null>(() => readCurrent(pid));
  const [conv, setConv] = useState<Conversation | null>(null);
  const [skill, setSkill] = useState<Skill>('modeling');
  const [draft, setDraft] = useState('');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  const [deleting, setDeleting] = useState(false);
  // The conversation a request was sent for, so that an answer arriving
  // after the owner switched does not replace the one in view.
  const currentRef = useRef(current);
  currentRef.current = current;

  const loadList = useCallback(() => {
    api.chats(pid).then(setList, (e: Error) => setError(e.message));
  }, [pid]);
  useEffect(loadList, [loadList]);

  const choose = useCallback(
    (id: string | null) => {
      writeCurrent(pid, id);
      setCurrent(id);
      setConv(null);
      setError('');
      setDeleting(false);
    },
    [pid],
  );
  useChatOpened(
    useCallback(
      (id: string) => {
        if (id !== currentRef.current) {
          choose(id);
        }
      },
      [choose],
    ),
  );

  // The conversation, followed while its turn runs, and read again when it
  // was decided elsewhere.
  const working = conv?.working ?? false;
  const [asked, setAsked] = useState(0);
  useChatChanged(useCallback(() => setAsked((n) => n + 1), []));
  useEffect(() => {
    if (!current) {
      return;
    }
    let stopped = false;
    let timer = 0;
    let wasWorking = false;
    const load = async () => {
      try {
        const c = await api.chat(pid, current);
        if (stopped) {
          return;
        }
        setConv(c);
        setError('');
        if (c.working) {
          wasWorking = true;
          timer = window.setTimeout(load, followEvery);
        } else {
          loadList();
          if (wasWorking) {
            // The turn's proposal may have changed what the views show.
            chatChanged();
          }
        }
      } catch (e) {
        if (stopped) {
          return;
        }
        if (e instanceof ApiError && e.status === 404) {
          choose(null);
          return;
        }
        // A read that failed is tried again a little later.
        setError(e instanceof Error ? e.message : String(e));
        timer = window.setTimeout(load, 2 * followEvery);
      }
    };
    void load();
    return () => {
      stopped = true;
      window.clearTimeout(timer);
    };
  }, [pid, current, working, asked, loadList, choose]);

  const send = async () => {
    const text = draft.trim();
    if (!text || busy || working) {
      return;
    }
    setBusy(true);
    setError('');
    try {
      const sentFor = current;
      const c = current ? await api.say(pid, current, text) : await api.newChat(pid, text, skill);
      setDraft('');
      if (currentRef.current === sentFor) {
        if (!sentFor) {
          writeCurrent(pid, c.id);
          setCurrent(c.id);
        }
        setConv(c);
      }
      loadList();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  };

  const stop = async () => {
    if (!current) {
      return;
    }
    const stoppedIn = current;
    setError('');
    try {
      const c = await api.stopChat(pid, stoppedIn);
      // A turn that has ended is left to the reading above, which tells the views.
      if (c.working && currentRef.current === stoppedIn) {
        setConv(c);
      }
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
  };

  const remove = async () => {
    if (!current) {
      return;
    }
    setBusy(true);
    try {
      await api.deleteChat(pid, current);
      choose(null);
      loadList();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  };

  const fresh = !current;
  const chosen = skills.find((s) => s.skill === (conv?.skill ?? skill))!;
  return (
    <aside className="chat" aria-label="Chat">
      <div className="chat__head">
        <select className="gl-select chat__pick" aria-label="Conversation" value={current ?? ''} onChange={(e) => choose(e.target.value || null)}>
          <option value="">New conversation</option>
          {list?.chats.map((c) => (
            <option key={c.id} value={c.id}>
              {c.working ? '… ' : c.proposal ? '• ' : ''}
              {c.title}
            </option>
          ))}
        </select>
        {current &&
          (deleting ? (
            <>
              <button className="gl-button" data-variant="danger" data-size="sm" disabled={busy || working} onClick={remove}>
                Delete it
              </button>
              <button className="gl-button" data-variant="ghost" data-size="sm" onClick={() => setDeleting(false)}>
                Keep
              </button>
            </>
          ) : (
            <button className="gl-button" data-variant="ghost" data-size="sm" disabled={busy || working} onClick={() => setDeleting(true)} title="Delete this conversation">
              Delete
            </button>
          ))}
        <button className="gl-button" data-variant="ghost" data-size="sm" data-icon-only aria-label="Close the chat" onClick={onClose}>
          <Icon name="x" />
        </button>
      </div>
      <div className="chat__body">
        {list && !list.available && (
          <Alert tone="info" title="No chat here yet" body="A chat runs its agent in a container, with Docker, the Manager image and your provider login. This Markitect was started without them." />
        )}
        {fresh ? (
          <div className="chat__start">
            <div className="gl-segmented" data-block role="radiogroup" aria-label="What the chat does">
              {skills.map((s) => (
                <button key={s.skill} type="button" className="gl-segmented__item" role="radio" aria-checked={skill === s.skill} onClick={() => setSkill(s.skill)}>
                  {s.title}
                </button>
              ))}
            </div>
            <p className="small muted">{chosen.body} Nothing reaches your working tree until you accept it.</p>
          </div>
        ) : (
          conv && <Messages pid={pid} conv={conv} onChanged={setConv} onShow={onClose} />
        )}
        {conv?.unsaved && <Alert tone="warning" title="Not saved" body={`This conversation stays only while Markitect runs: ${conv.unsaved}`} />}
        {error && <Alert tone="danger" title="That did not work" body={error} />}
      </div>
      <div className="chat__compose">
        <textarea
          className="gl-textarea chat__input"
          aria-label="Message"
          placeholder={fresh ? chosen.placeholder : 'Answer or ask for more'}
          value={draft}
          disabled={!list?.available}
          onChange={(e) => setDraft(e.target.value)}
          onKeyDown={(e: ReactKeyboardEvent) => {
            if (e.key === 'Enter' && !e.shiftKey && !e.nativeEvent.isComposing) {
              e.preventDefault();
              void send();
            }
          }}
        />
        {working ? (
          <button className="gl-button" onClick={() => void stop()} title="Stop this turn">
            Stop
          </button>
        ) : (
          <button className="gl-button" data-variant="primary" aria-busy={busy || undefined} disabled={busy || !draft.trim() || !list?.available} onClick={() => void send()}>
            Send
          </button>
        )}
      </div>
    </aside>
  );
}

function Messages({ pid, conv, onChanged, onShow }: { pid: string; conv: Conversation; onChanged: (c: Conversation) => void; onShow: () => void }) {
  const end = useRef<HTMLDivElement>(null);
  const count = conv.messages.length;
  // New messages keep the end in view, unless the owner scrolled up to read.
  const following = useRef(true);
  useEffect(() => {
    const box = end.current?.closest('.chat__body');
    if (!box) {
      return;
    }
    const onScroll = () => {
      following.current = box.scrollHeight - box.scrollTop - box.clientHeight < 80;
    };
    box.addEventListener('scroll', onScroll);
    return () => box.removeEventListener('scroll', onScroll);
  }, []);
  // What the owner sends brings them back to the end, to see the answer come.
  const sent = conv.messages.filter((m) => m.from === 'owner').length;
  useEffect(() => {
    following.current = true;
  }, [sent]);
  useEffect(() => {
    if (following.current) {
      end.current?.scrollIntoView({ block: 'end' });
    }
  }, [count, conv.working]);
  const open = conv.proposals.at(-1)?.state === 'open' ? conv.proposals.at(-1) : undefined;
  return (
    <div className="chat__messages" aria-live="polite">
      {conv.messages.map((m, i) =>
        m.from === 'agent' ? <AgentText key={i} pid={pid} text={m.text} /> : (
          <p key={i} className={m.from === 'owner' ? 'chat__owner' : 'chat__note small'}>
            {m.text}
          </p>
        ),
      )}
      {conv.working && (
        <p className="chat__note small">
          {/* Announced once a second it would drown the messages. */}
          <span className="gl-spinner" aria-hidden="true" /> <span className="chat__doing" aria-live="off">{conv.doing || 'The chat is at work.'}</span>
        </p>
      )}
      {open && !conv.working && <ProposalCard pid={pid} chat={conv.id} proposal={open} onChanged={onChanged} onShow={onShow} />}
      <div ref={end} />
    </div>
  );
}

function AgentText({ pid, text }: { pid: string; text: string }) {
  // A link of the model in the chat's words leads into the Reader.
  const html = useMemo(() => renderMessage(text, { resolve: (raw) => ({ target: raw }), href: (t) => modelHref(pid, 'reader', t) }), [pid, text]);
  return <div className="chat__agent prose" dangerouslySetInnerHTML={{ __html: html }} />;
}

function ProposalCard({ pid, chat, proposal, onChanged, onShow }: { pid: string; chat: string; proposal: Proposal; onChanged: (c: Conversation) => void; onShow: () => void }) {
  const shown = useShownChat(pid);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  const decide = async (what: 'accept' | 'reject') => {
    setBusy(true);
    setError('');
    try {
      const c = await api.decide(pid, chat, what);
      onChanged(c);
      if (c.proposals.at(-1)?.state !== 'open' && shown === chat) {
        showChat(pid, null);
      }
      chatChanged();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  };
  return (
    <section className="gl-card chat__proposal" aria-label="Proposal">
      <h3 className="gl-card__title">Proposal: {plural(proposal.files.length, 'file')}</h3>
      <ul className="chat__files">
        {proposal.files.map((f) => (
          <li key={f.path}>
            <Badge tone={changeTone[f.change]}>{changeLabel[f.change]}</Badge> <span className="path">{f.path.replace(/^\.markitect\//, '')}</span>
          </li>
        ))}
      </ul>
      {proposal.outside && proposal.outside.length > 0 && <p className="small muted">Also changed outside the model, and not proposed: {proposal.outside.join(', ')}</p>}
      {proposal.conflicts && proposal.conflicts.length > 0 && (
        <Alert tone="danger" title="Nothing was written" body={`These files changed in the working tree since the chat started: ${proposal.conflicts.join(', ')}. The chat's next turn starts from the working tree as it is; ask it to take the change into account, or reject the proposal.`} />
      )}
      {error && <Alert tone="danger" title="That did not work" body={error} />}
      <div className="chat__actions">
        {shown === chat ? (
          <button className="gl-button" data-size="sm" onClick={() => showChat(pid, null)}>
            Show the working tree
          </button>
        ) : (
          <a
            className="gl-button"
            data-size="sm"
            href={modelHref(pid, 'reader')}
            onClick={() => {
              showChat(pid, chat);
              // In a narrow window the panel covers the views; it steps aside.
              if (window.matchMedia('(max-width: 900px)').matches) {
                onShow();
              }
            }}
          >
            Show in the views
          </a>
        )}
        <button className="gl-button" data-variant="primary" data-size="sm" aria-busy={busy || undefined} disabled={busy} onClick={() => void decide('accept')}>
          Accept
        </button>
        <button className="gl-button" data-size="sm" disabled={busy} onClick={() => void decide('reject')}>
          Reject
        </button>
      </div>
    </section>
  );
}

// ShownProposal heads the Model area while it shows a chat's proposal.
export function ShownProposal({ pid, chat, refresh }: { pid: string; chat: string; refresh: () => void }) {
  const [conv, setConv] = useState<Conversation | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  // The banner follows the conversation: a later turn may replace its
  // proposal, and the panel may accept or reject it.
  const [asked, setAsked] = useState(0);
  useChatChanged(useCallback(() => setAsked((n) => n + 1), []));
  useEffect(() => {
    let stopped = false;
    let timer = 0;
    const load = () =>
      api.chat(pid, chat).then(
        (c) => {
          if (stopped) {
            return;
          }
          setConv(c);
          if (c.proposals.at(-1)?.state !== 'open' && !c.working) {
            showChat(pid, null);
            return;
          }
          timer = window.setTimeout(load, 2 * followEvery);
        },
        (e: unknown) => {
          if (stopped) {
            return;
          }
          if (e instanceof ApiError && e.status === 404) {
            showChat(pid, null);
          } else {
            timer = window.setTimeout(load, 2 * followEvery);
          }
        },
      );
    void load();
    return () => {
      stopped = true;
      window.clearTimeout(timer);
    };
  }, [pid, chat, asked]);
  const open = conv?.proposals.at(-1);
  const decide = async (what: 'accept' | 'reject') => {
    setBusy(true);
    setError('');
    try {
      const c = await api.decide(pid, chat, what);
      setConv(c);
      if (c.proposals.at(-1)?.state !== 'open') {
        showChat(pid, null);
      }
      chatChanged();
      refresh();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  };
  const conflicts = open?.state === 'open' ? (open.conflicts ?? []) : [];
  const closed = busy || !conv || conv.working || open?.state !== 'open';
  return (
    <>
      <Alert
        tone="info"
        title={`The views show the proposal of “${conv?.title ?? 'a chat'}”`}
        body={conv?.working ? 'The chat is at work; its next proposal replaces this one.' : open ? `${plural(open.files.length, 'file')} on top of the working tree. Turn the diff on to see what it changes.` : undefined}
      >
        <button className="gl-button" data-variant="primary" data-size="sm" aria-busy={busy || undefined} disabled={closed} onClick={() => void decide('accept')}>
          Accept
        </button>
        <button className="gl-button" data-size="sm" disabled={closed} onClick={() => void decide('reject')}>
          Reject
        </button>
        <button className="gl-button" data-variant="ghost" data-size="sm" onClick={() => showChat(pid, null)}>
          Show the working tree
        </button>
      </Alert>
      {conflicts.length > 0 && <Alert tone="danger" title="Nothing was written" body={`These files changed in the working tree since the chat started: ${conflicts.join(', ')}.`} />}
      {error && <Alert tone="danger" title="That did not work" body={error} />}
    </>
  );
}
