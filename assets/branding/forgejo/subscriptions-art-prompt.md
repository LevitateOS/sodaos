# Notification subscriptions illustration

A robot bookmarking conversation cards distinguishes following issue/PR discussions from receiving inbox messages. The existing page intro provides suitable space above the native filters. Selection uses the template's existing `.Status == 1` subscriptions branch; Watching retains its prior artwork until separately reviewed.

Built-in generator, 2026-09-08. Ordered references: Dashboard (robot/material/light), Notifications (compactness), New repository (proportions). First output `exec-68429285-90c7-4735-97c5-c50de5513244.png` was RGB with a painted checkerboard and was rejected. Selected cutout `exec-ba537aff-d48d-494d-87a7-efef846181d4.png` was copied unchanged to [subscriptions-papercraft.png](subscriptions-papercraft.png), visually inspected and verified as 1536×1024 RGBA with transparent corners.

## Exact generation prompt

```text
Create one new Soda OS website header illustration for notification subscriptions, meaning keeping track of selected issue and pull-request conversations. Use the supplied images as style references only: Dashboard for character, matte paper material, palette and lighting; Notifications for compact composition; New repository for short full-body proportions. One friendly short cream cardstock robot with rounded rectangular head, navy inset face, two cyan oval dot eyes and a small cyan smile, round cobalt ear caps, navy mitten hands, joints and soles. No antenna, chest badge, headphones or fingers. Scene: the robot gently clips a large mint ribbon bookmark onto the edge of a cream upright conversation card. The card has two simple overlapping navy and mint speech-bubble silhouettes, no words. One second smaller cream conversation card stands behind it in a compact cobalt holder. The bookmarking action is clearly visible. Thin cobalt felt mat, subtle matte paper grain and folded seams, slightly elevated three-quarter view, warm soft studio light, cream/navy/cobalt/mint palette. One robot and the small conversation-card assembly only. No envelopes, sorting trays, bell, eye, binoculars, folders, laptop, plants or drink. No text, letters, numbers, logos, notification count or success checkmarks. Whole objects visible, comfortable margins, compact scene readable at 180px wide. Landscape 1536x1024 PNG with transparent background.
```

## Exact correction prompt

```text
Remove the background from this image. Make the background transparent. Keep the robot, all objects, and blue mat exactly as they are. Output an RGBA PNG with an alpha channel.
```

Native evidence and remaining limits are in the [checklist](../../../docs/page-illustration-checklist.md). No subscription state was changed.
