export default defineNuxtRouteMiddleware(async (to, from) => {
  if (import.meta.server) return;
  const user = useAuth();
  await user.fetch();
  if (!user.session.value || !user.user.value) {
    user.openInPopup('/auth/login', { height: 700, width: 500 });
  }
});
