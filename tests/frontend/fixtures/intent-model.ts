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
    return undefined;
  };
}
