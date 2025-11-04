<script setup lang="ts">
  definePageMeta({ layout: "default", middleware: "auth" });
  import {
    OrganizationService,
    BillingService,
  } from "@obiente/proto";
  import { useConnectClient } from "~/lib/connect-client";
  import {
    PlusIcon,
    CreditCardIcon,
    ArrowDownTrayIcon,
    DocumentTextIcon,
  } from "@heroicons/vue/24/outline";

  const error = ref("");
  const auth = useAuth();
  const orgClient = useConnectClient(OrganizationService);
  const billingClient = useConnectClient(BillingService);
  const route = useRoute();
  const config = useRuntimeConfig();

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
  const selectedOrg = computed({
    get: () => auth.currentOrganizationId,
    set: (id: string) => {
      if (id) auth.switchOrganization(id);
    },
  });

  // Load organizations if not already loaded
  if (!organizations.value.length && auth.isAuthenticated) {
    const res = await orgClient.listOrganizations({ onlyMine: true });
    auth.setOrganizations(res.organizations || []);
    if (targetOrgId) {
      auth.switchOrganization(targetOrgId);
    }
  }

  async function syncOrganizations() {
    if (!auth.isAuthenticated) return;
    const res = await orgClient.listOrganizations({});
    auth.setOrganizations(res.organizations || []);
  }

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

  // Get current member record for permission checks
  const { data: membersData } = await useAsyncData(
    () =>
      selectedOrg.value
        ? `org-members-${selectedOrg.value}`
        : "org-members-none",
    async () => {
      if (!selectedOrg.value) return [];
      try {
        const res = await orgClient.listMembers({
          organizationId: selectedOrg.value,
        });
        return res.members || [];
      } catch {
        return [];
      }
    },
    { watch: [selectedOrg], server: false }
  );

  const members = computed(() => membersData.value || []);

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

  const currentMonth = computed(() => {
    const now = new Date();
    return now.toLocaleString("default", { month: "long", year: "numeric" });
  });

  // Fetch usage data
  const { data: usageData, refresh: refreshUsage } = await useAsyncData(
    () =>
      selectedOrg.value
        ? `org-usage-${selectedOrg.value}`
        : "org-usage-none",
    async () => {
      if (!selectedOrg.value) return null;
      try {
        const res = await orgClient.getUsage({
          organizationId: selectedOrg.value,
        });
        return res;
      } catch (err) {
        console.error("Failed to fetch usage:", err);
        return null;
      }
    },
    { watch: [selectedOrg], server: false }
  );

  const usage = computed(() => usageData.value);
  
  // Fetch credit transactions (billing history)
  const { data: creditLogData, refresh: refreshCreditLog } = await useAsyncData(
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
    { watch: [selectedOrg], server: false }
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
    await useAsyncData(
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
    await useAsyncData(
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
    await useAsyncData(
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
  
  // Get current organization object to access credits
  const currentOrganization = computed(() => {
    if (!selectedOrg.value) return null;
    return organizations.value.find((o) => o.id === selectedOrg.value) || null;
  });
  
  const creditsBalance = computed(() => {
    const credits = currentOrganization.value?.credits;
    if (credits === undefined || credits === null) return 0;
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

  async function addCredits() {
    if (!selectedOrg.value || !addCreditsAmount.value) return;
    const amount = parseFloat(addCreditsAmount.value);
    
    if (isNaN(amount) || amount < 0.50) {
      const { toast } = useToast();
      toast.error("Minimum purchase amount is $0.50 USD");
      error.value = "Minimum purchase amount is $0.50 USD";
      return;
    }
    
    addCreditsLoading.value = true;
    error.value = "";
    try {
      const response = await billingClient.createCheckoutSession({
        organizationId: selectedOrg.value,
        amountCents: BigInt(Math.round(amount * 100)),
      });
      
      if (response.checkoutUrl) {
        window.location.href = response.checkoutUrl;
      } else {
        throw new Error("No checkout URL received");
      }
    } catch (err: any) {
      error.value = err.message || "Failed to create checkout session";
      addCreditsLoading.value = false;
    }
  }

  // Initialize payment element when dialog opens
  watch(addPaymentMethodDialogOpen, async (isOpen) => {
    if (isOpen) {
      paymentElementLoading.value = true;
      error.value = "";

      if (!stripe.value) {
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
        const response = await billingClient.createSetupIntent({
          organizationId: selectedOrg.value,
        });
        setupIntentClientSecret.value = response.clientSecret;

        await nextTick();
        await new Promise(resolve => setTimeout(resolve, 500));

        let container = paymentElementContainer.value;
        if (!container) {
          container = document.getElementById("payment-element");
          if (container) {
            paymentElementContainer.value = container;
          }
        }

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

        if (stripeElements.value) {
          stripeElements.value.clear();
        }

        const { getStripeAppearance } = await import('~/utils/stripe-theme');
        
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
      const { setupIntent, error: confirmError } =
        await stripe.value.confirmSetup({
          elements: stripeElements.value,
          redirect: "if_required",
        });

      if (confirmError) {
        throw new Error(confirmError.message);
      }

      if (setupIntent && setupIntent.payment_method) {
        await billingClient.attachPaymentMethod({
          organizationId: selectedOrg.value,
          paymentMethodId: setupIntent.payment_method as string,
        });

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
    if (typeof window !== "undefined") {
      const publishableKey = config.public.stripePublishableKey;
      
      if (window.Stripe) {
        if (publishableKey) {
          stripe.value = window.Stripe(publishableKey);
        }
      } else {
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
      
      const initialCredits = currentOrganization.value?.credits 
        ? (typeof currentOrganization.value.credits === 'bigint' 
            ? Number(currentOrganization.value.credits) 
            : currentOrganization.value.credits)
        : 0;
      
      syncOrganizations().then(() => {
        refreshCreditLog();
        refreshBillingAccount();
        refreshPaymentMethods();
        refreshInvoices();
      });
      
      let retries = 0;
      const maxRetries = 5;
      const retryInterval = 2000;
      
      const retrySync = setInterval(async () => {
        retries++;
        await syncOrganizations();
        
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
    const hours = s / 3600;
    if (hours < 1) {
      return `${(s / 60).toFixed(1)} min`;
    }
    return `${hours.toFixed(1)} core-hrs`;
  };

  const formatCurrency = (cents: number | bigint) => {
    const c = Number(cents);
    return new Intl.NumberFormat("en-US", {
      style: "currency",
      currency: "USD",
    }).format(c / 100);
  };

  const getUsagePercentage = (
    current: number | bigint,
    quota: number | bigint
  ) => {
    const c = Number(current);
    const q = Number(quota);
    if (q === 0 || !Number.isFinite(q)) return 0;
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
    <OuiCard>
      <OuiCardHeader>
        <OuiFlex align="center" justify="between">
          <OuiStack gap="xs">
            <OuiText size="xl" weight="semibold">Billing & Usage</OuiText>
            <OuiText color="muted">Manage billing, payment methods, and view usage for your organization.</OuiText>
          </OuiStack>
        </OuiFlex>
      </OuiCardHeader>
      <OuiCardBody>
        <OuiStack gap="lg">
          <!-- Organization Selector -->
          <OuiGrid cols="1" colsLg="2" gap="md">
            <OuiStack gap="xs">
              <OuiText size="sm" weight="medium">Select Organization</OuiText>
              <OuiSelect
                v-model="selectedOrg"
                placeholder="Choose organization"
                :items="
                  organizations.map((o) => ({
                    label: o.name ?? o.slug ?? o.id,
                    value: o.id,
                  }))
                "
              />
            </OuiStack>
            <OuiStack gap="xs" v-if="currentMemberRecord">
              <OuiText size="sm" weight="medium">Your Role</OuiText>
              <OuiBadge tone="solid" variant="primary">
                {{ currentMemberRecord.role.charAt(0).toUpperCase() + currentMemberRecord.role.slice(1) }}
              </OuiBadge>
            </OuiStack>
          </OuiGrid>

          <template v-if="!selectedOrg">
            <OuiBox p="xl" class="text-center">
              <OuiStack gap="md" align="center">
                <CreditCardIcon class="h-12 w-12 text-muted" />
                <OuiText size="lg" weight="semibold">Select an Organization</OuiText>
                <OuiText size="sm" color="muted">
                  Please select an organization to view billing and usage information.
                </OuiText>
              </OuiStack>
            </OuiBox>
          </template>

          <template v-else>
            <!-- Current Usage -->
            <OuiStack gap="lg">
              <OuiText size="2xl" weight="bold">Current Usage</OuiText>
              <template v-if="!usage || !usage.current">
                <OuiGrid cols="1" cols-md="2" cols-lg="4" gap="lg">
                  <!-- vCPU Usage Skeleton -->
                  <OuiCard>
                    <OuiCardBody>
                      <OuiStack gap="md">
                        <OuiFlex justify="between" align="start">
                          <OuiStack gap="xs" class="flex-1">
                            <OuiSkeleton width="6rem" height="1rem" variant="text" />
                            <OuiSkeleton width="8rem" height="2rem" variant="text" />
                          </OuiStack>
                          <OuiSkeleton width="4rem" height="1.5rem" variant="rectangle" rounded />
                        </OuiFlex>
                        <OuiSkeleton width="100%" height="0.5rem" variant="rectangle" rounded />
                        <OuiSkeleton width="12rem" height="0.875rem" variant="text" />
                      </OuiStack>
                    </OuiCardBody>
                  </OuiCard>

                  <!-- Memory Usage Skeleton -->
                  <OuiCard>
                    <OuiCardBody>
                      <OuiStack gap="md">
                        <OuiFlex justify="between" align="start">
                          <OuiStack gap="xs" class="flex-1">
                            <OuiSkeleton width="5rem" height="1rem" variant="text" />
                            <OuiSkeleton width="9rem" height="2rem" variant="text" />
                          </OuiStack>
                          <OuiSkeleton width="4rem" height="1.5rem" variant="rectangle" rounded />
                        </OuiFlex>
                        <OuiSkeleton width="100%" height="0.5rem" variant="rectangle" rounded />
                        <OuiSkeleton width="12rem" height="0.875rem" variant="text" />
                      </OuiStack>
                    </OuiCardBody>
                  </OuiCard>

                  <!-- Bandwidth Usage Skeleton -->
                  <OuiCard>
                    <OuiCardBody>
                      <OuiStack gap="md">
                        <OuiFlex justify="between" align="start">
                          <OuiStack gap="xs" class="flex-1">
                            <OuiSkeleton width="6rem" height="1rem" variant="text" />
                            <OuiSkeleton width="7rem" height="2rem" variant="text" />
                          </OuiStack>
                          <OuiSkeleton width="4rem" height="1.5rem" variant="rectangle" rounded />
                        </OuiFlex>
                        <OuiSkeleton width="100%" height="0.5rem" variant="rectangle" rounded />
                        <OuiSkeleton width="12rem" height="0.875rem" variant="text" />
                      </OuiStack>
                    </OuiCardBody>
                  </OuiCard>

                  <!-- Storage Usage Skeleton -->
                  <OuiCard>
                    <OuiCardBody>
                      <OuiStack gap="md">
                        <OuiFlex justify="between" align="start">
                          <OuiStack gap="xs" class="flex-1">
                            <OuiSkeleton width="6rem" height="1rem" variant="text" />
                            <OuiSkeleton width="6rem" height="2rem" variant="text" />
                          </OuiStack>
                          <OuiSkeleton width="4rem" height="1.5rem" variant="rectangle" rounded />
                        </OuiFlex>
                        <OuiSkeleton width="100%" height="0.5rem" variant="rectangle" rounded />
                        <OuiSkeleton width="12rem" height="0.875rem" variant="text" />
                      </OuiStack>
                    </OuiCardBody>
                  </OuiCard>
                </OuiGrid>

                <!-- Credits Balance Skeleton -->
                <OuiCard variant="outline">
                  <OuiCardBody>
                    <OuiStack gap="lg">
                      <OuiFlex justify="between" align="start">
                        <OuiStack gap="xs" class="flex-1">
                          <OuiSkeleton width="8rem" height="1rem" variant="text" />
                          <OuiSkeleton width="10rem" height="3rem" variant="text" />
                          <OuiSkeleton width="20rem" height="0.875rem" variant="text" />
                        </OuiStack>
                        <OuiSkeleton width="8rem" height="2rem" variant="rectangle" rounded />
                      </OuiFlex>
                    </OuiStack>
                  </OuiCardBody>
                </OuiCard>

                <!-- Current Month Summary Skeleton -->
                <OuiCard variant="outline">
                  <OuiCardBody>
                    <OuiStack gap="lg">
                      <OuiSkeleton width="12rem" height="1.5rem" variant="text" />
                      <OuiFlex justify="between" align="center">
                        <OuiStack gap="xs" class="flex-1">
                          <OuiSkeleton width="10rem" height="1rem" variant="text" />
                          <OuiSkeleton width="8rem" height="3rem" variant="text" />
                          <OuiSkeleton width="18rem" height="0.875rem" variant="text" />
                        </OuiStack>
                        <OuiSkeleton width="6rem" height="2rem" variant="rectangle" rounded />
                      </OuiFlex>
                    </OuiStack>
                  </OuiCardBody>
                </OuiCard>
              </template>
              <template v-else>
                <OuiGrid cols="1" cols-md="2" cols-lg="4" gap="lg">
                  <!-- vCPU Usage -->
                  <OuiCard>
                    <OuiCardBody>
                      <OuiStack gap="md">
                        <OuiFlex justify="between" align="start">
                          <OuiStack gap="xs">
                            <OuiText size="sm" color="muted">vCPU Hours</OuiText>
                            <OuiText size="2xl" weight="bold">
                              {{ formatCoreSecondsToHours(usage.current.cpuCoreSeconds ?? 0) }}
                            </OuiText>
                          </OuiStack>
                          <OuiBadge 
                            :variant="getUsageBadgeVariant(
                              getUsagePercentage(
                                usage.current.cpuCoreSeconds ?? 0,
                                usage.quota?.cpuCoreSecondsMonthly || 0
                              )
                            )"
                          >
                            Active
                          </OuiBadge>
                        </OuiFlex>
                        <OuiProgress 
                          :value="getUsagePercentage(
                            usage.current.cpuCoreSeconds ?? 0,
                            usage.quota?.cpuCoreSecondsMonthly || 0
                          )" 
                          :max="100" 
                        />
                        <OuiText size="sm" color="muted">
                          <template v-if="Number(usage.quota?.cpuCoreSecondsMonthly || 0) === 0">
                            Unlimited allocation
                          </template>
                          <template v-else>
                            {{ getUsagePercentage(
                              usage.current.cpuCoreSeconds ?? 0,
                              usage.quota?.cpuCoreSecondsMonthly || 0
                            ) }}% of monthly allocation
                            ({{ formatCoreSecondsToHours(usage.quota?.cpuCoreSecondsMonthly || 0) }})
                          </template>
                        </OuiText>
                      </OuiStack>
                    </OuiCardBody>
                  </OuiCard>

                  <!-- Memory Usage -->
                  <OuiCard>
                    <OuiCardBody>
                      <OuiStack gap="md">
                        <OuiFlex justify="between" align="start">
                          <OuiStack gap="xs">
                            <OuiText size="sm" color="muted">Memory</OuiText>
                            <OuiText size="2xl" weight="bold">
                              {{ formatBytes(Number(usage.current.memoryByteSeconds ?? 0) / 3600) }}/hr avg
                            </OuiText>
                          </OuiStack>
                          <OuiBadge 
                            :variant="getUsageBadgeVariant(
                              getUsagePercentage(
                                usage.current.memoryByteSeconds ?? 0,
                                usage.quota?.memoryByteSecondsMonthly || 0
                              )
                            )"
                          >
                            <template v-if="getUsagePercentage(
                              usage.current.memoryByteSeconds ?? 0,
                              usage.quota?.memoryByteSecondsMonthly || 0
                            ) >= 90">High</template>
                            <template v-else-if="getUsagePercentage(
                              usage.current.memoryByteSeconds ?? 0,
                              usage.quota?.memoryByteSecondsMonthly || 0
                            ) >= 75">Warning</template>
                            <template v-else>Normal</template>
                          </OuiBadge>
                        </OuiFlex>
                        <OuiProgress 
                          :value="getUsagePercentage(
                            usage.current.memoryByteSeconds ?? 0,
                            usage.quota?.memoryByteSecondsMonthly || 0
                          )" 
                          :max="100" 
                        />
                        <OuiText size="sm" color="muted">
                          <template v-if="Number(usage.quota?.memoryByteSecondsMonthly || 0) === 0">
                            Unlimited allocation
                          </template>
                          <template v-else>
                            {{ getUsagePercentage(
                              usage.current.memoryByteSeconds ?? 0,
                              usage.quota?.memoryByteSecondsMonthly || 0
                            ) }}% of monthly allocation
                            ({{ formatMemoryByteSecondsToGB(usage.quota?.memoryByteSecondsMonthly || 0) }} GB)
                          </template>
                        </OuiText>
                      </OuiStack>
                    </OuiCardBody>
                  </OuiCard>

                  <!-- Bandwidth Usage -->
                  <OuiCard>
                    <OuiCardBody>
                      <OuiStack gap="md">
                        <OuiFlex justify="between" align="start">
                          <OuiStack gap="xs">
                            <OuiText size="sm" color="muted">Bandwidth</OuiText>
                            <OuiText size="2xl" weight="bold">
                              {{ formatBytes(Number(usage.current.bandwidthRxBytes ?? 0) + Number(usage.current.bandwidthTxBytes ?? 0)) }}
                            </OuiText>
                          </OuiStack>
                          <OuiBadge 
                            :variant="getUsageBadgeVariant(
                              getUsagePercentage(
                                Number(usage.current.bandwidthRxBytes ?? 0) + Number(usage.current.bandwidthTxBytes ?? 0),
                                usage.quota?.bandwidthBytesMonthly || 0
                              )
                            )"
                          >
                            <template v-if="getUsagePercentage(
                              Number(usage.current.bandwidthRxBytes ?? 0) + Number(usage.current.bandwidthTxBytes ?? 0),
                              usage.quota?.bandwidthBytesMonthly || 0
                            ) >= 90">High</template>
                            <template v-else-if="getUsagePercentage(
                              Number(usage.current.bandwidthRxBytes ?? 0) + Number(usage.current.bandwidthTxBytes ?? 0),
                              usage.quota?.bandwidthBytesMonthly || 0
                            ) >= 75">Warning</template>
                            <template v-else>Normal</template>
                          </OuiBadge>
                        </OuiFlex>
                        <OuiProgress 
                          :value="getUsagePercentage(
                            Number(usage.current.bandwidthRxBytes ?? 0) + Number(usage.current.bandwidthTxBytes ?? 0),
                            usage.quota?.bandwidthBytesMonthly || 0
                          )" 
                          :max="100" 
                        />
                        <OuiText size="sm" color="muted">
                          <template v-if="Number(usage.quota?.bandwidthBytesMonthly || 0) === 0">
                            Unlimited allocation
                          </template>
                          <template v-else>
                            {{ getUsagePercentage(
                              Number(usage.current.bandwidthRxBytes ?? 0) + Number(usage.current.bandwidthTxBytes ?? 0),
                              usage.quota?.bandwidthBytesMonthly || 0
                            ) }}% of monthly allocation
                            ({{ formatBytes(usage.quota?.bandwidthBytesMonthly || 0) }})
                          </template>
                        </OuiText>
                      </OuiStack>
                    </OuiCardBody>
                  </OuiCard>

                  <!-- Storage Usage -->
                  <OuiCard>
                    <OuiCardBody>
                      <OuiStack gap="md">
                        <OuiFlex justify="between" align="start">
                          <OuiStack gap="xs">
                            <OuiText size="sm" color="muted">Storage (GB)</OuiText>
                            <OuiText size="2xl" weight="bold">
                              {{ formatBytesToGB(usage.current.storageBytes) }}
                            </OuiText>
                          </OuiStack>
                          <OuiBadge 
                            :variant="getUsageBadgeVariant(
                              getUsagePercentage(
                                usage.current.storageBytes,
                                usage.quota?.storageBytes || 0
                              )
                            )"
                          >
                            <template v-if="getUsagePercentage(
                              usage.current.storageBytes,
                              usage.quota?.storageBytes || 0
                            ) >= 90">High</template>
                            <template v-else-if="getUsagePercentage(
                              usage.current.storageBytes,
                              usage.quota?.storageBytes || 0
                            ) >= 75">Warning</template>
                            <template v-else>Normal</template>
                          </OuiBadge>
                        </OuiFlex>
                        <OuiProgress 
                          :value="getUsagePercentage(
                            usage.current.storageBytes,
                            usage.quota?.storageBytes || 0
                          )" 
                          :max="100" 
                        />
                        <OuiText size="sm" color="muted">
                          <template v-if="Number(usage.quota?.storageBytes || 0) === 0">
                            Unlimited allocation
                          </template>
                          <template v-else>
                            {{ getUsagePercentage(
                              usage.current.storageBytes,
                              usage.quota?.storageBytes || 0
                            ) }}% of monthly allocation
                            ({{ formatBytesToGB(usage.quota?.storageBytes || 0) }} GB)
                          </template>
                        </OuiText>
                      </OuiStack>
                    </OuiCardBody>
                  </OuiCard>
                </OuiGrid>

                <!-- Credits Balance -->
                <OuiCard variant="outline">
                  <OuiCardBody>
                    <OuiStack gap="lg">
                      <OuiFlex justify="between" align="start">
                        <OuiStack gap="xs">
                          <OuiText size="sm" color="muted">Credits Balance</OuiText>
                          <OuiText size="3xl" weight="bold">
                            {{ formatCurrency(creditsBalance) }}
                          </OuiText>
                          <OuiText size="sm" color="muted">
                            Available credits for your organization
                          </OuiText>
                        </OuiStack>
                        <OuiButton 
                          variant="solid" 
                          size="sm" 
                          @click="addCreditsDialogOpen = true"
                          v-if="currentUserIsOwner"
                        >
                          <PlusIcon class="h-4 w-4 mr-2" />
                          Add Credits
                        </OuiButton>
                      </OuiFlex>
                    </OuiStack>
                  </OuiCardBody>
                </OuiCard>

                <!-- Current Month Summary -->
                <OuiCard variant="outline">
                  <OuiCardBody>
                    <OuiStack gap="lg">
                      <OuiText size="xl" weight="semibold">Current Month Estimate</OuiText>
                      <OuiFlex justify="between" align="center">
                        <OuiStack gap="xs">
                          <OuiText size="sm" color="muted">{{ currentMonth }}</OuiText>
                          <OuiText size="3xl" weight="bold">
                            {{ usage.estimatedMonthly?.estimatedCostCents 
                              ? formatCurrency(usage.estimatedMonthly.estimatedCostCents)
                              : '$0.00' }}
                          </OuiText>
                          <OuiText size="sm" color="muted">
                            Based on current usage patterns
                          </OuiText>
                        </OuiStack>
                        <OuiButton variant="outline" size="sm" @click="refreshUsage">
                          Refresh
                        </OuiButton>
                      </OuiFlex>
                    </OuiStack>
                  </OuiCardBody>
                </OuiCard>
              </template>
              </OuiStack>

            <!-- Payment Methods -->
            <OuiStack gap="lg">
              <OuiFlex justify="between" align="center">
                <OuiText size="2xl" weight="bold">Payment Methods</OuiText>
                <OuiFlex gap="sm">
                  <OuiButton 
                    variant="outline" 
                    size="sm"
                    @click="addPaymentMethodDialogOpen = true"
                    :disabled="!billingAccount?.stripeCustomerId && !currentUserIsOwner"
                    v-if="currentUserIsOwner"
                  >
                    <PlusIcon class="h-4 w-4 mr-2" />
                    Add Payment Method
                  </OuiButton>
                  <OuiButton 
                    variant="solid" 
                    size="sm"
                    @click="openCustomerPortal"
                    :disabled="!billingAccount?.stripeCustomerId"
                  >
                    <CreditCardIcon class="h-4 w-4 mr-2" />
                    Manage Billing
                  </OuiButton>
                </OuiFlex>
              </OuiFlex>

              <OuiCard>
                <OuiCardBody>
                  <OuiStack gap="md">
                    <template v-if="!billingAccount?.stripeCustomerId">
                      <OuiStack gap="sm" align="center">
                        <CreditCardIcon class="h-12 w-12 text-muted" />
                        <OuiText size="lg" weight="semibold">No Payment Methods</OuiText>
                        <OuiText size="sm" color="muted">
                          Add credits to create a Stripe customer account and add payment methods.
                        </OuiText>
                      </OuiStack>
                    </template>
                    <template v-else-if="paymentMethods.length === 0">
                      <OuiStack gap="sm" align="center">
                        <CreditCardIcon class="h-12 w-12 text-muted" />
                        <OuiText size="lg" weight="semibold">No Payment Methods</OuiText>
                        <OuiText size="sm" color="muted">
                          Add a payment method to make purchases easier.
                        </OuiText>
                      </OuiStack>
                    </template>
                    <template v-else>
                      <OuiStack gap="sm">
                        <OuiBox
                          v-for="pm in paymentMethods"
                          :key="pm.id"
                          p="md"
                          border="1"
                          borderColor="muted"
                          rounded="md"
                          :class="{ 'ring-2 ring-primary': pm.isDefault }"
                        >
                          <OuiFlex justify="between" align="center">
                            <OuiFlex gap="md" align="center">
                              <CreditCardIcon class="h-6 w-6 text-muted" />
                              <OuiStack gap="xs">
                                <OuiFlex gap="sm" align="center">
                                  <OuiText size="sm" weight="medium">
                                    {{ pm.card ? `${formatCardBrand(pm.card.brand)} •••• ${pm.card.last4}` : pm.type }}
                                  </OuiText>
                                  <OuiBadge v-if="pm.isDefault" variant="success" size="xs">
                                    Default
                                  </OuiBadge>
                                </OuiFlex>
                                <OuiText size="xs" color="muted" v-if="pm.card">
                                  Expires {{ pm.card.expMonth }}/{{ pm.card.expYear }}
                                </OuiText>
                              </OuiStack>
                            </OuiFlex>
                            <OuiFlex gap="sm" v-if="currentUserIsOwner">
                              <OuiButton
                                variant="ghost"
                                size="sm"
                                @click="setDefaultPaymentMethod(pm.id)"
                                :disabled="pm.isDefault"
                              >
                                Set as Default
                              </OuiButton>
                              <OuiButton
                                variant="ghost"
                                size="sm"
                                @click="removePaymentMethod(pm.id)"
                                :disabled="pm.isDefault && paymentMethods.length > 1"
                              >
                                Remove
                              </OuiButton>
                            </OuiFlex>
                          </OuiFlex>
                        </OuiBox>
                      </OuiStack>
                    </template>
                  </OuiStack>
                </OuiCardBody>
              </OuiCard>
            </OuiStack>

            <!-- Invoices -->
            <OuiStack gap="lg">
              <OuiFlex justify="between" align="center">
                <OuiText size="2xl" weight="bold">Invoices</OuiText>
                <OuiButton 
                  variant="outline" 
                  size="sm"
                  @click="refreshInvoices"
                >
                  Refresh
                </OuiButton>
              </OuiFlex>

              <OuiCard>
                <OuiCardBody class="p-0">
                  <OuiStack>
                    <!-- Table Header -->
                    <OuiBox p="md" borderBottom="1" borderColor="muted">
                      <OuiGrid cols="5" gap="md">
                        <OuiText size="sm" weight="medium" color="muted">Invoice</OuiText>
                        <OuiText size="sm" weight="medium" color="muted">Date</OuiText>
                        <OuiText size="sm" weight="medium" color="muted">Amount</OuiText>
                        <OuiText size="sm" weight="medium" color="muted">Status</OuiText>
                        <OuiText size="sm" weight="medium" color="muted">Actions</OuiText>
                      </OuiGrid>
                    </OuiBox>

                    <!-- Table Rows -->
                    <OuiStack>
                      <template v-if="!billingAccount?.stripeCustomerId">
                        <OuiBox p="md">
                          <OuiStack gap="sm" align="center">
                            <DocumentTextIcon class="h-8 w-8 text-muted" />
                            <OuiText size="sm" color="muted" align="center">
                              No invoices available. Add credits to create a Stripe customer account.
                            </OuiText>
                          </OuiStack>
                        </OuiBox>
                      </template>
                      <template v-else-if="invoices.length === 0">
                        <OuiBox p="md">
                          <OuiStack gap="sm" align="center">
                            <DocumentTextIcon class="h-8 w-8 text-muted" />
                            <OuiText size="sm" color="muted" align="center">
                              No invoices found. Invoices will appear here after purchases.
                            </OuiText>
                          </OuiStack>
                        </OuiBox>
                      </template>
                      <OuiBox
                        v-for="invoice in invoices"
                        :key="invoice.id"
                        p="md"
                        borderBottom="1"
                        borderColor="muted"
                        :class="{ 'last:border-b-0': invoice === invoices[invoices.length - 1] }"
                      >
                        <OuiGrid cols="5" gap="md" align="center">
                          <OuiStack gap="xs">
                            <OuiText size="sm" weight="medium">{{ invoice.number || `#${invoice.id.substring(0, 8).toUpperCase()}` }}</OuiText>
                            <OuiText size="xs" color="muted" v-if="invoice.description">
                              {{ invoice.description }}
                            </OuiText>
                          </OuiStack>
                          <OuiText size="sm" color="muted">
                            {{ formatInvoiceDate(invoice.date) }}
                          </OuiText>
                          <OuiText size="sm" weight="medium">
                            {{ formatCurrency(invoice.amountDue) }}
                            <span v-if="invoice.currency && invoice.currency.toLowerCase() !== 'usd'" class="text-xs text-muted">
                              {{ invoice.currency.toUpperCase() }}
                            </span>
                          </OuiText>
                          <OuiBadge :variant="getInvoiceStatusVariant(invoice.status)">
                            {{ invoice.status.charAt(0).toUpperCase() + invoice.status.slice(1) }}
                          </OuiBadge>
                          <OuiFlex gap="sm">
                            <OuiButton
                              v-if="invoice.hostedInvoiceUrl"
                              variant="ghost"
                              size="sm"
                              @click="openInvoiceUrl(invoice.hostedInvoiceUrl!)"
                            >
                              <DocumentTextIcon class="h-4 w-4 mr-1" />
                              View
                            </OuiButton>
                            <OuiButton
                              v-if="invoice.invoicePdf"
                              variant="ghost"
                              size="sm"
                              @click="openInvoiceUrl(invoice.invoicePdf!)"
                            >
                              <ArrowDownTrayIcon class="h-4 w-4 mr-1" />
                              PDF
                            </OuiButton>
                            <OuiText v-if="!invoice.hostedInvoiceUrl && !invoice.invoicePdf" size="xs" color="muted">
                              No actions
                            </OuiText>
                          </OuiFlex>
                        </OuiGrid>
                      </OuiBox>
                      <OuiBox v-if="hasMoreInvoices" p="md" borderTop="1" borderColor="muted">
                        <OuiText size="sm" color="muted" align="center">
                          More invoices available. Visit the Stripe Customer Portal to view all invoices.
                        </OuiText>
                        <OuiFlex justify="center" class="mt-2">
                          <OuiButton variant="outline" size="sm" @click="openCustomerPortal">
                            Open Customer Portal
                          </OuiButton>
                        </OuiFlex>
                      </OuiBox>
                    </OuiStack>
                  </OuiStack>
                </OuiCardBody>
              </OuiCard>
            </OuiStack>

            <!-- Transaction History -->
            <OuiStack gap="lg">
              <OuiFlex justify="between" align="center">
                <OuiText size="2xl" weight="bold">Transaction History</OuiText>
              </OuiFlex>

              <OuiCard>
                <OuiCardBody class="p-0">
                  <OuiStack>
                    <!-- Table Header -->
                    <OuiBox p="md" borderBottom="1" borderColor="muted">
                      <OuiGrid cols="5" gap="md">
                        <OuiText size="sm" weight="medium" color="muted">Transaction</OuiText>
                        <OuiText size="sm" weight="medium" color="muted">Date</OuiText>
                        <OuiText size="sm" weight="medium" color="muted">Amount</OuiText>
                        <OuiText size="sm" weight="medium" color="muted">Type</OuiText>
                        <OuiText size="sm" weight="medium" color="muted">Balance</OuiText>
                      </OuiGrid>
                    </OuiBox>

                    <!-- Table Rows -->
                    <OuiStack>
                      <template v-if="billingHistory.length === 0">
                        <OuiBox p="md">
                          <OuiText size="sm" color="muted" align="center">
                            No payment transactions yet. Add credits to see transaction history.
                          </OuiText>
                        </OuiBox>
                      </template>
                      <OuiBox
                        v-for="transaction in billingHistory"
                        :key="transaction.id"
                        p="md"
                        borderBottom="1"
                        borderColor="muted"
                      >
                        <OuiGrid cols="5" gap="md" align="center">
                          <OuiText size="sm" weight="medium">{{ transaction.number }}</OuiText>
                          <OuiText size="sm" color="muted">{{ transaction.date }}</OuiText>
                          <OuiText size="sm" weight="medium">{{ transaction.amount }}</OuiText>
                          <OuiBadge variant="success">{{ transaction.status }}</OuiBadge>
                          <OuiText size="sm" color="muted">
                            {{ formatCurrency(transaction.transaction.balanceAfter) }}
                          </OuiText>
                        </OuiGrid>
                      </OuiBox>
                    </OuiStack>
                  </OuiStack>
                </OuiCardBody>
              </OuiCard>
            </OuiStack>
          </template>
        </OuiStack>
      </OuiCardBody>
    </OuiCard>

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
