import { useEffect, useState, type ReactNode } from "react";
import { errorText } from "../api/client";
import type { User } from "../api/types";
import { searchUsers } from "../api/users";
import { nameOf } from "../lib/format";
import { ErrorText, Icon, UserAvatar } from "./ui";

export function PeoplePicker({
  me,
  contacts,
  exclude = [],
  actions,
}: {
  me: User;
  contacts: User[];
  exclude?: string[];
  actions: (user: User) => ReactNode;
}) {
  const [query, setQuery] = useState("");
  const [results, setResults] = useState<User[]>([]);
  const [searching, setSearching] = useState(false);
  const [error, setError] = useState("");

  useEffect(() => {
    const term = query.trim();
    if (!term) return;
    const controller = new AbortController();
    const timer = setTimeout(async () => {
      setSearching(true);
      try {
        const users = await searchUsers(term, controller.signal);
        setResults(users);
        setError("");
      } catch (err) {
        if (!controller.signal.aborted) setError(errorText(err));
      } finally {
        if (!controller.signal.aborted) setSearching(false);
      }
    }, 250);
    return () => {
      controller.abort();
      clearTimeout(timer);
    };
  }, [query]);

  const source = query.trim() ? results : contacts;
  const people = source.filter((user) => user.id !== me.id && !user.deleted && !exclude.includes(user.id));

  return (
    <div className="picker">
      <label className="search">
        <Icon name="search" size={16} />
        <input
          aria-label="Search people"
          placeholder="Search by name or email"
          value={query}
          onChange={(event) => setQuery(event.target.value)}
        />
      </label>
      {error && <ErrorText>{error}</ErrorText>}
      <p className="section-label">{query.trim() ? "Search results" : "Contacts"}</p>
      <ul className="people">
        {searching && !people.length && <li className="muted">Searching…</li>}
        {!searching && !people.length && <li className="muted">{query.trim() ? "Nobody found." : "No contacts yet."}</li>}
        {people.map((user) => (
          <li key={user.id} className="person">
            <UserAvatar user={user} size="sm" />
            <span className="person-name">
              <strong>{nameOf(user)}</strong>
              <small>{user.email}</small>
            </span>
            <span className="person-actions">{actions(user)}</span>
          </li>
        ))}
      </ul>
    </div>
  );
}
