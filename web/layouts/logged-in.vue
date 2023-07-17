<template>
  <div class="flex flex-col h-screen justify-between">
    <AppHeader />
    <div class="container mx-auto px-4">
      <slot />
    </div>
    <AppFooter />
  </div>
</template>
