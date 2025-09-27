import { useUser } from '~/stores/user';
import { defineNuxtRouteMiddleware } from '#app';
export default defineNuxtRouteMiddleware(async (to, from) => {
  const userStore = useUser();
  userStore.restoreSession();
  if (!userStore.isLoggedIn) {
    if (to.path === '/auth/callback') userStore.handleCallback(to);
    else window?.open(await userStore.login(), '_blank', 'width=500,height=700');
    //  router.(, {
    //     external: true,
    //     open: { target: '_blank', windowFeatures: { popup: true } },
    //   });
    //   navigateTo('test');
  }
});
