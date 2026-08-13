import confetti from "canvas-confetti";

// A generous celebration: a centre burst plus staggered volleys from both bottom
// corners. zIndex sits above the headlessui dialog (z-10) so the confetti flies
// over the modal; disableForReducedMotion honours the OS accessibility setting.
export function celebrate(): void {
  const defaults = { disableForReducedMotion: true, zIndex: 2000, ticks: 300 };
  confetti({ ...defaults, particleCount: 180, spread: 110, startVelocity: 45, origin: { x: 0.5, y: 0.6 } });
  for (const delay of [200, 400, 600]) {
    setTimeout(() => {
      confetti({ ...defaults, particleCount: 90, angle: 60, spread: 70, origin: { x: 0, y: 0.9 } });
      confetti({ ...defaults, particleCount: 90, angle: 120, spread: 70, origin: { x: 1, y: 0.9 } });
    }, delay);
  }
}
