export default defineNuxtRouteMiddleware(async (to, from) => {
  const user = useAuth();
  await user.fetch();
  if (import.meta.server) return;
  if (!user.session || !user.user) {
    user.popupLogin();
  }
});
