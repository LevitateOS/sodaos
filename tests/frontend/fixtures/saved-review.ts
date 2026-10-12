export const hflowCurrent = `{
  "intent": "d000000000000000000000000",
  "head_revision": "d222222222222222222222222",
  "status": "current",
  "review": {
    "intent": "d000000000000000000000000",
    "basis_revision": "d111111111111111111111111",
    "result_revision": "d222222222222222222222222",
    "content": {
      "producer": "soda-authored-v1",
      "supported": true,
      "intent": "d000000000000000000000000",
      "revision": "d111111111111111111111111",
      "objective": "Time-off requests my manager can approve",
      "facts": ["Time-off requests my manager can approve"],
      "alternatives": [
        {
          "key": "single_person",
          "name": "Small single-person workflow",
          "audience": "One person completing their own request without a separate decider.",
          "summary": "The person records what they need and sees the recorded result directly.",
          "steps": [
            {"actor": "The person", "action": "describes what they need", "result": "The request is recorded"},
            {"actor": "The person", "action": "confirms the recorded request", "result": "The request counts as submitted"}
          ],
          "visibility": "Only the person sees their own request unless the description says otherwise.",
          "consequences": "No waiting on another person, and no second check before the request counts.",
          "evidence": "Authored pattern; no external research was performed.",
          "outcomes": ["The person completes a time-off request"]
        },
        {
          "key": "shared_review",
          "name": "Shared review workflow",
          "audience": "A person making a request and another person deciding on it.",
          "summary": "The person submits a request; a reviewer decides, and the person sees the decision.",
          "steps": [
            {"actor": "The person", "action": "describes what they need and submits the request", "result": "The reviewer can see the request"},
            {"actor": "The reviewer", "action": "makes a decision", "result": "The person sees the decision", "branches": [
              {"outcome": "Approved", "result": "The person sees confirmation"},
              {"outcome": "Declined", "result": "The person sees the reason and can revise the request"}
            ]}
          ],
          "visibility": "The person and the reviewer see the request; who else can see it is not established.",
          "consequences": "A decision takes reviewer time, and a declined request needs revision.",
          "evidence": "Authored pattern; no external research was performed.",
          "outcomes": ["The person completes a time-off request"]
        }
      ],
      "excluded": ["d666666666666666666666666"],
      "exclusions": [
        {"idea": "d666666666666666666666666", "description": "HR reviews every request", "consequence": "Requests wait on HR", "provenance": "human"}
      ],
      "unknowns": ["Who can see a request beyond the people named in the walkthrough is not established."]
    },
    "challenges": [
      {"idea": "d444444444444444444444444", "assumption": "Everyone can see each request", "consequence": "Dates are exposed", "recommendation": "Confirm whether the visibility holds, or explain the correction."}
    ],
    "settled": ["d555555555555555555555555"],
    "recommendation": {
      "producer": "soda-authored-v1",
      "intent": "d000000000000000000000000",
      "revision": "d111111111111111111111111",
      "objective": "Time-off requests my manager can approve",
      "recommended": "shared_review",
      "alternative": {
        "key": "shared_review",
        "name": "Shared review workflow",
        "audience": "A person making a request and another person deciding on it.",
        "summary": "The person submits a request; a reviewer decides, and the person sees the decision.",
        "steps": [
          {"actor": "The person", "action": "describes what they need and submits the request", "result": "The reviewer can see the request"},
          {"actor": "The reviewer", "action": "makes a decision", "result": "The person sees the decision", "branches": [
            {"outcome": "Approved", "result": "The person sees confirmation"},
            {"outcome": "Declined", "result": "The person sees the reason and can revise the request"}
          ]}
        ],
        "visibility": "The person and the reviewer see the request; who else can see it is not established.",
        "consequences": "A decision takes reviewer time, and a declined request needs revision.",
        "evidence": "Authored pattern; no external research was performed.",
        "outcomes": ["The person completes a time-off request"]
      },
      "outcomes": ["The person completes a time-off request"],
      "constraints": [],
      "excluded": ["d666666666666666666666666"],
      "milestones": [
        {"name": "The person: describes what they need and submits the request", "result": "The reviewer can see the request"},
        {"name": "The reviewer: makes a decision", "result": "The person sees the decision"},
        {"name": "If Approved", "result": "The person sees confirmation"},
        {"name": "If Declined", "result": "The person sees the reason and can revise the request"}
      ],
      "advice": "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
    },
    "consistency": {
      "producer": "soda-authored-v1",
      "intent": "d000000000000000000000000",
      "revision": "d111111111111111111111111",
      "mismatches": [],
      "confidence": 0.9,
      "advice": "abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789"
    },
    "saved_unix": 1791824831
  }
}`;
