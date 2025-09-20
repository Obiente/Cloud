
import type { DefineComponent, SlotsType } from 'vue'
type IslandComponent<T extends DefineComponent> = T & DefineComponent<{}, {refresh: () => Promise<void>}, {}, {}, {}, {}, {}, {}, {}, {}, {}, {}, SlotsType<{ fallback: { error: unknown } }>>

type HydrationStrategies = {
  hydrateOnVisible?: IntersectionObserverInit | true
  hydrateOnIdle?: number | true
  hydrateOnInteraction?: keyof HTMLElementEventMap | Array<keyof HTMLElementEventMap> | true
  hydrateOnMediaQuery?: string
  hydrateAfter?: number
  hydrateWhen?: boolean
  hydrateNever?: true
}
type LazyComponent<T> = (T & DefineComponent<HydrationStrategies, {}, {}, {}, {}, {}, {}, { hydrated: () => void }>)


export const AppHeader: typeof import("../components/app/AppHeader.vue")['default']
export const AppSidebar: typeof import("../components/app/AppSidebar.vue")['default']
export const AppNavigationLink: typeof import("../components/app/NavigationLink.vue")['default']
export const AppUserProfile: typeof import("../components/app/UserProfile.vue")['default']
export const OuiAvatar: typeof import("../components/oui/Avatar.vue")['default']
export const OuiBadge: typeof import("../components/oui/Badge.vue")['default']
export const OuiButton: typeof import("../components/oui/Button.vue")['default']
export const OuiCard: typeof import("../components/oui/Card.vue")['default']
export const OuiCardBody: typeof import("../components/oui/CardBody.vue")['default']
export const OuiCardFooter: typeof import("../components/oui/CardFooter.vue")['default']
export const OuiCardHeader: typeof import("../components/oui/CardHeader.vue")['default']
export const OuiCombobox: typeof import("../components/oui/Combobox.vue")['default']
export const OuiDialog: typeof import("../components/oui/Dialog.vue")['default']
export const OuiInput: typeof import("../components/oui/Input.vue")['default']
export const OuiProgress: typeof import("../components/oui/Progress.vue")['default']
export const OuiSelect: typeof import("../components/oui/Select.vue")['default']
export const OuiSkeleton: typeof import("../components/oui/Skeleton.vue")['default']
export const OuiText: typeof import("../components/oui/Text.vue")['default']
export const OuiTooltip: typeof import("../components/oui/Tooltip.vue")['default']
export const NuxtWelcome: typeof import("../../../node_modules/.pnpm/nuxt@4.1.2_@parcel+watcher@2.5.1_@types+node@24.5.1_@vue+compiler-sfc@3.5.21_db0@0.3.2__929524fb83e5bf9e1b61116f26b07874/node_modules/nuxt/dist/app/components/welcome.vue")['default']
export const NuxtLayout: typeof import("../../../node_modules/.pnpm/nuxt@4.1.2_@parcel+watcher@2.5.1_@types+node@24.5.1_@vue+compiler-sfc@3.5.21_db0@0.3.2__929524fb83e5bf9e1b61116f26b07874/node_modules/nuxt/dist/app/components/nuxt-layout")['default']
export const NuxtErrorBoundary: typeof import("../../../node_modules/.pnpm/nuxt@4.1.2_@parcel+watcher@2.5.1_@types+node@24.5.1_@vue+compiler-sfc@3.5.21_db0@0.3.2__929524fb83e5bf9e1b61116f26b07874/node_modules/nuxt/dist/app/components/nuxt-error-boundary.vue")['default']
export const ClientOnly: typeof import("../../../node_modules/.pnpm/nuxt@4.1.2_@parcel+watcher@2.5.1_@types+node@24.5.1_@vue+compiler-sfc@3.5.21_db0@0.3.2__929524fb83e5bf9e1b61116f26b07874/node_modules/nuxt/dist/app/components/client-only")['default']
export const DevOnly: typeof import("../../../node_modules/.pnpm/nuxt@4.1.2_@parcel+watcher@2.5.1_@types+node@24.5.1_@vue+compiler-sfc@3.5.21_db0@0.3.2__929524fb83e5bf9e1b61116f26b07874/node_modules/nuxt/dist/app/components/dev-only")['default']
export const ServerPlaceholder: typeof import("../../../node_modules/.pnpm/nuxt@4.1.2_@parcel+watcher@2.5.1_@types+node@24.5.1_@vue+compiler-sfc@3.5.21_db0@0.3.2__929524fb83e5bf9e1b61116f26b07874/node_modules/nuxt/dist/app/components/server-placeholder")['default']
export const NuxtLink: typeof import("../../../node_modules/.pnpm/nuxt@4.1.2_@parcel+watcher@2.5.1_@types+node@24.5.1_@vue+compiler-sfc@3.5.21_db0@0.3.2__929524fb83e5bf9e1b61116f26b07874/node_modules/nuxt/dist/app/components/nuxt-link")['default']
export const NuxtLoadingIndicator: typeof import("../../../node_modules/.pnpm/nuxt@4.1.2_@parcel+watcher@2.5.1_@types+node@24.5.1_@vue+compiler-sfc@3.5.21_db0@0.3.2__929524fb83e5bf9e1b61116f26b07874/node_modules/nuxt/dist/app/components/nuxt-loading-indicator")['default']
export const NuxtTime: typeof import("../../../node_modules/.pnpm/nuxt@4.1.2_@parcel+watcher@2.5.1_@types+node@24.5.1_@vue+compiler-sfc@3.5.21_db0@0.3.2__929524fb83e5bf9e1b61116f26b07874/node_modules/nuxt/dist/app/components/nuxt-time.vue")['default']
export const NuxtRouteAnnouncer: typeof import("../../../node_modules/.pnpm/nuxt@4.1.2_@parcel+watcher@2.5.1_@types+node@24.5.1_@vue+compiler-sfc@3.5.21_db0@0.3.2__929524fb83e5bf9e1b61116f26b07874/node_modules/nuxt/dist/app/components/nuxt-route-announcer")['default']
export const NuxtImg: typeof import("../../../node_modules/.pnpm/nuxt@4.1.2_@parcel+watcher@2.5.1_@types+node@24.5.1_@vue+compiler-sfc@3.5.21_db0@0.3.2__929524fb83e5bf9e1b61116f26b07874/node_modules/nuxt/dist/app/components/nuxt-stubs")['NuxtImg']
export const NuxtPicture: typeof import("../../../node_modules/.pnpm/nuxt@4.1.2_@parcel+watcher@2.5.1_@types+node@24.5.1_@vue+compiler-sfc@3.5.21_db0@0.3.2__929524fb83e5bf9e1b61116f26b07874/node_modules/nuxt/dist/app/components/nuxt-stubs")['NuxtPicture']
export const NuxtPage: typeof import("../../../node_modules/.pnpm/nuxt@4.1.2_@parcel+watcher@2.5.1_@types+node@24.5.1_@vue+compiler-sfc@3.5.21_db0@0.3.2__929524fb83e5bf9e1b61116f26b07874/node_modules/nuxt/dist/pages/runtime/page")['default']
export const NoScript: typeof import("../../../node_modules/.pnpm/nuxt@4.1.2_@parcel+watcher@2.5.1_@types+node@24.5.1_@vue+compiler-sfc@3.5.21_db0@0.3.2__929524fb83e5bf9e1b61116f26b07874/node_modules/nuxt/dist/head/runtime/components")['NoScript']
export const Link: typeof import("../../../node_modules/.pnpm/nuxt@4.1.2_@parcel+watcher@2.5.1_@types+node@24.5.1_@vue+compiler-sfc@3.5.21_db0@0.3.2__929524fb83e5bf9e1b61116f26b07874/node_modules/nuxt/dist/head/runtime/components")['Link']
export const Base: typeof import("../../../node_modules/.pnpm/nuxt@4.1.2_@parcel+watcher@2.5.1_@types+node@24.5.1_@vue+compiler-sfc@3.5.21_db0@0.3.2__929524fb83e5bf9e1b61116f26b07874/node_modules/nuxt/dist/head/runtime/components")['Base']
export const Title: typeof import("../../../node_modules/.pnpm/nuxt@4.1.2_@parcel+watcher@2.5.1_@types+node@24.5.1_@vue+compiler-sfc@3.5.21_db0@0.3.2__929524fb83e5bf9e1b61116f26b07874/node_modules/nuxt/dist/head/runtime/components")['Title']
export const Meta: typeof import("../../../node_modules/.pnpm/nuxt@4.1.2_@parcel+watcher@2.5.1_@types+node@24.5.1_@vue+compiler-sfc@3.5.21_db0@0.3.2__929524fb83e5bf9e1b61116f26b07874/node_modules/nuxt/dist/head/runtime/components")['Meta']
export const Style: typeof import("../../../node_modules/.pnpm/nuxt@4.1.2_@parcel+watcher@2.5.1_@types+node@24.5.1_@vue+compiler-sfc@3.5.21_db0@0.3.2__929524fb83e5bf9e1b61116f26b07874/node_modules/nuxt/dist/head/runtime/components")['Style']
export const Head: typeof import("../../../node_modules/.pnpm/nuxt@4.1.2_@parcel+watcher@2.5.1_@types+node@24.5.1_@vue+compiler-sfc@3.5.21_db0@0.3.2__929524fb83e5bf9e1b61116f26b07874/node_modules/nuxt/dist/head/runtime/components")['Head']
export const Html: typeof import("../../../node_modules/.pnpm/nuxt@4.1.2_@parcel+watcher@2.5.1_@types+node@24.5.1_@vue+compiler-sfc@3.5.21_db0@0.3.2__929524fb83e5bf9e1b61116f26b07874/node_modules/nuxt/dist/head/runtime/components")['Html']
export const Body: typeof import("../../../node_modules/.pnpm/nuxt@4.1.2_@parcel+watcher@2.5.1_@types+node@24.5.1_@vue+compiler-sfc@3.5.21_db0@0.3.2__929524fb83e5bf9e1b61116f26b07874/node_modules/nuxt/dist/head/runtime/components")['Body']
export const NuxtIsland: typeof import("../../../node_modules/.pnpm/nuxt@4.1.2_@parcel+watcher@2.5.1_@types+node@24.5.1_@vue+compiler-sfc@3.5.21_db0@0.3.2__929524fb83e5bf9e1b61116f26b07874/node_modules/nuxt/dist/app/components/nuxt-island")['default']
export const NuxtRouteAnnouncer: typeof import("../../../node_modules/.pnpm/nuxt@4.1.2_@parcel+watcher@2.5.1_@types+node@24.5.1_@vue+compiler-sfc@3.5.21_db0@0.3.2__929524fb83e5bf9e1b61116f26b07874/node_modules/nuxt/dist/app/components/server-placeholder")['default']
export const LazyAppHeader: LazyComponent<typeof import("../components/app/AppHeader.vue")['default']>
export const LazyAppSidebar: LazyComponent<typeof import("../components/app/AppSidebar.vue")['default']>
export const LazyAppNavigationLink: LazyComponent<typeof import("../components/app/NavigationLink.vue")['default']>
export const LazyAppUserProfile: LazyComponent<typeof import("../components/app/UserProfile.vue")['default']>
export const LazyOuiAvatar: LazyComponent<typeof import("../components/oui/Avatar.vue")['default']>
export const LazyOuiBadge: LazyComponent<typeof import("../components/oui/Badge.vue")['default']>
export const LazyOuiButton: LazyComponent<typeof import("../components/oui/Button.vue")['default']>
export const LazyOuiCard: LazyComponent<typeof import("../components/oui/Card.vue")['default']>
export const LazyOuiCardBody: LazyComponent<typeof import("../components/oui/CardBody.vue")['default']>
export const LazyOuiCardFooter: LazyComponent<typeof import("../components/oui/CardFooter.vue")['default']>
export const LazyOuiCardHeader: LazyComponent<typeof import("../components/oui/CardHeader.vue")['default']>
export const LazyOuiCombobox: LazyComponent<typeof import("../components/oui/Combobox.vue")['default']>
export const LazyOuiDialog: LazyComponent<typeof import("../components/oui/Dialog.vue")['default']>
export const LazyOuiInput: LazyComponent<typeof import("../components/oui/Input.vue")['default']>
export const LazyOuiProgress: LazyComponent<typeof import("../components/oui/Progress.vue")['default']>
export const LazyOuiSelect: LazyComponent<typeof import("../components/oui/Select.vue")['default']>
export const LazyOuiSkeleton: LazyComponent<typeof import("../components/oui/Skeleton.vue")['default']>
export const LazyOuiText: LazyComponent<typeof import("../components/oui/Text.vue")['default']>
export const LazyOuiTooltip: LazyComponent<typeof import("../components/oui/Tooltip.vue")['default']>
export const LazyNuxtWelcome: LazyComponent<typeof import("../../../node_modules/.pnpm/nuxt@4.1.2_@parcel+watcher@2.5.1_@types+node@24.5.1_@vue+compiler-sfc@3.5.21_db0@0.3.2__929524fb83e5bf9e1b61116f26b07874/node_modules/nuxt/dist/app/components/welcome.vue")['default']>
export const LazyNuxtLayout: LazyComponent<typeof import("../../../node_modules/.pnpm/nuxt@4.1.2_@parcel+watcher@2.5.1_@types+node@24.5.1_@vue+compiler-sfc@3.5.21_db0@0.3.2__929524fb83e5bf9e1b61116f26b07874/node_modules/nuxt/dist/app/components/nuxt-layout")['default']>
export const LazyNuxtErrorBoundary: LazyComponent<typeof import("../../../node_modules/.pnpm/nuxt@4.1.2_@parcel+watcher@2.5.1_@types+node@24.5.1_@vue+compiler-sfc@3.5.21_db0@0.3.2__929524fb83e5bf9e1b61116f26b07874/node_modules/nuxt/dist/app/components/nuxt-error-boundary.vue")['default']>
export const LazyClientOnly: LazyComponent<typeof import("../../../node_modules/.pnpm/nuxt@4.1.2_@parcel+watcher@2.5.1_@types+node@24.5.1_@vue+compiler-sfc@3.5.21_db0@0.3.2__929524fb83e5bf9e1b61116f26b07874/node_modules/nuxt/dist/app/components/client-only")['default']>
export const LazyDevOnly: LazyComponent<typeof import("../../../node_modules/.pnpm/nuxt@4.1.2_@parcel+watcher@2.5.1_@types+node@24.5.1_@vue+compiler-sfc@3.5.21_db0@0.3.2__929524fb83e5bf9e1b61116f26b07874/node_modules/nuxt/dist/app/components/dev-only")['default']>
export const LazyServerPlaceholder: LazyComponent<typeof import("../../../node_modules/.pnpm/nuxt@4.1.2_@parcel+watcher@2.5.1_@types+node@24.5.1_@vue+compiler-sfc@3.5.21_db0@0.3.2__929524fb83e5bf9e1b61116f26b07874/node_modules/nuxt/dist/app/components/server-placeholder")['default']>
export const LazyNuxtLink: LazyComponent<typeof import("../../../node_modules/.pnpm/nuxt@4.1.2_@parcel+watcher@2.5.1_@types+node@24.5.1_@vue+compiler-sfc@3.5.21_db0@0.3.2__929524fb83e5bf9e1b61116f26b07874/node_modules/nuxt/dist/app/components/nuxt-link")['default']>
export const LazyNuxtLoadingIndicator: LazyComponent<typeof import("../../../node_modules/.pnpm/nuxt@4.1.2_@parcel+watcher@2.5.1_@types+node@24.5.1_@vue+compiler-sfc@3.5.21_db0@0.3.2__929524fb83e5bf9e1b61116f26b07874/node_modules/nuxt/dist/app/components/nuxt-loading-indicator")['default']>
export const LazyNuxtTime: LazyComponent<typeof import("../../../node_modules/.pnpm/nuxt@4.1.2_@parcel+watcher@2.5.1_@types+node@24.5.1_@vue+compiler-sfc@3.5.21_db0@0.3.2__929524fb83e5bf9e1b61116f26b07874/node_modules/nuxt/dist/app/components/nuxt-time.vue")['default']>
export const LazyNuxtRouteAnnouncer: LazyComponent<typeof import("../../../node_modules/.pnpm/nuxt@4.1.2_@parcel+watcher@2.5.1_@types+node@24.5.1_@vue+compiler-sfc@3.5.21_db0@0.3.2__929524fb83e5bf9e1b61116f26b07874/node_modules/nuxt/dist/app/components/nuxt-route-announcer")['default']>
export const LazyNuxtImg: LazyComponent<typeof import("../../../node_modules/.pnpm/nuxt@4.1.2_@parcel+watcher@2.5.1_@types+node@24.5.1_@vue+compiler-sfc@3.5.21_db0@0.3.2__929524fb83e5bf9e1b61116f26b07874/node_modules/nuxt/dist/app/components/nuxt-stubs")['NuxtImg']>
export const LazyNuxtPicture: LazyComponent<typeof import("../../../node_modules/.pnpm/nuxt@4.1.2_@parcel+watcher@2.5.1_@types+node@24.5.1_@vue+compiler-sfc@3.5.21_db0@0.3.2__929524fb83e5bf9e1b61116f26b07874/node_modules/nuxt/dist/app/components/nuxt-stubs")['NuxtPicture']>
export const LazyNuxtPage: LazyComponent<typeof import("../../../node_modules/.pnpm/nuxt@4.1.2_@parcel+watcher@2.5.1_@types+node@24.5.1_@vue+compiler-sfc@3.5.21_db0@0.3.2__929524fb83e5bf9e1b61116f26b07874/node_modules/nuxt/dist/pages/runtime/page")['default']>
export const LazyNoScript: LazyComponent<typeof import("../../../node_modules/.pnpm/nuxt@4.1.2_@parcel+watcher@2.5.1_@types+node@24.5.1_@vue+compiler-sfc@3.5.21_db0@0.3.2__929524fb83e5bf9e1b61116f26b07874/node_modules/nuxt/dist/head/runtime/components")['NoScript']>
export const LazyLink: LazyComponent<typeof import("../../../node_modules/.pnpm/nuxt@4.1.2_@parcel+watcher@2.5.1_@types+node@24.5.1_@vue+compiler-sfc@3.5.21_db0@0.3.2__929524fb83e5bf9e1b61116f26b07874/node_modules/nuxt/dist/head/runtime/components")['Link']>
export const LazyBase: LazyComponent<typeof import("../../../node_modules/.pnpm/nuxt@4.1.2_@parcel+watcher@2.5.1_@types+node@24.5.1_@vue+compiler-sfc@3.5.21_db0@0.3.2__929524fb83e5bf9e1b61116f26b07874/node_modules/nuxt/dist/head/runtime/components")['Base']>
export const LazyTitle: LazyComponent<typeof import("../../../node_modules/.pnpm/nuxt@4.1.2_@parcel+watcher@2.5.1_@types+node@24.5.1_@vue+compiler-sfc@3.5.21_db0@0.3.2__929524fb83e5bf9e1b61116f26b07874/node_modules/nuxt/dist/head/runtime/components")['Title']>
export const LazyMeta: LazyComponent<typeof import("../../../node_modules/.pnpm/nuxt@4.1.2_@parcel+watcher@2.5.1_@types+node@24.5.1_@vue+compiler-sfc@3.5.21_db0@0.3.2__929524fb83e5bf9e1b61116f26b07874/node_modules/nuxt/dist/head/runtime/components")['Meta']>
export const LazyStyle: LazyComponent<typeof import("../../../node_modules/.pnpm/nuxt@4.1.2_@parcel+watcher@2.5.1_@types+node@24.5.1_@vue+compiler-sfc@3.5.21_db0@0.3.2__929524fb83e5bf9e1b61116f26b07874/node_modules/nuxt/dist/head/runtime/components")['Style']>
export const LazyHead: LazyComponent<typeof import("../../../node_modules/.pnpm/nuxt@4.1.2_@parcel+watcher@2.5.1_@types+node@24.5.1_@vue+compiler-sfc@3.5.21_db0@0.3.2__929524fb83e5bf9e1b61116f26b07874/node_modules/nuxt/dist/head/runtime/components")['Head']>
export const LazyHtml: LazyComponent<typeof import("../../../node_modules/.pnpm/nuxt@4.1.2_@parcel+watcher@2.5.1_@types+node@24.5.1_@vue+compiler-sfc@3.5.21_db0@0.3.2__929524fb83e5bf9e1b61116f26b07874/node_modules/nuxt/dist/head/runtime/components")['Html']>
export const LazyBody: LazyComponent<typeof import("../../../node_modules/.pnpm/nuxt@4.1.2_@parcel+watcher@2.5.1_@types+node@24.5.1_@vue+compiler-sfc@3.5.21_db0@0.3.2__929524fb83e5bf9e1b61116f26b07874/node_modules/nuxt/dist/head/runtime/components")['Body']>
export const LazyNuxtIsland: LazyComponent<typeof import("../../../node_modules/.pnpm/nuxt@4.1.2_@parcel+watcher@2.5.1_@types+node@24.5.1_@vue+compiler-sfc@3.5.21_db0@0.3.2__929524fb83e5bf9e1b61116f26b07874/node_modules/nuxt/dist/app/components/nuxt-island")['default']>
export const LazyNuxtRouteAnnouncer: LazyComponent<typeof import("../../../node_modules/.pnpm/nuxt@4.1.2_@parcel+watcher@2.5.1_@types+node@24.5.1_@vue+compiler-sfc@3.5.21_db0@0.3.2__929524fb83e5bf9e1b61116f26b07874/node_modules/nuxt/dist/app/components/server-placeholder")['default']>

export const componentNames: string[]
