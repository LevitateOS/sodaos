# How to demo SodaOS and Canlang to founders

Demonstrate the product's actual journey: **Describe → Decide → Build → Use →
Improve**. Start with an empty request and let the product respond to what you
enter. The UI must explain the current state and the next useful action.

Maya, an Operations lead replacing a departmental SaaS workflow, remains a useful
audience persona. Equipment requests are an optional example problem. Neither
the persona nor the example supplies an application, accepted decisions, agent
comments, business records or successful outcomes.

## Walk through the real product

| Moment | What you do | What the product must show |
| --- | --- | --- |
| Describe | Open a new request and describe a process in your own words. | An empty input, a clear submit action and preserved text if setup interrupts you. A blank field explains itself: “Enter a description above to enable saving and exploring ideas.” |
| Decide | Read the actual proposed experience. Plan the draft, then Check walkthrough to ask JEV for review; correct and re-check until clean, then choose Accept and build when satisfied. | Concrete proposed behavior, the real JEV review result, unresolved questions and one clear next action. A proposal is distinct from a checked walkthrough, an accepted decision, or completed implementation. When you edit an accepted direction, the Update proposal button becomes the primary next step. |
| Build | Start the authorized work and follow Feed. | Actual role posts, progress, checks and interruptions. A wait names its cause and the supported next action. Technical controls remain available in detail. |
| Understand | Open a decision or an actual consultation in its post thread. | The agent's actual question to JEV, returned result and separate explanation. The agent's own explanation stays visible above the consultation block. Every non-returned JEV state names its next action in plain words. A failed discussion load offers “Retry loading discussion text” on that post. Other roles' comments appear only when they actually contribute. |
| Find | Filter the same feed to Decisions or Knowledge and open a post. | The same post, comments and history, with its current decision state. |
| Use | Open the app from App when a usable version is actually available. Complete an action yourself. | The real application at its own address in its own browser tab, never embedded in Forge. App shows its version, readiness and an “Open app ↗” link. Each row states the version and origin plus where app accounts and data live, separate from Soda Forge, and whether setup is operator- or administrator-handled. Setup, unavailable and access states explain how to continue. |
| Improve | Comment with a real change you want. Review any proposed scope change and follow its actual work. | A link from the comment to its decision and work. The existing app remains available where supported. |
| Return | Leave for Feed, return to App, and reload. | Your actual saved record and application context. “View project Feed” is available whenever a project is set, and Forge remembers your last project view. An update preserves data only when that behavior has been exercised. |

Your main navigation is **Feed · App · Work**. Feed carries conversations,
decisions and knowledge. App is where you use the result. Work provides progress,
checks and the existing development tools. Routine use should not require
deployment JSON, a terminal or replaying creation.

## Explain the products as they appear

- **SodaOS:** coordinates the work, environments, checks and application access.
  Soda Forge is where you follow the conversation and use the result.
- **Canlang:** expresses and runs the application's workflow, permissions,
  business rules and saved records.
- **JEV from TypeSafe AI:** makes recorded judgments from supplied context.
  The consulting agent explains its resulting choice in the same post thread.

Explain the screen you are actually showing. If work is pending or blocked, show
that state and its recovery action. If no application exists yet, show the empty
App state. Continue only when the actual prerequisites are satisfied.

## Before presenting

1. Check Forge access, provider sign-in and the finite execution limits required
   for the work you intend to start. Account setup does not fill in the request.
2. Verify that the empty, loading, unavailable and permission states are usable.
   Open the actual UI at the presentation's screen size.
3. Pick a recognizable business problem. Leave its input and subsequent choices
   for you to make through the product.
4. Rehearse navigation and recovery. Record observed build duration so you can
   explain timing without promising a fixed completion time.
5. Use actual accounts when demonstrating permissions — a fresh sign-in per
   user through the supported enrollment/role assignment, no shared-session
   shortcuts — actual app actions when demonstrating persistence, and actual
   work evidence when discussing checks.

The UX is ready when someone can follow this journey without an implementation
agent supplying missing clicks or secretly preparing its results.

## Rehearsal (pending, FUX24 — not yet passed)
When the live path works, rehearse exactly this sequence, only where the live
controls actually work, and record what happened — no invented results:
1. Verified result → prepare the exact preview → complete owner setup if
   needed → explicitly start/sign in/use it with isolated data.
2. Separately prepare/setup/start the company version — resume existing
   setup where it exists rather than assuming a repeat — then real company
   use with a retained record.
3. Try the exact verified changed preview at its own address first — before the
   separate company update, never skipping straight to company release —
   then stop/apply/start the company version → reopen the same
   record/history, confirming the data survived.
Note the concrete admitted application interface requirements honestly — the
source/app/model fields, roles, operations, stages and validation behavior
the accepted direction must disclose before acceptance. The fixture contract
lists them for the one specific Northstar rehearsal scenario only: they are
visible requirements of that scenario, not hidden constraints, and never
necessary conditions for an arbitrary user-entered problem. Choose the real
user-entered scenario for the rehearsal and demonstrate only capabilities
actually verified for it; the Northstar contract is optional specific test
coverage — do not present fixture apps/data as generated, do not imply every
user must demo Northstar, and do not claim this path has passed until it
actually has.

## Implementation scope and current limits

The [journey blueprint](../design/factory-demo-journey.md) defines the reusable
UX. The [full frontend-to-demo checklist](../development/factory-remaining-tasks.md#current-product-ux-tasks)
tracks the work and its verification.

The local frontend slice is reviewed, and the installed empty-request screen has
been checked at desktop, tablet and phone sizes. As of 2026-10-27 the matching
backend install is verified live (installed bytes and authenticated served
asset hashes match the reviewed build; FUX07), plus the fux14 planning-recovery
checkpoint (installed bytes and 29/29 served assets verified). One genuine
native planning run has been admitted under it and finished needs-human: the
worker session/turn started and made repeated attempts, but its finish was
refused on the facts[0]=objective byte-equality rule with zero accepted
finish_product_plan. Model turns have therefore run; there is no accepted plan
for the current successor (an earlier run submitted a genuine plan that JEV
reviewed with outcome_gap), and no complete journey. The finish-path prompt
repair is in progress and not installed. The real
generated-app journey, an actual improvement with retained records, rehearsal
and presenter handover must pass before the demo is called ready. No
end-to-end journey is claimed.

The hand-authored Northstar v1/v2 application and its records are integration
test fixtures. They are not the user's starting state, Factory output or
presentation content. Prepared scenario runners are stopped following the
owner's scope correction. No Factory-generated application or completed
Factory build has been established for this demo.

Local browser, PostgreSQL and native transport checks establish only the
behavior they exercised. The complete installed request-to-generated-app journey
remains unqualified. Keep those limits visible when describing readiness.
