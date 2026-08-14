<template>
  <!-- Covers the fitted image inside ZoomableImage's transformed wrapper, so the
       stamp sticks to the photo and zooms/pans with it like the face boxes. -->
  <div class="stamp-stage pointer-events-none absolute inset-0 flex items-center justify-center" data-testid="rejected-stamp">
    <div class="stamp">Rejected</div>
  </div>
</template>

<style scoped>
.stamp-stage {
  /* cqw units below size the stamp relative to the displayed image */
  container-type: size;
}

.stamp {
  font-size: 12cqw;
  line-height: 1;
  font-weight: 900;
  letter-spacing: 0.06em;
  text-transform: uppercase;
  user-select: none;
  color: rgb(220 38 38 / 0.92);
  border: 0.075em solid rgb(220 38 38 / 0.92);
  border-radius: 999px;
  padding: 0.2em 0.55em 0.24em;
  transform: rotate(-12deg);
  filter: blur(0) drop-shadow(0 0.02em 0.08em rgb(0 0 0 / 0.45));
  animation: stamp-slam 0.45s both;
}

@keyframes stamp-slam {
  0% {
    opacity: 0;
    transform: rotate(-12deg) scale(3);
    filter: blur(0.06em) drop-shadow(0 0.1em 0.2em rgb(0 0 0 / 0.25));
    /* accelerate into the hit */
    animation-timing-function: cubic-bezier(0.7, 0, 1, 0.5);
  }
  55% {
    opacity: 1;
    transform: rotate(-12deg) scale(0.9);
    filter: blur(0) drop-shadow(0 0.02em 0.08em rgb(0 0 0 / 0.45));
    /* recoil */
    animation-timing-function: cubic-bezier(0, 0.6, 0.4, 1);
  }
  75% {
    transform: rotate(-12deg) scale(1.05);
  }
  100% {
    opacity: 1;
    transform: rotate(-12deg) scale(1);
    filter: blur(0) drop-shadow(0 0.02em 0.08em rgb(0 0 0 / 0.45));
  }
}

@media (prefers-reduced-motion: reduce) {
  .stamp {
    animation: none;
  }
}
</style>
