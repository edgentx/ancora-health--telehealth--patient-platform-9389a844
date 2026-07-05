'use client';

import { useState } from 'react';

import {
  useMessageThreads,
  usePostMessage,
  useThreadMessages,
  type MessageThread,
} from '@/lib/api';
import { formatDateTime } from '@/lib/format';

import { QueryState } from './query-state';
import { StatusPill } from './status-pill';

/**
 * Secure patient/care-team messaging, shared by the patient portal and the
 * provider console. Threads come over GraphQL and messages over GraphQL too
 * (S-80's dual surface); posting a message optimistically marks it as "mine" and
 * relies on the hook's cache invalidation to reconcile with the server copy.
 *
 * The realtime push (new inbound messages over the WebSocket) is wired by the
 * `realtime` hooks elsewhere; here we render the authoritative cache so a live
 * push and a refetch converge on one list.
 */
export function SecureMessaging() {
  const threadsQuery = useMessageThreads();
  const [activeId, setActiveId] = useState<string | null>(null);

  return (
    <section>
      <div className="section-head">
        <div>
          <h1 className="page-heading">Secure messages</h1>
          <p className="page-subheading" style={{ margin: 0 }}>
            Encrypted conversations with your care team.
          </p>
        </div>
      </div>

      <QueryState
        query={threadsQuery}
        isEmpty={(threads) => threads.length === 0}
        loadingLabel="Loading conversations…"
        emptyLabel="You have no secure messages yet."
      >
        {(threads) => {
          const active =
            threads.find((t) => t.id === activeId) ?? threads[0] ?? null;
          return (
            <div className="split">
              <ul className="list" aria-label="Conversations">
                {threads.map((thread) => (
                  <li key={thread.id}>
                    <button
                      type="button"
                      className={`list__row${active?.id === thread.id ? ' is-active' : ''}`}
                      style={{ width: '100%', cursor: 'pointer', textAlign: 'left' }}
                      aria-pressed={active?.id === thread.id}
                      onClick={() => setActiveId(thread.id)}
                    >
                      <span className="list__primary">{thread.subject}</span>
                      <StatusPill status={thread.status} />
                    </button>
                  </li>
                ))}
              </ul>
              {active ? (
                <ThreadPane thread={active} />
              ) : (
                <div className="qs qs--empty">Select a conversation.</div>
              )}
            </div>
          );
        }}
      </QueryState>
    </section>
  );
}

function ThreadPane({ thread }: { thread: MessageThread }) {
  const messagesQuery = useThreadMessages(thread.id);
  const post = usePostMessage();
  const [draft, setDraft] = useState('');
  // Ids of messages we sent in this session, so they render right-aligned even
  // before the refetch confirms them.
  const [mine, setMine] = useState<Set<string>>(() => new Set());

  const canSend = draft.trim().length > 0 && !post.isPending;

  function send() {
    const body = draft.trim();
    if (!body) return;
    post.mutate(
      { threadId: thread.id, body },
      {
        onSuccess: (message) => {
          setMine((prev) => new Set(prev).add(message.id));
          setDraft('');
        },
      },
    );
  }

  return (
    <div className="card">
      <div className="section-head" style={{ marginBottom: 0 }}>
        <h2 style={{ margin: 0, fontSize: '1.125rem' }}>{thread.subject}</h2>
        <StatusPill status={thread.status} />
      </div>

      <QueryState
        query={messagesQuery}
        isEmpty={(messages) => messages.length === 0}
        loadingLabel="Loading messages…"
        emptyLabel="No messages in this thread yet — say hello."
      >
        {(messages) => (
          <div className="messages" aria-label="Messages">
            {messages.map((message) => (
              <div
                key={message.id}
                className={`message${mine.has(message.id) ? ' message--mine' : ''}`}
              >
                <div>{message.body}</div>
                <div className="message__meta">{formatDateTime(message.sentAt)}</div>
              </div>
            ))}
          </div>
        )}
      </QueryState>

      {post.isError ? (
        <p className="banner banner--conflict" role="alert" style={{ marginTop: 'var(--space-3)' }}>
          Your message couldn’t be sent. Please try again.
        </p>
      ) : null}

      <div className="composer">
        <input
          className="input"
          aria-label="Message"
          placeholder="Write a secure message…"
          value={draft}
          onChange={(e) => setDraft(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === 'Enter' && canSend) send();
          }}
        />
        <button
          type="button"
          className="btn btn--primary"
          disabled={!canSend}
          onClick={send}
        >
          {post.isPending ? 'Sending…' : 'Send'}
        </button>
      </div>
    </div>
  );
}
