export default defineNuxtPlugin({
  name: 'invite-notifications',
  async setup() {
    // Only run on client side
    if (!import.meta.client) return;

    const nuxtApp = useNuxtApp();
    const { addNotification } = useNotifications();
    const { toast } = useToast();
    const { OrganizationService } = await import("@obiente/proto");
    const { createClient } = await import("@connectrpc/connect");
    
    const orgClient = createClient(OrganizationService, nuxtApp.$connect);
    
    // Track which invites we've already notified about
    const notifiedInviteIds = new Set<string>();
    
    // Function to check for new invites and create notifications
    async function checkInvites() {
      try {
        const res = await orgClient.listMyInvites({});
        const invites = res.invites || [];
        
        // Create notifications for new invites we haven't seen yet
        for (const invite of invites) {
          if (!notifiedInviteIds.has(invite.id)) {
            notifiedInviteIds.add(invite.id);
            
            // Add notification
            addNotification({
              title: `Invitation to ${invite.organizationName}`,
              message: `You've been invited to join ${invite.organizationName} as ${invite.role.toUpperCase()}. Click to view your invitations.`,
            });
            
            // Also show a toast
            toast.info(
              `You've been invited to ${invite.organizationName}`,
              "View your invitations to accept or decline"
            );
          }
        }
        
        // Clean up invites that no longer exist
        const currentInviteIds = new Set(invites.map(i => i.id));
        for (const id of notifiedInviteIds) {
          if (!currentInviteIds.has(id)) {
            notifiedInviteIds.delete(id);
          }
        }
      } catch (error) {
        // Silently fail - don't spam errors if user is not authenticated yet
        console.debug("Failed to check invites:", error);
      }
    }
    
    // Wait a bit for auth to initialize, then check
    setTimeout(() => {
      checkInvites();
      
      // Check every 30 seconds for new invites
      setInterval(checkInvites, 30000);
    }, 2000);
  },
});

