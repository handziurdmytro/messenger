import { useEffect, useRef, useState, type ReactNode } from "react";
import type { User } from "../api/types";
import { initials, nameOf } from "../lib/format";

const icons: Record<string, ReactNode> = {
  chat: <path d="M20 12a8 8 0 0 1-8 8H4l1.5-3A8 8 0 1 1 20 12Z" />,
  people: (
    <>
      <circle cx="9" cy="8" r="3.2" />
      <path d="M3 20a6 6 0 0 1 12 0M16 5.2a3 3 0 0 1 0 5.6M18 14.5a5.5 5.5 0 0 1 3 5" />
    </>
  ),
  search: (
    <>
      <circle cx="11" cy="11" r="6.5" />
      <path d="m16 16 4.5 4.5" />
    </>
  ),
  plus: <path d="M12 5v14M5 12h14" />,
  send: <path d="M5 12h14m-6-6 6 6-6 6" />,
  back: <path d="m14 6-6 6 6 6" />,
  close: <path d="m6 6 12 12M18 6 6 18" />,
  settings: (
    <>
      <circle cx="12" cy="12" r="3" />
      <path d="M12 3v2.5M12 18.5V21M3 12h2.5M18.5 12H21M5.6 5.6l1.8 1.8M16.6 16.6l1.8 1.8M5.6 18.4l1.8-1.8M16.6 7.4l1.8-1.8" />
    </>
  ),
  logout: <path d="M10 5H5v14h5M14 8l4 4-4 4M18 12H9" />,
  edit: <path d="M5 19h4L19 9l-4-4L5 15v4ZM13 7l4 4" />,
  trash: <path d="M5 7h14M10 7V5h4v2M7 7l1 12h8l1-12" />,
  info: (
    <>
      <circle cx="12" cy="12" r="8.5" />
      <path d="M12 11v5M12 8h.01" />
    </>
  ),
  block: (
    <>
      <circle cx="12" cy="12" r="8.5" />
      <path d="m6 6 12 12" />
    </>
  ),
  check: <path d="m5 12.5 4.5 4.5L19 7" />,
  checks: <path d="m2 12.5 4.5 4.5L16 7M12 16l1 1 9.5-10" />,
  user: (
    <>
      <circle cx="12" cy="8" r="3.5" />
      <path d="M5 20a7 7 0 0 1 14 0" />
    </>
  ),
};

export function Icon({ name, size = 20 }: { name: string; size?: number }) {
  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.6"
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
    >
      {icons[name] ?? icons.chat}
    </svg>
  );
}

export function Avatar({ label, url, size = "md" }: { label: string; url?: string; size?: "sm" | "md" | "lg" }) {
  const [failed, setFailed] = useState(false);
  const safe = !!url && /^https?:\/\//.test(url);
  return (
    <span className={`avatar avatar-${size}`} aria-hidden="true">
      {safe && !failed ? (
        <img src={url} alt="" referrerPolicy="no-referrer" onError={() => setFailed(true)} />
      ) : (
        initials(label)
      )}
    </span>
  );
}

export function UserAvatar({ user, size }: { user?: User | null; size?: "sm" | "md" | "lg" }) {
  return <Avatar label={nameOf(user)} url={user?.deleted ? undefined : user?.avatar_url} size={size} />;
}

export function ErrorText({ children }: { children: ReactNode }) {
  return (
    <p className="error" role="alert">
      {children}
    </p>
  );
}

export function IconButton({
  icon,
  label,
  onClick,
  disabled,
  danger,
}: {
  icon: string;
  label: string;
  onClick: () => void;
  disabled?: boolean;
  danger?: boolean;
}) {
  return (
    <button
      type="button"
      className={`icon-button${danger ? " danger" : ""}`}
      aria-label={label}
      title={label}
      onClick={onClick}
      disabled={disabled}
    >
      <Icon name={icon} size={18} />
    </button>
  );
}

export function Modal({ title, children, close }: { title: string; children: ReactNode; close: () => void }) {
  const ref = useRef<HTMLDialogElement>(null);
  useEffect(() => {
    const dialog = ref.current;
    dialog?.showModal();
    return () => dialog?.close();
  }, []);
  return (
    <dialog
      ref={ref}
      aria-label={title}
      onCancel={(event) => {
        event.preventDefault();
        close();
      }}
      onClick={(event) => {
        if (event.target === event.currentTarget) close();
      }}
    >
      <header className="modal-head">
        <h2>{title}</h2>
        <IconButton icon="close" label="Close" onClick={close} />
      </header>
      <div className="modal-body">{children}</div>
    </dialog>
  );
}
