# Browser tailnet

[Responsibility map index](README.md) · [Coverage snapshot and limits](../README.md).
Page grouping is navigation; the slice IDs retain their individual review ownership.

<a id="coverage-17f064172432"></a>

<a id="frontendtailnetsoda-tailnet-pagets-1"></a>

## [frontend/tailnet/soda-tailnet-page.ts](../../../../../frontend/tailnet/soda-tailnet-page.ts)



| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 1–234, 249–305, 317–318, 321–412, 458–467, 471–485, 489–515, 559–697, 810–836 | Operator host Tailnet observation/actions; declarations/fields: `TailnetRequestError`, `constructor`, `Scope`, `Confirmation`, `HostResult`, `ExitChoice`, `unknownOutcome`, `observedExitNode`, `advertisedExitPeer`, `exitPeerChoice`, `exitPeerOptions`, `exitNodeMissing`, `hostOutcomeNotice`, `mutationFailureNotice`, `hostWarning`, `formText`, `renderPeerItem`, `SodaTailnet`, `configure`, `hostDirty`, `createRenderRoot`, `connectedCallback`, `disconnectedCallback`, `clearSecrets`, `retire`, `resume`, `current`, `requireCurrent`, `loseAuthorization`, `request`, `resetHost`, `applyObservedSettings`, `refreshFailed`, `refresh`, `clearAuthURL`, `clearHostDraft`, `applyHostMutation`, `applyMutation`, `mutationFailed`, `mutate`, `choose`, `cancel`, `confirm`, `hostActionBody`, `hostAction`, `controlsDisabled`, `refreshObservations`, `discardHostDraft`, `onConfirmKeydown`, `onSignin`, `onAuthentication`, `onLogout`, `onApplyExitNode`, `onApplyAdvertise`, `onExitNodeChange`, `onAllowLANChange`, `onAdvertiseChange`, `renderReconnect`, `renderPending`, `renderHostStatus`, `renderAuthLink`, `renderUnavailableExitOption`, `renderExitPreferences`, `renderPeerList`, `renderHostControls`, `renderHostSection`, `renderAuthorized`, `render`, `mountTailnetPage` |
| [N04](../../slices/networking.md#n04-project-enrollment-policy) / active | 235–248, 306–316, 413–457, 468–470, 516–558, 698–809 | Project enrollment policy/secret and admission controls; declarations/fields: `copyEnrollmentDraft`, `resetEnrollment`, `applyEnrollmentMutation`, `enrollmentSubmitBlocked`, `enrollmentNeedsReview`, `enrollmentPayload`, `submitEnrollment`, `markEnrollmentDirty`, `onModeChange`, `onNetworkInput`, `onTagsInput`, `onPreauthorizedChange`, `enrollmentRevision`, `onCloseAdmission`, `onOfferManagedDefault`, `onKeepDefaultOff`, `renderEnrollmentSummary`, `renderEnrollmentForm`, `renderEnrollmentAdmission`, `renderEnrollmentSection` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 319 | Host mutation response branch; declarations/fields: `applyMutation` |
| [N04](../../slices/networking.md#n04-project-enrollment-policy) / active | 320 | Enrollment policy response branch; declarations/fields: `applyMutation` |
| [N07](../../slices/networking.md#n07-git-endpoint-advertisement) / active | 486–488 | Capability contract/reference; declarations/fields: `onRefreshForgejo` |

<a id="coverage-77a48bf8d0f6"></a>

## [frontend/tailnet/soda-tailnet-response.ts](../../../../../frontend/tailnet/soda-tailnet-response.ts)



| Slice / lifecycle | Current lines | Responsibility and declarations |
| --- | --- | --- |
| [H03](../../slices/shared-supporting-slices.md#h03-encoding-and-parsing) / active | 1–2 | Bounded representation parsing and encoding |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 3–106 | Operator host Tailnet observation/actions; declarations/fields: `hostView` |
| [N04](../../slices/networking.md#n04-project-enrollment-policy) / active | 107–169 | Project enrollment policy/secret and admission controls; declarations/fields: `enrollmentView`, `projectOptions` |
| [N05](../../slices/networking.md#n05-project-tailnet-selection) / active | 170–236 | Project Tailnet selection and revision-bound intent; declarations/fields: `projectView` |
| [N03](../../slices/networking.md#n03-host-tailnet-control) / active | 237–262 | Settings aggregation, native auth URL and host mutation response; declarations/fields: `settingsView`, `authenticationURL`, `hostResult` |
| [N04](../../slices/networking.md#n04-project-enrollment-policy) / active | 263–270 | Enrollment policy mutation response; declarations/fields: `enrollmentResult` |

