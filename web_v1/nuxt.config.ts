// https://nuxt.com/docs/api/configuration/nuxt-config
export default defineNuxtConfig({
  devServer: {
    port: 8080,
  },
  loadingIndicator: {
    name: "circle",
    color: "#3B8070",
    background: "white",
  },
  ssr: false,
  devtools: { enabled: true },
  modules: ["@nuxtjs/tailwindcss", "@pinia/nuxt", "@pinia-plugin-persistedstate/nuxt", "nuxt-icon"],
  css: [`assets/dropzone.css`],
  imports: {
    dirs: ["stores"],
  },
});
