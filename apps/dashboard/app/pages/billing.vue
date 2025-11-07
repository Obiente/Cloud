<script setup lang="ts">
  definePageMeta({ layout: "default", middleware: "auth" });
  
  // Redirect if billing is disabled
  const appConfig = useConfig();
  await appConfig.fetchConfig();
  if (appConfig.billingEnabled.value !== true) {
    throw createError({
      statusCode: 404,
      statusMessage: "Billing is disabled",
    });
  }
  
  import {
    OrganizationService,
    BillingService,
    type Organization,
  } from "@obiente/proto";
  import { useConnectClient } from "~/lib/connect-client";
  import { useOrganizationLabels } from "~/composables/useOrganizationLabels";
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
  const config = useRuntimeConfig(); // For stripePublishableKey

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
    const res = await orgClient.listOrganizations({ onlyMine: true });
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
      .filter((t) => {
        // Include all payment transactions (from Stripe or otherwise)
        // This includes both credit purchases and other payments
        return t.type === "payment";
      })
      .map((t) => {
        let dateStr = "";
        if (t.createdAt) {
          if (typeof t.createdAt === 'object' && t.createdAt !== null && 'seconds' in t.createdAt) {
            dateStr = new Date(Number((t.createdAt as any).seconds) * 1000).toLocaleDateString();
          } else if (typeof t.createdAt === 'object' && t.createdAt !== null && 'toMillis' in t.createdAt) {
            const timestamp = t.createdAt as any;
            if (typeof timestamp.toMillis === 'function') {
              dateStr = new Date(timestamp.toMillis()).toLocaleDateString();
            }
          }
        }
        const amountCents = typeof t.amountCents === 'bigint' ? Number(t.amountCents) : Number(t.amountCents || 0);
        const balanceAfter = typeof t.balanceAfter === 'bigint' ? Number(t.balanceAfter) : Number(t.balanceAfter || 0);
        return {
          id: t.id,
          number: `#${t.id.substring(0, 8).toUpperCase()}`,
          date: dateStr,
          amount: formatCurrency(Math.abs(amountCents)), // Show absolute value for display
          status: amountCents > 0 ? "Paid" : "Refunded",
          transaction: t,
          balanceAfter,
        };
      });
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
    if (!billingAccount.value?.stripeCustomerId) {
      const { toast } = useToast();
      toast.error("No billing account found. Please add credits first.");
      return;
    }
    try {
      const response = await billingClient.createPortalSession({
        organizationId: selectedOrg.value,
      });
      if (response.portalUrl) {
        window.location.href = response.portalUrl;
      } else {
        throw new Error("No portal URL received");
      }
    } catch (err: any) {
      let errorMessage = err.message || "Failed to open customer portal";
      
      // Provide helpful message for portal configuration errors
      if (errorMessage.includes("not configured") || errorMessage.includes("configuration")) {
        errorMessage = "Stripe Customer Portal is not configured. Please contact support or configure it in your Stripe Dashboard.";
      }
      
      error.value = errorMessage;
      const { toast } = useToast();
      toast.error(errorMessage);
    }
  }
  
  // Get current organization object to access credits
  const currentOrganization = computed((): (Organization & { planInfo?: Organization['planInfo'] }) | null => {
    if (!selectedOrg.value) return null;
    return organizations.value.find((o) => o.id === selectedOrg.value) as (Organization & { planInfo?: Organization['planInfo'] }) | null || null;
  });
  
  const creditsBalance = computed(() => {
    const credits = currentOrganization.value?.credits;
    if (credits === undefined || credits === null) return 0;
    return typeof credits === 'bigint' ? Number(credits) : credits;
  });
  
  const addCreditsDialogOpen = ref(false);
  const addCreditsAmount = ref("");
  const addCreditsLoading = ref(false);
  const removePaymentMethodDialogOpen = ref(false);
  const paymentMethodToRemove = ref<string | null>(null);

  const addPaymentMethodDialogOpen = ref(false);
  
  // Billing information management
  const editBillingInfoDialogOpen = ref(false);
  const billingInfoLoading = ref(false);
  const billingInfoForm = ref({
    billingEmail: "",
    companyName: "",
    taxId: "",
    address: {
      line1: "",
      line2: "",
      city: "",
      state: "",
      postalCode: "",
      country: "",
    },
  });

  // Subscription management
  const subscriptions = ref<any[]>([]);
  const subscriptionsLoading = ref(false);
  const cancelSubscriptionDialogOpen = ref(false);
  const subscriptionToCancel = ref<string | null>(null);
  const updateSubscriptionPaymentDialogOpen = ref(false);
  const subscriptionToUpdate = ref<any | null>(null);
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
      // Check if we have a default payment method
      const defaultPaymentMethod = paymentMethods.value.find(pm => pm.isDefault);
      
      if (defaultPaymentMethod && billingAccount.value?.stripeCustomerId) {
        // Use PaymentIntent with existing payment method
        try {
          const response = await billingClient.createPaymentIntent({
            organizationId: selectedOrg.value,
            amountCents: BigInt(Math.round(amount * 100)),
            paymentMethodId: defaultPaymentMethod.id,
          });
          
          if (response.clientSecret && stripe.value) {
            // PaymentIntent is already confirmed on the backend, but we need to check status
            // If 3D Secure is required, we'll need to handle it
            const { error: retrieveError, paymentIntent } = await stripe.value.retrievePaymentIntent(response.clientSecret);
            
            if (retrieveError) {
              throw new Error(retrieveError.message);
            }
            
            // Check if payment requires action (3D Secure)
            if (paymentIntent.status === 'requires_action' || paymentIntent.status === 'requires_payment_method') {
              // Confirm with 3D Secure if needed
              const { error: confirmError } = await stripe.value.confirmCardPayment(
                response.clientSecret,
                {
                  payment_method: defaultPaymentMethod.id,
                }
              );
              
              if (confirmError) {
                throw new Error(confirmError.message);
              }
            } else if (paymentIntent.status !== 'succeeded') {
              throw new Error(`Payment status: ${paymentIntent.status}`);
            }
            
            // Payment succeeded - refresh data
            const { toast } = useToast();
            toast.success("Payment successful! Credits are being added...");
            
            // Refresh data
            await syncOrganizations();
            await refreshCreditLog();
            await refreshBillingAccount();
            await refreshPaymentMethods();
            await refreshInvoices();
            
            addCreditsDialogOpen.value = false;
            addCreditsAmount.value = "";
            addCreditsLoading.value = false;
          } else {
            throw new Error("No client secret received");
          }
        } catch (paymentIntentError: any) {
          // If PaymentIntent fails, fall back to CheckoutSession
          console.warn("PaymentIntent failed, falling back to CheckoutSession:", paymentIntentError);
          error.value = paymentIntentError.message || "Failed to process payment with saved card. Redirecting to checkout...";
          
          // Fall back to checkout session
          const response = await billingClient.createCheckoutSession({
            organizationId: selectedOrg.value,
            amountCents: BigInt(Math.round(amount * 100)),
          });
          
          if (response.checkoutUrl) {
            window.location.href = response.checkoutUrl;
          } else {
            throw new Error("No checkout URL received");
          }
        }
      } else {
        // No default payment method, use CheckoutSession
        const response = await billingClient.createCheckoutSession({
          organizationId: selectedOrg.value,
          amountCents: BigInt(Math.round(amount * 100)),
        });
        
        if (response.checkoutUrl) {
          window.location.href = response.checkoutUrl;
        } else {
          throw new Error("No checkout URL received");
        }
      }
    } catch (err: any) {
      error.value = err.message || "Failed to process payment";
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

  function openRemovePaymentMethodDialog(paymentMethodId: string) {
    paymentMethodToRemove.value = paymentMethodId;
    removePaymentMethodDialogOpen.value = true;
  }

  async function removePaymentMethod() {
    if (!selectedOrg.value || !paymentMethodToRemove.value) return;
    try {
      await billingClient.detachPaymentMethod({
        organizationId: selectedOrg.value,
        paymentMethodId: paymentMethodToRemove.value,
      });
      await refreshPaymentMethods();
      await refreshBillingAccount();
      useToast().toast.success("Payment method removed");
      removePaymentMethodDialogOpen.value = false;
      paymentMethodToRemove.value = null;
    } catch (err: any) {
      error.value = err.message || "Failed to remove payment method";
      useToast().toast.error(error.value);
    }
  }

  // Load billing information into form
  function loadBillingInfo() {
    if (!billingAccount.value) return;
    billingInfoForm.value.billingEmail = billingAccount.value.billingEmail || "";
    billingInfoForm.value.companyName = billingAccount.value.companyName || "";
    billingInfoForm.value.taxId = billingAccount.value.taxId || "";
    if (billingAccount.value.address) {
      billingInfoForm.value.address = {
        line1: billingAccount.value.address.line1 || "",
        line2: billingAccount.value.address.line2 || "",
        city: billingAccount.value.address.city || "",
        state: billingAccount.value.address.state || "",
        postalCode: billingAccount.value.address.postalCode || "",
        country: billingAccount.value.address.country || "",
      };
    }
  }

  // Update billing information
  async function updateBillingInfo() {
    if (!selectedOrg.value) return;
    billingInfoLoading.value = true;
    error.value = "";
    try {
      await billingClient.updateBillingAccount({
        organizationId: selectedOrg.value,
        billingEmail: billingInfoForm.value.billingEmail || undefined,
        companyName: billingInfoForm.value.companyName || undefined,
        taxId: billingInfoForm.value.taxId || undefined,
        address: {
          line1: billingInfoForm.value.address.line1,
          line2: billingInfoForm.value.address.line2 || undefined,
          city: billingInfoForm.value.address.city,
          state: billingInfoForm.value.address.state || undefined,
          postalCode: billingInfoForm.value.address.postalCode,
          country: billingInfoForm.value.address.country,
        },
      });
      await refreshBillingAccount();
      editBillingInfoDialogOpen.value = false;
      useToast().toast.success("Billing information updated");
    } catch (err: any) {
      error.value = err.message || "Failed to update billing information";
      useToast().toast.error(error.value);
    } finally {
      billingInfoLoading.value = false;
    }
  }

  // Load subscriptions
  async function loadSubscriptions() {
    if (!selectedOrg.value) return;
    subscriptionsLoading.value = true;
    try {
      const res = await (billingClient as any).listSubscriptions({
        organizationId: selectedOrg.value,
      });
      subscriptions.value = res.subscriptions || [];
    } catch (err: any) {
      console.error("Failed to load subscriptions:", err);
      subscriptions.value = [];
    } finally {
      subscriptionsLoading.value = false;
    }
  }

  // Cancel subscription
  function openCancelSubscriptionDialog(subscriptionId: string) {
    subscriptionToCancel.value = subscriptionId;
    cancelSubscriptionDialogOpen.value = true;
  }

  async function cancelSubscription() {
    if (!selectedOrg.value || !subscriptionToCancel.value) return;
    subscriptionsLoading.value = true;
    try {
      const res = await (billingClient as any).cancelSubscription({
        organizationId: selectedOrg.value,
        subscriptionId: subscriptionToCancel.value,
      });
      await loadSubscriptions();
      cancelSubscriptionDialogOpen.value = false;
      subscriptionToCancel.value = null;
      useToast().toast.success(res.message || "Subscription will be canceled at the end of the billing period");
    } catch (err: any) {
      error.value = err.message || "Failed to cancel subscription";
      useToast().toast.error(error.value);
    } finally {
      subscriptionsLoading.value = false;
    }
  }

  // Update subscription payment method
  function openUpdateSubscriptionPaymentDialog(subscription: any) {
    subscriptionToUpdate.value = subscription;
    updateSubscriptionPaymentDialogOpen.value = true;
  }

  async function updateSubscriptionPaymentMethod(paymentMethodId: string) {
    if (!selectedOrg.value || !subscriptionToUpdate.value) return;
    subscriptionsLoading.value = true;
    try {
      await (billingClient as any).updateSubscriptionPaymentMethod({
        organizationId: selectedOrg.value,
        subscriptionId: subscriptionToUpdate.value.id,
        paymentMethodId,
      });
      await loadSubscriptions();
      await refreshPaymentMethods();
      updateSubscriptionPaymentDialogOpen.value = false;
      subscriptionToUpdate.value = null;
      useToast().toast.success("Subscription payment method updated");
    } catch (err: any) {
      error.value = err.message || "Failed to update subscription payment method";
      useToast().toast.error(error.value);
    } finally {
      subscriptionsLoading.value = false;
    }
  }

  // Watch for billing account changes to load form
  watch(billingAccount, () => {
    if (editBillingInfoDialogOpen.value) {
      loadBillingInfo();
    }
  });

  // Load subscriptions when org changes
  watch(selectedOrg, () => {
    if (selectedOrg.value) {
      loadSubscriptions();
    }
  });

  // Load subscriptions on mount
  if (selectedOrg.value) {
    loadSubscriptions();
    refreshCurrentOrganization();
  }

  // Watch for org changes to refresh plan info
  watch(selectedOrg, () => {
    if (selectedOrg.value) {
      refreshCurrentOrganization();
      loadSubscriptions();
    }
  });

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
    let seconds = 0;
    if (typeof date === 'object' && date !== null) {
      if ('seconds' in date) {
        seconds = Number(date.seconds);
      } else if ('toMillis' in date && typeof date.toMillis === 'function') {
        return new Date(date.toMillis()).toLocaleDateString();
      }
    } else if (typeof date === 'number') {
      seconds = date;
    }
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
    // memoryByteSecondsMonthly is the total byte-seconds quota for the full month
    // To convert to GB, we need to divide by the total seconds in the month
    const now = new Date();
    const monthStart = new Date(now.getFullYear(), now.getMonth(), 1);
    const monthEnd = new Date(now.getFullYear(), now.getMonth() + 1, 0, 23, 59, 59, 999);
    const secondsInFullMonth = Math.max(1, Math.floor((monthEnd.getTime() - monthStart.getTime()) / 1000));
    const avgBytes = bs / secondsInFullMonth;
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
                :items="organizationSelectItems"
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
            <!-- Plan Information -->
            <OuiCard v-if="currentOrganization?.planInfo" variant="outline">
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
                  <OuiGrid cols="1" cols-md="2" cols-lg="5" gap="md">
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
                                @click="openRemovePaymentMethodDialog(pm.id)"
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

            <!-- Billing Information -->
            <OuiStack gap="lg" v-if="currentUserIsOwner">
              <OuiFlex justify="between" align="center">
                <OuiText size="2xl" weight="bold">Billing Information</OuiText>
                <OuiButton 
                  variant="outline" 
                  size="sm"
                  @click="loadBillingInfo(); editBillingInfoDialogOpen = true"
                  :disabled="!billingAccount?.stripeCustomerId"
                >
                  Edit Billing Info
                </OuiButton>
              </OuiFlex>

              <OuiCard>
                <OuiCardBody>
                  <OuiStack gap="md">
                    <template v-if="!billingAccount?.stripeCustomerId">
                      <OuiStack gap="sm" align="center">
                        <OuiText size="sm" color="muted">
                          Add credits to create a Stripe customer account and set billing information.
                        </OuiText>
                      </OuiStack>
                    </template>
                    <template v-else>
                      <OuiStack gap="sm">
                        <OuiFlex justify="between">
                          <OuiText size="sm" color="muted">Billing Email</OuiText>
                          <OuiText size="sm" weight="medium">{{ billingAccount.billingEmail || "Not set" }}</OuiText>
                        </OuiFlex>
                        <OuiFlex justify="between">
                          <OuiText size="sm" color="muted">Company Name</OuiText>
                          <OuiText size="sm" weight="medium">{{ billingAccount.companyName || "Not set" }}</OuiText>
                        </OuiFlex>
                        <OuiFlex justify="between">
                          <OuiText size="sm" color="muted">Tax ID</OuiText>
                          <OuiText size="sm" weight="medium">{{ billingAccount.taxId || "Not set" }}</OuiText>
                        </OuiFlex>
                        <OuiFlex justify="between" v-if="billingAccount.address">
                          <OuiText size="sm" color="muted">Billing Address</OuiText>
                          <OuiText size="sm" weight="medium">
                            {{ billingAccount.address.line1 }}{{ billingAccount.address.line2 ? `, ${billingAccount.address.line2}` : "" }}<br/>
                            {{ billingAccount.address.city }}{{ billingAccount.address.state ? `, ${billingAccount.address.state}` : "" }} {{ billingAccount.address.postalCode }}<br/>
                            {{ billingAccount.address.country }}
                          </OuiText>
                        </OuiFlex>
                        <OuiFlex justify="between" v-else>
                          <OuiText size="sm" color="muted">Billing Address</OuiText>
                          <OuiText size="sm" weight="medium">Not set</OuiText>
                        </OuiFlex>
                      </OuiStack>
                    </template>
                  </OuiStack>
                </OuiCardBody>
              </OuiCard>
            </OuiStack>

            <!-- Subscriptions -->
            <OuiStack gap="lg" v-if="currentUserIsOwner">
              <OuiFlex justify="between" align="center">
                <OuiText size="2xl" weight="bold">Subscriptions</OuiText>
                <OuiButton 
                  variant="outline" 
                  size="sm"
                  @click="loadSubscriptions"
                  :disabled="subscriptionsLoading || !billingAccount?.stripeCustomerId"
                >
                  Refresh
                </OuiButton>
              </OuiFlex>

              <OuiCard>
                <OuiCardBody>
                  <OuiStack gap="md">
                    <template v-if="!billingAccount?.stripeCustomerId">
                      <OuiStack gap="sm" align="center">
                        <OuiText size="sm" color="muted">
                          Add credits to create a Stripe customer account and view subscriptions.
                        </OuiText>
                      </OuiStack>
                    </template>
                    <template v-else-if="subscriptionsLoading">
                      <OuiStack gap="sm" align="center">
                        <OuiText size="sm" color="muted">Loading subscriptions...</OuiText>
                      </OuiStack>
                    </template>
                    <template v-else-if="subscriptions.length === 0">
                      <OuiStack gap="sm" align="center">
                        <OuiText size="sm" color="muted">No active subscriptions found.</OuiText>
                      </OuiStack>
                    </template>
                    <template v-else>
                      <OuiStack gap="sm">
                        <OuiBox
                          v-for="sub in subscriptions"
                          :key="sub.id"
                          p="md"
                          border="1"
                          borderColor="muted"
                          rounded="md"
                        >
                          <OuiStack gap="sm">
                            <OuiFlex justify="between" align="start">
                              <OuiStack gap="xs">
                                <OuiText size="sm" weight="semibold">{{ sub.description || "Subscription" }}</OuiText>
                                <OuiText size="xs" color="muted">ID: {{ sub.id }}</OuiText>
                              </OuiStack>
                              <OuiBadge :variant="sub.status === 'active' ? 'success' : sub.status === 'canceled' ? 'danger' : 'warning'">
                                {{ sub.status }}
                              </OuiBadge>
                            </OuiFlex>
                            <OuiFlex justify="between">
                              <OuiText size="sm" color="muted">Amount</OuiText>
                              <OuiText size="sm" weight="medium">
                                {{ formatCurrency(sub.amount || 0) }} {{ sub.currency?.toUpperCase() || 'USD' }}
                                <span v-if="sub.interval">/ {{ sub.interval }}</span>
                              </OuiText>
                            </OuiFlex>
                            <OuiFlex justify="between" v-if="sub.currentPeriodStart">
                              <OuiText size="sm" color="muted">Current Period</OuiText>
                              <OuiText size="sm">
                                {{ new Date(sub.currentPeriodStart.seconds * 1000).toLocaleDateString() }} - 
                                {{ sub.currentPeriodEnd ? new Date(sub.currentPeriodEnd.seconds * 1000).toLocaleDateString() : 'N/A' }}
                              </OuiText>
                            </OuiFlex>
                            <OuiFlex justify="between" v-if="sub.cancelAtPeriodEnd">
                              <OuiText size="sm" color="muted">Status</OuiText>
                              <OuiText size="sm" color="warning">Will cancel at period end</OuiText>
                            </OuiFlex>
                            <OuiFlex gap="sm" v-if="sub.status === 'active'">
                              <OuiButton
                                variant="ghost"
                                size="sm"
                                @click="openUpdateSubscriptionPaymentDialog(sub)"
                                :disabled="paymentMethods.length === 0"
                              >
                                Update Payment Method
                              </OuiButton>
                              <OuiButton
                                variant="ghost"
                                size="sm"
                                color="danger"
                                @click="openCancelSubscriptionDialog(sub.id)"
                                :disabled="sub.cancelAtPeriodEnd"
                              >
                                Cancel Subscription
                              </OuiButton>
                            </OuiFlex>
                          </OuiStack>
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
                            {{ formatCurrency(invoice.amountDue ?? 0) }}
                            <span v-if="invoice.currency && invoice.currency.toLowerCase() !== 'usd'" class="text-xs text-muted">
                              {{ invoice.currency.toUpperCase() }}
                            </span>
                          </OuiText>
                          <OuiBadge v-if="invoice.status && invoice.status.trim()" :variant="getInvoiceStatusVariant(invoice.status)">
                            {{ invoice.status.charAt(0).toUpperCase() + invoice.status.slice(1) }}
                          </OuiBadge>
                          <OuiText v-else size="sm" color="muted">—</OuiText>
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
                            {{ formatCurrency(transaction.balanceAfter ?? 0) }}
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

    <!-- Remove Payment Method Dialog -->
    <OuiDialog v-model:open="removePaymentMethodDialogOpen" title="Remove Payment Method">
      <OuiStack gap="lg">
        <OuiStack gap="xs">
          <OuiText size="sm" color="muted">
            Are you sure you want to remove this payment method? This action cannot be undone.
          </OuiText>
          <OuiText v-if="error" size="sm" color="danger">{{ error }}</OuiText>
        </OuiStack>
        
        <OuiFlex justify="end" gap="sm">
          <OuiButton variant="ghost" @click="removePaymentMethodDialogOpen = false">
            Cancel
          </OuiButton>
          <OuiButton 
            variant="solid" 
            color="danger"
            @click="removePaymentMethod"
          >
            Remove Payment Method
          </OuiButton>
        </OuiFlex>
      </OuiStack>
    </OuiDialog>

    <!-- Add Credits Dialog -->
    <OuiDialog v-model:open="addCreditsDialogOpen" title="Purchase Credits">
      <OuiStack gap="lg">
        <OuiStack gap="xs">
          <OuiText size="sm" color="muted">
            <template v-if="paymentMethods.find(pm => pm.isDefault)">
              Purchase credits using your default payment method. If 3D Secure authentication is required, you'll be prompted to complete it.
            </template>
            <template v-else>
              Purchase credits for your organization. You will be redirected to Stripe Checkout to complete the payment.
            </template>
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

    <!-- Edit Billing Information Dialog -->
    <OuiDialog v-model:open="editBillingInfoDialogOpen" title="Edit Billing Information">
      <OuiStack gap="lg">
        <OuiStack gap="xs">
          <OuiText size="sm" color="muted">
            Update your billing information. This will be synced to your Stripe customer account.
          </OuiText>
          <OuiText v-if="error" size="sm" color="danger">{{ error }}</OuiText>
        </OuiStack>

        <OuiStack gap="md">
          <OuiStack gap="xs">
            <OuiText size="sm" weight="medium">Billing Email</OuiText>
            <OuiInput
              v-model="billingInfoForm.billingEmail"
              type="email"
              placeholder="billing@example.com"
            />
          </OuiStack>

          <OuiStack gap="xs">
            <OuiText size="sm" weight="medium">Company Name</OuiText>
            <OuiInput
              v-model="billingInfoForm.companyName"
              placeholder="Company Name"
            />
          </OuiStack>

          <OuiStack gap="xs">
            <OuiText size="sm" weight="medium">Tax ID</OuiText>
            <OuiInput
              v-model="billingInfoForm.taxId"
              placeholder="Tax ID (optional)"
            />
          </OuiStack>

          <OuiStack gap="xs">
            <OuiText size="sm" weight="medium">Address Line 1</OuiText>
            <OuiInput
              v-model="billingInfoForm.address.line1"
              placeholder="Street address"
            />
          </OuiStack>

          <OuiStack gap="xs">
            <OuiText size="sm" weight="medium">Address Line 2</OuiText>
            <OuiInput
              v-model="billingInfoForm.address.line2"
              placeholder="Apartment, suite, etc. (optional)"
            />
          </OuiStack>

          <OuiGrid cols="2" gap="md">
            <OuiStack gap="xs">
              <OuiText size="sm" weight="medium">City</OuiText>
              <OuiInput
                v-model="billingInfoForm.address.city"
                placeholder="City"
              />
            </OuiStack>

            <OuiStack gap="xs">
              <OuiText size="sm" weight="medium">State</OuiText>
              <OuiInput
                v-model="billingInfoForm.address.state"
                placeholder="State (optional)"
              />
            </OuiStack>
          </OuiGrid>

          <OuiGrid cols="2" gap="md">
            <OuiStack gap="xs">
              <OuiText size="sm" weight="medium">Postal Code</OuiText>
              <OuiInput
                v-model="billingInfoForm.address.postalCode"
                placeholder="Postal code"
              />
            </OuiStack>

            <OuiStack gap="xs">
              <OuiText size="sm" weight="medium">Country</OuiText>
              <OuiInput
                v-model="billingInfoForm.address.country"
                placeholder="Country code (e.g., US)"
              />
            </OuiStack>
          </OuiGrid>
        </OuiStack>

        <OuiFlex justify="end" gap="sm">
          <OuiButton variant="ghost" @click="editBillingInfoDialogOpen = false">
            Cancel
          </OuiButton>
          <OuiButton 
            variant="solid" 
            @click="updateBillingInfo"
            :disabled="billingInfoLoading"
          >
            {{ billingInfoLoading ? "Saving..." : "Save Changes" }}
          </OuiButton>
        </OuiFlex>
      </OuiStack>
    </OuiDialog>

    <!-- Cancel Subscription Dialog -->
    <OuiDialog v-model:open="cancelSubscriptionDialogOpen" title="Cancel Subscription">
      <OuiStack gap="lg">
        <OuiStack gap="xs">
          <OuiText size="sm" color="muted">
            Are you sure you want to cancel this subscription? The subscription will remain active until the end of the current billing period, and you will continue to have access until then.
          </OuiText>
          <OuiText v-if="error" size="sm" color="danger">{{ error }}</OuiText>
        </OuiStack>

        <OuiFlex justify="end" gap="sm">
          <OuiButton variant="ghost" @click="cancelSubscriptionDialogOpen = false">
            Keep Subscription
          </OuiButton>
          <OuiButton 
            variant="solid" 
            color="danger"
            @click="cancelSubscription"
            :disabled="subscriptionsLoading"
          >
            {{ subscriptionsLoading ? "Canceling..." : "Cancel Subscription" }}
          </OuiButton>
        </OuiFlex>
      </OuiStack>
    </OuiDialog>

    <!-- Update Subscription Payment Method Dialog -->
    <OuiDialog v-model:open="updateSubscriptionPaymentDialogOpen" title="Update Subscription Payment Method">
      <OuiStack gap="lg">
        <OuiStack gap="xs">
          <OuiText size="sm" color="muted">
            Select a payment method to use for this subscription.
          </OuiText>
          <OuiText v-if="error" size="sm" color="danger">{{ error }}</OuiText>
        </OuiStack>

        <OuiStack gap="sm">
          <OuiBox
            v-for="pm in paymentMethods"
            :key="pm.id"
            p="md"
            border="1"
            borderColor="muted"
            rounded="md"
            :class="{ 'ring-2 ring-primary': pm.isDefault }"
            @click="updateSubscriptionPaymentMethod(pm.id)"
            class="cursor-pointer hover:bg-muted/50 transition-colors"
          >
            <OuiFlex justify="between" align="center">
              <OuiFlex gap="md" align="center">
                <CreditCardIcon class="h-6 w-6 text-muted" />
                <OuiStack gap="xs">
                  <OuiText size="sm" weight="medium">
                    {{ pm.card ? `${formatCardBrand(pm.card.brand)} •••• ${pm.card.last4}` : pm.type }}
                  </OuiText>
                  <OuiText size="xs" color="muted" v-if="pm.card">
                    Expires {{ pm.card.expMonth }}/{{ pm.card.expYear }}
                  </OuiText>
                </OuiStack>
              </OuiFlex>
              <OuiBadge v-if="pm.isDefault" variant="success" size="xs">
                Default
              </OuiBadge>
            </OuiFlex>
          </OuiBox>
        </OuiStack>

        <OuiFlex justify="end" gap="sm">
          <OuiButton variant="ghost" @click="updateSubscriptionPaymentDialogOpen = false">
            Cancel
          </OuiButton>
        </OuiFlex>
      </OuiStack>
    </OuiDialog>
  </OuiStack>
</template>
