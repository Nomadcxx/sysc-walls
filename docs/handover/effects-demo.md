# Effect demo handover

Finish the showcase in animated WebP: convert the current README GIFs, then capture every active sysc-Go effect without a showcase image. The current GIF assets total about 247 MiB, so use WebP links in both READMEs and remove old GIFs once nothing references them. Keep each WebP at or below 5 MiB where the animation remains readable.

I checked the current sysc-Go source and its local worktrees: there is no all-effects demo command. The old README instruction to run `examples/demo` points to a directory that has never existed in the repository. `syscgo-tui` offers live previews, and sysc-walls has a separate `sysc-walls-daemon -demo` mode; for individual captures, build `syscgo` and run `syscgo -effect <effect-id>`.

## Effects without a current showcase image

Standalone effects:

- `ak-47`
- `great-wave`
- `wasteland`
- `skull`
- `sonar`
- `cracktro`
- `sysc-logo`
- `cross-logo`
- `logo-morph`
- `pour`
- `print`

Effects that use ASCII input:

- `ak-47-text`
- `decrypt`

The existing README already shows `fire`, `matrix`, `rain`, `fireworks`, `beams`, `aquarium`, `fire-text`, `matrix-art`, `rain-art`, `beam-text`, `ring-text`, and `blackhole`. Re-encode these GIFs to WebP; also convert `showcase_tui.gif` because it is embedded in the README. Name the text previews `matrix-art.webp` and `rain-art.webp`, matching their effect IDs.

## Capture requirements

- Build `syscgo` from the sysc-Go repository root and run each effect in a real terminal. Record the live terminal window with GPU Screen Recorder, OBS, or another compositor-aware screen recorder; do not use VHS, terminal emulation, or reconstructed frames.
- Capture at 1920×1080 or the monitor's native resolution when it is higher. Use a readable terminal font and keep desktop chrome and notifications out of frame.
- Use one supported theme per clip to keep file sizes down; `nord` works across the catalog. Capture a full cycle for `great-wave` (8 seconds), `wasteland` (16 seconds), `ak-47` (12 seconds), and `ak-47-text` (14 seconds).
- Pass `-file assets/SYSC.txt` to `ak-47-text`, `decrypt`, `pour`, and `print`.
- Encode the screen recording as animated WebP with FFmpeg's `libwebp_anim` encoder. Start with 15 fps, the `text` preset, and quality 85. Review each file in a browser, check that it loops and the text stays legible, and confirm it stays at or below 5 MiB. Shorten the clip or reduce frame rate before lowering the capture resolution.
- Keep credits visible in the `ak-47` and `ak-47-text` captures. Do not crop the `wasteland` credit.
- Do not add or show `ak-47-rama` or `justice-cross`; both remain excluded until their animations are fixed. `cross-logo` is a separate supported effect and stays in scope.

## Handoff sequence

1. Convert the existing showcase GIFs to `assets/<name>.webp` and verify them in a browser. Record each effect listed above in a real terminal and encode the screen capture as `assets/<effect-id>.webp`.
2. Inspect every recording and WebP for clipping, readable motion, visible credits, animation, and file size. Follow `sysc-Go/demos/README.md` for the screen-capture and encoding steps. Update sysc-Go README links to WebP and remove superseded GIFs after confirming no references remain.
3. Open and merge the sysc-Go PR. Then update the sysc-walls Effects Demo links to `.webp`, following its existing `raw.githubusercontent.com/Nomadcxx/sysc-Go/master/assets/...` links.

Do not change effect implementations or add effect IDs as part of the demo work.
