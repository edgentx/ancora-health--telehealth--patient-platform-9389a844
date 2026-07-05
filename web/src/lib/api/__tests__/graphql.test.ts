import { graphql, HttpResponse } from 'msw';
import { describe, expect, it } from 'vitest';

import { server } from '@/test/msw/server';

import { ApiError } from '../errors';
import { GraphQLClient } from '../graphql';
import { HttpClient, defaultConfig } from '../http';

const BASE = 'http://api.test';

function makeGraphQL() {
  const http = new HttpClient({ ...defaultConfig(), baseUrl: BASE, retryBackoffMs: 1 });
  return new GraphQLClient(http, '/graphql');
}

describe('GraphQLClient', () => {
  it('returns typed data on success', async () => {
    server.use(
      graphql.query('Ping', () =>
        HttpResponse.json({ data: { ping: 'pong' } }),
      ),
    );
    const data = await makeGraphQL().execute<{ ping: string }>({
      query: 'query Ping { ping }',
      operationName: 'Ping',
    });
    expect(data).toEqual({ ping: 'pong' });
  });

  it('maps a GraphQL errors array to an ApiError', async () => {
    server.use(
      graphql.query('Ping', () =>
        HttpResponse.json({
          errors: [
            { message: 'forbidden', extensions: { code: 'FORBIDDEN', retryable: false } },
          ],
        }),
      ),
    );
    const error = await makeGraphQL()
      .execute({ query: 'query Ping { ping }', operationName: 'Ping' })
      .catch((e: unknown) => e);
    expect(error).toBeInstanceOf(ApiError);
    expect((error as ApiError).kind).toBe('graphql');
    expect((error as ApiError).code).toBe('FORBIDDEN');
    expect((error as ApiError).retryable).toBe(false);
  });
});
