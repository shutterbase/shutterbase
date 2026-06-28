/** @type {import('tailwindcss').Config} */
module.exports = {
  darkMode: "class",
  content: ["./index.html", "./src/**/*.{js,ts,jsx,tsx,vue}", "./node_modules/flowbite/**/*.js"],
  theme: {
    extend: {
      fontFamily: {
        sans: ['Inter', 'ui-sans-serif', 'system-ui', 'sans-serif'],
        mono: ['"JetBrains Mono"', 'ui-monospace', 'SFMono-Regular', 'monospace'],
      },
      scale: {
        flip: "-1",
      },
      colors: {
        // surface — raised panels, cards, inputs. Deliberately NOT named `white`:
        // Quasar ships `.bg-white { background:#fff !important }`, so a Tailwind
        // `bg-white` can never be flipped by a `dark:` variant. These collision-free
        // names own every raised surface across the app. Pair: bg-surface dark:bg-surface-dark.
        surface: {
          DEFAULT: "#ffffff",
          muted: "#f3f5f9",
          dark: "#1b2230",
          "dark-muted": "#11161f",
        },
        // primary — a refined, low-chroma cool-neutral blue. Carries the app's
        // surfaces, borders and structure: dark surfaces stay near-neutral so the
        // photographs supply the color. NOT Material blue.
        primary: {
          50: "#f3f5f9",
          100: "#e7ebf2",
          200: "#cdd4e1",
          300: "#a6b1c7",
          400: "#7986a1",
          500: "#586580",
          600: "#444f66",
          700: "#353e52",
          800: "#222a38",
          900: "#161c27",
          925: "#11161f",
          950: "#0b0f17",
        },
        // accent — the one confident cobalt. Actions, selection, links, focus.
        accent: {
          50: "#eef3ff",
          100: "#dde6ff",
          200: "#c1d2ff",
          300: "#9bb4ff",
          400: "#6f8dff",
          500: "#4569f3",
          600: "#3251e3",
          700: "#2a41c0",
          800: "#27399b",
          900: "#26357a",
          950: "#1a2249",
        },
        success: {
          50: "#eef8f1", 100: "#d6efdd", 200: "#aedfbd", 300: "#79c894",
          400: "#48ad6c", 500: "#2c9152", 600: "#1f7341", 700: "#1c5b35",
          800: "#1a482c", 900: "#163b25", 950: "#0a2113",
        },
        warning: {
          50: "#fbf6ec", 100: "#f5e9cd", 200: "#ead29e", 300: "#dcb45f",
          400: "#d29a3c", 500: "#bd7f2c", 600: "#a26326", 700: "#834a22",
          800: "#6d3e23", 900: "#5d3623", 950: "#351a10",
        },
        error: {
          50: "#fdf3f2", 100: "#fbe4e2", 200: "#f7cdc9", 300: "#f0a8a1",
          400: "#e5766c", 500: "#d44e42", 600: "#bd3729", 700: "#9e2c20",
          800: "#83271e", 900: "#6d2620", 950: "#3b0f0a",
        },
      },
      borderRadius: {
        DEFAULT: "0.375rem",
      },
      boxShadow: {
        panel: "0 1px 2px 0 rgb(0 0 0 / 0.04), 0 1px 3px 0 rgb(0 0 0 / 0.06)",
        "panel-dark": "0 1px 2px 0 rgb(0 0 0 / 0.4), 0 0 0 1px rgb(255 255 255 / 0.04)",
      },
    },
  },
  plugins: [require("flowbite/plugin")],
};
