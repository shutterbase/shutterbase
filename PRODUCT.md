# Product

## Register

product

## Users

Collaborative photography teams — event and sports photographers, their editors, and
project admins. They shoot in volume (thousands of frames per event across multiple
photographers and cameras), then upload, time-sync, tag, search, and download. Context
is long culling/tagging sessions after a shoot, sometimes on-site on a laptop. The work
is repetitive and high-volume, so speed and legibility matter as much as polish.

## Product Purpose

A collaborative photo pipeline: client-side processing + upload to S3, reconciling each
camera's clock drift via a QR time-sync so frames from different photographers line up
chronologically, then tagging, searching (AND-filtered tags), and bulk download. Success
is moving thousands of photos through the pipeline quickly and reliably, with the images
themselves always the hero.

## Brand Personality

Professional, precise, unobtrusive. A serious tool that gets out of the way of the
photographs. Quietly confident, not loud; warm only through its existing character art
(the potato + ghost), never through decoration. Three words: pro, precise, content-first.

## Anti-references

- Stock Quasar / Material default look (Material blue `#1976D2`, default components, Roboto).
- Generic SaaS dashboards — card-grid-everything, hero-metric tiles, templated admin panels.
- Glassmorphism, decorative blurs, gradient text/buttons.
- Playful / rounded / cute — big radii, pastel friendliness, mascot-forward UI.

## Design Principles

- **The photo is the hero.** UI surfaces recede to near-neutral; chrome is quiet so images
  carry the color and attention. Selection, actions, and state are the only things that earn accent.
- **Pro density and gallery presentation are both first-class.** The photo grid must flex
  from Immich-style near-zero-margin dense/masonry to a relaxed fine-art gallery layout —
  the same content, two legitimate reading speeds.
- **Intentional restraint.** Refined neutral surfaces + one confident, deliberate blue ramp
  (not Material). No decoration for its own sake; every element earns its place.
- **Preserve identity.** The shutterbase logo, icon, wordmark, and the potato/ghost character
  art are kept as-is — they carry the product's existing warmth and recognition.
- **Earn every pixel.** Keyboard-fast, dense where it helps the high-volume workflow, legible
  everywhere, accessible by default.

## Accessibility & Inclusion

WCAG AA: body text ≥4.5:1, large/UI text ≥3:1, including on tinted and accent surfaces.
Dark by default with a real light theme toggle. Every animation has a
`prefers-reduced-motion` alternative. Selection and status never rely on color alone.
