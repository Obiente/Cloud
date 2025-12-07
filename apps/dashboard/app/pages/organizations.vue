<script setup lang="ts">
  definePageMeta({ layout: "default", middleware: "auth" });
  import {
    OrganizationService,
    AdminService,
    BillingService,
    VPSService,
    type OrganizationMember,
    type Organization,
  } from "@obiente/proto";
  import { useConnectClient } from "~/lib/connect-client";
  import { useOrganizationId } from "~/composables/useOrganizationId";
  import { useToast } from "~/composables/useToast";
  import { useOrganizationLabels } from "~/composables/useOrganizationLabels";
  import AuditLogs from "~/components/audit/AuditLogs.vue";
  import OrganizationSSHKeys from "~/components/organizations/OrganizationSSHKeys.vue";
  import {
    CheckIcon,
    PlusIcon,
    CreditCardIcon,
    PencilIcon,
    ArrowDownTrayIcon,
    DocumentTextIcon,
    ArrowPathIcon,
    KeyIcon,
  } from "@heroicons/vue/24/outline";

  const name = ref("");
  const slug = ref("");
  const inviteEmail = ref("");
  const inviteRole = ref("");
  const error = ref("");

  const auth = useAuth();
  const orgClient = useConnectClient(OrganizationService);
  const adminClient = useConnectClient(AdminService);
  const billingClient = useConnectClient(BillingService);
  const route = useRoute();
  const config = useRuntimeConfig();
  const { toast } = useToast();

  // Store the target org from query params before any other logic
  const targetOrgId =
    route.query.organizationId && typeof route.query.organizationId === "string"
      ? route.query.organizationId
      : null;

  // Check for organizationId in query params (from superadmin navigation)
  if (targetOrgId) {
    auth.switchOrganization(targetOrgId);
  }

  const organizations = computed(() => auth.organizations || []);
  const { organizationSelectItems } = useOrganizationLabels(organizations);
  
  // Get organizationId using SSR-compatible composable
  const organizationId = useOrganizationId();
  
  const selectedOrg = computed({
    get: () => organizationId.value || auth.currentOrganizationId || "",
    set: (id: string) => {
      if (id) auth.switchOrganization(id);
    },
  });

  const activeTab = ref("members");
  const transferDialogOpen = ref(false);
  const transferCandidate = ref<OrganizationMember | null>(null);

  // Check for tab in query params
  if (route.query.tab && typeof route.query.tab === "string") {
    activeTab.value = route.query.tab;
  }

  if (!organizations.value.length && auth.isAuthenticated) {
    // Only show user's own organizations in the select, even for superadmins
    const res = await orgClient.listOrganizations({ onlyMine: true });
    auth.setOrganizations(res.organizations || []);
    // Ensure target org is set after organizations are loaded
    if (targetOrgId) {
      auth.switchOrganization(targetOrgId);
    }
  }

  const { data: membersData, refresh: refreshMembers } = await useClientFetch(
    () =>
      organizationId.value
        ? `org-members-${organizationId.value}`
        : "org-members-none",
    async () => {
      const orgId = organizationId.value;
      if (!orgId) return [] as OrganizationMember[];
      const res = await orgClient.listMembers({
        organizationId: orgId,
      });
      return res.members || [];
    },
    { watch: [selectedOrg] }
  );
  const members = computed(() => membersData.value || []);

  const DEFAULT_INVITE_ROLE = "member";
  const OWNER_TRANSFER_FALLBACK_ROLE = "admin";
  const SYSTEM_ROLES = [
    { value: "owner", label: "Owner", disabled: true },
    { value: "admin", label: "Admin", disabled: false },
    { value: "member", label: "Member", disabled: false },
    { value: "viewer", label: "Viewer", disabled: false },
    { value: "none", label: "None", disabled: false },
  ];

  const defaultRoleItems = computed(() =>
    SYSTEM_ROLES.map((role) => ({
      label: `${role.label} (system)`,
      value: role.value,
      disabled: role.disabled ?? false,
    }))
  );

  const { data: roleCatalogData, refresh: refreshRoleCatalog } =
    await useClientFetch(
      () =>
        organizationId.value
          ? `org-role-catalog-${organizationId.value}`
          : "org-role-catalog-none",
      async () => {
        const orgId = organizationId.value;
        if (!orgId) return [] as { id: string; name: string; permissionsJson: string }[];
        const res = await adminClient.listRoles({
          organizationId: orgId,
        });
        return (res.roles || []).map((r) => ({ id: r.id, name: r.name, permissionsJson: r.permissionsJson || "[]" }));
      },
      { watch: [selectedOrg] }
    );

  const roleDisplayMap = computed(() => {
    const map = new Map<string, string>();
    SYSTEM_ROLES.forEach((role) => map.set(role.value, role.label));
    (roleCatalogData.value || []).forEach((role) => {
      map.set(role.id, role.name);
    });
    return map;
  });

  const customRoleItems = computed(() =>
    (roleCatalogData.value || []).map((r) => ({
      label: `${r.name} (custom)`,
      value: r.id,
      disabled: false,
    }))
  );

  const roleItems = computed(() => {
    const items = [...defaultRoleItems.value, ...customRoleItems.value];
    const order = new Map(SYSTEM_ROLES.map((role, idx) => [role.value, idx]));
    return items.sort((a, b) => {
      const aIdx = order.has(a.value) ? order.get(a.value)! : order.size + 1;
      const bIdx = order.has(b.value) ? order.get(b.value)! : order.size + 1;
      if (aIdx !== bIdx) return aIdx - bIdx;
      return a.label.localeCompare(b.label);
    });
  });

  const selectableRoleItems = computed(() =>
    roleItems.value.filter((item) => !item.disabled)
  );

  const currentUserIdentifiers = computed(() => {
    const identifiers = new Set<string>();
    const sessionUser: any = auth.user || null;
    if (!sessionUser) {
      return identifiers;
    }
    [sessionUser.id, sessionUser.sub, sessionUser.userId].forEach((id) => {
      if (id) {
        identifiers.add(String(id));
      }
    });
    return identifiers;
  });

  const currentMemberRecord = computed(
    () =>
      members.value.find((member) => {
        const memberUserId = member.user?.id;
        if (!memberUserId) return false;
        return currentUserIdentifiers.value.has(memberUserId);
      }) || null
  );

  const currentUserIsOwner = computed(
    () => currentMemberRecord.value?.role === "owner"
  );

  // Fetch user's permissions for the current organization
  const { data: userPermissionsData } = await useClientFetch(
    () =>
      organizationId.value
        ? `org-permissions-${organizationId.value}`
        : "org-permissions-none",
    async () => {
      const orgId = organizationId.value;
      if (!orgId) return [] as string[];
      try {
        const res = await orgClient.getMyPermissions({
          organizationId: orgId,
        });
        return res.permissions || [];
      } catch {
        return [] as string[];
      }
    },
    { watch: [selectedOrg] }
  );
  const userPermissions = computed(() => userPermissionsData.value || []);

  // Helper function to check if a permission matches (supports wildcards)
  function matchesPermission(perm: string, required: string): boolean {
    // Special case: "*" matches everything
    if (perm === "*") return true;
    if (required === "*") return true;
    
    if (perm === required) return true;
    if (perm.endsWith(".*")) {
      const prefix = perm.slice(0, -2);
      return required.startsWith(prefix + ".");
    }
    if (required.endsWith(".*")) {
      const prefix = required.slice(0, -2);
      return perm.startsWith(prefix + ".");
    }
    return false;
  }

  // Check if user has a specific permission
  function hasPermission(permission: string): boolean {
    const perms = userPermissions.value;
    return perms.some((perm) => matchesPermission(perm, permission));
  }

  // Check if current user can update member roles
  const canUpdateMembers = computed(() => {
    return hasPermission("organization.members.update") || hasPermission("organization.members.*");
  });

  watch(
    [selectedOrg, roleItems],
    () => {
      inviteEmail.value = "";
      const exists = selectableRoleItems.value.find(
        (item) => item.value === inviteRole.value
      );
      if (!exists) {
        const preferred = selectableRoleItems.value.find(
          (item) => item.value === DEFAULT_INVITE_ROLE
        );
        inviteRole.value =
          (preferred || selectableRoleItems.value[0])?.value ?? "";
      }
    },
    { immediate: true }
  );

  // Load invitations when invitations tab is active
  watch(activeTab, (tab) => {
    if (tab === "invitations") {
      refreshInvitations();
    }
  }, { immediate: true });

  // Watch for org changes to refresh plan info
  watch(selectedOrg, () => {
    if (selectedOrg.value) {
      refreshCurrentOrganization();
    }
  });

  async function syncOrganizations() {
    if (!auth.isAuthenticated) return;
    const res = await orgClient.listOrganizations({});
    auth.setOrganizations(res.organizations || []);
  }

  // Refresh current organization to get plan info
  async function refreshCurrentOrganization() {
    if (!selectedOrg.value) return;
    try {
      const res = await orgClient.getOrganization({ organizationId: selectedOrg.value });
      if (res.organization) {
        // Update the organization in the list
        const orgs = organizations.value;
        const index = orgs.findIndex((o) => o.id === selectedOrg.value);
        if (index >= 0) {
          orgs[index] = res.organization;
          auth.setOrganizations([...orgs]);
        } else {
          auth.setOrganizations([...orgs, res.organization]);
        }
      }
    } catch (err) {
      console.error("Failed to refresh organization:", err);
    }
  }

  async function createOrg() {
    error.value = "";
    try {
      const res = await orgClient.createOrganization({
        name: name.value,
        slug: slug.value || undefined,
      });
      await syncOrganizations();
      if (res.organization?.id) {
        await auth.switchOrganization(res.organization.id);
        await refreshMembers();
        await refreshRoleCatalog();
      }
      auth.notifyOrganizationsUpdated();
      name.value = "";
      slug.value = "";
    } catch (e: any) {
      error.value = e?.message || "Error creating organization";
    }
  }

  const inviting = ref(false);
  const resendingInvite = ref<string | null>(null);
  
  // My invitations (for invitations tab)
  const myInvitations = ref<any[]>([]);
  const loadingInvitations = ref(false);
  const processingInvite = ref<string | null>(null);
  const decliningInvite = ref(false);

  async function invite() {
    if (!selectedOrg.value || !inviteEmail.value || !inviteRole.value || inviting.value) return;
    inviting.value = true;
    try {
      await orgClient.inviteMember({
        organizationId: selectedOrg.value,
        email: inviteEmail.value,
        role: inviteRole.value,
      });
      toast.success(`Invitation email sent to ${inviteEmail.value}`);
      inviteEmail.value = "";
      await refreshMembers();
    } catch (error: any) {
      toast.error(error?.message || "Failed to send invitation email");
    } finally {
      inviting.value = false;
    }
  }

  async function resendInvite(member: OrganizationMember) {
    if (!selectedOrg.value || !member.id || resendingInvite.value === member.id) return;
    resendingInvite.value = member.id;
    try {
      await orgClient.resendInvite({
        organizationId: selectedOrg.value,
        memberId: member.id,
      });
      const email = member.user?.email || "the user";
      toast.success(`Invitation email resent to ${email}`);
      await refreshMembers();
    } catch (error: any) {
      toast.error(error?.message || "Failed to resend invitation email");
    } finally {
      resendingInvite.value = null;
    }
  }

  async function setRole(memberId: string, role: string) {
    if (!selectedOrg.value) return;
    await orgClient.updateMember({
      organizationId: selectedOrg.value,
      memberId,
      role,
    });
    await refreshMembers();
  }

  function roleLabel(role: string) {
    return roleDisplayMap.value.get(role) || role;
  }

  const { getInitials, formatCurrency: formatCurrencyUtil } = useUtils();

  function openTransferDialog(member: OrganizationMember) {
    transferCandidate.value = member;
    transferDialogOpen.value = true;
  }

  async function confirmTransferOwnership() {
    if (!selectedOrg.value || !transferCandidate.value) return;
    await orgClient.transferOwnership({
      organizationId: selectedOrg.value,
      newOwnerMemberId: transferCandidate.value.id,
      fallbackRole: OWNER_TRANSFER_FALLBACK_ROLE,
    });
    transferDialogOpen.value = false;
    transferCandidate.value = null;
    await refreshAll();
  }

  async function remove(memberId: string) {
    await orgClient.removeMember({
      organizationId: selectedOrg.value,
      memberId,
    });
    await refreshMembers();
  }

  async function refreshAll() {
    await syncOrganizations();
    await Promise.all([refreshMembers(), refreshRoleCatalog(), refreshInvitations()]);
  }
  
  async function refreshInvitations() {
    loadingInvitations.value = true;
    try {
      const res = await orgClient.listMyInvites({});
      myInvitations.value = res.invites || [];
    } catch (error: any) {
      const { toast } = useToast();
      toast.error(error?.message || "Failed to load invitations");
    } finally {
      loadingInvitations.value = false;
    }
  }
  
  async function acceptInvite(invite: any) {
    if (processingInvite.value === invite.id) return;
    processingInvite.value = invite.id;
    decliningInvite.value = false;
    
    try {
      const res = await orgClient.acceptInvite({
        organizationId: invite.organizationId,
        memberId: invite.id,
      });
      
      const { toast } = useToast();
      toast.success(`You've joined ${res.organization?.name || 'the organization'}!`);
      
      // Refresh user's organizations list
      const orgRes = await orgClient.listOrganizations({ onlyMine: true });
      auth.setOrganizations(orgRes.organizations || []);
      auth.notifyOrganizationsUpdated();
      
      // Switch to the new organization
      if (res.organization?.id) {
        await auth.switchOrganization(res.organization.id);
        selectedOrg.value = res.organization.id;
      }
      
      // Remove from list
      myInvitations.value = myInvitations.value.filter(i => i.id !== invite.id);
      
      // Refresh members if we switched to that org
      if (res.organization?.id) {
        await refreshMembers();
      }
    } catch (error: any) {
      const { toast } = useToast();
      toast.error(error?.message || "Failed to accept invitation");
    } finally {
      processingInvite.value = null;
    }
  }
  
  async function declineInvite(invite: any) {
    if (processingInvite.value === invite.id) return;
    processingInvite.value = invite.id;
    decliningInvite.value = true;
    
    try {
      await orgClient.declineInvite({
        organizationId: invite.organizationId,
        memberId: invite.id,
      });
      
      const { toast } = useToast();
      toast.success("Invitation declined");
      
      // Remove from list
      myInvitations.value = myInvitations.value.filter(i => i.id !== invite.id);
    } catch (error: any) {
      const { toast } = useToast();
      toast.error(error?.message || "Failed to decline invitation");
    } finally {
      processingInvite.value = null;
      decliningInvite.value = false;
    }
  }
  
  async function addCredits() {
    if (!selectedOrg.value || !addCreditsAmount.value) return;
    const amount = parseFloat(addCreditsAmount.value);
    
    // Validate minimum amount ($0.50 USD)
    if (isNaN(amount) || amount < 0.50) {
      const { toast } = useToast();
      toast.error("Minimum purchase amount is $0.50 USD");
      error.value = "Minimum purchase amount is $0.50 USD";
      return;
    }
    
    addCreditsLoading.value = true;
    error.value = "";
    try {
      // Create Stripe Checkout Session
      const response = await billingClient.createCheckoutSession({
        organizationId: selectedOrg.value,
        amountCents: BigInt(Math.round(amount * 100)), // Convert dollars to cents
      });
      
      if (response.checkoutUrl) {
        // Redirect to Stripe Checkout
        window.location.href = response.checkoutUrl;
      } else {
        throw new Error("No checkout URL received");
      }
    } catch (err: any) {
      error.value = err.message || "Failed to create checkout session";
      addCreditsLoading.value = false;
    }
  }

  const memberStats = computed(() => {
    const activeMembers = members.value.filter((m) => m.status === "active");
    const pendingInvites = members.value.filter((m) => m.status === "invited");
    return {
      total: activeMembers.length,
      owners: activeMembers.filter((m) => m.role === "owner").length,
      admins: activeMembers.filter((m) => m.role === "admin").length,
      members: activeMembers.filter((m) => m.role === "member").length,
      viewers: activeMembers.filter((m) => m.role === "viewer").length,
      pending: pendingInvites.length,
    };
  });

  const tabs = [
    { id: "members", label: "Members" },
    { id: "invitations", label: "Invitations" },
    { id: "roles", label: "Roles" },
    { id: "ssh-keys", label: "SSH Keys", icon: KeyIcon },
    { id: "audit-logs", label: "Audit Logs" },
  ];

  const inviteDisabled = computed(
    () => !selectedOrg.value || !inviteEmail.value || !inviteRole.value
  );

  const transferDialogSummary = computed(() => {
    if (!transferCandidate.value) return "";
    const target = transferCandidate.value;
    const label =
      target.user?.name ||
      target.user?.email ||
      target.user?.preferredUsername ||
      target.user?.id ||
      "this member";
    return `Ownership will move to ${label}. You will become ${OWNER_TRANSFER_FALLBACK_ROLE}.`;
  });

  const currentMonth = computed(() => {
    const now = new Date();
    return now.toLocaleString("default", { month: "long", year: "numeric" });
  });

  // Fetch usage data
  const { data: usageData, refresh: refreshUsage } = await useClientFetch(
    () =>
      selectedOrg.value
        ? `org-usage-${selectedOrg.value}`
        : "org-usage-none",
    async () => {
      if (!selectedOrg.value) return null;
      try {
        // Connect RPC returns the response message directly
        const res = await orgClient.getUsage({
          organizationId: selectedOrg.value,
        });
        // Connect returns the message directly (not wrapped in .msg)
        // The response is GetUsageResponse with properties: organizationId, month, current, estimatedMonthly, quota
        return res;
      } catch (err) {
        console.error("Failed to fetch usage:", err);
        // Return null to show loading state, but log the error
        return null;
      }
    },
    { watch: [selectedOrg] }
  );

  const usage = computed(() => usageData.value);
  
  // Fetch credit transactions (billing history)
  const { data: creditLogData, refresh: refreshCreditLog } = await useClientFetch(
    () =>
      selectedOrg.value
        ? `org-credit-log-${selectedOrg.value}`
        : "org-credit-log-none",
    async () => {
      if (!selectedOrg.value) return { transactions: [], pagination: null };
      const res = await orgClient.getCreditLog({
        organizationId: selectedOrg.value,
        page: 1,
        perPage: 50,
      });
      return res;
    },
    { watch: [selectedOrg] }
  );

  const billingHistory = computed(() => {
    const transactions = creditLogData.value?.transactions || [];
    return transactions
      .filter((t) => t.type === "payment" && t.source === "stripe")
      .map((t) => ({
        id: t.id,
        number: `#${t.id.substring(0, 8).toUpperCase()}`,
        date: t.createdAt
          ? new Date(Number(t.createdAt.seconds) * 1000).toLocaleDateString()
          : "",
        amount: formatCurrency(t.amountCents),
        status: t.amountCents > 0 ? "Paid" : "Refunded",
        transaction: t,
      }));
  });

  // Fetch billing account
  const { data: billingAccountData, refresh: refreshBillingAccount } =
    await useClientFetch(
      () =>
        selectedOrg.value
          ? `billing-account-${selectedOrg.value}`
          : "billing-account-none",
      async () => {
        if (!selectedOrg.value) return null;
        try {
          const res = await billingClient.getBillingAccount({
            organizationId: selectedOrg.value,
          });
          return res.account;
        } catch (err) {
          console.error("Failed to fetch billing account:", err);
          return null;
        }
      },
      { watch: [selectedOrg], server: false }
    );

  const billingAccount = computed(() => billingAccountData.value);

  // Fetch payment methods
  const { data: paymentMethodsData, refresh: refreshPaymentMethods } =
    await useClientFetch(
      () =>
        selectedOrg.value
          ? `payment-methods-${selectedOrg.value}`
          : "payment-methods-none",
      async () => {
        if (!selectedOrg.value) return [];
        try {
          const res = await billingClient.listPaymentMethods({
            organizationId: selectedOrg.value,
          });
          return res.paymentMethods || [];
        } catch (err) {
          console.error("Failed to fetch payment methods:", err);
          return [];
        }
      },
      { watch: [selectedOrg], server: false }
    );

  const paymentMethods = computed(() => paymentMethodsData.value || []);

  // Fetch invoices
  const { data: invoicesData, refresh: refreshInvoices } =
    await useClientFetch(
      () =>
        selectedOrg.value
          ? `invoices-${selectedOrg.value}`
          : "invoices-none",
      async () => {
        if (!selectedOrg.value) return { invoices: [], hasMore: false };
        try {
          const res = await billingClient.listInvoices({
            organizationId: selectedOrg.value,
            limit: 20,
          });
          return { invoices: res.invoices || [], hasMore: res.hasMore || false };
        } catch (err) {
          console.error("Failed to fetch invoices:", err);
          return { invoices: [], hasMore: false };
        }
      },
      { watch: [selectedOrg], server: false }
    );

  const invoices = computed(() => invoicesData.value?.invoices || []);
  const hasMoreInvoices = computed(() => invoicesData.value?.hasMore || false);

  // Access Stripe Customer Portal
  async function openCustomerPortal() {
    if (!selectedOrg.value) return;
    try {
      const response = await billingClient.createPortalSession({
        organizationId: selectedOrg.value,
      });
      if (response.portalUrl) {
        window.location.href = response.portalUrl;
      }
    } catch (err: any) {
      error.value = err.message || "Failed to open customer portal";
    }
  }
  
  // Get current organization object to access credits and plan info
  const currentOrganization = computed((): (Organization & { planInfo?: Organization['planInfo'] }) | null => {
    if (!selectedOrg.value) return null;
    return organizations.value.find((o) => o.id === selectedOrg.value) as (Organization & { planInfo?: Organization['planInfo'] }) | null || null;
  });
  
  const creditsBalance = computed(() => {
    const credits = currentOrganization.value?.credits;
    if (credits === undefined || credits === null) return 0;
    // Handle both bigint (from proto) and number types
    return typeof credits === 'bigint' ? Number(credits) : credits;
  });
  
  const addCreditsDialogOpen = ref(false);
  const addCreditsAmount = ref("");
  const addCreditsLoading = ref(false);

  const addPaymentMethodDialogOpen = ref(false);
  const addPaymentMethodLoading = ref(false);
  const paymentElementLoading = ref(false);
  const stripe = ref<any>(null);
  const stripeElements = ref<any>(null);
  const paymentElement = ref<any>(null);
  const setupIntentClientSecret = ref<string>("");
  const paymentElementContainer = ref<HTMLElement | null>(null);

  // Initialize payment element when dialog opens
  watch(addPaymentMethodDialogOpen, async (isOpen) => {
    if (isOpen) {
      paymentElementLoading.value = true;
      error.value = "";

      // Wait for Stripe to be loaded
      if (!stripe.value) {
        // Wait a bit for Stripe to load
        let attempts = 0;
        while (!stripe.value && attempts < 10) {
          await new Promise(resolve => setTimeout(resolve, 100));
          attempts++;
        }
        
        if (!stripe.value) {
          error.value = "Stripe.js is not loaded. Please ensure NUXT_PUBLIC_STRIPE_PUBLISHABLE_KEY is set and refresh the page.";
          paymentElementLoading.value = false;
          addPaymentMethodDialogOpen.value = false;
          return;
        }
      }

      if (!selectedOrg.value) {
        error.value = "Please select an organization first.";
        paymentElementLoading.value = false;
        addPaymentMethodDialogOpen.value = false;
        return;
      }

      try {
        // Create setup intent
        const response = await billingClient.createSetupIntent({
          organizationId: selectedOrg.value,
        });
        setupIntentClientSecret.value = response.clientSecret;

        // Wait for DOM to be ready - dialog uses ClientOnly so needs extra time
        await nextTick();
        // Additional delay to ensure dialog is fully rendered
        await new Promise(resolve => setTimeout(resolve, 500));

        // Wait for container to be available with retries
        let container = paymentElementContainer.value;
        if (!container) {
          // Try to find it by ID as fallback
          container = document.getElementById("payment-element");
          if (container) {
            paymentElementContainer.value = container;
          }
        }

        // Retry finding the container if still not found
        if (!container) {
          let retries = 0;
          while (!container && retries < 10) {
            await new Promise(resolve => setTimeout(resolve, 100));
            container = paymentElementContainer.value || document.getElementById("payment-element");
            if (container && !paymentElementContainer.value) {
              paymentElementContainer.value = container;
            }
            retries++;
          }
        }

        if (!container) {
          console.error("Payment element container not found after retries");
          error.value = "Payment form container not found. Please try again.";
          paymentElementLoading.value = false;
          return;
        }

        // Clear any existing elements
        if (stripeElements.value) {
          stripeElements.value.clear();
        }

        // Import Stripe theme helper
        const { getStripeAppearance } = await import('~/utils/stripe-theme');
        
        // Create elements with Oui-themed appearance
        stripeElements.value = stripe.value.elements({
          clientSecret: setupIntentClientSecret.value,
          appearance: getStripeAppearance(),
        });

        paymentElement.value = stripeElements.value.create("payment");
        paymentElement.value.mount(container);
        paymentElementLoading.value = false;
      } catch (err: any) {
        console.error("Failed to initialize payment form:", err);
        error.value = err.message || "Failed to initialize payment form";
        paymentElementLoading.value = false;
        addPaymentMethodDialogOpen.value = false;
      }
    } else if (!isOpen && paymentElement.value) {
      // Cleanup when dialog closes
      paymentElement.value?.unmount();
      paymentElement.value = null;
      stripeElements.value = null;
      setupIntentClientSecret.value = "";
      paymentElementLoading.value = false;
    }
  });

  async function addPaymentMethod() {
    if (!stripe.value || !paymentElement.value || !selectedOrg.value) return;

    addPaymentMethodLoading.value = true;
    error.value = "";

    try {
      // Confirm setup intent
      const { setupIntent, error: confirmError } =
        await stripe.value.confirmSetup({
          elements: stripeElements.value,
          redirect: "if_required",
        });

      if (confirmError) {
        throw new Error(confirmError.message);
      }

      if (setupIntent && setupIntent.payment_method) {
        // Attach payment method to customer
        await billingClient.attachPaymentMethod({
          organizationId: selectedOrg.value,
          paymentMethodId: setupIntent.payment_method as string,
        });

        // Refresh payment methods
        await refreshPaymentMethods();
        await refreshBillingAccount();

        addPaymentMethodDialogOpen.value = false;
        useToast().toast.success("Payment method added successfully");
      }
    } catch (err: any) {
      error.value = err.message || "Failed to add payment method";
    } finally {
      addPaymentMethodLoading.value = false;
    }
  }

  async function setDefaultPaymentMethod(paymentMethodId: string) {
    if (!selectedOrg.value) return;
    try {
      await billingClient.setDefaultPaymentMethod({
        organizationId: selectedOrg.value,
        paymentMethodId,
      });
      await refreshPaymentMethods();
      await refreshBillingAccount();
      useToast().toast.success("Default payment method updated");
    } catch (err: any) {
      error.value = err.message || "Failed to set default payment method";
      useToast().toast.error(error.value);
    }
  }

  async function removePaymentMethod(paymentMethodId: string) {
    if (!selectedOrg.value) return;
    if (!confirm("Are you sure you want to remove this payment method?")) {
      return;
    }
    try {
      await billingClient.detachPaymentMethod({
        organizationId: selectedOrg.value,
        paymentMethodId,
      });
      await refreshPaymentMethods();
      await refreshBillingAccount();
      useToast().toast.success("Payment method removed");
    } catch (err: any) {
      error.value = err.message || "Failed to remove payment method";
      useToast().toast.error(error.value);
    }
  }

  function formatCardBrand(brand: string): string {
    const brands: Record<string, string> = {
      visa: "Visa",
      mastercard: "Mastercard",
      amex: "American Express",
      discover: "Discover",
      jcb: "JCB",
      diners: "Diners Club",
      unionpay: "UnionPay",
    };
    return brands[brand.toLowerCase()] || brand;
  }

  function getInvoiceStatusVariant(status: string): "success" | "warning" | "danger" | "secondary" {
    switch (status.toLowerCase()) {
      case "paid":
        return "success";
      case "open":
      case "draft":
        return "warning";
      case "uncollectible":
      case "void":
        return "danger";
      default:
        return "secondary";
    }
  }

  function formatInvoiceDate(date: any): string {
    if (!date) return "";
    const seconds = typeof date === 'object' && 'seconds' ? Number(date.seconds) : typeof date === 'number' ? date : 0;
    if (seconds === 0) return "";
    return new Date(seconds * 1000).toLocaleDateString();
  }

  function openInvoiceUrl(url: string) {
    if (typeof window !== 'undefined' && url) {
      window.open(url, '_blank');
    }
  }

  // Load Stripe.js and handle payment redirects
  onMounted(async () => {
    // Load Stripe.js
    if (typeof window !== "undefined") {
      const publishableKey = config.public.stripePublishableKey;
      
      // Check if Stripe is already loaded
      if (window.Stripe) {
        if (publishableKey) {
          stripe.value = window.Stripe(publishableKey);
        }
      } else {
        // Load Stripe.js dynamically
        const stripeScript = document.createElement("script");
        stripeScript.src = "https://js.stripe.com/v3/";
        stripeScript.async = true;
        document.head.appendChild(stripeScript);

        stripeScript.onload = () => {
          if (window.Stripe && publishableKey) {
            stripe.value = window.Stripe(publishableKey);
          } else {
            console.error("Stripe.js loaded but publishable key not configured");
          }
        };

        stripeScript.onerror = () => {
          console.error("Failed to load Stripe.js");
          error.value = "Failed to load payment form. Please refresh the page.";
        };
      }
    }

    // Handle payment success/cancel redirects
    if (route.query.payment === "success") {
      const { toast } = useToast();
      
      // Store initial credits balance for comparison
      const initialCredits = currentOrganization.value?.credits 
        ? (typeof currentOrganization.value.credits === 'bigint' 
            ? Number(currentOrganization.value.credits) 
            : currentOrganization.value.credits)
        : 0;
      
      // Refresh data immediately
      syncOrganizations().then(() => {
        refreshCreditLog();
        refreshBillingAccount();
        refreshPaymentMethods();
        refreshInvoices();
      });
      
      // Retry syncing organizations a few times to catch webhook delay
      // Webhooks might take a moment to process
      let retries = 0;
      const maxRetries = 5;
      const retryInterval = 2000; // 2 seconds
      
      const retrySync = setInterval(async () => {
        retries++;
        await syncOrganizations();
        
        // Check if credits have updated
        const currentOrg = organizations.value.find((o) => o.id === selectedOrg.value);
        if (currentOrg?.credits) {
          const currentCredits = typeof currentOrg.credits === 'bigint' 
            ? Number(currentOrg.credits) 
            : currentOrg.credits;
          
          if (currentCredits > initialCredits) {
            clearInterval(retrySync);
            toast.success("Payment successful! Credits have been added to your account.");
            navigateTo({ query: { ...route.query, payment: undefined } });
            return;
          }
        }
        
        if (retries >= maxRetries) {
          clearInterval(retrySync);
          toast.success("Payment successful! Credits are being processed and will appear shortly.");
          navigateTo({ query: { ...route.query, payment: undefined } });
        }
      }, retryInterval);
      
      // Initial success message
      toast.success("Payment successful! Updating credits...");
    } else if (route.query.payment === "canceled") {
      const { toast } = useToast();
      toast.info("Payment canceled.");
      navigateTo({ query: { ...route.query, payment: undefined } });
    }
  });
  
  // Format helper functions
  const formatBytes = (bytes: number | bigint | null | undefined) => {
    if (bytes === null || bytes === undefined) return "0 B";
    const b = Number(bytes);
    if (b === 0 || !Number.isFinite(b)) return "0 B";
    const k = 1024;
    const sizes = ["B", "KB", "MB", "GB", "TB"];
    const i = Math.floor(Math.log(b) / Math.log(k));
    return `${(b / Math.pow(k, i)).toFixed(2)} ${sizes[i]}`;
  };

  const formatBytesToGB = (bytes: number | bigint) => {
    const b = Number(bytes);
    if (b === 0) return "0.00";
    return (b / (1024 * 1024 * 1024)).toFixed(2);
  };

  const formatMemoryByteSecondsToGB = (byteSeconds: number | bigint | null | undefined) => {
    if (byteSeconds === null || byteSeconds === undefined) return "0.00";
    const bs = Number(byteSeconds);
    if (bs === 0 || !Number.isFinite(bs)) return "0.00";
    // Convert byte-seconds to GB-hours, then show as GB (for average memory usage)
    const now = new Date();
    const monthStart = new Date(now.getFullYear(), now.getMonth(), 1);
    const secondsInMonth = Math.max(1, Math.floor((now.getTime() - monthStart.getTime()) / 1000));
    const avgBytes = bs / secondsInMonth;
    return formatBytesToGB(avgBytes);
  };

  const formatCoreSecondsToHours = (coreSeconds: number | bigint | null | undefined) => {
    if (coreSeconds === null || coreSeconds === undefined) return "0.00";
    const s = Number(coreSeconds);
    if (!Number.isFinite(s) || s === 0) return "0.00";
    return (s / 3600).toFixed(2);
  };

  const formatCPUUsage = (coreSeconds: number | bigint | null | undefined): string => {
    if (coreSeconds === null || coreSeconds === undefined) return "0.00";
    const s = Number(coreSeconds);
    if (!Number.isFinite(s) || s === 0) return "0.00";
    const hours = s / 3600;
    if (hours < 1) {
      return `${(s / 60).toFixed(1)} min`;
    }
    return `${hours.toFixed(1)} core-hrs`;
  };

  const formatCurrency = (cents: number | bigint) => {
    return formatCurrencyUtil(cents);
  };

  const getUsagePercentage = (
    current: number | bigint,
    quota: number | bigint
  ) => {
    const c = Number(current);
    const q = Number(quota);
    if (q === 0 || !Number.isFinite(q)) return 0; // Unlimited quota
    if (!Number.isFinite(c) || c === 0) return 0;
    return Math.min(100, Math.max(0, Math.round((c / q) * 100)));
  };

  const getUsageBadgeVariant = (percentage: number) => {
    if (percentage >= 90) return "danger";
    if (percentage >= 75) return "warning";
    return "success";
  };
</script>

<template>
  <OuiStack gap="lg">
    <OuiGrid cols="1" colsLg="3" gap="lg">
      <OuiCard class="col-span-2">
        <OuiCardHeader>
          <OuiFlex align="center" justify="between">
            <OuiStack gap="xs">
              <OuiText size="xl" weight="semibold">Organizations</OuiText>
              <OuiText color="muted">Create and manage your teams.</OuiText>
            </OuiStack>
            <OuiButton variant="ghost" @click="refreshAll">Refresh</OuiButton>
          </OuiFlex>
        </OuiCardHeader>
        <OuiCardBody>
          <OuiStack gap="lg">
            <OuiGrid cols="1" colsLg="2" gap="md">
              <OuiStack gap="xs">
                <OuiText size="sm" weight="medium">Select Organization</OuiText>
                <OuiSelect
                  v-model="selectedOrg"
                  placeholder="Choose organization"
                  :items="organizationSelectItems"
                />
              </OuiStack>
              <OuiStack gap="xs">
                <OuiText size="sm" weight="medium">Your Role</OuiText>
                <OuiBadge
                  v-if="currentMemberRecord"
                  tone="solid"
                  variant="primary"
                >
                  {{ roleLabel(currentMemberRecord.role) }}
                </OuiBadge>
                <OuiText v-else color="muted" size="sm">
                  You are not a member of this organization.
                </OuiText>
              </OuiStack>
            </OuiGrid>
            <div class="border border-border-muted/40 rounded-xl" />
            
            <!-- Plan Information -->
            <OuiCard v-if="selectedOrg && currentOrganization?.planInfo" variant="outline">
              <OuiCardHeader>
                <OuiFlex align="center" justify="between">
                  <OuiStack gap="xs">
                    <OuiText size="lg" weight="semibold">Current Plan: {{ currentOrganization.planInfo.planName }}</OuiText>
                    <OuiText size="sm" color="muted" v-if="currentOrganization.planInfo.description">
                      {{ currentOrganization.planInfo.description }}
                    </OuiText>
                  </OuiStack>
                </OuiFlex>
              </OuiCardHeader>
              <OuiCardBody>
                <OuiStack gap="md">
                  <OuiText size="sm" color="muted">
                    Your organization has resource limits based on your plan. Adding credits or making payments may automatically upgrade your plan.
                  </OuiText>
                  <OuiGrid cols="1" cols-md="2" cols-lg="6" gap="md">
                    <OuiStack gap="xs">
                      <OuiText size="xs" color="muted">CPU Cores</OuiText>
                      <OuiText size="sm" weight="medium">
                        {{ currentOrganization.planInfo.cpuCores || 'Unlimited' }}
                      </OuiText>
                    </OuiStack>
                    <OuiStack gap="xs">
                      <OuiText size="xs" color="muted">Memory</OuiText>
                      <OuiText size="sm" weight="medium">
                        {{ formatBytes(Number(currentOrganization.planInfo.memoryBytes || 0)) || 'Unlimited' }}
                      </OuiText>
                    </OuiStack>
                    <OuiStack gap="xs">
                      <OuiText size="xs" color="muted">Max Deployments</OuiText>
                      <OuiText size="sm" weight="medium">
                        {{ currentOrganization.planInfo.deploymentsMax || 'Unlimited' }}
                      </OuiText>
                    </OuiStack>
                    <OuiStack gap="xs">
                      <OuiText size="xs" color="muted">Max VPS Instances</OuiText>
                      <OuiText size="sm" weight="medium">
                        {{ currentOrganization.planInfo.maxVpsInstances || 'Unlimited' }}
                      </OuiText>
                    </OuiStack>
                    <OuiStack gap="xs">
                      <OuiText size="xs" color="muted">Bandwidth/Month</OuiText>
                      <OuiText size="sm" weight="medium">
                        {{ formatBytes(Number(currentOrganization.planInfo.bandwidthBytesMonth || 0)) || 'Unlimited' }}
                      </OuiText>
                    </OuiStack>
                    <OuiStack gap="xs">
                      <OuiText size="xs" color="muted">Storage</OuiText>
                      <OuiText size="sm" weight="medium">
                        {{ formatBytes(Number(currentOrganization.planInfo.storageBytes || 0)) || 'Unlimited' }}
                      </OuiText>
                    </OuiStack>
                  </OuiGrid>
                  <OuiAlert v-if="currentOrganization.planInfo.minimumPaymentCents > 0" variant="info">
                    <OuiText size="sm">
                      <strong>Auto-Upgrade:</strong> Organizations that pay at least 
                      {{ formatCurrency(Number(currentOrganization.planInfo.minimumPaymentCents)) }} 
                      will automatically be upgraded to this plan.
                    </OuiText>
                  </OuiAlert>
                  <OuiAlert v-if="currentOrganization.planInfo.monthlyFreeCreditsCents > 0" variant="success">
                    <OuiText size="sm">
                      <strong>Monthly Free Credits:</strong> This plan includes 
                      {{ formatCurrency(Number(currentOrganization.planInfo.monthlyFreeCreditsCents)) }} 
                      in free credits automatically added to your account on the 1st of each month.
                    </OuiText>
                  </OuiAlert>
                </OuiStack>
              </OuiCardBody>
            </OuiCard>
            
            <OuiStack gap="md" as="form" @submit.prevent="createOrg">
              <OuiText size="sm" weight="medium">Create Organization</OuiText>
              <OuiGrid cols="1" colsLg="2" gap="md">
                <OuiInput
                  v-model="name"
                  label="Name"
                  placeholder="Acme Inc"
                  required
                />
                <OuiInput v-model="slug" label="Slug" placeholder="acme" />
              </OuiGrid>
              <OuiFlex gap="sm">
                <OuiButton type="submit">Create</OuiButton>
                <OuiText v-if="error" color="danger">{{ error }}</OuiText>
              </OuiFlex>
            </OuiStack>
          </OuiStack>
        </OuiCardBody>
      </OuiCard>

      <OuiCard>
        <OuiCardHeader>
          <OuiText size="lg" weight="semibold">Member Stats</OuiText>
        </OuiCardHeader>
        <OuiCardBody>
          <OuiStack gap="lg">
            <OuiStack gap="xs">
              <OuiText size="2xl" weight="semibold">{{
                memberStats.total
              }}</OuiText>
              <OuiText color="muted" size="sm">Active members</OuiText>
            </OuiStack>
            <OuiGrid cols="2" gap="md">
              <OuiStack gap="xs">
                <OuiText size="lg" weight="semibold">{{ memberStats.owners }}</OuiText>
                <OuiText color="muted" size="sm">Owners</OuiText>
              </OuiStack>
              <OuiStack gap="xs">
                <OuiText size="lg" weight="semibold">{{ memberStats.admins }}</OuiText>
                <OuiText color="muted" size="sm">Admins</OuiText>
              </OuiStack>
              <OuiStack gap="xs">
                <OuiText size="lg" weight="semibold">{{ memberStats.members }}</OuiText>
                <OuiText color="muted" size="sm">Members</OuiText>
              </OuiStack>
              <OuiStack gap="xs">
                <OuiText size="lg" weight="semibold">{{ memberStats.viewers }}</OuiText>
                <OuiText color="muted" size="sm">Viewers</OuiText>
              </OuiStack>
            </OuiGrid>
            <OuiStack gap="xs">
              <OuiText size="lg" weight="semibold">{{ memberStats.pending }}</OuiText>
              <OuiText color="muted" size="sm">Pending invites</OuiText>
            </OuiStack>
          </OuiStack>
        </OuiCardBody>
      </OuiCard>
    </OuiGrid>

    <OuiTabs v-model="activeTab" :tabs="tabs" />

    <OuiCard>
      <OuiCardBody>
        <OuiTabs v-model="activeTab" :tabs="tabs" :content-only="true">
          <template #members>
            <OuiFlex align="center" justify="between" class="mb-4">
              <OuiStack gap="xs">
                <OuiText size="lg" weight="semibold">Members</OuiText>
                <OuiText color="muted" size="sm">
                  Manage member roles and ownership.
                </OuiText>
              </OuiStack>
              <OuiButton
                variant="ghost"
                size="sm"
                @click="refreshMembers"
                :disabled="!selectedOrg"
              >
                <ArrowPathIcon class="h-4 w-4 mr-1" />
                Refresh
              </OuiButton>
            </OuiFlex>

            <div class="overflow-x-auto">
              <table class="min-w-full text-left text-sm">
                <thead>
                  <tr class="text-text-muted uppercase text-xs tracking-wide">
                    <th class="px-4 py-2">Member</th>
                    <th class="px-4 py-2">Email</th>
                    <th class="px-4 py-2">Role</th>
                    <th class="px-4 py-2">Status</th>
                    <th class="px-4 py-2 w-64">Actions</th>
                  </tr>
                </thead>
                <tbody>
                  <tr
                    v-for="member in members"
                    :key="member.id"
                    class="border-t border-border-muted/40"
                  >
                    <td class="px-4 py-3">
                      <OuiFlex gap="sm" align="center">
                        <OuiAvatar
                          :name="
                            member.user?.name ||
                            member.user?.email ||
                            member.user?.id ||
                            ''
                          "
                          :src="member.user?.avatarUrl"
                        />
                        <div>
                          <OuiText weight="medium">
                            {{
                              member.user?.name ||
                              member.user?.email ||
                              member.user?.id
                            }}
                          </OuiText>
                          <OuiText color="muted" size="xs">
                            {{
                              member.user?.preferredUsername || member.user?.id
                            }}
                          </OuiText>
                        </div>
                      </OuiFlex>
                    </td>
                    <td class="px-4 py-3 text-text-secondary">
                      {{ member.user?.email || "—" }}
                    </td>
                    <td class="px-4 py-3">
                      <OuiSelect
                        :model-value="member.role"
                        :items="roleItems"
                        :disabled="
                          !canUpdateMembers ||
                          (member.role === 'owner' && currentUserIsOwner)
                        "
                        @update:model-value="(r) => setRole(member.id, r as string)"
                      />
                    </td>
                    <td class="px-4 py-3">
                      <OuiBadge
                        :tone="member.status === 'active' ? 'solid' : 'soft'"
                        :variant="
                          member.status === 'active' ? 'success' : 'secondary'
                        "
                      >
                        {{ member.status }}
                      </OuiBadge>
                    </td>
                    <td class="px-4 py-3">
                      <OuiFlex gap="sm">
                        <OuiButton
                          v-if="
                            currentUserIsOwner &&
                            member.status === 'active' &&
                            member.role !== 'owner'
                          "
                          size="sm"
                          variant="ghost"
                          @click="openTransferDialog(member)"
                        >
                          Transfer Ownership
                        </OuiButton>
                        <OuiButton
                          v-if="member.status === 'invited' && (currentUserIsOwner || currentMemberRecord?.role === 'admin')"
                          size="sm"
                          variant="ghost"
                          @click="resendInvite(member)"
                          :disabled="resendingInvite === member.id"
                        >
                          {{ resendingInvite === member.id ? 'Sending...' : 'Resend' }}
                        </OuiButton>
                        <OuiButton
                          v-if="currentUserIsOwner && member.role !== 'owner'"
                          size="sm"
                          variant="ghost"
                          color="danger"
                          @click="remove(member.id)"
                        >
                          {{ member.status === 'invited' ? 'Uninvite' : 'Remove' }}
                        </OuiButton>
                      </OuiFlex>
                    </td>
                  </tr>
                  <tr v-if="!members.length">
                    <td
                      colspan="5"
                      class="px-4 py-6 text-center text-text-muted"
                    >
                      No members yet. Invite someone to get started.
                    </td>
                  </tr>
                </tbody>
              </table>
            </div>

            <OuiStack gap="md" class="mt-6">
              <OuiText size="md" weight="semibold">Invite Member</OuiText>
              <OuiGrid cols="1" colsLg="3" gap="md">
                <OuiInput
                  label="Email"
                  v-model="inviteEmail"
                  placeholder="user@example.com"
                />
                <OuiSelect
                  label="Role"
                  v-model="inviteRole"
                  :items="roleItems"
                  :disabled="!canUpdateMembers"
                />
                <OuiFlex align="end">
                  <OuiButton @click="invite" :disabled="inviteDisabled || inviting">
                    {{ inviting ? 'Sending...' : 'Invite' }}
                  </OuiButton>
                </OuiFlex>
              </OuiGrid>
            </OuiStack>
          </template>

          <template #invitations>
            <OuiStack gap="md">
              <OuiFlex align="center" justify="between" class="mb-4">
                <OuiStack gap="xs">
                  <OuiText size="lg" weight="semibold">My Invitations</OuiText>
                  <OuiText color="muted" size="sm">
                    Accept or decline invitations to join organizations.
                  </OuiText>
                </OuiStack>
                <OuiButton
                  variant="ghost"
                  size="sm"
                  @click="refreshInvitations"
                  :disabled="loadingInvitations"
                >
                  <ArrowPathIcon class="h-4 w-4 mr-1" :class="{ 'animate-spin': loadingInvitations }" />
                  Refresh
                </OuiButton>
              </OuiFlex>

              <div v-if="loadingInvitations" class="text-center py-8">
                <OuiText color="muted">Loading invitations...</OuiText>
              </div>

              <div v-else-if="myInvitations.length === 0" class="text-center py-8">
                <OuiText color="muted">You have no pending invitations.</OuiText>
              </div>

              <OuiStack v-else gap="md">
                <OuiCard
                  v-for="invite in myInvitations"
                  :key="invite.id"
                  class="border border-border-muted rounded-xl"
                >
                  <OuiCardBody>
                    <OuiFlex align="center" justify="between" wrap="wrap" gap="md">
                      <OuiStack gap="xs" class="flex-1 min-w-0">
                        <OuiText size="lg" weight="semibold">{{ invite.organizationName }}</OuiText>
                        <OuiText color="muted" size="sm">
                          You've been invited to join as <span class="uppercase font-medium">{{ invite.role }}</span>
                        </OuiText>
                        <OuiText color="muted" size="xs">
                          Invited <OuiDate :value="invite.invitedAt" />
                        </OuiText>
                      </OuiStack>
                      <OuiFlex gap="sm" wrap="wrap">
                        <OuiButton
                          variant="ghost"
                          color="danger"
                          @click="declineInvite(invite)"
                          :disabled="processingInvite === invite.id"
                        >
                          {{ processingInvite === invite.id && decliningInvite ? 'Declining...' : 'Decline' }}
                        </OuiButton>
                        <OuiButton
                          @click="acceptInvite(invite)"
                          :disabled="processingInvite === invite.id"
                        >
                          {{ processingInvite === invite.id && !decliningInvite ? 'Accepting...' : 'Accept' }}
                        </OuiButton>
                      </OuiFlex>
                    </OuiFlex>
                  </OuiCardBody>
                </OuiCard>
              </OuiStack>
            </OuiStack>
          </template>

          <template #roles>
            <OuiStack gap="md">
              <OuiText size="lg" weight="semibold">Roles</OuiText>
              <OuiText color="muted" size="sm">
                System roles appear first. Any custom roles you create follow
                below.
              </OuiText>
              <OuiGrid cols="1" colsLg="2" gap="md">
                <OuiCard v-for="item in roleItems" :key="item.value">
                  <OuiCardBody>
                    <OuiStack gap="xs">
                      <OuiFlex align="center" justify="between">
                        <OuiText weight="medium">{{ item.label }}</OuiText>
                        <CheckIcon
                          v-if="item.disabled"
                          class="h-4 w-4 text-text-muted"
                        />
                      </OuiFlex>
                      <OuiText color="muted" size="sm">
                        {{
                          item.disabled ? "Reserved system role" : "Assignable"
                        }}
                      </OuiText>
                    </OuiStack>
                  </OuiCardBody>
                </OuiCard>
              </OuiGrid>
            </OuiStack>
          </template>
          <template #ssh-keys>
            <OrganizationSSHKeys :organization-id="selectedOrg" />
          </template>
          <template #audit-logs>
            <AuditLogs
              :organization-id="selectedOrg"
              resource-type="organization"
              :resource-id="selectedOrg"
            />
          </template>
        </OuiTabs>
      </OuiCardBody>
    </OuiCard>

    <OuiDialog v-model:open="transferDialogOpen" title="Transfer Ownership">
      <p class="text-sm text-text-muted">
        {{ transferDialogSummary }}
      </p>
      <template #footer>
        <OuiFlex gap="sm" justify="end">
          <OuiButton variant="ghost" @click="transferDialogOpen = false">
            Cancel
          </OuiButton>
          <OuiButton
            color="primary"
            @click="confirmTransferOwnership"
            :disabled="!transferCandidate"
          >
            Confirm transfer
          </OuiButton>
        </OuiFlex>
      </template>
    </OuiDialog>

    <!-- Add Payment Method Dialog -->
    <OuiDialog v-model:open="addPaymentMethodDialogOpen" title="Add Payment Method">
      <OuiStack gap="lg">
        <OuiStack gap="xs">
          <OuiText size="sm" color="muted">
            Add a new payment method to your account. You can use this for future purchases.
          </OuiText>
          <OuiText v-if="error" size="sm" color="danger">{{ error }}</OuiText>
          <OuiText v-if="!config.public.stripePublishableKey" size="sm" color="warning">
            Warning: Stripe publishable key not configured. Please set NUXT_PUBLIC_STRIPE_PUBLISHABLE_KEY.
          </OuiText>
        </OuiStack>
        
        <!-- Payment Element Container -->
        <div v-if="paymentElementLoading" class="min-h-[200px] flex items-center justify-center">
          <OuiText size="sm" color="muted">Loading payment form...</OuiText>
        </div>
        <div 
          ref="paymentElementContainer" 
          id="payment-element" 
          class="min-h-[200px]"
          :class="{ 'hidden': paymentElementLoading }"
        ></div>

        <OuiFlex justify="end" gap="sm">
          <OuiButton variant="ghost" @click="addPaymentMethodDialogOpen = false">
            Cancel
          </OuiButton>
          <OuiButton 
            variant="solid" 
            @click="addPaymentMethod"
            :disabled="addPaymentMethodLoading || paymentElementLoading || !paymentElement"
          >
            {{ addPaymentMethodLoading ? "Processing..." : "Add Payment Method" }}
          </OuiButton>
        </OuiFlex>
      </OuiStack>
    </OuiDialog>

    <!-- Add Credits Dialog -->
    <OuiDialog v-model:open="addCreditsDialogOpen" title="Purchase Credits">
      <OuiStack gap="lg">
        <OuiStack gap="xs">
          <OuiText size="sm" color="muted">
            Purchase credits for your organization. You will be redirected to Stripe Checkout to complete the payment.
          </OuiText>
          <OuiText v-if="error" size="sm" color="danger">{{ error }}</OuiText>
        </OuiStack>
        
        <OuiStack gap="md">
          <OuiStack gap="xs">
            <OuiText size="sm" weight="medium">Amount (USD)</OuiText>
            <OuiInput
              v-model="addCreditsAmount"
              type="number"
              step="0.01"
              min="0.50"
              placeholder="0.50"
              :error="addCreditsAmount && parseFloat(addCreditsAmount) < 0.50 ? 'Minimum amount is $0.50 USD' : undefined"
            />
            <OuiText size="xs" color="muted">
              Minimum purchase amount is $0.50 USD
            </OuiText>
          </OuiStack>
        </OuiStack>

        <OuiFlex justify="end" gap="sm">
          <OuiButton variant="ghost" @click="addCreditsDialogOpen = false">
            Cancel
          </OuiButton>
          <OuiButton 
            variant="solid" 
            @click="addCredits"
            :disabled="addCreditsLoading || !addCreditsAmount || parseFloat(addCreditsAmount) < 0.50 || isNaN(parseFloat(addCreditsAmount))"
          >
            {{ addCreditsLoading ? "Processing..." : "Continue to Payment" }}
          </OuiButton>
        </OuiFlex>
      </OuiStack>
    </OuiDialog>
  </OuiStack>
</template>
