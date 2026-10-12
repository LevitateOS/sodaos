import {intentRevision} from '../../../frontend/spaces/sodaspaces-intent-response';
import type {IntentRevision} from '../../../frontend/spaces/sodaspaces-intent-response';

// Synthetic planning peer for browser checks; no AI, funding or native mutations.
export function createIntentModel() {
  const revisions = new Map<string, IntentRevision>();
  const departments: Record<string, unknown>[] = [];
  const allowances = new Map<string, Record<string, unknown>>();
  let serial = 100;
  const nextID = () => 'd' + (++serial).toString(16).padStart(24, '0');
  return (path: string, method: string, body: Record<string, unknown> | null): Response | undefined => {
    if (path.endsWith('/api/factory/departments')) {
      if (method === 'GET') return Response.json(departments);
      if (!body) return new Response(null, {status: 400});
      departments.push({...body, owner: '1'});
      return Response.json(departments.at(-1), {status: 201});
    }
    if (path.endsWith('/api/factory/intents') && method === 'POST' && body) {
      const revision = intentRevision(body);
      revisions.set(revision.id, revision);
      return Response.json({...body, owner: '1', created_unix: 1}, {status: 201});
    }
    if (method === 'GET' && /\/api\/factory\/intents(\?.*)?$/.test(path)) {
      const limit = Number(new URL(path, 'http://fixture').searchParams.get('limit') ?? '20');
      const heads = new Map<string, IntentRevision>();
      for (const revision of revisions.values()) heads.set(revision.intent, revision);
      const all = [...heads.values()].map((revision, index) => ({
        intent: revision.intent,
        revision: revision.id,
        state: revision.state,
        objective: revision.objective,
        repository: revision.repository,
        created_unix: index + 1,
        effective_count: revision.effective_outcomes?.length ?? 0,
        task_count: revision.tasks?.length ?? 0,
      }));
      return Response.json({intents: Number.isFinite(limit) ? all.slice(0, Math.max(0, limit)) : all});
    }
    if (method === 'GET') {
      const allowance = path.match(/\/api\/factory\/intents\/(d[0-9a-f]{24})\/allowance$/);
      if (allowance) {
        const key = allowance[1] || '';
        const stored = key ? allowances.get(key) : undefined;
        if (!stored) return new Response(null, {status: 404});
        return Response.json(stored);
      }
      const overview = path.match(/\/api\/factory\/intents\/(d[0-9a-f]{24})\/overview$/);
      if (overview) {
        const key = overview[1] || '';
        const head = key ? [...revisions.values()].reverse().find((revision) => revision.intent === key) : undefined;
        if (!head || !/^(0|[1-9][0-9]{0,18})$/.test(head.repository)) return new Response(null, {status: 404});
        const ideas = head.ideas ?? [];
        const tasks = head.tasks ?? [];
        const byDecision: Record<string, number> = {};
        for (const idea of ideas) byDecision[idea.decision] = (byDecision[idea.decision] ?? 0) + 1;
        return Response.json({
          intent: head.intent,
          revision: head.id,
          state: head.state,
          direction: {
            objective: head.objective,
            ideas_by_decision: byDecision,
            ideas_by_source: {},
            effective_outcomes: (head.effective_outcomes ?? []).filter((outcome) => /^d[0-9a-f]{24}$/.test(outcome)),
            ideas_total: ideas.length,
            tasks_total: tasks.length,
            unresolved_assumptions: ideas.filter((idea) => idea.assumption && idea.decision === 'pending').length,
            ...(head.approach === undefined ? {} : {approach: head.approach}),
          },
          context: {repository: head.repository},
          generation: 'completed',
          setup: {state: 'ready'},
          work_items: {
            by_state: {delivered: 0, working: 0, waiting: tasks.length, attention: 0},
            total: tasks.length,
            outcomes_total: 0,
            outcomes_evidenced: 0,
          },
          mappings: tasks.map((task, index) => ({
            task_key: task.key,
            issue_id: String(5 + index),
            issue_index: String(5 + index),
            acceptance_id: task.outcomes[0] || 'Demonstrate the selected outcome',
            created_unix: 1728670000,
            effective: true,
          })),
          attention: [],
          observed_unix: 1728670000,
        });
      }
    }
    if (path.endsWith('/api/factory/intent-allowances') && method === 'POST' && body) {
      const intent = String(body.intent || '');
      if (!/^d[0-9a-f]{24}$/.test(intent)) return new Response(null, {status: 400});
      const record = {
        id: String(body.id || nextID()),
        intent,
        unit: 'planning_calls',
        amount: body.amount ?? 0,
        reserved: body.reserved ?? 0,
        consumed: body.consumed ?? 0,
      };
      allowances.set(intent, record);
      return Response.json(record, {status: 201});
    }
    const single = method === 'GET' ? path.match(/\/api\/factory\/intents\/(d[0-9a-f]{24})$/) : null;
    if (single) {
      const stored = [...revisions.values()].reverse().find((revision) => revision.intent === single[1]);
      if (!stored) return new Response(null, {status: 404});
      return Response.json(stored);
    }
    if (path.includes('/api/factory/intents/') && path.endsWith('/explore') && body) {
      const prior = revisions.get(String(body.revision));
      if (!prior) return new Response(null, {status: 409});
      const ideas = prior.ideas.length
        ? prior.ideas
        : [
            {
              id: nextID(),
              description: prior.objective,
              consequence: 'Build a small version of this outcome.',
              assumption: false,
              decision: 'pending' as const,
            },
          ];
      const revision: IntentRevision = {
        ...prior,
        id: nextID(),
        predecessor: prior.id,
        ideas,
        tasks: ideas
          .filter((idea) => idea.decision !== 'decline')
          .map((idea) => ({
            key: nextID(),
            objective: idea.correction || idea.description,
            outcomes: ['Demonstrate the selected outcome'],
            idea_ids: [idea.id],
          })),
      };
      revisions.set(revision.id, revision);
      return Response.json({...revision, owner: '1', created_unix: 1});
    }
    if (path.includes('/api/factory/intents/') && path.endsWith('/generate') && body) {
      const intentID = path.split('/').at(-2);
      const accepted = [...revisions.values()]
        .reverse()
        .find((revision) => revision.intent === intentID && revision.state === 'accepted');
      if (!accepted) return new Response(null, {status: 409});
      return Response.json({intent_id: accepted.intent, revision_id: accepted.id, state: 'completed'});
    }
    return undefined;
  };
}
