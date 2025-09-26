import { useUserStore } from '~/stores/user';
import { defineNuxtRouteMiddleware } from '#app';
export default defineNuxtRouteMiddleware(async (to, from) => {
  if (import.meta.server) return;
  const userStore = useUserStore();
  console.log(to.fullPath);
  if (!userStore.user) {
    if (to.fullPath === '/callback/auth') userStore.handleCallback();
    else window?.open(userStore.login(), '_blank', 'width=500,height=700');
    //  router.(, {
    //     external: true,
    //     open: { target: '_blank', windowFeatures: { popup: true } },
    //   });
    //   navigateTo('test');
  }
});
