import { useEffect, useRef, type ReactNode } from 'react';
import type { Tone } from './status';

// Small wrappers over the design system's markup; the classes and attributes
// are those of its spec.

export function Icon({ name, size }: { name: string; size?: 'xs' | 'md' }) {
  return <span className="gl-icon" data-size={size} style={{ ['--gl-icon' as string]: `var(--gl-icon-${name})` }} aria-hidden="true" />;
}

const alertIcon = { info: 'info', success: 'circle-check', warning: 'triangle-alert', danger: 'octagon-x' } as const;

export function Alert(props: { tone: Exclude<Tone, 'neutral'>; title: string; body?: ReactNode; children?: ReactNode }) {
  return (
    <div className="gl-alert" data-tone={props.tone === 'info' ? undefined : props.tone} role={props.tone === 'danger' ? 'alert' : 'status'}>
      <Icon name={alertIcon[props.tone]} />
      <p className="gl-alert__title">{props.title}</p>
      {props.body && <p className="gl-alert__body">{props.body}</p>}
      {props.children && <div className="gl-alert__actions">{props.children}</div>}
    </div>
  );
}

export function Badge({ tone, children }: { tone: Tone; children: ReactNode }) {
  return (
    <span className="gl-badge" data-tone={tone === 'neutral' ? undefined : tone}>
      {children}
    </span>
  );
}

// Dialog is a native modal dialog, open while it is rendered: removing it
// from the page also takes it off the top layer. showModal focuses the first
// focusable element, the close button, so the first field takes the focus
// back; when the dialog goes, the focus returns to what opened it.
export function Dialog(props: { title: string; onClose: () => void; children: ReactNode; footer: ReactNode }) {
  const ref = useRef<HTMLDialogElement>(null);
  useEffect(() => {
    const opener = document.activeElement instanceof HTMLElement ? document.activeElement : null;
    const d = ref.current;
    if (d && !d.open) {
      d.showModal();
      d.querySelector<HTMLElement>('input, textarea, select')?.focus();
    }
    return () => opener?.focus();
  }, []);
  return (
    <dialog ref={ref} className="gl-dialog" aria-labelledby="dialog-title" onClose={props.onClose}>
      <div className="gl-dialog__header">
        <h2 className="gl-dialog__title" id="dialog-title">
          {props.title}
        </h2>
        <button type="button" className="gl-button" data-variant="ghost" data-size="sm" data-icon-only aria-label="Close" onClick={props.onClose}>
          <Icon name="x" />
        </button>
      </div>
      <div className="gl-dialog__body">{props.children}</div>
      <div className="gl-dialog__footer">{props.footer}</div>
    </dialog>
  );
}

export function Field(props: {
  id: string;
  label: string;
  hint?: string;
  optional?: boolean;
  value: string;
  onChange: (v: string) => void;
  mono?: boolean;
  // A control beside the input, such as a button that fills it.
  action?: ReactNode;
}) {
  const input = (
    <input
      className={props.mono ? 'gl-input mono' : 'gl-input'}
      id={props.id}
      value={props.value}
      onChange={(e) => props.onChange(e.target.value)}
      aria-describedby={props.hint ? `${props.id}-hint` : undefined}
      spellCheck={false}
      autoComplete="off"
    />
  );
  return (
    <div className="gl-field">
      <label className="gl-field__label" htmlFor={props.id}>
        {props.label} {props.optional && <span data-optional>(optional)</span>}
      </label>
      {props.action ? (
        <div className="field-row">
          {input}
          {props.action}
        </div>
      ) : (
        input
      )}
      {props.hint && (
        <p className="gl-field__hint" id={`${props.id}-hint`}>
          {props.hint}
        </p>
      )}
    </div>
  );
}
