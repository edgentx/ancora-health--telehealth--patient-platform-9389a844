/**
 * Patient-engagement hooks (secure messaging, notifications).
 *
 * Threads and messages are served over GraphQL here to demonstrate the client's
 * dual REST/GraphQL surface behind one hook API; posting a message invalidates
 * the thread's message list so the new message appears immediately. The realtime
 * message/notification push is wired separately in `./realtime`.
 */
'use client';

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import { apiClient } from '../client';
import type {
  Message,
  MessageThread,
  Notification,
  PostMessageRequest,
  StartMessageThreadRequest,
} from '../models/engagement';
import { queryKeys } from './keys';

const THREADS_QUERY = /* GraphQL */ `
  query Threads {
    messageThreads {
      id
      status
      patientId
      careTeamMemberIds
      subject
      version
    }
  }
`;

const MESSAGES_QUERY = /* GraphQL */ `
  query ThreadMessages($threadId: ID!) {
    threadMessages(threadId: $threadId) {
      id
      threadId
      authorId
      body
      sentAt
    }
  }
`;

export function useMessageThreads() {
  return useQuery({
    queryKey: queryKeys.engagement.threads(),
    queryFn: async () => {
      const data = await apiClient.graphql.execute<{ messageThreads: MessageThread[] }>({
        query: THREADS_QUERY,
        operationName: 'Threads',
      });
      return data.messageThreads;
    },
  });
}

export function useThreadMessages(threadId: string) {
  return useQuery({
    queryKey: queryKeys.engagement.messages(threadId),
    queryFn: async () => {
      const data = await apiClient.graphql.execute<
        { threadMessages: Message[] },
        { threadId: string }
      >({
        query: MESSAGES_QUERY,
        operationName: 'ThreadMessages',
        variables: { threadId },
      });
      return data.threadMessages;
    },
    enabled: threadId.length > 0,
  });
}

export function useNotifications() {
  return useQuery({
    queryKey: queryKeys.engagement.notifications(),
    queryFn: () =>
      apiClient.rest.get<Notification[]>('/api/engagement/notifications'),
  });
}

export function useStartMessageThread() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: StartMessageThreadRequest) =>
      apiClient.rest.post<MessageThread>('/api/engagement/threads', input),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: queryKeys.engagement.threads() });
    },
  });
}

export function usePostMessage() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: PostMessageRequest) =>
      apiClient.rest.post<Message>(
        `/api/engagement/threads/${input.threadId}/messages`,
        input,
      ),
    onSuccess: (message) => {
      void qc.invalidateQueries({
        queryKey: queryKeys.engagement.messages(message.threadId),
      });
    },
  });
}
