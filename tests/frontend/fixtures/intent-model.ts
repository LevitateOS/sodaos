import {intentRevision} from '../../../frontend/spaces/sodaspaces-intent-response';
import type {IntentRevision} from '../../../frontend/spaces/sodaspaces-intent-response';

// Synthetic planning peer for browser checks; no AI, funding or native mutations.
export function createIntentModel() {
  const revisions = new Map<string, IntentRevision>();
  const departments: Record<string, unknown>[] = [];
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
    if (method === 'GET' && path.split('?')[0]?.endsWith('/api/factory/intents')) {
      const heads = new Map<string, IntentRevision>();
      for (const revision of revisions.values()) heads.set(revision.intent, revision);
      return Response.json({
        intents: [...heads.values()].map((revision, index) => ({
          intent: revision.intent,
          revision: revision.id,
          state: revision.state,
          objective: revision.objective,
          repository: revision.repository,
          created_unix: index + 1,
          effective_count: revision.effective_outcomes?.length ?? 0,
          task_count: revision.tasks?.length ?? 0,
        })),
      });
    }
    const single = method === 'GET' ? path.match(/\/api\/factory\/intents\/(d[0-9a-f]{24})$/) : null;
    if (single) {
      const stored = [...revisions.values()].find((revision) => revision.intent === single[1]);
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
