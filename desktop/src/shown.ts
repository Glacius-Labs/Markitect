import { useEffect, useState } from 'react';

// The chat whose proposal the Model views show on top of the working tree,
// per project. It lives in this window's session, so moving around the views
// keeps it, and every part of the window that shows it hears a change.

const key = (pid: string) => `markitect.shown.${pid}`;
const changed = 'markitect-shown';

function read(pid: string): string | null {
  try {
    return sessionStorage.getItem(key(pid));
  } catch {
    return null;
  }
}

export function showChat(pid: string, chat: string | null) {
  try {
    if (chat === null) {
      sessionStorage.removeItem(key(pid));
    } else {
      sessionStorage.setItem(key(pid), chat);
    }
  } catch {
    // No storage: the views show the working tree.
  }
  window.dispatchEvent(new Event(changed));
}

// A conversation changed outside the panel, as by an accept from the Model
// area's banner; the panel reads it again.
const decided = 'markitect-chat-changed';

export function chatChanged() {
  window.dispatchEvent(new Event(decided));
}

export function useChatChanged(react: () => void) {
  useEffect(() => {
    window.addEventListener(decided, react);
    return () => window.removeEventListener(decided, react);
  }, [react]);
}

export function useShownChat(pid: string): string | null {
  const [chat, setChat] = useState(() => read(pid));
  useEffect(() => {
    const update = () => setChat(read(pid));
    update();
    window.addEventListener(changed, update);
    return () => window.removeEventListener(changed, update);
  }, [pid]);
  return chat;
}
