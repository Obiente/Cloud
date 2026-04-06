// Fix vue-tsc template resolution for Nuxt auto-imports
// vue-tsc resolves template identifiers against @vue/runtime-core's ComponentCustomProperties
// but Nuxt's generated types augment 'vue' module. This bridge ensures navigateTo is available.

export {}

declare module '@vue/runtime-core' {
  interface ComponentCustomProperties {
    navigateTo: (...args: any[]) => any
    $router: import('vue-router').Router
    $route: import('vue-router').RouteLocationNormalizedLoaded
  }
}
