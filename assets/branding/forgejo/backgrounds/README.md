# Brutalist subway welcome backgrounds

Six original backgrounds generated with OpenAI’s built-in image-generation tool
for Forgejo, independently of the soda-bar website artwork. The owner selected a
Wes Anderson-style brutalist subway entrance with a red soda vending machine.

`masters/` keeps the six approved PNGs. `manifest.json` records dimensions,
master/export hashes and WebP encoding. Only the optimized `subway-*.webp` files
enter `forgejo-payload.json`; original masters are never served. To regenerate an
export, run `cwebp -q 86 -mt masters/NAME.png -o subway-NAME.webp` here.

## Generation prompt summaries

- Desktop: frontal, symmetrical empty New York subway entrance; raw concrete
  pillars/beams, off-white square tiles, stainless stair rails, charcoal terrazzo,
  red steel and one illuminated red soda vending machine on the right edge.
  Quiet central wall for an opaque website panel, details around the perimeter.
  Warm practical fluorescent lighting and restrained cinematic realism. No people,
  robots, readable signs, branding, baked-in interface or website rectangle.
- Tablet: recompose the same space in 3:4 portrait, preserving materials and light;
  move the stairs and vending machine toward the outside edges and lower band.
- Mobile: recompose in 9:16 portrait, with a tall quiet central wall, symmetrical
  overhead portal and the vending machine in the lower-right part of the scene.
- Night edits: preserve each approved frame’s geometry, object placement, machine
  and aspect ratio. Remove daylight/ambient fill, deepen stair shadows and gray
  concrete, leaving warm ceiling-light pools and the illuminated soda machine as
  the principal bright elements. No blanket filter, orange wash or new objects.

`home.css` selects mobile below 48rem, tablet from 48rem and landscape desktop from 75rem. Tall desktop windows
keep the tablet composition.
Guest theme selection chooses the day/night exposure. Only the public welcome
page paints this background; native repository and account pages stay unchanged.
The fixed background uses cover sizing anchored at the right to retain the soda
machine. Mobile leaves a lower photographic band after the footer so the machine
can appear when scrolling to the bottom; the foreground covers the quiet center.
