// https://nuxt.com/docs/api/configuration/nuxt-config
export default defineNuxtConfig({
  devServer: {
    port: 8080,
  },
  ssr: false,
  debug: true,
  devtools: { enabled: true },
  modules: ["@nuxtjs/tailwindcss", "@pinia/nuxt", "@pinia-plugin-persistedstate/nuxt", "nuxt-icon"],
  css: [`assets/dropzone.css`],
  imports: {
    dirs: ["stores"],
  },
});
