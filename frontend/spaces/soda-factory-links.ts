import type {FactoryAcceptanceLinks, FactoryWorkLinks} from '@factory/product/frontend/factory/app/navigation.js';
import type {WorkLinks} from '@factory/product/frontend/factory/api/work-response.js';
import type {ActivityItem, FactoryActivity} from '@factory/product/frontend/factory/api/activity-response.js';
import {id} from '@factory/product/frontend/factory/api/models.js';
import {nativePath} from '@factory/product/frontend/factory/shared/format.js';
import {repositoryWorkPath} from './soda-places.js';

export function fountainIssueLink(repositoryPath: string, issue: string): string | undefined {
  if (!nativePath(repositoryPath) || !id(issue)) return;
  return repositoryWorkPath(repositoryPath, 'issues', issue);
}

// Fountain alone owns these native URL shapes; shared views receive callbacks.
export function nativeIssueLink(repositoryPath: string, issue: string): string | undefined {
  if (!nativePath(repositoryPath) || !id(issue)) return undefined;
  return `${repositoryPath}/issues/${issue}`;
}

export function nativePullLink(repositoryPath: string, prNumber: string): string | undefined {
  if (!nativePath(repositoryPath) || !id(prNumber)) return undefined;
  return `${repositoryPath}/pulls/${prNumber}`;
}

/** Adopt only the exact native destination admitted with this repository's activity. */
export function fountainActivityItem(item: ActivityItem, activity: FactoryActivity): {href: string} | undefined {
  const expected =
    item.kind === 'pr_created'
      ? nativePullLink(activity.repository_path, item.pr ?? '')
      : nativeIssueLink(activity.repository_path, item.issue);
  return expected && item.url === expected
    ? {href: item.kind === 'pr_created' ? expected : fountainIssueLink(activity.repository_path, item.issue)!}
    : undefined;
}

export function fountainWorkLinks(repositoryPath: string, recorded?: WorkLinks): FactoryWorkLinks {
  return {
    issue: (number) => {
      const native = nativeIssueLink(repositoryPath, String(number));
      return recorded && recorded.issue_url !== native
        ? recorded.issue_url
        : fountainIssueLink(repositoryPath, String(number));
    },
    pull: (number) => (recorded ? recorded.pull_url : nativePullLink(repositoryPath, String(number))),
  };
}

export function fountainAcceptanceLinks(repositoryPath: string): FactoryAcceptanceLinks {
  return {
    issue: (number) => fountainIssueLink(repositoryPath, number),
    comment: (issue, comment) => {
      const link = fountainIssueLink(repositoryPath, issue);
      return link && id(comment) ? `${link}#issuecomment-${comment}` : undefined;
    },
  };
}
