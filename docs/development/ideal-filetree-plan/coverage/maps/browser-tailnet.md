# Browser tailnet

[Responsibility map index](README.md) · [Coverage scope](../README.md).

Current responsibility accounting at `519b76bd` (2026-10-07).
One slice owns each named duty; disjoint complete symbols may share an owner.
Compound fields/clauses may share a physical line with distinct selectors.
Disposition concerns the duty, not source validity or installed qualification.

<a id="coverage-bb0e180b1ba3"></a>

## [frontend/tailnet/soda-tailnet-actions.ts](../../../../../frontend/tailnet/soda-tailnet-actions.ts)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–99, 122–224; file scaffold; unknownOutcome; Confirmation; HostResult; hostOutcomeNotice; mutationFailureNotice; hostWarning; formText; ActionsInput; clearHostDraft; applyHostMutation; mutationFailed; mutate; choose; cancel; confirm; hostActionBody; hostAction | [N03](../../slices/networking.md#n03-host-tailnet-control) | retained | Current scaffold duty: file scaffold; 18 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 100–121, 225–274; applyEnrollmentMutation; applyMutation; enrollmentSubmitBlocked; enrollmentNeedsReview; enrollmentPayload; submitEnrollment | [N04](../../slices/networking.md#n04-project-enrollment-policy) | retained | Current declaration duty: applyEnrollmentMutation; 6 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-c5dfd8f4798a"></a>

## [frontend/tailnet/soda-tailnet-confirmation-view.ts](../../../../../frontend/tailnet/soda-tailnet-confirmation-view.ts)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–36; file scaffold; ConfirmationViewInput; renderReconnect; renderPending | [N03](../../slices/networking.md#n03-host-tailnet-control) | retained | Current scaffold duty: file scaffold; 4 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-fb7589dfbe55"></a>

## [frontend/tailnet/soda-tailnet-enrollment-view.ts](../../../../../frontend/tailnet/soda-tailnet-enrollment-view.ts)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–137; file scaffold; EnrollmentViewInput; renderEnrollmentSummary; renderEnrollmentForm; renderEnrollmentAdmission; renderEnrollmentSection | [N04](../../slices/networking.md#n04-project-enrollment-policy) | retained | Current scaffold duty: file scaffold; 6 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-3955de1168bc"></a>

## [frontend/tailnet/soda-tailnet-entry.ts](../../../../../frontend/tailnet/soda-tailnet-entry.ts)

exact-blob current maintained map; full spans retained

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–25; file scaffold; mount | [N03](../../slices/networking.md#n03-host-tailnet-control) | retained | Current scaffold duty: file scaffold; Current declaration duty: mount — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-a516f2c0aad6"></a>

## [frontend/tailnet/soda-tailnet-host-view.ts](../../../../../frontend/tailnet/soda-tailnet-host-view.ts)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–35, 58–183; file scaffold; ExitChoice; advertisedExitPeer; exitPeerChoice; exitPeerOptions; exitNodeMissing; renderPeerItem; renderHostStatus; renderAuthLink; renderUnavailableExitOption; renderExitOption; renderExitPreferences; renderPeerList; renderHostControls; renderHostSection | [N03](../../slices/networking.md#n03-host-tailnet-control) | retained | Current scaffold duty: file scaffold; 15 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 36–57; HostViewInput | [N07](../../slices/networking.md#n07-git-endpoint-advertisement) | retained | Current declaration duty: HostViewInput — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-b46a7d157b5b"></a>

## [frontend/tailnet/soda-tailnet-observation.ts](../../../../../frontend/tailnet/soda-tailnet-observation.ts)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–98, 112–155; file scaffold; TailnetRequestError; constructor; Scope; observedExitNode; ObservationInput; request; resetHost; applyObservedSettings; refreshFailed; refresh | [N03](../../slices/networking.md#n03-host-tailnet-control) | retained | Current scaffold duty: file scaffold; 12 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 99–111; copyEnrollmentDraft; resetEnrollment | [N04](../../slices/networking.md#n04-project-enrollment-policy) | retained | Current declaration duty: copyEnrollmentDraft; Current declaration duty: resetEnrollment; Current method duty: copyEnrollmentDraft — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-17f064172432"></a>
<a id="frontendtailnetsoda-tailnet-pagets-1"></a>

## [frontend/tailnet/soda-tailnet-page.ts](../../../../../frontend/tailnet/soda-tailnet-page.ts)

current source declaration/method inspection; receiver methods normalized by method name; unmatched helpers assigned by inspected consumer duty

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–33, 35–37, 39, 41–43, 45–50, 60–273, 277–291, 295–321, 365–397, 418–448; file scaffold; SodaTailnet; configure; private busy; private stale; private blocked; private sent; private notice; private message; private authURL; private exitDirty; hostDirty; private exitRevision; private policyRevision; private exitNode; private allowLAN; private network; private tags; private preauthorized; private readonly hiddenPage; private readonly shownPage; createRenderRoot; connectedCallback; disconnectedCallback; clearSecrets; retire; resume; current; requireCurrent; loseAuthorization; observationInput; clearAuthURL; actionsInput; controlsDisabled; refreshObservations; discardHostDraft; onConfirmKeydown; onSignin; onAuthentication; onLogout; onApplyExitNode; onApplyAdvertise; onExitNodeChange; onAllowLANChange; onAdvertiseChange; confirmationViewInput; hostViewInput; renderAuthorized; render; mountTailnetPage | [N03](../../slices/networking.md#n03-host-tailnet-control) | retained | Current scaffold duty: file scaffold; 50 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 34, 40, 44, 292–294; private advertiseDirty; private advertiseRevision; private advertise; onRefreshForgejo | [N07](../../slices/networking.md#n07-git-endpoint-advertisement) | retained | Current field duty: private advertiseDirty; 4 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |
| 38, 51–59, 274–276, 322–364, 398–417; private enrollmentDirty; private readonly visibility; private readonly departure; markEnrollmentDirty; onModeChange; onNetworkInput; onTagsInput; onPreauthorizedChange; enrollmentRevision; onCloseAdmission; onOfferManagedDefault; onKeepDefaultOff; enrollmentViewInput | [N04](../../slices/networking.md#n04-project-enrollment-policy) | retained | Current field duty: private enrollmentDirty; 13 named units assigned here; remaining selectors preserve each duty — Current complete declaration/method or file span inspected; owner transferred by exact current symbol match where available, otherwise by traced package consumer and duty. |

<a id="coverage-77a48bf8d0f6"></a>

## [frontend/tailnet/soda-tailnet-response.ts](../../../../../frontend/tailnet/soda-tailnet-response.ts)

exact-blob current maintained map; full spans retained

| Current spans and named units | Owner | Disposition | Responsibility / evidence |
| --- | --- | --- | --- |
| 1–2; whole file | [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) | retained | Bounded representation parsing and encoding — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 3–106, 237–262; hostView; settingsView, authenticationURL, hostResult | [N03](../../slices/networking.md#n03-host-tailnet-control) | retained | Operator host Tailnet observation/actions; declarations/fields: `hostView`; Settings aggregation, native auth URL and host mutation response; declarations/fields: `settingsView`, `authenticationURL`, `hostResult` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 107–169, 263–270; enrollmentView, projectOptions; enrollmentResult | [N04](../../slices/networking.md#n04-project-enrollment-policy) | retained | Project enrollment policy/secret and admission controls; declarations/fields: `enrollmentView`, `projectOptions`; Enrollment policy mutation response; declarations/fields: `enrollmentResult` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
| 170–236; projectView | [N05](../../slices/networking.md#n05-project-tailnet-selection) | retained | Project Tailnet selection and revision-bound intent; declarations/fields: `projectView` — Manifest confirms byte identity; current maintained responsibility map spans reused and clipped only to current file bounds. |
