/** Tailwind for the public gallery. Colors come from CSS custom properties the
 * server derives from the gallery theme, so one stylesheet serves every
 * white-label deployment. */
module.exports = {
  content: ["../*.templ", "../static/gallery.js"],
  darkMode: ["class", '[data-theme="dark"]'],
  theme: {
    extend: {
      colors: {
        primary: "rgb(var(--c-primary) / <alpha-value>)",
        accent: "rgb(var(--c-accent) / <alpha-value>)",
        surface: "rgb(var(--c-surface) / <alpha-value>)",
        ink: "rgb(var(--c-ink) / <alpha-value>)",
        muted: "rgb(var(--c-muted) / <alpha-value>)",
        line: "rgb(var(--c-line) / <alpha-value>)",
        card: "rgb(var(--c-card) / <alpha-value>)",
      },
      fontFamily: {
        heading: ["var(--font-heading)", "ui-sans-serif", "system-ui", "sans-serif"],
        body: ["var(--font-body)", "ui-sans-serif", "system-ui", "sans-serif"],
      },
      borderRadius: {
        theme: "var(--radius)",
      },
    },
  },
  plugins: [],
};
