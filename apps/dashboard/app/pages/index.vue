<template>
  <div class="min-h-screen bg-background">
    <!-- Hero Section -->
    <div class="fixed top-0 left-0 right-0 z-30 pt-2 md:pt-5 px-3 md:px-4">
      <OuiContainer
        as="nav"
        :size="isScrolled ? '5xl' : '7xl'"
        class="flex items-center rounded-xl justify-between w-full mx-auto transform-gpu transition-all duration-500 ease-out border will-change-transform"
        :class="
          isScrolled
            ? 'py-1.5 px-2 md:py-2 md:px-2 bg-background/70 backdrop-blur-sm shadow-sm border-border-muted'
            : 'p-0 border-transparent'
        "
      >
        <!-- Navigation -->

        <OuiFlex align="center" gap="md" class="flex-1 min-w-0">
          <OuiFlex align="center" gap="sm" class="min-w-0">
            <ObienteLogo size="md" />
            <OuiText
              size="xl"
              weight="bold"
              color="primary"
              class="truncate text-lg md:text-xl"
              >Obiente Cloud</OuiText
            >
          </OuiFlex>
        </OuiFlex>

        <OuiFlex align="center" gap="md" class="shrink-0 hidden sm:flex">
          <OuiButton variant="ghost" size="sm">Features</OuiButton>
          <OuiButton variant="ghost" size="sm">Pricing</OuiButton>
          <OuiButton variant="ghost" size="sm" @click="navigateTo('/docs')"
            >Docs</OuiButton
          >
          <OuiButton variant="outline" size="sm" @click="handleSignUp">
            Sign Up
          </OuiButton>
          <OuiButton
            variant="outline"
            size="sm"
            @click="navigateTo('/dashboard')"
            >Sign In</OuiButton
          >
        </OuiFlex>
      </OuiContainer>
    </div>
    <OuiContainer size="7xl" class="relative px-3 md:px-4 pt-20 md:pt-32">
      <!-- Hero Content -->
      <OuiStack
        gap="xl"
        align="center"
        class="text-center py-16 md:pb-32 lg:pb-32 md:pt-20 md:gap-16 relative z-10"
      >
        <OuiStack gap="md" align="center" class="max-w-4xl md:gap-6">
          <OuiText
            as="h1"
            size="6xl"
            weight="bold"
            color="primary"
            class="text-3xl md:text-5xl lg:text-7xl leading-tight tracking-tight"
          >
            Deploy, Scale, and Manage
            <br class="hidden md:block" />
            Your Cloud Infrastructure
            <br class="hidden md:block" />
            <span
              class="bg-clip-text text-transparent bg-gradient-to-r from-accent-primary to-accent-primary/70 bg-[length:150%_auto] animate-gradient-subtle"
            >
              With Obiente Cloud
            </span>
          </OuiText>

          <OuiText
            size="xl"
            color="secondary"
            class="max-w-2xl px-2 md:px-0 text-md md:text-xl lg:text-2xl leading-relaxed"
          >
            Deploy containers, game servers, and VPS instances.
            <span class="text-primary font-semibold"
              >Pay only for what you use</span
            >
            with transparent, resource-based pricing. No more overpaying for
            unused resources.
          </OuiText>
        </OuiStack>

        <OuiFlex
          gap="sm"
          wrap="wrap"
          justify="center"
          class="mt-6 md:mt-8 md:gap-4 animate-slide-up-delay"
        >
          <OuiButton
            size="lg"
            color="primary"
            class="gap-2 shadow-lg shadow-accent-primary/25 w-full sm:w-auto text-sm md:text-base"
            @click="navigateTo('/dashboard')"
          >
            <RocketLaunchIcon class="h-5 w-5" />
            Get Started
          </OuiButton>
          <OuiButton
            variant="outline"
            size="lg"
            class="gap-2 w-full sm:w-auto text-sm md:text-base px-8 py-4 border-2 hover:bg-accent-primary/10 transform hover:scale-105 transition-all duration-200 hover:-translate-y-1"
            @click="navigateTo('/docs')"
          >
            <ChatBubbleLeftRightIcon class="h-5 w-5 md:h-6 md:w-6" />
            <span class="hidden sm:inline">View Documentation</span>
            <span class="sm:hidden">Docs</span>
          </OuiButton>
        </OuiFlex>

        <!-- Stats -->
        <OuiFlex
          gap="md"
          wrap="wrap"
          justify="center"
          class="mt-8 md:mt-16 opacity-75 md:gap-8"
        >
          <OuiStack gap="xs" align="center">
            <OuiText
              size="2xl"
              weight="bold"
              color="primary"
              class="text-xl md:text-2xl"
              >Fast</OuiText
            >
            <OuiText size="sm" color="secondary" class="text-xs md:text-sm"
              >Deployments</OuiText
            >
          </OuiStack>
          <OuiStack gap="xs" align="center">
            <OuiText
              size="2xl"
              weight="bold"
              color="primary"
              class="text-xl md:text-2xl lg:text-3xl"
              >Reliable</OuiText
            >
            <OuiText size="sm" color="secondary" class="text-xs md:text-sm"
              >Infrastructure</OuiText
            >
          </OuiStack>
          <OuiStack gap="xs" align="center">
            <OuiText
              size="2xl"
              weight="bold"
              color="primary"
              class="text-xl md:text-2xl lg:text-3xl"
              >Quality</OuiText
            >
            <OuiText size="sm" color="secondary" class="text-xs md:text-sm"
              >Hardware</OuiText
            >
          </OuiStack>
          <OuiStack gap="xs" align="center">
            <OuiText
              size="2xl"
              weight="bold"
              color="primary"
              class="text-xl md:text-2xl lg:text-3xl"
              >Personal</OuiText
            >
            <OuiText size="sm" color="secondary" class="text-xs md:text-sm"
              >Support</OuiText
            >
          </OuiStack>
        </OuiFlex>
      </OuiStack>
    </OuiContainer>

    <!-- Features Section -->
    <OuiContainer py="3xl" size="7xl" class="px-3 md:px-4 md:py-24">
      <OuiStack gap="2xl" align="center">
        <OuiStack gap="lg" align="center" class="text-center max-w-3xl">
          <OuiText
            as="h2"
            size="3xl"
            weight="bold"
            color="primary"
            class="md:text-5xl"
          >
            Everything you need to deploy
          </OuiText>
          <OuiText size="lg" color="secondary" class="md:text-xl">
            Deploy containers, game servers, and VPS instances. Pay-as-you-go
            pricing means you only pay for actual resource usage, not idle time.
          </OuiText>
        </OuiStack>

        <OuiGrid :cols="{ sm: 1, md: 2, lg: 3 }" gap="xl" class="w-full">
          <!-- GitHub Integration -->
          <OuiCard
            variant="default"
            hoverable
            class="group cursor-pointer transition-all duration-200 hover:-translate-y-1 hover:shadow-xl border-accent-primary/10 hover:border-accent-primary/30"
          >
            <OuiCardBody>
              <OuiStack gap="md">
                <OuiBox
                  class="w-12 h-12 bg-accent-primary/20 rounded-lg flex items-center justify-center"
                >
                  <RocketLaunchIcon class="h-6 w-6 text-accent-primary" />
                </OuiBox>
                <OuiStack gap="sm">
                  <OuiText as="h3" size="lg" weight="semibold" color="primary">
                    GitHub Integration
                  </OuiText>
                  <OuiText size="sm" color="secondary">
                    Deploy directly from GitHub repositories. Automatic builds
                    on push with support for Docker and Docker Compose.
                  </OuiText>
                </OuiStack>
              </OuiStack>
            </OuiCardBody>
          </OuiCard>

          <!-- Custom Domains -->
          <OuiCard
            variant="default"
            hoverable
            class="group cursor-pointer transition-all duration-200 hover:-translate-y-1 hover:shadow-xl border-accent-success/10 hover:border-accent-success/30"
          >
            <OuiCardBody>
              <OuiStack gap="md">
                <OuiBox
                  class="w-12 h-12 bg-accent-success/20 rounded-lg flex items-center justify-center"
                >
                  <ShieldCheckIcon class="h-6 w-6 text-accent-success" />
                </OuiBox>
                <OuiStack gap="sm">
                  <OuiText as="h3" size="lg" weight="semibold" color="primary">
                    Custom Domains
                  </OuiText>
                  <OuiText size="sm" color="secondary">
                    Configure custom domains with automatic SSL certificates.
                    Set up routing rules for multiple paths and domains.
                  </OuiText>
                </OuiStack>
              </OuiStack>
            </OuiCardBody>
          </OuiCard>

          <!-- Container Deployments -->
          <OuiCard
            variant="default"
            hoverable
            class="group cursor-pointer transition-all duration-200 hover:-translate-y-1 hover:shadow-xl border-accent-secondary/10 hover:border-accent-secondary/30"
          >
            <OuiCardBody>
              <OuiStack gap="md">
                <OuiBox
                  class="w-12 h-12 bg-accent-secondary/20 rounded-lg flex items-center justify-center"
                >
                  <ServerIcon class="h-6 w-6 text-accent-secondary" />
                </OuiBox>
                <OuiStack gap="sm">
                  <OuiText as="h3" size="lg" weight="semibold" color="primary">
                    Container Deployments
                  </OuiText>
                  <OuiText size="sm" color="secondary">
                    Deploy single containers or multi-container applications
                    with Docker Compose. Full control over your application
                    stack.
                  </OuiText>
                </OuiStack>
              </OuiStack>
            </OuiCardBody>
          </OuiCard>

          <!-- Environment Variables -->
          <OuiCard
            variant="default"
            hoverable
            class="group cursor-pointer transition-all duration-200 hover:-translate-y-1 hover:shadow-xl border-accent-info/10 hover:border-accent-info/30"
          >
            <OuiCardBody>
              <OuiStack gap="md">
                <OuiBox
                  class="w-12 h-12 bg-accent-info/20 rounded-lg flex items-center justify-center"
                >
                  <CircleStackIcon class="h-6 w-6 text-accent-info" />
                </OuiBox>
                <OuiStack gap="sm">
                  <OuiText as="h3" size="lg" weight="semibold" color="primary">
                    Environment Variables
                  </OuiText>
                  <OuiText size="sm" color="secondary">
                    Manage environment variables per deployment. Secure secrets
                    management with easy configuration through the dashboard.
                  </OuiText>
                </OuiStack>
              </OuiStack>
            </OuiCardBody>
          </OuiCard>

          <!-- Real-Time Monitoring -->
          <OuiCard
            variant="default"
            hoverable
            class="group cursor-pointer transition-all duration-200 hover:-translate-y-1 hover:shadow-xl border-accent-warning/10 hover:border-accent-warning/30"
          >
            <OuiCardBody>
              <OuiStack gap="md">
                <OuiBox
                  class="w-12 h-12 bg-accent-warning/20 rounded-lg flex items-center justify-center"
                >
                  <ChartBarIcon class="h-6 w-6 text-accent-warning" />
                </OuiBox>
                <OuiStack gap="sm">
                  <OuiText as="h3" size="lg" weight="semibold" color="primary">
                    Real-Time Monitoring
                  </OuiText>
                  <OuiText size="sm" color="secondary">
                    Track vCPU, memory, bandwidth, and storage usage in
                    real-time. View build logs, application logs, and
                    performance metrics.
                  </OuiText>
                </OuiStack>
              </OuiStack>
            </OuiCardBody>
          </OuiCard>

          <!-- File Management -->
          <OuiCard
            variant="default"
            hoverable
            class="group cursor-pointer transition-all duration-200 hover:-translate-y-1 hover:shadow-xl border-accent-danger/10 hover:border-accent-danger/30"
          >
            <OuiCardBody>
              <OuiStack gap="md">
                <OuiBox
                  class="w-12 h-12 bg-accent-danger/20 rounded-lg flex items-center justify-center"
                >
                  <FolderIcon class="h-6 w-6 text-accent-danger" />
                </OuiBox>
                <OuiStack gap="sm">
                  <OuiText as="h3" size="lg" weight="semibold" color="primary">
                    File Management
                  </OuiText>
                  <OuiText size="sm" color="secondary">
                    Upload, edit, and manage files directly through the
                    dashboard. Terminal access for advanced configuration and
                    debugging.
                  </OuiText>
                </OuiStack>
              </OuiStack>
            </OuiCardBody>
          </OuiCard>

          <!-- Pay-as-You-Go Pricing -->
          <OuiCard
            variant="default"
            hoverable
            class="group cursor-pointer transition-all duration-200 hover:-translate-y-1 hover:shadow-xl border-accent-success/10 hover:border-accent-success/30"
          >
            <OuiCardBody>
              <OuiStack gap="md">
                <OuiBox
                  class="w-12 h-12 bg-accent-success/20 rounded-lg flex items-center justify-center"
                >
                  <CircleStackIcon class="h-6 w-6 text-accent-success" />
                </OuiBox>
                <OuiStack gap="sm">
                  <OuiText as="h3" size="lg" weight="semibold" color="primary">
                    Pay-as-You-Go Pricing
                  </OuiText>
                  <OuiText size="sm" color="secondary">
                    Only pay for resources you actually use. Perfect for game
                    servers and VPSs where you often overpay for idle time. Stop
                    wasting money on unused capacity.
                  </OuiText>
                </OuiStack>
              </OuiStack>
            </OuiCardBody>
          </OuiCard>

          <!-- Game Server Hosting -->
          <OuiCard
            variant="default"
            hoverable
            class="group cursor-pointer transition-all duration-200 hover:-translate-y-1 hover:shadow-xl border-accent-primary/10 hover:border-accent-primary/30"
          >
            <OuiCardBody>
              <OuiStack gap="md">
                <OuiBox
                  class="w-12 h-12 bg-accent-primary/20 rounded-lg flex items-center justify-center"
                >
                  <ServerIcon class="h-6 w-6 text-accent-primary" />
                </OuiBox>
                <OuiStack gap="sm">
                  <OuiText as="h3" size="lg" weight="semibold" color="primary">
                    Game Server Hosting
                  </OuiText>
                  <OuiText size="sm" color="secondary">
                    Deploy dedicated game servers with Docker support. Pay only
                    when your server is running - perfect for Minecraft, game
                    servers, and multiplayer applications.
                  </OuiText>
                </OuiStack>
              </OuiStack>
            </OuiCardBody>
          </OuiCard>

          <!-- VPS Instances -->
          <OuiCard
            variant="default"
            hoverable
            class="group cursor-pointer transition-all duration-200 hover:-translate-y-1 hover:shadow-xl border-accent-info/10 hover:border-accent-info/30"
          >
            <OuiCardBody>
              <OuiStack gap="md">
                <OuiBox
                  class="w-12 h-12 bg-accent-info/20 rounded-lg flex items-center justify-center"
                >
                  <ServerIcon class="h-6 w-6 text-accent-info" />
                </OuiBox>
                <OuiStack gap="sm">
                  <OuiText as="h3" size="lg" weight="semibold" color="primary">
                    VPS Instances
                  </OuiText>
                  <OuiText size="sm" color="secondary">
                    Get full root access VPS instances with complete control.
                    Pay only for actual CPU and memory usage - perfect for
                    development, hosting, and personal projects where you need
                    full flexibility.
                  </OuiText>
                </OuiStack>
              </OuiStack>
            </OuiCardBody>
          </OuiCard>
        </OuiGrid>
      </OuiStack>
    </OuiContainer>

    <!-- Infrastructure Section -->
    <OuiContainer
      as="section"
      py="3xl"
      size="7xl"
      class="px-3 md:px-4 md:py-24"
    >
      <OuiStack gap="2xl" align="center">
        <OuiStack gap="lg" align="center" class="text-center max-w-3xl">
          <OuiText
            as="h2"
            size="3xl"
            weight="bold"
            color="primary"
            class="md:text-5xl"
          >
            Enterprise-Grade Infrastructure
          </OuiText>
          <OuiText size="lg" color="secondary" class="md:text-xl">
            Built on premium hardware with redundancy and reliability at its
            core.
          </OuiText>
        </OuiStack>

        <OuiGrid :cols="{ sm: 1, md: 2, lg: 3 }" gap="xl" class="w-full">
          <!-- Premium Hardware -->
          <OuiCard variant="default" class="h-full border-accent-primary/10 hover:border-accent-primary/20 transition-colors">
            <OuiCardBody>
              <OuiStack gap="md">
                <OuiBox
                  class="w-12 h-12 bg-accent-primary/20 rounded-lg flex items-center justify-center"
                >
                  <ServerIcon class="h-6 w-6 text-accent-primary" />
                </OuiBox>
                <OuiStack gap="sm">
                  <OuiText as="h3" size="lg" weight="semibold" color="primary">
                    Premium Hardware
                  </OuiText>
                  <OuiText size="sm" color="secondary">
                    Powered by AMD Ryzen 7 7900X processors with DDR5 RAM and
                    NVMe SSDs for exceptional performance.
                  </OuiText>
                </OuiStack>
              </OuiStack>
            </OuiCardBody>
          </OuiCard>

          <!-- High Availability -->
          <OuiCard variant="default" class="h-full border-accent-success/10 hover:border-accent-success/20 transition-colors">
            <OuiCardBody>
              <OuiStack gap="md">
                <OuiBox
                  class="w-12 h-12 bg-accent-success/20 rounded-lg flex items-center justify-center"
                >
                  <ShieldCheckIcon class="h-6 w-6 text-accent-success" />
                </OuiBox>
                <OuiStack gap="sm">
                  <OuiText as="h3" size="lg" weight="semibold" color="primary">
                    High Availability
                  </OuiText>
                  <OuiText size="sm" color="secondary">
                    Automatic failover to our fallback node ensures your
                    services stay online even during main node outages.
                  </OuiText>
                </OuiStack>
              </OuiStack>
            </OuiCardBody>
          </OuiCard>

          <!-- Free Support -->
          <OuiCard variant="default" class="h-full border-accent-secondary/10 hover:border-accent-secondary/20 transition-colors">
            <OuiCardBody>
              <OuiStack gap="md">
                <OuiBox
                  class="w-12 h-12 bg-accent-secondary/20 rounded-lg flex items-center justify-center"
                >
                  <ChatBubbleLeftRightIcon
                    class="h-6 w-6 text-accent-secondary"
                  />
                </OuiBox>
                <OuiStack gap="sm">
                  <OuiText as="h3" size="lg" weight="semibold" color="primary">
                    Free Support
                  </OuiText>
                  <OuiText size="sm" color="secondary">
                    Get help when you need it. Our support team is available to
                    assist with deployment, configuration, and troubleshooting.
                  </OuiText>
                </OuiStack>
              </OuiStack>
            </OuiCardBody>
          </OuiCard>

          <!-- Security -->
          <OuiCard variant="default" class="h-full border-accent-danger/10 hover:border-accent-danger/20 transition-colors">
            <OuiCardBody>
              <OuiStack gap="md">
                <OuiBox
                  class="w-12 h-12 bg-accent-danger/20 rounded-lg flex items-center justify-center"
                >
                  <ShieldCheckIcon class="h-6 w-6 text-accent-danger" />
                </OuiBox>
                <OuiStack gap="sm">
                  <OuiText as="h3" size="lg" weight="semibold" color="primary">
                    Security
                  </OuiText>
                  <OuiText size="sm" color="secondary">
                    SSL certificates, automated backups, and comprehensive audit
                    logs to keep your data secure.
                  </OuiText>
                </OuiStack>
              </OuiStack>
            </OuiCardBody>
          </OuiCard>

          <!-- DNS Management -->
          <OuiCard variant="default" class="h-full border-accent-info/10 hover:border-accent-info/20 transition-colors">
            <OuiCardBody>
              <OuiStack gap="md">
                <OuiBox
                  class="w-12 h-12 bg-accent-info/20 rounded-lg flex items-center justify-center"
                >
                  <GlobeAltIcon class="h-6 w-6 text-accent-info" />
                </OuiBox>
                <OuiStack gap="sm">
                  <OuiText as="h3" size="lg" weight="semibold" color="primary">
                    DNS Management
                  </OuiText>
                  <OuiText size="sm" color="secondary">
                    Custom domain support with automatic DNS configuration.
                    Verify domains and set up routing rules for your
                    deployments.
                  </OuiText>
                </OuiStack>
              </OuiStack>
            </OuiCardBody>
          </OuiCard>

          <!-- Real-Time Monitoring -->
          <OuiCard variant="default" class="h-full border-accent-warning/10 hover:border-accent-warning/20 transition-colors">
            <OuiCardBody>
              <OuiStack gap="md">
                <OuiBox
                  class="w-12 h-12 bg-accent-warning/20 rounded-lg flex items-center justify-center"
                >
                  <ChartBarIcon class="h-6 w-6 text-accent-warning" />
                </OuiBox>
                <OuiStack gap="sm">
                  <OuiText as="h3" size="lg" weight="semibold" color="primary">
                    Real-Time Monitoring
                  </OuiText>
                  <OuiText size="sm" color="secondary">
                    Track vCPU, memory, bandwidth, and storage usage in
                    real-time. View build logs, application logs, and
                    performance metrics.
                  </OuiText>
                </OuiStack>
              </OuiStack>
            </OuiCardBody>
          </OuiCard>
        </OuiGrid>
      </OuiStack>
    </OuiContainer>

    <!-- Pricing Section -->
    <OuiContainer
      as="section"
      py="3xl"
      size="7xl"
      class="px-3 md:px-4 md:py-24"
    >
      <OuiStack gap="2xl" align="center">
        <OuiStack gap="lg" align="center" class="text-center max-w-3xl">
          <OuiText
            as="h2"
            size="3xl"
            weight="bold"
            color="primary"
            class="md:text-4xl"
          >
            Simple, transparent pricing
          </OuiText>
          <OuiText size="lg" color="secondary">
            Pay only for what you use. No fixed plans, no hidden fees. Free
            support included. Stop overpaying for unused resources.
          </OuiText>
          <OuiText size="sm" color="secondary" class="opacity-60 italic">
            Students, open-source projects, non-profits, and early-stage
            startups may qualify for reduced pricing. Contact us to discuss.
          </OuiText>
        </OuiStack>

        <OuiGrid
          cols="1"
          cols-md="2"
          cols-lg="4"
          gap="xl"
          class="w-full max-w-6xl"
        >
          <!-- Small App Example -->
          <OuiCard
            variant="default"
            hoverable
            class="text-center group hover:-translate-y-1 transition-all duration-200 h-full flex flex-col border-accent-secondary/10 hover:border-accent-secondary/30"
          >
            <OuiCardBody class="flex flex-col h-full">
              <OuiStack gap="lg" class="flex-1">
                <OuiStack gap="md">
                  <OuiText size="lg" weight="semibold" color="primary"
                    >Small App</OuiText
                  >
                  <OuiText size="sm" color="secondary"
                    >Perfect for side projects</OuiText
                  >
                </OuiStack>

                <OuiStack gap="xs" align="center">
                  <OuiText size="3xl" weight="bold" color="primary"
                    >~$5</OuiText
                  >
                  <OuiText size="sm" color="secondary">per month</OuiText>
                </OuiStack>

                <OuiStack gap="sm" align="start" class="text-left flex-1">
                  <OuiFlex align="center" gap="sm">
                    <CheckIcon class="h-4 w-4 text-accent-success shrink-0" />
                    <OuiText size="sm" color="secondary"
                      >0.5 GB RAM running 24/7</OuiText
                    >
                  </OuiFlex>
                  <OuiFlex align="center" gap="sm">
                    <CheckIcon class="h-4 w-4 text-accent-success shrink-0" />
                    <OuiText size="sm" color="secondary"
                      >0.25 vCPU cores running 24/7</OuiText
                    >
                  </OuiFlex>
                  <OuiFlex align="center" gap="sm">
                    <CheckIcon class="h-4 w-4 text-accent-success shrink-0" />
                    <OuiText size="sm" color="secondary"
                      >~10 GB bandwidth/month</OuiText
                    >
                  </OuiFlex>
                  <OuiFlex align="center" gap="sm">
                    <CheckIcon class="h-4 w-4 text-accent-success shrink-0" />
                    <OuiText size="sm" color="secondary">~5 GB storage</OuiText>
                  </OuiFlex>
                  <OuiFlex align="center" gap="sm">
                    <CheckIcon class="h-4 w-4 text-accent-success shrink-0" />
                    <OuiText size="sm" color="secondary">Free support</OuiText>
                  </OuiFlex>
                  <OuiFlex align="center" gap="sm">
                    <CheckIcon class="h-4 w-4 text-accent-success shrink-0" />
                    <OuiText size="sm" color="secondary"
                      >Automatic failover to fallback node</OuiText
                    >
                  </OuiFlex>
                </OuiStack>
              </OuiStack>
            </OuiCardBody>
          </OuiCard>

          <!-- Game Server Example -->
          <OuiCard
            variant="default"
            hoverable
            class="text-center group hover:-translate-y-1 transition-all duration-200 ring-2 ring-accent-success/50 h-full flex flex-col"
          >
            <OuiCardBody class="flex flex-col h-full">
              <OuiStack gap="lg" class="flex-1">
                <OuiStack gap="md">
                  <OuiFlex align="center" justify="center" gap="sm">
                    <OuiText size="lg" weight="semibold" color="primary"
                      >Game Server</OuiText
                    >
                    <OuiBadge color="success" size="sm">Save 50%</OuiBadge>
                  </OuiFlex>
                  <OuiText size="sm" color="secondary"
                    >12 hours/day runtime</OuiText
                  >
                </OuiStack>

                <OuiStack gap="xs" align="center">
                  <OuiText size="3xl" weight="bold" color="primary"
                    >~$15</OuiText
                  >
                  <OuiText size="sm" color="secondary">per month</OuiText>
                  <OuiText
                    size="xs"
                    color="secondary"
                    class="opacity-75 italic line-through"
                  >
                    Traditional hosting: $20-30/month
                  </OuiText>
                </OuiStack>

                <OuiStack gap="sm" align="start" class="text-left flex-1">
                  <OuiFlex align="center" gap="sm">
                    <CheckIcon class="h-4 w-4 text-accent-success shrink-0" />
                    <OuiText size="sm" color="secondary"
                      >4 GB RAM running 12h/day</OuiText
                    >
                  </OuiFlex>
                  <OuiFlex align="center" gap="sm">
                    <CheckIcon class="h-4 w-4 text-accent-success shrink-0" />
                    <OuiText size="sm" color="secondary"
                      >2 vCPU cores average</OuiText
                    >
                  </OuiFlex>
                  <OuiFlex align="center" gap="sm">
                    <CheckIcon class="h-4 w-4 text-accent-success shrink-0" />
                    <OuiText size="sm" color="secondary"
                      >~100 GB bandwidth/month</OuiText
                    >
                  </OuiFlex>
                  <OuiFlex align="center" gap="sm">
                    <CheckIcon class="h-4 w-4 text-accent-success shrink-0" />
                    <OuiText size="sm" color="secondary"
                      >~20 GB storage</OuiText
                    >
                  </OuiFlex>
                  <OuiFlex align="center" gap="sm">
                    <CheckIcon class="h-4 w-4 text-accent-success shrink-0" />
                    <OuiText size="sm" color="secondary"
                      >Low costs when idle or offline</OuiText
                    >
                  </OuiFlex>
                  <OuiFlex align="center" gap="sm">
                    <CheckIcon class="h-4 w-4 text-accent-success shrink-0" />
                    <OuiText size="sm" color="secondary">Free support</OuiText>
                  </OuiFlex>
                </OuiStack>
              </OuiStack>
            </OuiCardBody>
          </OuiCard>

          <!-- VPS Example -->
          <OuiCard
            variant="default"
            hoverable
            class="text-center group hover:-translate-y-1 transition-all duration-200 h-full flex flex-col border-accent-info/10 hover:border-accent-info/30"
          >
            <OuiCardBody class="flex flex-col h-full">
              <OuiStack gap="lg" class="flex-1">
                <OuiStack gap="md">
                  <OuiText size="lg" weight="semibold" color="primary"
                    >VPS Instance</OuiText
                  >
                  <OuiText size="sm" color="secondary"
                    >Competitive pricing</OuiText
                  >
                </OuiStack>

                <OuiStack gap="xs" align="center">
                  <OuiText size="3xl" weight="bold" color="primary"
                    >~$8</OuiText
                  >
                  <OuiText size="sm" color="secondary">per month</OuiText>
                  <OuiText
                    size="xs"
                    color="secondary"
                    class="opacity-75 italic"
                  >
                    Traditional VPS: $10-12/month
                  </OuiText>
                </OuiStack>

                <OuiStack gap="sm" align="start" class="text-left flex-1">
                  <OuiFlex align="center" gap="sm">
                    <CheckIcon class="h-4 w-4 text-accent-success shrink-0" />
                    <OuiText size="sm" color="secondary"
                      >2 GB RAM running 24/7</OuiText
                    >
                  </OuiFlex>
                  <OuiFlex align="center" gap="sm">
                    <CheckIcon class="h-4 w-4 text-accent-success shrink-0" />
                    <OuiText size="sm" color="secondary"
                      >1 vCPU core running 24/7</OuiText
                    >
                  </OuiFlex>
                  <OuiFlex align="center" gap="sm">
                    <CheckIcon class="h-4 w-4 text-accent-success shrink-0" />
                    <OuiText size="sm" color="secondary"
                      >~50 GB bandwidth/month</OuiText
                    >
                  </OuiFlex>
                  <OuiFlex align="center" gap="sm">
                    <CheckIcon class="h-4 w-4 text-accent-success shrink-0" />
                    <OuiText size="sm" color="secondary"
                      >~10 GB storage</OuiText
                    >
                  </OuiFlex>
                  <OuiFlex align="center" gap="sm">
                    <CheckIcon class="h-4 w-4 text-accent-success shrink-0" />
                    <OuiText size="sm" color="secondary"
                      >Pay for actual usage, not idle time</OuiText
                    >
                  </OuiFlex>
                  <OuiFlex align="center" gap="sm">
                    <CheckIcon class="h-4 w-4 text-accent-success shrink-0" />
                    <OuiText size="sm" color="secondary">Free support</OuiText>
                  </OuiFlex>
                </OuiStack>
              </OuiStack>
            </OuiCardBody>
          </OuiCard>

          <!-- Medium App Example -->
          <OuiCard
            variant="default"
            hoverable
            class="text-center group hover:-translate-y-1 transition-all duration-200 ring-2 ring-accent-primary/50 h-full flex flex-col"
          >
            <OuiCardBody class="flex flex-col h-full">
              <OuiStack gap="lg" class="flex-1">
                <OuiStack gap="md">
                  <OuiFlex align="center" justify="center" gap="sm">
                    <OuiText size="lg" weight="semibold" color="primary"
                      >Medium App</OuiText
                    >
                    <OuiBadge color="primary" size="sm">Popular</OuiBadge>
                  </OuiFlex>
                  <OuiText size="sm" color="secondary"
                    >For growing teams</OuiText
                  >
                </OuiStack>

                <OuiStack gap="xs" align="center">
                  <OuiText size="3xl" weight="bold" color="primary"
                    >~$14</OuiText
                  >
                  <OuiText size="sm" color="secondary">per month</OuiText>
                </OuiStack>

                <OuiStack gap="sm" align="start" class="text-left flex-1">
                  <OuiFlex align="center" gap="sm">
                    <CheckIcon class="h-4 w-4 text-accent-success shrink-0" />
                    <OuiText size="sm" color="secondary"
                      >2 GB RAM running 24/7</OuiText
                    >
                  </OuiFlex>
                  <OuiFlex align="center" gap="sm">
                    <CheckIcon class="h-4 w-4 text-accent-success shrink-0" />
                    <OuiText size="sm" color="secondary"
                      >1 vCPU core running 24/7</OuiText
                    >
                  </OuiFlex>
                  <OuiFlex align="center" gap="sm">
                    <CheckIcon class="h-4 w-4 text-accent-success shrink-0" />
                    <OuiText size="sm" color="secondary"
                      >~50 GB bandwidth/month</OuiText
                    >
                  </OuiFlex>
                  <OuiFlex align="center" gap="sm">
                    <CheckIcon class="h-4 w-4 text-accent-success shrink-0" />
                    <OuiText size="sm" color="secondary"
                      >~25 GB storage</OuiText
                    >
                  </OuiFlex>
                  <OuiFlex align="center" gap="sm">
                    <CheckIcon class="h-4 w-4 text-accent-success shrink-0" />
                    <OuiText size="sm" color="secondary"
                      >Custom configurations</OuiText
                    >
                  </OuiFlex>
                  <OuiFlex align="center" gap="sm">
                    <CheckIcon class="h-4 w-4 text-accent-success shrink-0" />
                    <OuiText size="sm" color="secondary">Free support</OuiText>
                  </OuiFlex>
                  <OuiFlex align="center" gap="sm">
                    <CheckIcon class="h-4 w-4 text-accent-success shrink-0" />
                    <OuiText size="sm" color="secondary"
                      >Automatic failover to fallback node</OuiText
                    >
                  </OuiFlex>
                </OuiStack>
              </OuiStack>
            </OuiCardBody>
          </OuiCard>
        </OuiGrid>

        <OuiFlex gap="md" wrap="wrap" justify="center" class="mt-8">
          <OuiButton
            size="lg"
            color="primary"
            class="gap-2 shadow-lg shadow-accent-primary/25"
            @click="navigateTo('/dashboard')"
          >
            <RocketLaunchIcon class="h-5 w-5" />
            Get Started
          </OuiButton>
        </OuiFlex>

        <OuiText
          size="sm"
          color="secondary"
          class="opacity-75 text-center max-w-2xl"
        >
          * Example costs based on 24/7 usage. You only pay for actual runtime -
          if your app runs part-time, you'll pay less. Use the calculator below
          to estimate your exact costs.
        </OuiText>

        <OuiCard variant="default" class="w-full max-w-4xl mt-4">
          <OuiCardBody>
            <OuiStack gap="md">
              <OuiText
                size="lg"
                weight="semibold"
                color="primary"
                class="text-center"
              >
                Why Pay-as-You-Go Matters
              </OuiText>
              <OuiGrid cols="1" cols-md="2" gap="md">
                <OuiFlex gap="sm" align="start">
                  <ServerIcon
                    class="h-5 w-5 text-accent-success shrink-0 mt-0.5"
                  />
                  <OuiStack gap="xs">
                    <OuiText size="sm" weight="semibold" color="primary">
                      Game Servers
                    </OuiText>
                    <OuiText size="sm" color="secondary">
                      Low costs when idle or offline. Traditional hosting
                      charges $20-40/month even when your server is empty.
                    </OuiText>
                  </OuiStack>
                </OuiFlex>
                <OuiFlex gap="sm" align="start">
                  <ServerIcon
                    class="h-5 w-5 text-accent-primary shrink-0 mt-0.5"
                  />
                  <OuiStack gap="xs">
                    <OuiText size="sm" weight="semibold" color="primary">
                      VPS Instances
                    </OuiText>
                    <OuiText size="sm" color="secondary">
                      Pay for actual CPU and memory usage, not idle time. Most
                      VPS providers charge full price regardless of utilization.
                    </OuiText>
                  </OuiStack>
                </OuiFlex>
                <OuiFlex gap="sm" align="start">
                  <RocketLaunchIcon
                    class="h-5 w-5 text-accent-secondary shrink-0 mt-0.5"
                  />
                  <OuiStack gap="xs">
                    <OuiText size="sm" weight="semibold" color="primary">
                      Development Environments
                    </OuiText>
                    <OuiText size="sm" color="secondary">
                      Stop paying for resources that sit idle overnight or on
                      weekends.
                    </OuiText>
                  </OuiStack>
                </OuiFlex>
                <OuiFlex gap="sm" align="start">
                  <ChartBarIcon
                    class="h-5 w-5 text-accent-warning shrink-0 mt-0.5"
                  />
                  <OuiStack gap="xs">
                    <OuiText size="sm" weight="semibold" color="primary">
                      Variable Workloads
                    </OuiText>
                    <OuiText size="sm" color="secondary">
                      Scale costs automatically with demand - no
                      over-provisioning required.
                    </OuiText>
                  </OuiStack>
                </OuiFlex>
              </OuiGrid>
            </OuiStack>
          </OuiCardBody>
        </OuiCard>
      </OuiStack>
    </OuiContainer>

    <!-- Pricing Calculator Section -->
    <PricingCalculator />

    <!-- CTA Section -->
    <OuiContainer
      as="section"
      size="7xl"
      py="3xl"
      class="px-3 md:px-4 md:py-24"
    >
      <OuiStack gap="xl" align="center" class="text-center">
        <OuiStack gap="lg" align="center" class="max-w-3xl">
          <OuiText
            as="h2"
            size="3xl"
            weight="bold"
            color="primary"
            class="md:text-4xl"
          >
            Ready to get started?
          </OuiText>
          <OuiText size="lg" color="secondary" class="md:text-xl">
            Join our growing community of developers building and deploying
            containerized applications.
          </OuiText>
        </OuiStack>
        <OuiFlex gap="md" wrap="wrap" justify="center">
          <OuiButton
            size="lg"
            color="primary"
            class="gap-2 shadow-lg shadow-accent-primary/25"
            @click="navigateTo('/dashboard')"
          >
            <RocketLaunchIcon class="h-5 w-5" />
            Get Started
          </OuiButton>
          <OuiButton variant="outline" size="lg" class="gap-2">
            <ChatBubbleLeftRightIcon class="h-5 w-5" />
            Talk to Sales
          </OuiButton>
        </OuiFlex>

        <OuiText size="sm" color="secondary" class="opacity-75">
          Resource-based pricing • Cancel anytime
        </OuiText>
      </OuiStack>
    </OuiContainer>

    <!-- Footer -->
    <AppFooter />
  </div>
</template>

<script setup lang="ts">
  import { computed } from "vue";
  import { useWindowScroll } from "@vueuse/core";
  import {
    RocketLaunchIcon,
    ServerIcon,
    CircleStackIcon,
    ChartBarIcon,
    ShieldCheckIcon,
    CheckIcon,
    ChatBubbleLeftRightIcon,
    FolderIcon,
    GlobeAltIcon,
    BoltIcon,
  } from "@heroicons/vue/24/outline";
  import PricingCalculator from "~/components/pricing/PricingCalculator.vue";
  import ObienteLogo from "~/components/app/ObienteLogo.vue";
  import { useConfig } from "~/composables/useConfig";

  // Page meta - no auth required for homepage
  definePageMeta({
    layout: false, // Use custom layout for homepage
  });

  // Check if self-hosted and redirect to dashboard (non-blocking)
  const config = useConfig();
  // On SSR: use timeout to prevent blocking page render
  // On client: no timeout - let slow connections complete
  if (import.meta.server) {
    const configPromise = config.fetchConfig();
    const timeoutPromise = new Promise(resolve => setTimeout(resolve, 500));
    await Promise.race([configPromise, timeoutPromise]);
  } else {
    // Client-side: fetch without timeout, don't block
    config.fetchConfig().catch(() => null);
  }

  if (config.selfHosted.value === true) {
    // Redirect to dashboard for self-hosted instances
    await navigateTo("/dashboard");
  }

  // Scroll state for header effect using VueUse
  const { y: scrollY } = useWindowScroll();
  const isScrolled = computed(() => scrollY.value > 10);

  // Auth composable for sign-up
  const auth = useAuth();

  // Handle sign-up click
  const handleSignUp = () => {
    if (import.meta.client) {
      auth.popupSignup();
    }
    // Server-side: signup is handled via popup, no navigation needed
  };

  // SEO meta
  useHead({
    title: "Obiente Cloud - Container, Game Server & VPS Hosting",
    meta: [
      {
        name: "description",
        content:
          "Deploy containers, game servers, and VPS instances with pay-as-you-go pricing. Only pay for actual usage - no more overpaying for unused resources. Competitive with VPS providers.",
      },
      {
        name: "keywords",
        content:
          "container hosting, docker deployment, game server hosting, vps hosting, pay as you go, cloud platform, minecraft hosting, competitive pricing",
      },
      {
        property: "og:title",
        content: "Obiente Cloud - Container, Game Server & VPS Hosting",
      },
      {
        property: "og:description",
        content:
          "Deploy containers, game servers, and VPS instances with pay-as-you-go pricing. Only pay for actual usage - no more overpaying for unused resources.",
      },
      { property: "og:type", content: "website" },
      { name: "twitter:card", content: "summary_large_image" },
    ],
  });
</script>

<style>
  /* Ensure full page background coverage for homepage */
  html,
  body {
    background-color: var(--oui-background) !important;
  }

  /* Landing page animations */
  @keyframes fade-in {
    from {
      opacity: 0;
      transform: translateY(10px);
    }
    to {
      opacity: 1;
      transform: translateY(0);
    }
  }

  @keyframes slide-up {
    from {
      opacity: 0;
      transform: translateY(30px);
    }
    to {
      opacity: 1;
      transform: translateY(0);
    }
  }

  @keyframes gradient {
    0%,
    100% {
      background-position: 0% 50%;
    }
    50% {
      background-position: 100% 50%;
    }
  }

  @keyframes gradient-subtle {
    0%,
    100% {
      background-position: 0% 50%;
    }
    50% {
      background-position: 100% 50%;
    }
  }

  .animate-fade-in {
    animation: fade-in 0.6s ease-out forwards;
  }

  .animate-slide-up {
    animation: slide-up 0.8s ease-out forwards;
    animation-delay: 0.2s;
    opacity: 0;
  }

  .animate-slide-up-delay {
    animation: slide-up 0.8s ease-out forwards;
    animation-delay: 0.4s;
    opacity: 0;
  }

  .animate-fade-in-delay {
    animation: fade-in 0.8s ease-out forwards;
    animation-delay: 0.6s;
    opacity: 0;
  }

  .animate-gradient {
    animation: gradient 3s ease infinite;
  }

  .animate-gradient-subtle {
    animation: gradient-subtle 8s ease-in-out infinite;
  }
</style>
