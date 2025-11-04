<template>
  <div class="min-h-screen bg-background">
    <!-- Hero Section -->

    <OuiContainer
      as="nav"
      :size="isScrolled ? '5xl' : '7xl'"
      class="flex items-center rounded-xl justify-between w-full mx-auto transform-gpu transition-all duration-500 ease-out border relative sticky top-5 z-30 transition-all duration-500 ease-out transform-gpu will-change-transform"
      :class="
        isScrolled
          ? 'py-2 px-2 bg-background/80 backdrop-blur-md shadow-sm border-border-muted'
          : 'p-0 bg-background border-transparent'
      "
    >
      <!-- Navigation -->

      <OuiFlex align="center" gap="md">
        <OuiBox class="flex items-center gap-2">
          <OuiText size="xl" weight="bold" color="primary"
            >Obiente Cloud</OuiText
          >
        </OuiBox>
      </OuiFlex>

      <OuiFlex align="center" gap="md" class="md:flex">
        <OuiButton variant="ghost" size="sm">Features</OuiButton>
        <OuiButton variant="ghost" size="sm">Pricing</OuiButton>
        <OuiButton variant="ghost" size="sm">Docs</OuiButton>
        <OuiButton variant="outline" size="sm" @click="navigateTo('/dashboard')"
          >Sign In</OuiButton
        >
      </OuiFlex>
    </OuiContainer>
    <OuiContainer size="7xl" class="relative">
      <!-- Hero Content -->
      <OuiStack gap="2xl" align="center" class="text-center py-32 md:py-40">
        <OuiStack gap="lg" align="center" class="max-w-4xl">
          <OuiText as="h1" size="6xl" weight="bold" color="primary">
            Deploy, Scale, and Manage Your Cloud Infrastructure <br />
            With
            <span
              class="bg-clip-text text-transparent bg-linear-to-r from-accent-primary to-accent-primary"
            >
              Obiente Cloud
            </span>
          </OuiText>

          <OuiText size="xl" color="secondary" class="max-w-2xl md:text-2xl">
            Deploy containerized applications with GitHub integration. Pay only for what you use with transparent, resource-based pricing.
          </OuiText>
        </OuiStack>

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
          <OuiButton variant="outline" size="lg" class="gap-2">
            <PlayIcon class="h-5 w-5" />
            Watch Demo
          </OuiButton>
        </OuiFlex>

        <!-- Stats -->
        <OuiFlex gap="xl" wrap="wrap" justify="center" class="mt-16 opacity-75">
          <OuiStack gap="xs" align="center">
            <OuiText size="2xl" weight="bold" color="primary">Fast</OuiText>
            <OuiText size="sm" color="secondary">Deployments</OuiText>
          </OuiStack>
          <OuiStack gap="xs" align="center">
            <OuiText size="2xl" weight="bold" color="primary">Reliable</OuiText>
            <OuiText size="sm" color="secondary">Infrastructure</OuiText>
          </OuiStack>
          <OuiStack gap="xs" align="center">
            <OuiText size="2xl" weight="bold" color="primary">Quality</OuiText>
            <OuiText size="sm" color="secondary">Hardware</OuiText>
          </OuiStack>
          <OuiStack gap="xs" align="center">
            <OuiText size="2xl" weight="bold" color="primary">Personal</OuiText>
            <OuiText size="sm" color="secondary">Support</OuiText>
          </OuiStack>
        </OuiFlex>
      </OuiStack>
    </OuiContainer>

    <!-- Features Section -->
    <OuiContainer py="6xl" size="7xl">
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
            Container-based deployments with GitHub integration, custom domains, and powerful monitoring tools.
          </OuiText>
        </OuiStack>

        <OuiGrid cols="1" cols-md="2" cols-lg="3" gap="xl" class="w-full">
          <!-- GitHub Integration -->
          <OuiCard
            variant="default"
            hoverable
            class="group cursor-pointer transition-all duration-200 hover:-translate-y-1 hover:shadow-xl"
          >
            <OuiCardBody>
              <OuiStack gap="md">
                <OuiBox class="w-12 h-12 bg-accent-primary/20 rounded-lg flex items-center justify-center">
                  <RocketLaunchIcon class="h-6 w-6 text-accent-primary" />
                </OuiBox>
                <OuiStack gap="sm">
                  <OuiText as="h3" size="lg" weight="semibold" color="primary">
                    GitHub Integration
                  </OuiText>
                  <OuiText size="sm" color="secondary">
                    Deploy directly from GitHub repositories. Automatic builds on push with support for Docker and Docker Compose.
                  </OuiText>
                </OuiStack>
              </OuiStack>
            </OuiCardBody>
          </OuiCard>

          <!-- Custom Domains -->
          <OuiCard
            variant="default"
            hoverable
            class="group cursor-pointer transition-all duration-200 hover:-translate-y-1 hover:shadow-xl"
          >
            <OuiCardBody>
              <OuiStack gap="md">
                <OuiBox class="w-12 h-12 bg-accent-success/20 rounded-lg flex items-center justify-center">
                  <ShieldCheckIcon class="h-6 w-6 text-accent-success" />
                </OuiBox>
                <OuiStack gap="sm">
                  <OuiText as="h3" size="lg" weight="semibold" color="primary">
                    Custom Domains
                  </OuiText>
                  <OuiText size="sm" color="secondary">
                    Configure custom domains with automatic SSL certificates. Set up routing rules for multiple paths and domains.
                  </OuiText>
                </OuiStack>
              </OuiStack>
            </OuiCardBody>
          </OuiCard>

          <!-- Container Deployments -->
          <OuiCard
            variant="default"
            hoverable
            class="group cursor-pointer transition-all duration-200 hover:-translate-y-1 hover:shadow-xl"
          >
            <OuiCardBody>
              <OuiStack gap="md">
                <OuiBox class="w-12 h-12 bg-accent-secondary/20 rounded-lg flex items-center justify-center">
                  <ServerIcon class="h-6 w-6 text-accent-secondary" />
                </OuiBox>
                <OuiStack gap="sm">
                  <OuiText as="h3" size="lg" weight="semibold" color="primary">
                    Container Deployments
                  </OuiText>
                  <OuiText size="sm" color="secondary">
                    Deploy single containers or multi-container applications with Docker Compose. Full control over your application stack.
                  </OuiText>
                </OuiStack>
              </OuiStack>
            </OuiCardBody>
          </OuiCard>

          <!-- Environment Variables -->
          <OuiCard
            variant="default"
            hoverable
            class="group cursor-pointer transition-all duration-200 hover:-translate-y-1 hover:shadow-xl"
          >
            <OuiCardBody>
              <OuiStack gap="md">
                <OuiBox class="w-12 h-12 bg-accent-info/20 rounded-lg flex items-center justify-center">
                  <CircleStackIcon class="h-6 w-6 text-accent-info" />
                </OuiBox>
                <OuiStack gap="sm">
                  <OuiText as="h3" size="lg" weight="semibold" color="primary">
                    Environment Variables
                  </OuiText>
                  <OuiText size="sm" color="secondary">
                    Manage environment variables per deployment. Secure secrets management with easy configuration through the dashboard.
                  </OuiText>
                </OuiStack>
              </OuiStack>
            </OuiCardBody>
          </OuiCard>

          <!-- Real-Time Monitoring -->
          <OuiCard
            variant="default"
            hoverable
            class="group cursor-pointer transition-all duration-200 hover:-translate-y-1 hover:shadow-xl"
          >
            <OuiCardBody>
              <OuiStack gap="md">
                <OuiBox class="w-12 h-12 bg-accent-warning/20 rounded-lg flex items-center justify-center">
                  <ChartBarIcon class="h-6 w-6 text-accent-warning" />
                </OuiBox>
                <OuiStack gap="sm">
                  <OuiText as="h3" size="lg" weight="semibold" color="primary">
                    Real-Time Monitoring
                  </OuiText>
                  <OuiText size="sm" color="secondary">
                    Track vCPU, memory, bandwidth, and storage usage in real-time. View build logs, application logs, and performance metrics.
                  </OuiText>
                </OuiStack>
              </OuiStack>
            </OuiCardBody>
          </OuiCard>

          <!-- File Management -->
          <OuiCard
            variant="default"
            hoverable
            class="group cursor-pointer transition-all duration-200 hover:-translate-y-1 hover:shadow-xl"
          >
            <OuiCardBody>
              <OuiStack gap="md">
                <OuiBox class="w-12 h-12 bg-accent-danger/20 rounded-lg flex items-center justify-center">
                  <FolderIcon class="h-6 w-6 text-accent-danger" />
                </OuiBox>
                <OuiStack gap="sm">
                  <OuiText as="h3" size="lg" weight="semibold" color="primary">
                    File Management
                  </OuiText>
                  <OuiText size="sm" color="secondary">
                    Upload, edit, and manage files directly through the dashboard. Terminal access for advanced configuration and debugging.
                  </OuiText>
                </OuiStack>
              </OuiStack>
            </OuiCardBody>
          </OuiCard>
        </OuiGrid>
      </OuiStack>
    </OuiContainer>

    <!-- Infrastructure Section -->
    <OuiContainer as="section" py="6xl" size="7xl">
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
            Built on premium hardware with redundancy and reliability at its core.
          </OuiText>
        </OuiStack>

        <OuiGrid cols="1" cols-md="2" cols-lg="3" gap="xl" class="w-full">
          <!-- Premium Hardware -->
          <OuiCard variant="default" class="h-full">
            <OuiCardBody>
              <OuiStack gap="md">
                <OuiBox class="w-12 h-12 bg-accent-primary/20 rounded-lg flex items-center justify-center">
                  <ServerIcon class="h-6 w-6 text-accent-primary" />
                </OuiBox>
                <OuiStack gap="sm">
                  <OuiText as="h3" size="lg" weight="semibold" color="primary">
                    Premium Hardware
                  </OuiText>
                  <OuiText size="sm" color="secondary">
                    Powered by AMD Ryzen 7 7900X processors with DDR5 RAM and NVMe SSDs for exceptional performance.
                  </OuiText>
                </OuiStack>
              </OuiStack>
            </OuiCardBody>
          </OuiCard>

          <!-- High Availability -->
          <OuiCard variant="default" class="h-full">
            <OuiCardBody>
              <OuiStack gap="md">
                <OuiBox class="w-12 h-12 bg-accent-success/20 rounded-lg flex items-center justify-center">
                  <ShieldCheckIcon class="h-6 w-6 text-accent-success" />
                </OuiBox>
                <OuiStack gap="sm">
                  <OuiText as="h3" size="lg" weight="semibold" color="primary">
                    High Availability
                  </OuiText>
                  <OuiText size="sm" color="secondary">
                    Automatic failover to our fallback node ensures your services stay online even during main node outages.
                  </OuiText>
                </OuiStack>
              </OuiStack>
            </OuiCardBody>
          </OuiCard>

          <!-- Free Support -->
          <OuiCard variant="default" class="h-full">
            <OuiCardBody>
              <OuiStack gap="md">
                <OuiBox class="w-12 h-12 bg-accent-secondary/20 rounded-lg flex items-center justify-center">
                  <ChatBubbleLeftRightIcon class="h-6 w-6 text-accent-secondary" />
                </OuiBox>
                <OuiStack gap="sm">
                  <OuiText as="h3" size="lg" weight="semibold" color="primary">
                    Free Support
                  </OuiText>
                  <OuiText size="sm" color="secondary">
                    Get help when you need it. Our support team is available to assist with deployment, configuration, and troubleshooting.
                  </OuiText>
                </OuiStack>
              </OuiStack>
            </OuiCardBody>
          </OuiCard>

          <!-- Security -->
          <OuiCard variant="default" class="h-full">
            <OuiCardBody>
              <OuiStack gap="md">
                <OuiBox class="w-12 h-12 bg-accent-danger/20 rounded-lg flex items-center justify-center">
                  <ShieldCheckIcon class="h-6 w-6 text-accent-danger" />
                </OuiBox>
                <OuiStack gap="sm">
                  <OuiText as="h3" size="lg" weight="semibold" color="primary">
                    Security
                  </OuiText>
                  <OuiText size="sm" color="secondary">
                    SSL certificates, automated backups, and comprehensive audit logs to keep your data secure.
                  </OuiText>
                </OuiStack>
              </OuiStack>
            </OuiCardBody>
          </OuiCard>

          <!-- DNS Management -->
          <OuiCard variant="default" class="h-full">
            <OuiCardBody>
              <OuiStack gap="md">
                <OuiBox class="w-12 h-12 bg-accent-info/20 rounded-lg flex items-center justify-center">
                  <GlobeAltIcon class="h-6 w-6 text-accent-info" />
                </OuiBox>
                <OuiStack gap="sm">
                  <OuiText as="h3" size="lg" weight="semibold" color="primary">
                    DNS Management
                  </OuiText>
                  <OuiText size="sm" color="secondary">
                    Custom domain support with automatic DNS configuration. Verify domains and set up routing rules for your deployments.
                  </OuiText>
                </OuiStack>
              </OuiStack>
            </OuiCardBody>
          </OuiCard>

          <!-- Real-Time Monitoring -->
          <OuiCard variant="default" class="h-full">
            <OuiCardBody>
              <OuiStack gap="md">
                <OuiBox class="w-12 h-12 bg-accent-warning/20 rounded-lg flex items-center justify-center">
                  <ChartBarIcon class="h-6 w-6 text-accent-warning" />
                </OuiBox>
                <OuiStack gap="sm">
                  <OuiText as="h3" size="lg" weight="semibold" color="primary">
                    Real-Time Monitoring
                  </OuiText>
                  <OuiText size="sm" color="secondary">
                    Track vCPU, memory, bandwidth, and storage usage in real-time. View build logs, application logs, and performance metrics.
                  </OuiText>
                </OuiStack>
              </OuiStack>
            </OuiCardBody>
          </OuiCard>
        </OuiGrid>
      </OuiStack>
    </OuiContainer>

    <!-- Pricing Section -->
    <OuiContainer as="section" py="6xl" size="7xl">
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
            Pay only for what you use. No fixed plans, no hidden fees. Free support included.
          </OuiText>
          <OuiText size="sm" color="secondary" class="opacity-60 italic">
            Students, open-source projects, non-profits, and early-stage startups may qualify for reduced pricing. Contact us to discuss.
          </OuiText>
        </OuiStack>

        <OuiGrid cols="1" cols-md="3" gap="xl" class="w-full max-w-5xl">
          <!-- Small App Example -->
          <OuiCard
            variant="default"
            hoverable
            class="text-center group hover:-translate-y-1 transition-all duration-200 h-full flex flex-col"
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
                  <OuiText size="3xl" weight="bold" color="primary">~$5</OuiText>
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
                    <OuiText size="sm" color="secondary">Automatic failover to fallback node</OuiText>
                  </OuiFlex>
                </OuiStack>
              </OuiStack>

              <OuiButton
                variant="outline"
                size="md"
                class="w-full mt-auto"
                @click="navigateTo('/dashboard')"
              >
                Get Started
              </OuiButton>
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
                    <OuiText size="sm" color="secondary">~25 GB storage</OuiText>
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
                    <OuiText size="sm" color="secondary">Automatic failover to fallback node</OuiText>
                  </OuiFlex>
                </OuiStack>
              </OuiStack>

              <OuiButton
                color="primary"
                size="md"
                class="w-full mt-auto shadow-lg shadow-accent-primary/25"
                @click="navigateTo('/dashboard')"
              >
                Get Started
              </OuiButton>
            </OuiCardBody>
          </OuiCard>

          <!-- Large App Example -->
          <OuiCard
            variant="default"
            hoverable
            class="text-center group hover:-translate-y-1 transition-all duration-200 h-full flex flex-col"
          >
            <OuiCardBody class="flex flex-col h-full">
              <OuiStack gap="lg" class="flex-1">
                <OuiStack gap="md">
                  <OuiText size="lg" weight="semibold" color="primary"
                    >Large App</OuiText
                  >
                  <OuiText size="sm" color="secondary"
                    >For large organizations</OuiText
                  >
                </OuiStack>

                <OuiStack gap="xs" align="center">
                  <OuiText size="3xl" weight="bold" color="primary"
                    >~$44</OuiText
                  >
                  <OuiText size="sm" color="secondary">per month</OuiText>
                </OuiStack>

                <OuiStack gap="sm" align="start" class="text-left flex-1">
                  <OuiFlex align="center" gap="sm">
                    <CheckIcon class="h-4 w-4 text-accent-success shrink-0" />
                    <OuiText size="sm" color="secondary"
                      >8 GB RAM running 24/7</OuiText
                    >
                  </OuiFlex>
                  <OuiFlex align="center" gap="sm">
                    <CheckIcon class="h-4 w-4 text-accent-success shrink-0" />
                    <OuiText size="sm" color="secondary"
                      >2 vCPU cores running 24/7</OuiText
                    >
                  </OuiFlex>
                  <OuiFlex align="center" gap="sm">
                    <CheckIcon class="h-4 w-4 text-accent-success shrink-0" />
                    <OuiText size="sm" color="secondary"
                      >~200 GB bandwidth/month</OuiText
                    >
                  </OuiFlex>
                  <OuiFlex align="center" gap="sm">
                    <CheckIcon class="h-4 w-4 text-accent-success shrink-0" />
                    <OuiText size="sm" color="secondary">~100 GB storage</OuiText>
                  </OuiFlex>
                  <OuiFlex align="center" gap="sm">
                    <CheckIcon class="h-4 w-4 text-accent-success shrink-0" />
                    <OuiText size="sm" color="secondary"
                      >Scale resources as needed</OuiText
                    >
                  </OuiFlex>
                  <OuiFlex align="center" gap="sm">
                    <CheckIcon class="h-4 w-4 text-accent-success shrink-0" />
                    <OuiText size="sm" color="secondary"
                      >Premium hardware</OuiText
                    >
                  </OuiFlex>
                  <OuiFlex align="center" gap="sm">
                    <CheckIcon class="h-4 w-4 text-accent-success shrink-0" />
                    <OuiText size="sm" color="secondary">Free support</OuiText>
                  </OuiFlex>
                  <OuiFlex align="center" gap="sm">
                    <CheckIcon class="h-4 w-4 text-accent-success shrink-0" />
                    <OuiText size="sm" color="secondary">Automatic failover to fallback node</OuiText>
                  </OuiFlex>
                </OuiStack>
              </OuiStack>

              <OuiButton variant="outline" size="md" class="w-full mt-auto">
                Get Started
              </OuiButton>
            </OuiCardBody>
          </OuiCard>
        </OuiGrid>
        
        <OuiText size="sm" color="secondary" class="opacity-75 text-center max-w-2xl">
          * Example costs based on 24/7 usage. You only pay for actual runtime - if your app runs part-time, you'll pay less. Use the calculator below to estimate your exact costs.
        </OuiText>
      </OuiStack>
    </OuiContainer>

    <!-- Pricing Calculator Section -->
    <PricingCalculator />

    <!-- CTA Section -->
    <OuiContainer as="section" size="7xl" py="6xl">
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
            Join our growing community of developers building and deploying containerized applications.
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
  PlayIcon,
  ServerIcon,
  CircleStackIcon,
  ChartBarIcon,
  ShieldCheckIcon,
  CheckIcon,
  ChatBubbleLeftRightIcon,
  FolderIcon,
  GlobeAltIcon,
} from "@heroicons/vue/24/outline";
import PricingCalculator from "~/components/pricing/PricingCalculator.vue";

// Page meta - no auth required for homepage
definePageMeta({
  layout: false, // Use custom layout for homepage
});

// Scroll state for header effect using VueUse
const { y: scrollY } = useWindowScroll();
const isScrolled = computed(() => scrollY.value > 10);

// SEO meta
useHead({
  title: "Obiente Cloud - Container Deployment Platform",
  meta: [
    {
      name: "description",
      content:
        "Deploy containerized applications with GitHub integration. Pay-as-you-go pricing with transparent resource-based costs. Docker and Docker Compose support.",
    },
    {
      name: "keywords",
      content:
        "container hosting, docker deployment, docker compose, github deployment, cloud platform, pay as you go, vercel alternative",
    },
    {
      property: "og:title",
      content: "Obiente Cloud - Container Deployment Platform",
    },
    {
      property: "og:description",
      content:
        "Deploy containerized applications with GitHub integration. Pay-as-you-go pricing with transparent resource-based costs.",
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
</style>
