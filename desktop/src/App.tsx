import { useCallback, useEffect, useState } from 'react';
import { api, ApiError, type AppInfo } from './api';
import { ChatPanel, useChatOpened } from './Chat';
import { Hub, useEscapeTo, type InitRun } from './Hub';
import { ModelArea } from './Model';
import { Projects } from './Projects';
import { hub, parse } from './route';
import { Settings } from './Settings';
import { Statistics } from './Statistics';

const place = () => parse(window.location.hash);

// Whether the chat panel is open lives in this window's session.
const chatKey = 'markitect.chat.open';

function chatOpen(): boolean {
  try {
    return sessionStorage.getItem(chatKey) === '1';
  } catch {
    return false;
  }
}

export function App() {
  const [info, setInfo] = useState<AppInfo | null>(null);
  const [closed, setClosed] = useState(false);
  const [where, setWhere] = useState(place);
  const [chat, setChat] = useState(chatOpen);
  const toggleChat = useCallback(() => {
    setChat((open) => {
      try {
        sessionStorage.setItem(chatKey, open ? '0' : '1');
      } catch {
        // No storage: the panel stays as it is until the window closes.
      }
      return !open;
    });
  }, []);
  useChatOpened(
    useCallback(() => {
      try {
        sessionStorage.setItem(chatKey, '1');
      } catch {
        // No storage: the panel stays open until the window closes.
      }
      setChat(true);
    }, []),
  );
  // The init that created a project, shown on its hub until the owner leaves.
  const [created, setCreated] = useState<{ id: string; init: InitRun } | null>(null);

  const loadInfo = useCallback(() => {
    api.app().then(setInfo, (e: Error) => setClosed(e instanceof ApiError && e.status === 401));
  }, []);

  useEffect(() => {
    loadInfo();
    const onHash = () => {
      const next = place();
      setWhere(next);
      setCreated((c) => (c && c.id === next.project ? c : null));
    };
    window.addEventListener('hashchange', onHash);
    return () => window.removeEventListener('hashchange', onHash);
  }, [loadInfo]);

  const home = useCallback(() => {
    window.location.hash = '#/';
  }, []);
  // Escape goes up one level: from an area to its project, from a project to
  // the list.
  const up = useCallback(() => {
    window.location.hash = where.area && where.project ? hub(where.project) : '#/';
  }, [where]);
  useEscapeTo(up);

  const panel = chat && where.project && !closed;
  return (
    <div className={panel ? 'app app--chat' : 'app'}>
      <header className="topbar">
        <a className="topbar__brand" href="#/">
          <img src="/markitect-mark.svg" alt="" className="topbar__mark" />
          <span>Markitect</span>
        </a>
        {info && <span className="topbar__version small">{info.version}</span>}
        {where.project && !closed && (
          <button className="gl-button" data-size="sm" aria-pressed={chat} onClick={toggleChat}>
            Chat
          </button>
        )}
      </header>
      <div className="workspace">
        <main className="page">
          {closed ? (
            <div className="gl-empty">
              <h3 className="gl-empty__title">This window lost its connection</h3>
              <p className="gl-empty__body">Markitect was started again. Open it with markitect desktop.</p>
            </div>
          ) : where.project && where.area === 'statistics' ? (
            <Statistics key={where.project} id={where.project} />
          ) : where.project && where.area === 'settings' ? (
            <Settings key={where.project} id={where.project} />
          ) : where.project && where.area === 'model' ? (
            <ModelArea key={where.project} id={where.project} view={where.view ?? 'reader'} item={where.item} />
          ) : where.project ? (
            <Hub key={where.project} id={where.project} init={created?.id === where.project ? created.init : undefined} onGone={home} />
          ) : (
            <Projects
              info={info}
              onAdded={(r) => {
                setCreated(r.output || !r.ok ? { id: r.project.id, init: { ok: r.ok, output: r.output } } : null);
                loadInfo();
                window.location.hash = `#/p/${r.project.id}`;
              }}
            />
          )}
        </main>
        {panel && <ChatPanel key={where.project} pid={where.project!} onClose={toggleChat} />}
      </div>
    </div>
  );
}
