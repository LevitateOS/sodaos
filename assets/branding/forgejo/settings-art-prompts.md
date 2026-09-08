# Settings page illustrations

Generated with the built-in image tool on 2026-09-08 after visually reviewing all twelve existing page illustrations and their prompt records. Existing artwork and canonical logos are unchanged.

## Shared references

Every initial generation used the same ordered references, with explicit roles:

1. [Dashboard](dashboard-papercraft.png): robot face, cardstock material, palette and light.
2. [Notifications](notifications-papercraft.png): compact composition and readable interaction.
3. [New repository](new-repo-papercraft.png): full-body proportions.

For consistency, reuse these originals and the shared prompt below. Change only the scene. Do not mix in the cream-faced Login robot, crescent eyes from Pull requests/Milestones, or the smoother New organization materials. Compare character, apparent scale and actual alpha before accepting new outputs.

## Exact shared prompt

```text
Create one new original Soda OS settings-page header illustration in the existing paper-workshop family. Input images are STYLE REFERENCES only: image 1 Dashboard defines the robot face, matte cardstock construction, palette and lighting; image 2 Notifications defines compact composition and clear interaction; image 3 New repository defines full-body proportions. Image 1 wins for material and face. Do not reproduce their scenes or incidental props. One short friendly cream cardstock robot with rounded rectangular head, inset navy face, two cyan oval dot eyes and a small cyan smile, round cobalt ear caps with restrained cyan inset, short cream limbs, navy joints, navy mitten hands and navy soles. No antenna, chest badge, headphones, individual fingers or crescent eyes. Fine matte paper grain and folded seams, restrained felt, not glossy plastic, metal, clay or plush. One clear action with one main prop/compact assembly, at most two meaningful supporting props. Thin cobalt felt mat. Cream, navy, cobalt and muted mint, only tiny amber accent if needed. Slightly elevated three-quarter view and soft warm upper-left studio lighting with restrained contact shadows. Whole objects visible, centered compact visual weight, roughly 80-85% occupied canvas width, breathing room on all sides. Landscape 1536x1024. TRUE TRANSPARENT PNG ALPHA outside objects and in gaps. Do not paint a checkerboard, solid background, halo or vignette. Preserve cream surfaces. Scene must read at 180px wide. No words, letters, numbers, logos, interface screenshot or code. No extra plants, drinks, laptops or furniture unless scene explicitly calls for it.
```

## Exact scene suffixes and selected outputs

### settings-profile

[settings-profile-papercraft.png](settings-profile-papercraft.png)

```text
Scene: the robot carefully straightens a cream avatar silhouette inside a small freestanding cobalt portrait frame. A single mint backing card gives depth. The framing gesture is the focus. No envelope, mailbox, screen or mirror.
```

Selected generated source: `exec-382a8f6c-fde6-4392-af7b-35d2f7174a52.png`.

### settings-account

[settings-account-papercraft.png](settings-account-papercraft.png)

```text
Scene: the robot places one mint envelope into a compact cream personal mailbox with a cobalt base and a simple avatar-shaped emblem. The delivery gesture is the focus. No bell, sorting trays or piles of mail.
```

Selected generated source: `exec-fee1d496-1cb9-4524-9be0-fc7ddf5b280a.png`.

### settings-appearance

[settings-appearance-papercraft.png](settings-appearance-papercraft.png)

```text
Scene: the robot compares two small cream and navy cardstock swatches beside one standing sample panel split into cream and navy halves with a small mint accent. The comparison gesture is the focus. No paint pots, brush collection or interface controls.
```

Selected generated source: `exec-0b23f17a-2879-48b0-afbf-67a06a495d91.png`.

### settings-security

[settings-security-papercraft.png](settings-security-papercraft.png)

```text
Scene: the robot fastens a cream paper shield onto the front of a closed cobalt storage box with mint trim. The protective gesture is the focus. No key, keyhole, checkmark or success badge.
```

Selected generated source: `exec-59072821-0bc5-41b4-a29d-5d11cf84a84c.png`.

### settings-keys

[settings-keys-papercraft.png](settings-keys-papercraft.png)

```text
Scene: the robot hangs one large mint paper key token beside one cream key token on a compact cobalt rack. Use two simple distinct key silhouettes with no writing. The hanging gesture is the focus. No shield, lockbox or fingerprints.
```

Selected generated source: `exec-0bfeb839-7138-406d-bb8f-58e1b5038bdc.png`.

### settings-applications

[settings-applications-papercraft.png](settings-applications-papercraft.png)

```text
Scene: the robot connects one rounded mint app block to a central cream identity medallion using a short cobalt paper ribbon. One cobalt block is already connected. The medallion has a simple avatar silhouette. The connecting gesture is the focus. No laptop, folder, code, key or lock.
```

Selected generated source: `exec-154453b8-0bda-45ec-8ca6-7cb281d136e4.png`.

## Background corrections

Security, Keys and Applications generated with real alpha on the first pass. Profile, Account and Appearance initially contained painted checkerboards and no alpha. A first edit with the following exact prompt also returned opaque backgrounds and was rejected:

```text
Edit this existing illustration only to remove the painted checkerboard background. Replace ALL checkerboard pixels outside the robot, props and cobalt felt mat with genuine transparent alpha, including gaps between objects. Return a transparent RGBA PNG cutout, landscape 1536x1024. Preserve the exact composition, character, colors, face, paper texture, cream objects, silhouette, scale and margins. Do not add a background, checkerboard, glow, scenery or shadow cloud. Do not make the cream robot or cream props transparent. Change only the background to true alpha.
```

A second edit of each original candidate used this exact prompt and produced the selected cutouts:

```text
Remove the background from this image. Make the background transparent. Keep the robot, all objects, and blue mat exactly as they are. Output an RGBA PNG with an alpha channel.
```

## Checks and integration

All six selected outputs were visually inspected and copied unchanged into this directory. PNG dimensions, RGBA alpha and preservation of the twelve earlier illustrations were checked with read-only image metadata/hash inspection. Background pixels must be assessed using normal alpha compositing; RGB color hidden under zero alpha can look like a glow in viewers that ignore alpha. Native page placement and theme checks belong to the concurrent template integration task; this generation record does not claim those checks passed.

