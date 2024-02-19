module.exports = {
  //...
  plugins: [require("daisyui")],

  daisyui: {
    themes: [
      {
        shutterbaseLight: {
          primary: "#051A37",
          secondary: "#37465D",
          accent: "#9DC02E",
          neutral: "#F2F2F2",
          "base-100": "#F2F2F2",
          info: "#0092d6",
          success: "#6cb288",
          warning: "#daad58",
          error: "#ab3d30",
        },
      },
      // "dark",
    ],
    darkTheme: "shutterbaseLight", // name of one of the included themes for dark mode
    base: true, // applies background color and foreground color for root element by default
    styled: true, // include daisyUI colors and design decisions for all components
    utils: true, // adds responsive and modifier utility classes
    rtl: false, // rotate style direction from left-to-right to right-to-left. You also need to add dir="rtl" to your html tag and install `tailwindcss-flip` plugin for Tailwind CSS.
    prefix: "", // prefix for daisyUI classnames (components, modifiers and responsive class names. Not colors)
    logs: true, // Shows info about daisyUI version and used config in the console when building your CSS
  },
};
