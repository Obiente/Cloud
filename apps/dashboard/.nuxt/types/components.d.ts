
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

interface _GlobalComponents {
      'AppHeader': typeof import("../../components/app/AppHeader.vue")['default']
    'AppSidebar': typeof import("../../components/app/AppSidebar.vue")['default']
    'AppNavigationLink': typeof import("../../components/app/NavigationLink.vue")['default']
    'AppUserProfile': typeof import("../../components/app/UserProfile.vue")['default']
    'OuiAvatar': typeof import("../../components/oui/Avatar.vue")['default']
    'OuiBadge': typeof import("../../components/oui/Badge.vue")['default']
    'OuiButton': typeof import("../../components/oui/Button.vue")['default']
    'OuiCard': typeof import("../../components/oui/Card.vue")['default']
    'OuiCardBody': typeof import("../../components/oui/CardBody.vue")['default']
    'OuiCardFooter': typeof import("../../components/oui/CardFooter.vue")['default']
    'OuiCardHeader': typeof import("../../components/oui/CardHeader.vue")['default']
    'OuiCombobox': typeof import("../../components/oui/Combobox.vue")['default']
    'OuiDialog': typeof import("../../components/oui/Dialog.vue")['default']
    'OuiInput': typeof import("../../components/oui/Input.vue")['default']
    'OuiProgress': typeof import("../../components/oui/Progress.vue")['default']
    'OuiSelect': typeof import("../../components/oui/Select.vue")['default']
    'OuiSkeleton': typeof import("../../components/oui/Skeleton.vue")['default']
    'OuiText': typeof import("../../components/oui/Text.vue")['default']
    'OuiTooltip': typeof import("../../components/oui/Tooltip.vue")['default']
    'NuxtWelcome': typeof import("../../../../node_modules/.pnpm/nuxt@4.1.2_@parcel+watcher@2.5.1_@types+node@24.5.1_@vue+compiler-sfc@3.5.21_db0@0.3.2__929524fb83e5bf9e1b61116f26b07874/node_modules/nuxt/dist/app/components/welcome.vue")['default']
    'NuxtLayout': typeof import("../../../../node_modules/.pnpm/nuxt@4.1.2_@parcel+watcher@2.5.1_@types+node@24.5.1_@vue+compiler-sfc@3.5.21_db0@0.3.2__929524fb83e5bf9e1b61116f26b07874/node_modules/nuxt/dist/app/components/nuxt-layout")['default']
    'NuxtErrorBoundary': typeof import("../../../../node_modules/.pnpm/nuxt@4.1.2_@parcel+watcher@2.5.1_@types+node@24.5.1_@vue+compiler-sfc@3.5.21_db0@0.3.2__929524fb83e5bf9e1b61116f26b07874/node_modules/nuxt/dist/app/components/nuxt-error-boundary.vue")['default']
    'ClientOnly': typeof import("../../../../node_modules/.pnpm/nuxt@4.1.2_@parcel+watcher@2.5.1_@types+node@24.5.1_@vue+compiler-sfc@3.5.21_db0@0.3.2__929524fb83e5bf9e1b61116f26b07874/node_modules/nuxt/dist/app/components/client-only")['default']
    'DevOnly': typeof import("../../../../node_modules/.pnpm/nuxt@4.1.2_@parcel+watcher@2.5.1_@types+node@24.5.1_@vue+compiler-sfc@3.5.21_db0@0.3.2__929524fb83e5bf9e1b61116f26b07874/node_modules/nuxt/dist/app/components/dev-only")['default']
    'ServerPlaceholder': typeof import("../../../../node_modules/.pnpm/nuxt@4.1.2_@parcel+watcher@2.5.1_@types+node@24.5.1_@vue+compiler-sfc@3.5.21_db0@0.3.2__929524fb83e5bf9e1b61116f26b07874/node_modules/nuxt/dist/app/components/server-placeholder")['default']
    'NuxtLink': typeof import("../../../../node_modules/.pnpm/nuxt@4.1.2_@parcel+watcher@2.5.1_@types+node@24.5.1_@vue+compiler-sfc@3.5.21_db0@0.3.2__929524fb83e5bf9e1b61116f26b07874/node_modules/nuxt/dist/app/components/nuxt-link")['default']
    'NuxtLoadingIndicator': typeof import("../../../../node_modules/.pnpm/nuxt@4.1.2_@parcel+watcher@2.5.1_@types+node@24.5.1_@vue+compiler-sfc@3.5.21_db0@0.3.2__929524fb83e5bf9e1b61116f26b07874/node_modules/nuxt/dist/app/components/nuxt-loading-indicator")['default']
    'NuxtTime': typeof import("../../../../node_modules/.pnpm/nuxt@4.1.2_@parcel+watcher@2.5.1_@types+node@24.5.1_@vue+compiler-sfc@3.5.21_db0@0.3.2__929524fb83e5bf9e1b61116f26b07874/node_modules/nuxt/dist/app/components/nuxt-time.vue")['default']
    'NuxtRouteAnnouncer': typeof import("../../../../node_modules/.pnpm/nuxt@4.1.2_@parcel+watcher@2.5.1_@types+node@24.5.1_@vue+compiler-sfc@3.5.21_db0@0.3.2__929524fb83e5bf9e1b61116f26b07874/node_modules/nuxt/dist/app/components/nuxt-route-announcer")['default']
    'NuxtImg': typeof import("../../../../node_modules/.pnpm/nuxt@4.1.2_@parcel+watcher@2.5.1_@types+node@24.5.1_@vue+compiler-sfc@3.5.21_db0@0.3.2__929524fb83e5bf9e1b61116f26b07874/node_modules/nuxt/dist/app/components/nuxt-stubs")['NuxtImg']
    'NuxtPicture': typeof import("../../../../node_modules/.pnpm/nuxt@4.1.2_@parcel+watcher@2.5.1_@types+node@24.5.1_@vue+compiler-sfc@3.5.21_db0@0.3.2__929524fb83e5bf9e1b61116f26b07874/node_modules/nuxt/dist/app/components/nuxt-stubs")['NuxtPicture']
    'NuxtPage': typeof import("../../../../node_modules/.pnpm/nuxt@4.1.2_@parcel+watcher@2.5.1_@types+node@24.5.1_@vue+compiler-sfc@3.5.21_db0@0.3.2__929524fb83e5bf9e1b61116f26b07874/node_modules/nuxt/dist/pages/runtime/page")['default']
    'NoScript': typeof import("../../../../node_modules/.pnpm/nuxt@4.1.2_@parcel+watcher@2.5.1_@types+node@24.5.1_@vue+compiler-sfc@3.5.21_db0@0.3.2__929524fb83e5bf9e1b61116f26b07874/node_modules/nuxt/dist/head/runtime/components")['NoScript']
    'Link': typeof import("../../../../node_modules/.pnpm/nuxt@4.1.2_@parcel+watcher@2.5.1_@types+node@24.5.1_@vue+compiler-sfc@3.5.21_db0@0.3.2__929524fb83e5bf9e1b61116f26b07874/node_modules/nuxt/dist/head/runtime/components")['Link']
    'Base': typeof import("../../../../node_modules/.pnpm/nuxt@4.1.2_@parcel+watcher@2.5.1_@types+node@24.5.1_@vue+compiler-sfc@3.5.21_db0@0.3.2__929524fb83e5bf9e1b61116f26b07874/node_modules/nuxt/dist/head/runtime/components")['Base']
    'Title': typeof import("../../../../node_modules/.pnpm/nuxt@4.1.2_@parcel+watcher@2.5.1_@types+node@24.5.1_@vue+compiler-sfc@3.5.21_db0@0.3.2__929524fb83e5bf9e1b61116f26b07874/node_modules/nuxt/dist/head/runtime/components")['Title']
    'Meta': typeof import("../../../../node_modules/.pnpm/nuxt@4.1.2_@parcel+watcher@2.5.1_@types+node@24.5.1_@vue+compiler-sfc@3.5.21_db0@0.3.2__929524fb83e5bf9e1b61116f26b07874/node_modules/nuxt/dist/head/runtime/components")['Meta']
    'Style': typeof import("../../../../node_modules/.pnpm/nuxt@4.1.2_@parcel+watcher@2.5.1_@types+node@24.5.1_@vue+compiler-sfc@3.5.21_db0@0.3.2__929524fb83e5bf9e1b61116f26b07874/node_modules/nuxt/dist/head/runtime/components")['Style']
    'Head': typeof import("../../../../node_modules/.pnpm/nuxt@4.1.2_@parcel+watcher@2.5.1_@types+node@24.5.1_@vue+compiler-sfc@3.5.21_db0@0.3.2__929524fb83e5bf9e1b61116f26b07874/node_modules/nuxt/dist/head/runtime/components")['Head']
    'Html': typeof import("../../../../node_modules/.pnpm/nuxt@4.1.2_@parcel+watcher@2.5.1_@types+node@24.5.1_@vue+compiler-sfc@3.5.21_db0@0.3.2__929524fb83e5bf9e1b61116f26b07874/node_modules/nuxt/dist/head/runtime/components")['Html']
    'Body': typeof import("../../../../node_modules/.pnpm/nuxt@4.1.2_@parcel+watcher@2.5.1_@types+node@24.5.1_@vue+compiler-sfc@3.5.21_db0@0.3.2__929524fb83e5bf9e1b61116f26b07874/node_modules/nuxt/dist/head/runtime/components")['Body']
    'NuxtIsland': typeof import("../../../../node_modules/.pnpm/nuxt@4.1.2_@parcel+watcher@2.5.1_@types+node@24.5.1_@vue+compiler-sfc@3.5.21_db0@0.3.2__929524fb83e5bf9e1b61116f26b07874/node_modules/nuxt/dist/app/components/nuxt-island")['default']
    'NuxtRouteAnnouncer': typeof import("../../../../node_modules/.pnpm/nuxt@4.1.2_@parcel+watcher@2.5.1_@types+node@24.5.1_@vue+compiler-sfc@3.5.21_db0@0.3.2__929524fb83e5bf9e1b61116f26b07874/node_modules/nuxt/dist/app/components/server-placeholder")['default']
      'LazyAppHeader': LazyComponent<typeof import("../../components/app/AppHeader.vue")['default']>
    'LazyAppSidebar': LazyComponent<typeof import("../../components/app/AppSidebar.vue")['default']>
    'LazyAppNavigationLink': LazyComponent<typeof import("../../components/app/NavigationLink.vue")['default']>
    'LazyAppUserProfile': LazyComponent<typeof import("../../components/app/UserProfile.vue")['default']>
    'LazyOuiAvatar': LazyComponent<typeof import("../../components/oui/Avatar.vue")['default']>
    'LazyOuiBadge': LazyComponent<typeof import("../../components/oui/Badge.vue")['default']>
    'LazyOuiButton': LazyComponent<typeof import("../../components/oui/Button.vue")['default']>
    'LazyOuiCard': LazyComponent<typeof import("../../components/oui/Card.vue")['default']>
    'LazyOuiCardBody': LazyComponent<typeof import("../../components/oui/CardBody.vue")['default']>
    'LazyOuiCardFooter': LazyComponent<typeof import("../../components/oui/CardFooter.vue")['default']>
    'LazyOuiCardHeader': LazyComponent<typeof import("../../components/oui/CardHeader.vue")['default']>
    'LazyOuiCombobox': LazyComponent<typeof import("../../components/oui/Combobox.vue")['default']>
    'LazyOuiDialog': LazyComponent<typeof import("../../components/oui/Dialog.vue")['default']>
    'LazyOuiInput': LazyComponent<typeof import("../../components/oui/Input.vue")['default']>
    'LazyOuiProgress': LazyComponent<typeof import("../../components/oui/Progress.vue")['default']>
    'LazyOuiSelect': LazyComponent<typeof import("../../components/oui/Select.vue")['default']>
    'LazyOuiSkeleton': LazyComponent<typeof import("../../components/oui/Skeleton.vue")['default']>
    'LazyOuiText': LazyComponent<typeof import("../../components/oui/Text.vue")['default']>
    'LazyOuiTooltip': LazyComponent<typeof import("../../components/oui/Tooltip.vue")['default']>
    'LazyNuxtWelcome': LazyComponent<typeof import("../../../../node_modules/.pnpm/nuxt@4.1.2_@parcel+watcher@2.5.1_@types+node@24.5.1_@vue+compiler-sfc@3.5.21_db0@0.3.2__929524fb83e5bf9e1b61116f26b07874/node_modules/nuxt/dist/app/components/welcome.vue")['default']>
    'LazyNuxtLayout': LazyComponent<typeof import("../../../../node_modules/.pnpm/nuxt@4.1.2_@parcel+watcher@2.5.1_@types+node@24.5.1_@vue+compiler-sfc@3.5.21_db0@0.3.2__929524fb83e5bf9e1b61116f26b07874/node_modules/nuxt/dist/app/components/nuxt-layout")['default']>
    'LazyNuxtErrorBoundary': LazyComponent<typeof import("../../../../node_modules/.pnpm/nuxt@4.1.2_@parcel+watcher@2.5.1_@types+node@24.5.1_@vue+compiler-sfc@3.5.21_db0@0.3.2__929524fb83e5bf9e1b61116f26b07874/node_modules/nuxt/dist/app/components/nuxt-error-boundary.vue")['default']>
    'LazyClientOnly': LazyComponent<typeof import("../../../../node_modules/.pnpm/nuxt@4.1.2_@parcel+watcher@2.5.1_@types+node@24.5.1_@vue+compiler-sfc@3.5.21_db0@0.3.2__929524fb83e5bf9e1b61116f26b07874/node_modules/nuxt/dist/app/components/client-only")['default']>
    'LazyDevOnly': LazyComponent<typeof import("../../../../node_modules/.pnpm/nuxt@4.1.2_@parcel+watcher@2.5.1_@types+node@24.5.1_@vue+compiler-sfc@3.5.21_db0@0.3.2__929524fb83e5bf9e1b61116f26b07874/node_modules/nuxt/dist/app/components/dev-only")['default']>
    'LazyServerPlaceholder': LazyComponent<typeof import("../../../../node_modules/.pnpm/nuxt@4.1.2_@parcel+watcher@2.5.1_@types+node@24.5.1_@vue+compiler-sfc@3.5.21_db0@0.3.2__929524fb83e5bf9e1b61116f26b07874/node_modules/nuxt/dist/app/components/server-placeholder")['default']>
    'LazyNuxtLink': LazyComponent<typeof import("../../../../node_modules/.pnpm/nuxt@4.1.2_@parcel+watcher@2.5.1_@types+node@24.5.1_@vue+compiler-sfc@3.5.21_db0@0.3.2__929524fb83e5bf9e1b61116f26b07874/node_modules/nuxt/dist/app/components/nuxt-link")['default']>
    'LazyNuxtLoadingIndicator': LazyComponent<typeof import("../../../../node_modules/.pnpm/nuxt@4.1.2_@parcel+watcher@2.5.1_@types+node@24.5.1_@vue+compiler-sfc@3.5.21_db0@0.3.2__929524fb83e5bf9e1b61116f26b07874/node_modules/nuxt/dist/app/components/nuxt-loading-indicator")['default']>
    'LazyNuxtTime': LazyComponent<typeof import("../../../../node_modules/.pnpm/nuxt@4.1.2_@parcel+watcher@2.5.1_@types+node@24.5.1_@vue+compiler-sfc@3.5.21_db0@0.3.2__929524fb83e5bf9e1b61116f26b07874/node_modules/nuxt/dist/app/components/nuxt-time.vue")['default']>
    'LazyNuxtRouteAnnouncer': LazyComponent<typeof import("../../../../node_modules/.pnpm/nuxt@4.1.2_@parcel+watcher@2.5.1_@types+node@24.5.1_@vue+compiler-sfc@3.5.21_db0@0.3.2__929524fb83e5bf9e1b61116f26b07874/node_modules/nuxt/dist/app/components/nuxt-route-announcer")['default']>
    'LazyNuxtImg': LazyComponent<typeof import("../../../../node_modules/.pnpm/nuxt@4.1.2_@parcel+watcher@2.5.1_@types+node@24.5.1_@vue+compiler-sfc@3.5.21_db0@0.3.2__929524fb83e5bf9e1b61116f26b07874/node_modules/nuxt/dist/app/components/nuxt-stubs")['NuxtImg']>
    'LazyNuxtPicture': LazyComponent<typeof import("../../../../node_modules/.pnpm/nuxt@4.1.2_@parcel+watcher@2.5.1_@types+node@24.5.1_@vue+compiler-sfc@3.5.21_db0@0.3.2__929524fb83e5bf9e1b61116f26b07874/node_modules/nuxt/dist/app/components/nuxt-stubs")['NuxtPicture']>
    'LazyNuxtPage': LazyComponent<typeof import("../../../../node_modules/.pnpm/nuxt@4.1.2_@parcel+watcher@2.5.1_@types+node@24.5.1_@vue+compiler-sfc@3.5.21_db0@0.3.2__929524fb83e5bf9e1b61116f26b07874/node_modules/nuxt/dist/pages/runtime/page")['default']>
    'LazyNoScript': LazyComponent<typeof import("../../../../node_modules/.pnpm/nuxt@4.1.2_@parcel+watcher@2.5.1_@types+node@24.5.1_@vue+compiler-sfc@3.5.21_db0@0.3.2__929524fb83e5bf9e1b61116f26b07874/node_modules/nuxt/dist/head/runtime/components")['NoScript']>
    'LazyLink': LazyComponent<typeof import("../../../../node_modules/.pnpm/nuxt@4.1.2_@parcel+watcher@2.5.1_@types+node@24.5.1_@vue+compiler-sfc@3.5.21_db0@0.3.2__929524fb83e5bf9e1b61116f26b07874/node_modules/nuxt/dist/head/runtime/components")['Link']>
    'LazyBase': LazyComponent<typeof import("../../../../node_modules/.pnpm/nuxt@4.1.2_@parcel+watcher@2.5.1_@types+node@24.5.1_@vue+compiler-sfc@3.5.21_db0@0.3.2__929524fb83e5bf9e1b61116f26b07874/node_modules/nuxt/dist/head/runtime/components")['Base']>
    'LazyTitle': LazyComponent<typeof import("../../../../node_modules/.pnpm/nuxt@4.1.2_@parcel+watcher@2.5.1_@types+node@24.5.1_@vue+compiler-sfc@3.5.21_db0@0.3.2__929524fb83e5bf9e1b61116f26b07874/node_modules/nuxt/dist/head/runtime/components")['Title']>
    'LazyMeta': LazyComponent<typeof import("../../../../node_modules/.pnpm/nuxt@4.1.2_@parcel+watcher@2.5.1_@types+node@24.5.1_@vue+compiler-sfc@3.5.21_db0@0.3.2__929524fb83e5bf9e1b61116f26b07874/node_modules/nuxt/dist/head/runtime/components")['Meta']>
    'LazyStyle': LazyComponent<typeof import("../../../../node_modules/.pnpm/nuxt@4.1.2_@parcel+watcher@2.5.1_@types+node@24.5.1_@vue+compiler-sfc@3.5.21_db0@0.3.2__929524fb83e5bf9e1b61116f26b07874/node_modules/nuxt/dist/head/runtime/components")['Style']>
    'LazyHead': LazyComponent<typeof import("../../../../node_modules/.pnpm/nuxt@4.1.2_@parcel+watcher@2.5.1_@types+node@24.5.1_@vue+compiler-sfc@3.5.21_db0@0.3.2__929524fb83e5bf9e1b61116f26b07874/node_modules/nuxt/dist/head/runtime/components")['Head']>
    'LazyHtml': LazyComponent<typeof import("../../../../node_modules/.pnpm/nuxt@4.1.2_@parcel+watcher@2.5.1_@types+node@24.5.1_@vue+compiler-sfc@3.5.21_db0@0.3.2__929524fb83e5bf9e1b61116f26b07874/node_modules/nuxt/dist/head/runtime/components")['Html']>
    'LazyBody': LazyComponent<typeof import("../../../../node_modules/.pnpm/nuxt@4.1.2_@parcel+watcher@2.5.1_@types+node@24.5.1_@vue+compiler-sfc@3.5.21_db0@0.3.2__929524fb83e5bf9e1b61116f26b07874/node_modules/nuxt/dist/head/runtime/components")['Body']>
    'LazyNuxtIsland': LazyComponent<typeof import("../../../../node_modules/.pnpm/nuxt@4.1.2_@parcel+watcher@2.5.1_@types+node@24.5.1_@vue+compiler-sfc@3.5.21_db0@0.3.2__929524fb83e5bf9e1b61116f26b07874/node_modules/nuxt/dist/app/components/nuxt-island")['default']>
    'LazyNuxtRouteAnnouncer': LazyComponent<typeof import("../../../../node_modules/.pnpm/nuxt@4.1.2_@parcel+watcher@2.5.1_@types+node@24.5.1_@vue+compiler-sfc@3.5.21_db0@0.3.2__929524fb83e5bf9e1b61116f26b07874/node_modules/nuxt/dist/app/components/server-placeholder")['default']>
}

declare module 'vue' {
  export interface GlobalComponents extends _GlobalComponents { }
}

export {}
