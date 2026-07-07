// Ancora async data service — the backend the frontend's async client expects.
//
//   • GraphQL queries (POST /graphql)          — messageThreads, threadMessages.
//   • REST (GET/POST /api/engagement/*)         — notifications, start thread,
//     post message; a post publishes to subscribers.
//   • WebSocket (/realtime, Ancora envelope)    — {type,channel,payload} frames;
//     new messages push on the `messaging` channel (no polling).
//
// Seeded with sample telehealth data (design/demo). Real domain data lands with
// the Ancora backend build.

import { createServer } from 'node:http'
import { createYoga, createSchema } from 'graphql-yoga'
import { WebSocketServer } from 'ws'

// ── Seed state ────────────────────────────────────────────────────────────────
const THREADS = [
  { id: 'th.1', status: 'open', patientId: 'pat.dana', careTeamMemberIds: ['prov.rivera'], subject: 'Post-visit follow-up: lab results', version: 1 },
  { id: 'th.2', status: 'open', patientId: 'pat.dana', careTeamMemberIds: ['prov.rivera', 'sched.cortez'], subject: 'Rescheduling next week’s appointment', version: 1 },
  { id: 'th.3', status: 'resolved', patientId: 'pat.dana', careTeamMemberIds: ['prov.rivera'], subject: 'Prescription refill — lisinopril', version: 2 },
]
const MESSAGES = {
  'th.1': [
    { id: 'm.1', threadId: 'th.1', authorId: 'prov.rivera', body: 'Your labs came back normal — cholesterol is down from last quarter. Nice work.', sentAt: '2026-07-06T14:02:00Z' },
    { id: 'm.2', threadId: 'th.1', authorId: 'pat.dana', body: 'That’s great to hear. Anything I should keep doing?', sentAt: '2026-07-06T14:10:00Z' },
    { id: 'm.3', threadId: 'th.1', authorId: 'prov.rivera', body: 'Keep up the walking routine and we’ll recheck in 3 months.', sentAt: '2026-07-06T14:12:00Z' },
  ],
  'th.2': [
    { id: 'm.4', threadId: 'th.2', authorId: 'pat.dana', body: 'Can we move Thursday’s visit to the afternoon?', sentAt: '2026-07-06T09:30:00Z' },
    { id: 'm.5', threadId: 'th.2', authorId: 'sched.cortez', body: 'We have 3:15pm open Thursday — booked it for you.', sentAt: '2026-07-06T09:45:00Z' },
  ],
  'th.3': [
    { id: 'm.6', threadId: 'th.3', authorId: 'pat.dana', body: 'Requesting a refill on my lisinopril.', sentAt: '2026-07-02T11:00:00Z' },
    { id: 'm.7', threadId: 'th.3', authorId: 'prov.rivera', body: 'Sent to your pharmacy — should be ready today.', sentAt: '2026-07-02T11:20:00Z' },
  ],
}
const NOTIFICATIONS = [
  { id: 'n.1', kind: 'message', title: 'New message from Dr. Rivera', unread: true, at: '2026-07-06T14:12:00Z' },
  { id: 'n.2', kind: 'appointment', title: 'Appointment confirmed: Thu 3:15pm', unread: true, at: '2026-07-06T09:45:00Z' },
  { id: 'n.3', kind: 'billing', title: 'Statement available', unread: false, at: '2026-07-05T08:00:00Z' },
]

// ── Realtime hub (Ancora envelope: {type, channel, payload}) ──────────────────
const sockets = new Set()
function broadcast(channel, type, payload) {
  const frame = JSON.stringify({ type, channel, payload })
  for (const ws of sockets) { try { ws.send(frame) } catch { /* dropped */ } }
}

// ── GraphQL ───────────────────────────────────────────────────────────────────
const schema = createSchema({
  typeDefs: /* GraphQL */ `
    type MessageThread { id: ID!, status: String!, patientId: ID!, careTeamMemberIds: [ID!]!, subject: String!, version: Int! }
    type Message { id: ID!, threadId: ID!, authorId: ID!, body: String!, sentAt: String! }
    type Query { messageThreads: [MessageThread!]!, threadMessages(threadId: ID!): [Message!]! }
  `,
  resolvers: {
    Query: {
      messageThreads: () => THREADS,
      threadMessages: (_, { threadId }) => MESSAGES[threadId] || [],
    },
  },
})

const yoga = createYoga({
  schema,
  graphqlEndpoint: '/graphql',
  cors: { origin: ['https://dev.anco.vforce360.ai'], credentials: true },
  landingPage: false,
})

const server = createServer(async (req, res) => {
  const url = new URL(req.url, 'http://x')
  res.setHeader('Access-Control-Allow-Origin', 'https://dev.anco.vforce360.ai')
  res.setHeader('Access-Control-Allow-Credentials', 'true')
  res.setHeader('Access-Control-Allow-Methods', 'GET,POST,PUT,OPTIONS')
  res.setHeader('Access-Control-Allow-Headers', 'Content-Type,Accept,x-ancora-role,x-ancora-user')
  if (req.method === 'OPTIONS') { res.writeHead(204); return res.end() }
  if (url.pathname === '/health') { res.writeHead(200); return res.end('ok\n') }

  const json = (v) => { res.writeHead(200, { 'Content-Type': 'application/json' }); res.end(JSON.stringify(v)) }

  if (req.method === 'GET' && url.pathname === '/api/engagement/notifications') return json(NOTIFICATIONS)

  if (req.method === 'POST' && url.pathname === '/api/engagement/threads') {
    const body = await readJson(req)
    const th = { id: 'th.' + (THREADS.length + 1), status: 'open', patientId: body.patientId ?? 'pat.dana', careTeamMemberIds: body.careTeamMemberIds ?? ['prov.rivera'], subject: body.subject ?? 'New conversation', version: 1 }
    THREADS.unshift(th); MESSAGES[th.id] = []
    broadcast('messaging', 'thread.created', th)
    return json(th)
  }
  // Post a message into a thread — publishes to the messaging channel (live).
  const pm = url.pathname.match(/^\/api\/engagement\/threads\/([^/]+)\/messages$/)
  if (req.method === 'POST' && pm) {
    const threadId = pm[1]
    const body = await readJson(req)
    const msg = { id: 'm.' + Math.random().toString(36).slice(2, 8), threadId, authorId: body.authorId ?? 'pat.dana', body: body.body ?? '', sentAt: new Date(0).toISOString() }
    ;(MESSAGES[threadId] = MESSAGES[threadId] || []).push(msg)
    broadcast('messaging', 'message.created', msg) // ← live push, no polling
    return json(msg)
  }
  if (url.pathname.startsWith('/api/')) return json([]) // benign default for other REST reads

  return yoga(req, res)
})

// Ancora's RealtimeClient connects and receives channel-tagged frames.
const wss = new WebSocketServer({ server, path: '/realtime' })
wss.on('connection', (ws) => {
  sockets.add(ws)
  ws.send(JSON.stringify({ type: 'connected', channel: 'system', payload: { ok: true } }))
  ws.on('close', () => sockets.delete(ws))
  ws.on('message', () => { /* client subscribe frames are advisory here */ })
})

function readJson(req) {
  return new Promise((resolve) => { let b = ''; req.on('data', (d) => (b += d)); req.on('end', () => { try { resolve(JSON.parse(b || '{}')) } catch { resolve({}) } }) })
}

const PORT = process.env.PORT || 8080
server.listen(PORT, '0.0.0.0', () => console.log(`Ancora data service on :${PORT} (GraphQL /graphql, REST /api/engagement, realtime ws /realtime)`))
