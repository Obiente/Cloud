<template>
  <!-- Setup mock WebSocket for terminal preview and metrics -->
  <PreviewProviders mode="deployment" :deployment-mock="exampleWebDeployment" headless />
  <PreviewProviders mode="vps" :vps-mock="exampleVPS" headless />

  <OuiContainer size="full" minH="screen">
    <!-- Hero Section -->
    <OuiBox as="div" position="fixed" class="top-5 left-0 right-0 z-30" :pt="{ sm: 'xs', md: 'lg' }"
      :px="{ sm: 'sm', md: 'md' }">
      <OuiContainer as="nav" :size="isScrolled ? '5xl' : '7xl'" w="full" mx="auto"
        class="flex items-center rounded-xl justify-between transform-gpu transition-all duration-500 ease-out" :class="isScrolled
          ? 'bg-background/70 backdrop-blur-sm shadow-sm border border-muted'
          : 'border-transparent'
          " py="xs" px='xs'>
        <!-- Navigation -->

        <OuiFlex align="center" gap="md" grow>
          <OuiFlex align="center" gap="sm">
            <ObienteLogo size="md" />
            <OuiText size="xl" weight="bold" color="primary" class="truncate text-lg md:text-xl">Obiente Cloud</OuiText>
          </OuiFlex>
        </OuiFlex>

        <OuiFloatingPanel v-model="previewPanelOpen" :title="previewTitle" :resizable="true"
          :default-position="previewDefaultPosition"
          content-class="!min-w-[95vw] !w-[95vw] !max-w-[1800px] !max-h-[92vh] !min-h-[80vh]" body-class="p-0"
          @close="closePreview">
          <div v-if="activePreview === 'web'" :key="'web-' + previewKey" class="p-2 md:p-4 h-full">
            <PreviewProviders :component="DeploymentPage" mode="deployment" :deployment-mock="exampleWebDeployment" />
          </div>

          <div v-else-if="activePreview === 'game'" :key="'game-' + previewKey" class="p-2 md:p-4 h-full">
            <PreviewProviders :component="GameServerPage" mode="game" :game-server-mock="exampleGameServer"
              :game-usage-mock="exampleGameUsageData" :game-metrics-mock="exampleGameLiveMetrics" />
          </div>

          <div v-else-if="activePreview === 'vps'" :key="'vps-' + previewKey" class="p-2 md:p-4 h-full">
            <PreviewProviders :component="VpsPage" mode="vps" :vps-mock="exampleVPS"
              :vps-metrics-mock="exampleVpsLiveMetrics" />
          </div>
        </OuiFloatingPanel>

        <OuiFlex align="center" gap="md" :shrink="false" class="hidden sm:flex">
          <OuiButton variant="ghost" size="sm">Features</OuiButton>
          <OuiButton variant="ghost" size="sm">Pricing</OuiButton>
          <OuiButton variant="ghost" size="sm" @click="navigateTo('/docs')">Docs</OuiButton>
          <OuiButton variant="outline" size="sm" @click="handleSignUp">
            Sign Up
          </OuiButton>
          <OuiButton variant="outline" size="sm" @click="navigateTo('/dashboard')">Sign In</OuiButton>
        </OuiFlex>
      </OuiContainer>
    </OuiBox>
    <OuiStack as="div" w="full" gap="4xl">
      <OuiContainer size="full" w="full" position="relative" px="md" py="4xl" maxW="7xl" mx="auto"
        class="min-h-screen flex items-center">
        <!-- Hero Content -->
        <OuiStack gap="2xl" align="center" py="3xl" w="full" class="relative z-10">
          <!-- Hero background - dual color dots with spotlights -->
          <OuiBox position="absolute" overflow="hidden" class="inset-0 -z-10 pointer-events-none">
            <!-- Primary dots with spotlight mask -->
            <OuiBox position="absolute" class="inset-0 hero-dots-primary" style="
              --mask: radial-gradient(ellipse 80% 70% at 20% 30%, black 0%, transparent 70%);
              -webkit-mask-image: var(--mask);
              mask-image: var(--mask);
            "></OuiBox>
            <!-- Secondary dots with spotlight mask -->
            <OuiBox position="absolute" class="inset-0 hero-dots-secondary" style="
              --mask: radial-gradient(ellipse 70% 80% at 80% 65%, black 0%, transparent 70%);
              -webkit-mask-image: var(--mask);
              mask-image: var(--mask);
            "></OuiBox>
            <!-- Fade at bottom -->
          </OuiBox>

          <OuiFlex gap="2xl" direction="row" align="center" justify="center" maxW="7xl" w="full" mx="auto"
            class="text-left">
            <OuiStack gap="lg" class="flex-[3] text-left">
              <OuiFlex gap="sm" align="center">
                <OuiText as="span" size="lg" weight="semibold" color="accent" class="text-sm md:text-lg">Obiente Cloud
                </OuiText>
              </OuiFlex>

              <OuiText as="h1" size="6xl" weight="bold" color="primary"
                class="text-4xl md:text-6xl lg:text-7xl leading-tight tracking-tight">
                Deploy Anything.
                <br />
                Pay for What You Use.
              </OuiText>

              <OuiText size="lg" color="secondary" maxW="2xl" mt="md"
                class="text-base md:text-lg lg:text-xl leading-relaxed">
                Deploy any containerized application from GitHub, host game servers, and launch VPS instances with
                transparent pay-as-you-go pricing. Pay only for CPU, memory, storage, and bandwidth you actually use.
              </OuiText>

              <OuiFlex gap="md" wrap="wrap" mt="xl">
                <OuiButton size="lg" color="primary" @click="navigateTo('/dashboard')">
                  <RocketLaunchIcon class="h-5 w-5" />
                  Get Started
                </OuiButton>
                <OuiButton variant="outline" size="lg" @click="navigateTo('/docs')">
                  <ChatBubbleLeftRightIcon class="h-5 w-5" />
                  <OuiText class="hidden sm:inline" size="sm">Documentation</OuiText>
                  <OuiText class="sm:hidden" size="sm">Docs</OuiText>
                </OuiButton>
              </OuiFlex>
            </OuiStack>

            <!-- Right side visual element -->
            <OuiStack gap="lg" align="center" justify="center" class="flex-[2] hidden lg:flex">
              <OuiCard variant="default" w="full" class="bg-surface-muted/30 border-border-muted/40">
                <OuiCardBody p="lg">
                  <OuiStack gap="md">
                    <OuiFlex gap="sm">
                      <OuiBox w="3" h="3" rounded="full" class="bg-accent-danger/60"></OuiBox>
                      <OuiBox w="3" h="3" rounded="full" class="bg-accent-warning/60"></OuiBox>
                      <OuiBox w="3" h="3" rounded="full" class="bg-accent-success/60"></OuiBox>
                    </OuiFlex>
                    <OuiStack gap="xs" class="font-mono text-sm">
                      <OuiFlex gap="sm">
                        <OuiText as="span" size="sm" color="accent">$</OuiText>
                        <OuiText as="span" size="sm" color="primary">git push origin main</OuiText>
                      </OuiFlex>
                      <OuiText as="div" size="sm" color="success" class="pl-4">→ Deploying...</OuiText>
                      <OuiText as="div" size="sm" color="success" class="pl-4">→ Building Docker image</OuiText>
                      <OuiText as="div" size="sm" color="success" class="pl-4">→ Deployed in 3m 42s</OuiText>
                      <OuiText as="div" size="sm" color="primary" class="pl-4 mt-2">✓ Live at app.my.obiente.cloud
                      </OuiText>
                    </OuiStack>
                  </OuiStack>
                </OuiCardBody>
              </OuiCard>
            </OuiStack>
          </OuiFlex>

        </OuiStack>
      </OuiContainer>

      <!-- Features Section -->
      <OuiContainer size="full" w="full" maxW="7xl" mx="auto" px="md" py="4xl">
        <OuiStack gap="2xl" align="center" w="full" maxW="7xl" mx="auto">
          <OuiStack gap="lg" maxW="3xl" align="center" class="text-center">
            <OuiText as="h2" size="3xl" weight="bold" color="primary" class="md:text-4xl">
              Complete Cloud Platform Features
            </OuiText>
            <OuiText color="secondary" class="md:text-base">
              Everything you need to deploy, monitor, and scale your applications. From simple Docker containers to
              complex multi-service deployments, game servers, and full VPS instances.
            </OuiText>
          </OuiStack>

          <OuiGrid :cols="{ sm: 1, md: 2, lg: 3 }" gap="xl" w="full">
            <!-- GitHub Integration -->
            <OuiCard variant="default" h="full">
              <OuiCardBody p="lg">
                <OuiFlex gap="md" align="start">
                  <OuiBox w="10" h="10" rounded="lg" :shrink="false" class="bg-accent-primary/10">
                    <OuiFlex align="center" justify="center" h="full">
                      <RocketLaunchIcon class="h-5 w-5 text-accent-primary" />
                    </OuiFlex>
                  </OuiBox>
                  <OuiStack gap="xs" grow>
                    <OuiText as="h3" size="md" weight="semibold" color="primary">
                      GitHub Integration & CI/CD
                    </OuiText>
                    <OuiText size="sm" color="secondary">
                      Connect your GitHub repository and deploy automatically on every push. Supports Docker, Docker
                      Compose, and custom build strategies.
                    </OuiText>
                  </OuiStack>
                </OuiFlex>
              </OuiCardBody>
            </OuiCard>

            <!-- Custom Domains -->
            <OuiCard variant="default" h="full">
              <OuiCardBody p="lg">
                <OuiFlex gap="sm" align="start">
                  <OuiBox w="10" h="10" rounded="lg" :shrink="false" class="bg-accent-success/10">
                    <OuiFlex align="center" justify="center" h="full">
                      <ShieldCheckIcon class="h-5 w-5 text-accent-success" />
                    </OuiFlex>
                  </OuiBox>
                  <OuiStack gap="xs" grow>
                    <OuiText as="h3" size="md" weight="semibold" color="primary">
                      Custom Domains & SSL
                    </OuiText>
                    <OuiText size="sm" color="secondary">
                      Add unlimited custom domains with automatic Let's Encrypt SSL certificates. Configure advanced
                      routing rules and path-based routing.
                    </OuiText>
                  </OuiStack>
                </OuiFlex>
              </OuiCardBody>
            </OuiCard>

            <!-- Container Deployments -->
            <OuiCard variant="default" h="full">
              <OuiCardBody p="lg">
                <OuiFlex gap="sm" align="start">
                  <OuiBox w="10" h="10" rounded="lg" :shrink="false" class="bg-accent-secondary/10">
                    <OuiFlex align="center" justify="center" h="full">
                      <ServerIcon class="h-5 w-5 text-accent-secondary" />
                    </OuiFlex>
                  </OuiBox>
                  <OuiStack gap="xs" grow>
                    <OuiText as="h3" size="md" weight="semibold" color="primary">
                      Docker & Docker Compose
                    </OuiText>
                    <OuiText size="sm" color="secondary">
                      Deploy single containers or complex multi-service applications with Docker Compose. Manage
                      multiple
                      services, volumes, and networks from one dashboard.
                    </OuiText>
                  </OuiStack>
                </OuiFlex>
              </OuiCardBody>
            </OuiCard>

            <!-- Environment Variables -->
            <OuiCard variant="default" h="full">
              <OuiCardBody p="lg">
                <OuiFlex gap="sm" align="start">
                  <OuiBox w="10" h="10" rounded="lg" :shrink="false" class="bg-accent-info/10">
                    <OuiFlex align="center" justify="center" h="full">
                      <CircleStackIcon class="h-5 w-5 text-accent-info" />
                    </OuiFlex>
                  </OuiBox>
                  <OuiStack gap="xs" grow>
                    <OuiText as="h3" size="md" weight="semibold" color="primary">
                      Environment Variables
                    </OuiText>
                    <OuiText size="sm" color="secondary">
                      Manage environment variables per deployment. Secure secrets management with easy configuration
                      through the dashboard.
                    </OuiText>
                  </OuiStack>
                </OuiFlex>
              </OuiCardBody>
            </OuiCard>

            <!-- Real-Time Monitoring -->
            <OuiCard variant="default" h="full">
              <OuiCardBody p="lg">
                <OuiFlex gap="sm" align="start">
                  <OuiBox w="10" h="10" rounded="lg" :shrink="false" class="bg-accent-warning/10">
                    <OuiFlex align="center" justify="center" h="full">
                      <ChartBarIcon class="h-5 w-5 text-accent-warning" />
                    </OuiFlex>
                  </OuiBox>
                  <OuiStack gap="xs" grow>
                    <OuiText as="h3" size="md" weight="semibold" color="primary">
                      Real-Time Monitoring & Logs
                    </OuiText>
                    <OuiText size="sm" color="secondary">
                      Live metrics dashboard with CPU, memory, network, and disk usage. Real-time log streaming, build
                      logs, and container logs.
                    </OuiText>
                  </OuiStack>
                </OuiFlex>
              </OuiCardBody>
            </OuiCard>

            <!-- File Management -->
            <OuiCard variant="default" h="full">
              <OuiCardBody p="lg">
                <OuiFlex gap="sm" align="start">
                  <OuiBox w="10" h="10" rounded="lg" :shrink="false" class="bg-accent-danger/10">
                    <OuiFlex align="center" justify="center" h="full">
                      <FolderIcon class="h-5 w-5 text-accent-danger" />
                    </OuiFlex>
                  </OuiBox>
                  <OuiStack gap="xs" grow>
                    <OuiText as="h3" size="md" weight="semibold" color="primary">
                      File Management & Terminal
                    </OuiText>
                    <OuiText size="sm" color="secondary">
                      Web-based file browser with upload, edit, and delete capabilities. Built-in terminal access for
                      SSH-like experience.
                    </OuiText>
                  </OuiStack>
                </OuiFlex>
              </OuiCardBody>
            </OuiCard>

            <!-- Pay-as-You-Go Pricing -->
            <OuiCard variant="default" h="full">
              <OuiCardBody p="lg">
                <OuiFlex gap="sm" align="start">
                  <OuiBox w="10" h="10" rounded="lg" :shrink="false" class="bg-accent-success/10">
                    <OuiFlex align="center" justify="center" h="full">
                      <CircleStackIcon class="h-5 w-5 text-accent-success" />
                    </OuiFlex>
                  </OuiBox>
                  <OuiStack gap="xs" grow>
                    <OuiText as="h3" size="md" weight="semibold" color="primary">
                      Pay-as-You-Go Pricing
                    </OuiText>
                    <OuiText size="sm" color="secondary">
                      Only pay for resources you actually use. Perfect for game servers and VPSs where you often overpay
                      for idle time.
                    </OuiText>
                  </OuiStack>
                </OuiFlex>
              </OuiCardBody>
            </OuiCard>

            <!-- Game Server Hosting -->
            <OuiCard variant="default" h="full">
              <OuiCardBody p="lg">
                <OuiFlex gap="sm" align="start">
                  <OuiBox w="10" h="10" rounded="lg" :shrink="false" class="bg-accent-primary/10">
                    <OuiFlex align="center" justify="center" h="full">
                      <ServerIcon class="h-5 w-5 text-accent-primary" />
                    </OuiFlex>
                  </OuiBox>
                  <OuiStack gap="xs" grow>
                    <OuiText as="h3" size="md" weight="semibold" color="primary">
                      Game Server Hosting
                    </OuiText>
                    <OuiText size="sm" color="secondary">
                      Deploy Minecraft and other game servers with Docker. Built-in Minecraft server management. Pay
                      only
                      when running - save 50%+ vs traditional hosting.
                    </OuiText>
                  </OuiStack>
                </OuiFlex>
              </OuiCardBody>
            </OuiCard>

            <!-- VPS Instances -->
            <OuiCard variant="default" h="full">
              <OuiCardBody p="lg">
                <OuiFlex gap="sm" align="start">
                  <OuiBox w="10" h="10" rounded="lg" :shrink="false" class="bg-accent-info/10">
                    <OuiFlex align="center" justify="center" h="full">
                      <ServerIcon class="h-5 w-5 text-accent-info" />
                    </OuiFlex>
                  </OuiBox>
                  <OuiStack gap="xs" grow>
                    <OuiText as="h3" size="md" weight="semibold" color="primary">
                      VPS Instances with Root Access
                    </OuiText>
                    <OuiText size="sm" color="secondary">
                      Full root access Linux VPS instances. Web console, SSH access, firewall management, and snapshots.
                      Pay for actual usage, not idle time.
                    </OuiText>
                  </OuiStack>
                </OuiFlex>
              </OuiCardBody>
            </OuiCard>
          </OuiGrid>
        </OuiStack>
      </OuiContainer>

      <!-- Dashboard Showcase -->
      <OuiContainer as="section" size="full" w="full" maxW="7xl" mx="auto" px="md" py="4xl">
        <OuiStack gap="2xl" align="center" w="full" maxW="7xl" mx="auto" mb="3xl">
          <OuiStack gap="lg" align="center" maxW="3xl" class="text-center">
            <OuiText as="h2" size="3xl" weight="bold" color="primary" class="md:text-4xl">
              Interactive Dashboard Preview
            </OuiText>
            <OuiText color="secondary" class="md:text-base">
              Experience our cloud platform dashboard with live previews. Click any card below to explore real-time
              metrics, deployment management, game server controls, and VPS terminal access.
            </OuiText>
          </OuiStack>
        </OuiStack>

        <OuiStack gap="2xl" w="full" maxW="7xl" mx="auto">
          <!-- Web App Section -->
          <OuiBox>
            <OuiFlex gap="2xl" direction="row" align="start" w="full">
              <OuiBox
                class="flex-1 min-w-0 cursor-pointer focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent-primary/60 rounded-xl"
                role="button" tabindex="0" @click="openPreview('web')" @keydown.enter.prevent="openPreview('web')"
                @keydown.space.prevent="openPreview('web')">
                <OuiStack gap="md">
                  <DeploymentCard :deployment="exampleWebDeployment" :loading="false">
                    <template #actions>
                      <OuiFlex gap="xs">
                        <OuiBadge variant="secondary" size="sm">{{ exampleWebDeployment.containersRunning }}/{{
                          exampleWebDeployment.containersTotal }} running</OuiBadge>
                        <OuiBadge variant="secondary" size="sm">web</OuiBadge>
                      </OuiFlex>
                    </template>
                  </DeploymentCard>
                  <LiveMetrics :isStreaming="homepageMetricsStreaming" :latestMetric="homepageLatestMetric"
                    :currentCpuUsage="homepageCurrentCpuUsage" :currentMemoryUsage="homepageCurrentMemoryUsage"
                    :currentNetworkRx="homepageCurrentNetworkRx" :currentNetworkTx="homepageCurrentNetworkTx" />
                </OuiStack>
              </OuiBox>
              <OuiStack gap="lg" grow class="flex-1">
                <OuiText size="2xl" weight="bold" color="primary">Container Deployment</OuiText>
                <OuiText color="secondary">Deploy any containerized application from GitHub - web apps, APIs, background
                  workers,
                  databases, or anything that runs in Docker. Automatic builds, custom domains, and SSL certificates
                  included.
                </OuiText>

                <OuiStack gap="sm" mt="md">
                  <OuiFlex align="center" gap="sm">
                    <CheckIcon class="h-4 w-4 text-accent-success shrink-0" />
                    <OuiText size="sm" color="secondary">Automatic GitHub builds</OuiText>
                  </OuiFlex>
                  <OuiFlex align="center" gap="sm">
                    <CheckIcon class="h-4 w-4 text-accent-success shrink-0" />
                    <OuiText size="sm" color="secondary">Custom domains & SSL</OuiText>
                  </OuiFlex>
                  <OuiFlex align="center" gap="sm">
                    <CheckIcon class="h-4 w-4 text-accent-success shrink-0" />
                    <OuiText size="sm" color="secondary">Real-time monitoring</OuiText>
                  </OuiFlex>
                  <OuiFlex align="center" gap="sm">
                    <CheckIcon class="h-4 w-4 text-accent-success shrink-0" />
                    <OuiText size="sm" color="secondary">Docker Compose</OuiText>
                  </OuiFlex>
                  <OuiFlex align="center" gap="sm">
                    <CheckIcon class="h-4 w-4 text-accent-success shrink-0" />
                    <OuiText size="sm" color="secondary">Environment variables</OuiText>
                  </OuiFlex>
                  <OuiFlex align="center" gap="sm">
                    <CheckIcon class="h-4 w-4 text-accent-success shrink-0" />
                    <OuiText size="sm" color="secondary">Zero-downtime deploys</OuiText>
                  </OuiFlex>
                </OuiStack>
              </OuiStack>
            </OuiFlex>
          </OuiBox>

          <!-- Game Server Section -->
          <OuiBox>
            <OuiFlex gap="2xl" direction="row-reverse" align="start" w="full">
              <OuiBox rounded="xl"
                class="flex-1 cursor-pointer focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent-primary/60"
                role="button" tabindex="0" @click="openPreview('game')" @keydown.enter.prevent="openPreview('game')"
                @keydown.space.prevent="openPreview('game')">
                <GameServerCard :gameServer="exampleGameServerForCard" :loading="false">
                  <template #actions>
                    <OuiFlex gap="xs">
                      <OuiBadge variant="secondary" size="sm">minecraft</OuiBadge>
                      <OuiBadge variant="secondary" size="sm">port: 25565</OuiBadge>
                    </OuiFlex>
                  </template>
                </GameServerCard>
              </OuiBox>
              <OuiStack gap="lg" grow class="flex-1">
                <OuiText size="2xl" weight="bold" color="primary">Game Server Hosting</OuiText>
                <OuiText color="secondary">Host game servers with pay-as-you-go pricing. Only pay when your server is
                  running.
                  Perfect for Minecraft, Valheim, and other multiplayer games.</OuiText>

                <OuiStack gap="sm" mt="md">
                  <OuiFlex align="center" gap="sm">
                    <CheckIcon class="h-4 w-4 text-accent-success shrink-0" />
                    <OuiText size="sm" color="secondary">Pay-as-you-go pricing</OuiText>
                  </OuiFlex>
                  <OuiFlex align="center" gap="sm">
                    <CheckIcon class="h-4 w-4 text-accent-success shrink-0" />
                    <OuiText size="sm" color="secondary">Low-cost when idle</OuiText>
                  </OuiFlex>
                  <OuiFlex align="center" gap="sm">
                    <CheckIcon class="h-4 w-4 text-accent-success shrink-0" />
                    <OuiText size="sm" color="secondary">Backups & snapshots</OuiText>
                  </OuiFlex>
                  <OuiFlex align="center" gap="sm">
                    <CheckIcon class="h-4 w-4 text-accent-success shrink-0" />
                    <OuiText size="sm" color="secondary">Docker support</OuiText>
                  </OuiFlex>
                  <OuiFlex align="center" gap="sm">
                    <CheckIcon class="h-4 w-4 text-accent-success shrink-0" />
                    <OuiText size="sm" color="secondary">Real-time monitoring</OuiText>
                  </OuiFlex>
                  <OuiFlex align="center" gap="sm">
                    <CheckIcon class="h-4 w-4 text-accent-success shrink-0" />
                    <OuiText size="sm" color="secondary">File management</OuiText>
                  </OuiFlex>
                </OuiStack>
              </OuiStack>
            </OuiFlex>
          </OuiBox>

          <!-- VPS Section -->
          <OuiBox>
            <OuiFlex gap="2xl" direction="row" align="start" w="full">
              <OuiStack gap="md" rounded="xl"
                class="flex-1 cursor-pointer focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent-primary/60"
                role="button" tabindex="0" @click="openPreview('vps')" @keydown.enter.prevent="openPreview('vps')"
                @keydown.space.prevent="openPreview('vps')">
                <VPSCard :vps="exampleVPS" :loading="false">
                  <template #actions>
                    <OuiFlex gap="xs">
                      <OuiBadge variant="secondary" size="sm">{{ Number(exampleVPS.memoryBytes / BigInt(1024 * 1024 *
                        1024))
                        }} GB RAM</OuiBadge>
                      <OuiBadge variant="secondary" size="sm">{{ exampleVPS.cpuCores }} vCPU</OuiBadge>
                    </OuiFlex>
                  </template>
                </VPSCard>

              </OuiStack>
              <OuiStack gap="lg" grow class="flex-1">
                <OuiText size="2xl" weight="bold" color="primary">VPS Instances</OuiText>
                <OuiText color="secondary">Get full root access VPS instances with complete control over your
                  environment.
                  Perfect
                  for development, hosting, and personal projects.</OuiText>

                <OuiStack gap="sm" mt="md">
                  <OuiFlex align="center" gap="sm">
                    <CheckIcon class="h-4 w-4 text-accent-success shrink-0" />
                    <OuiText size="sm" color="secondary">Full root access & SSH</OuiText>
                  </OuiFlex>
                  <OuiFlex align="center" gap="sm">
                    <CheckIcon class="h-4 w-4 text-accent-success shrink-0" />
                    <OuiText size="sm" color="secondary">Web console & snapshots</OuiText>
                  </OuiFlex>
                  <OuiFlex align="center" gap="sm">
                    <CheckIcon class="h-4 w-4 text-accent-success shrink-0" />
                    <OuiText size="sm" color="secondary">Usage-based pricing</OuiText>
                  </OuiFlex>
                  <OuiFlex align="center" gap="sm">
                    <CheckIcon class="h-4 w-4 text-accent-success shrink-0" />
                    <OuiText size="sm" color="secondary">Multiple Linux distros</OuiText>
                  </OuiFlex>
                  <OuiFlex align="center" gap="sm">
                    <CheckIcon class="h-4 w-4 text-accent-success shrink-0" />
                    <OuiText size="sm" color="secondary">Real-time metrics</OuiText>
                  </OuiFlex>
                  <OuiFlex align="center" gap="sm">
                    <CheckIcon class="h-4 w-4 text-accent-success shrink-0" />
                    <OuiText size="sm" color="secondary">Firewall management</OuiText>
                  </OuiFlex>
                </OuiStack>
              </OuiStack>
            </OuiFlex>
          </OuiBox>
        </OuiStack>
      </OuiContainer>

      <!-- Infrastructure & Support Section -->
      <OuiContainer as="section" size="full" w="full" maxW="7xl" mx="auto" px="md" py="4xl">
        <OuiStack gap="2xl" align="center" w="full" maxW="7xl" mx="auto">
          <OuiStack gap="lg" align="center" maxW="3xl" class="text-center">
            <OuiText as="h2" size="3xl" weight="bold" color="primary" class="md:text-4xl">
              Enterprise Infrastructure & Support
            </OuiText>
            <OuiText color="secondary" class="md:text-base">
              Built on modern, high-performance infrastructure with 99.9% uptime SLA, automatic failover, and
              comprehensive
              security. Get help when you need it with our free support team.
            </OuiText>
          </OuiStack>

          <OuiGrid :cols="{ sm: 1, md: 2, lg: 3 }" gap="xl" w="full">
            <!-- Enterprise Infrastructure -->
            <OuiCard variant="default" h="full">
              <OuiCardBody p="lg">
                <OuiFlex gap="sm" align="start">
                  <OuiBox w="10" h="10" rounded="lg" :shrink="false" class="bg-accent-primary/10">
                    <OuiFlex align="center" justify="center" h="full">
                      <ServerIcon class="h-5 w-5 text-accent-primary" />
                    </OuiFlex>
                  </OuiBox>
                  <OuiStack gap="xs" grow>
                    <OuiText as="h3" size="md" weight="semibold" color="primary">
                      Enterprise Infrastructure
                    </OuiText>
                    <OuiText size="sm" color="secondary">
                      Modern cloud infrastructure with high-performance compute, SSD storage, and low-latency
                      networking.
                      Deployed across multiple regions for optimal performance.
                    </OuiText>
                  </OuiStack>
                </OuiFlex>
              </OuiCardBody>
            </OuiCard>

            <!-- High Availability -->
            <OuiCard variant="default" h="full">
              <OuiCardBody p="lg">
                <OuiFlex gap="sm" align="start">
                  <OuiBox w="10" h="10" rounded="lg" :shrink="false" class="bg-accent-success/10">
                    <OuiFlex align="center" justify="center" h="full">
                      <ShieldCheckIcon class="h-5 w-5 text-accent-success" />
                    </OuiFlex>
                  </OuiBox>
                  <OuiStack gap="xs" grow>
                    <OuiText as="h3" size="md" weight="semibold" color="primary">
                      High Availability & Redundancy
                    </OuiText>
                    <OuiText size="sm" color="secondary">
                      99.9% uptime SLA with built-in redundancy and automatic failover. Multi-region deployment options
                      ensure
                      your services stay online.
                    </OuiText>
                  </OuiStack>
                </OuiFlex>
              </OuiCardBody>
            </OuiCard>

            <!-- Free Support -->
            <OuiCard variant="default" h="full">
              <OuiCardBody p="lg">
                <OuiFlex gap="sm" align="start">
                  <OuiBox w="10" h="10" rounded="lg" :shrink="false" class="bg-accent-secondary/10">
                    <OuiFlex align="center" justify="center" h="full">
                      <ChatBubbleLeftRightIcon class="h-5 w-5 text-accent-secondary" />
                    </OuiFlex>
                  </OuiBox>
                  <OuiStack gap="xs" grow>
                    <OuiText as="h3" size="md" weight="semibold" color="primary">
                      Free Support
                    </OuiText>
                    <OuiText size="sm" color="secondary">
                      Get help when you need it. Our support team is available to assist with deployment, configuration,
                      and
                      troubleshooting.
                    </OuiText>
                  </OuiStack>
                </OuiFlex>
              </OuiCardBody>
            </OuiCard>
          </OuiGrid>
        </OuiStack>
      </OuiContainer>

      <!-- Pricing Section -->
      <OuiContainer as="section" size="full" w="full" maxW="7xl" mx="auto" px="md" py="4xl">
        <OuiStack gap="2xl" align="center" w="full" maxW="7xl" mx="auto" mb="3xl">
          <OuiStack gap="lg" align="center" maxW="3xl" class="text-center">
            <OuiText as="h2" size="3xl" weight="bold" color="primary" class="md:text-4xl">
              Simple, Transparent Pricing
            </OuiText>
            <OuiText color="secondary" class="md:text-base">
              Pay only for what you use. No fixed plans or hidden fees. Just transparent, usage-based pricing for CPU,
              memory,
              storage, and bandwidth.
            </OuiText>
            <OuiText size="sm" color="secondary" class="opacity-75 mt-2">
              Students, open-source projects, non-profits, and early-stage startups may qualify for reduced pricing.
              Contact
              us to discuss.
            </OuiText>
          </OuiStack>
        </OuiStack>

        <OuiGrid :cols="{ sm: 1, md: 2, lg: 4 }" gap="xl" w="full" maxW="7xl" mx="auto">
          <!-- Small App Example -->
          <OuiCard variant="default" hoverable h="full"
            class="text-center flex flex-col transition-all hover:shadow-lg hover:scale-[1.02]">
            <OuiCardBody p="xl" h="full" class="flex flex-col">
              <OuiStack gap="lg" grow>
                <OuiStack gap="md">
                  <OuiBox w="12" h="12" rounded="xl" mx="auto"
                    class="bg-accent-primary/10 flex items-center justify-center">
                    <RocketLaunchIcon class="h-6 w-6 text-accent-primary" />
                  </OuiBox>
                  <OuiText size="lg" weight="semibold" color="primary">Small App</OuiText>
                  <OuiText size="sm" color="secondary">Perfect for side projects</OuiText>
                </OuiStack>

                <OuiStack gap="xs" align="center">
                  <OuiText size="4xl" weight="bold" color="primary">~$5</OuiText>
                  <OuiText size="sm" color="secondary">per month</OuiText>
                </OuiStack>

                <OuiStack gap="sm" align="start" class="text-left flex-1">
                  <OuiFlex align="center" gap="sm">
                    <CheckIcon class="h-4 w-4 text-accent-success shrink-0" />
                    <OuiText size="sm" color="secondary">0.5 GB RAM running 24/7</OuiText>
                  </OuiFlex>
                  <OuiFlex align="center" gap="sm">
                    <CheckIcon class="h-4 w-4 text-accent-success shrink-0" />
                    <OuiText size="sm" color="secondary">0.25 vCPU cores running 24/7</OuiText>
                  </OuiFlex>
                  <OuiFlex align="center" gap="sm">
                    <CheckIcon class="h-4 w-4 text-accent-success shrink-0" />
                    <OuiText size="sm" color="secondary">~10 GB bandwidth/month</OuiText>
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
                    <OuiText size="sm" color="secondary">Built-in redundancy</OuiText>
                  </OuiFlex>
                </OuiStack>
              </OuiStack>
            </OuiCardBody>
          </OuiCard>

          <!-- Game Server Example -->
          <OuiCard variant="default" hoverable h="full"
            class="text-center ring-2 ring-accent-success/50 flex flex-col transition-all hover:shadow-lg hover:scale-[1.02] hover:ring-accent-success/70">
            <OuiCardBody p="xl" h="full" class="flex flex-col">
              <OuiStack gap="lg" grow>
                <OuiStack gap="md">
                  <OuiBox w="12" h="12" rounded="xl" mx="auto" class="bg-accent-success/10">
                    <OuiFlex align="center" justify="center" h="full">
                      <ServerIcon class="h-6 w-6 text-accent-success" />
                    </OuiFlex>
                  </OuiBox>
                  <OuiFlex align="center" justify="center" gap="sm">
                    <OuiText size="lg" weight="semibold" color="primary">Game Server</OuiText>
                    <OuiBadge color="success" size="sm">Save 50%</OuiBadge>
                  </OuiFlex>
                  <OuiText size="sm" color="secondary">12 hours/day runtime</OuiText>
                </OuiStack>

                <OuiStack gap="xs" align="center">
                  <OuiText size="4xl" weight="bold" color="primary">~$15</OuiText>
                  <OuiText size="sm" color="secondary">per month</OuiText>
                  <OuiText size="xs" color="secondary" class="opacity-75 italic line-through">
                    Traditional hosting: $5-15/month fixed
                  </OuiText>
                </OuiStack>

                <OuiStack gap="sm" align="start" class="text-left flex-1">
                  <OuiFlex align="center" gap="sm">
                    <CheckIcon class="h-4 w-4 text-accent-success shrink-0" />
                    <OuiText size="sm" color="secondary">4 GB RAM running 12h/day</OuiText>
                  </OuiFlex>
                  <OuiFlex align="center" gap="sm">
                    <CheckIcon class="h-4 w-4 text-accent-success shrink-0" />
                    <OuiText size="sm" color="secondary">2 vCPU cores average</OuiText>
                  </OuiFlex>
                  <OuiFlex align="center" gap="sm">
                    <CheckIcon class="h-4 w-4 text-accent-success shrink-0" />
                    <OuiText size="sm" color="secondary">~100 GB bandwidth/month</OuiText>
                  </OuiFlex>
                  <OuiFlex align="center" gap="sm">
                    <CheckIcon class="h-4 w-4 text-accent-success shrink-0" />
                    <OuiText size="sm" color="secondary">~20 GB storage</OuiText>
                  </OuiFlex>
                  <OuiFlex align="center" gap="sm">
                    <CheckIcon class="h-4 w-4 text-accent-success shrink-0" />
                    <OuiText size="sm" color="secondary">Low costs when idle or offline</OuiText>
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
          <OuiCard variant="default" hoverable h="full"
            class="text-center flex flex-col transition-all hover:shadow-lg hover:scale-[1.02]">
            <OuiCardBody p="xl" h="full" class="flex flex-col">
              <OuiStack gap="lg" grow>
                <OuiStack gap="md">
                  <OuiBox w="12" h="12" rounded="xl" mx="auto" class="bg-accent-info/10">
                    <OuiFlex align="center" justify="center" h="full">
                      <ServerIcon class="h-6 w-6 text-accent-info" />
                    </OuiFlex>
                  </OuiBox>
                  <OuiText size="lg" weight="semibold" color="primary">VPS Instance</OuiText>
                  <OuiText size="sm" color="secondary">Competitive pricing</OuiText>
                </OuiStack>

                <OuiStack gap="xs" align="center">
                  <OuiText size="4xl" weight="bold" color="primary">~$8</OuiText>
                  <OuiText size="sm" color="secondary">per month</OuiText>
                  <OuiText size="xs" color="secondary" class="opacity-75 italic">
                    Traditional VPS: $10-12/month
                  </OuiText>
                </OuiStack>

                <OuiStack gap="sm" align="start" class="text-left flex-1">
                  <OuiFlex align="center" gap="sm">
                    <CheckIcon class="h-4 w-4 text-accent-success shrink-0" />
                    <OuiText size="sm" color="secondary">2 GB RAM running 24/7</OuiText>
                  </OuiFlex>
                  <OuiFlex align="center" gap="sm">
                    <CheckIcon class="h-4 w-4 text-accent-success shrink-0" />
                    <OuiText size="sm" color="secondary">1 vCPU core running 24/7</OuiText>
                  </OuiFlex>
                  <OuiFlex align="center" gap="sm">
                    <CheckIcon class="h-4 w-4 text-accent-success shrink-0" />
                    <OuiText size="sm" color="secondary">~50 GB bandwidth/month</OuiText>
                  </OuiFlex>
                  <OuiFlex align="center" gap="sm">
                    <CheckIcon class="h-4 w-4 text-accent-success shrink-0" />
                    <OuiText size="sm" color="secondary">~10 GB storage</OuiText>
                  </OuiFlex>
                  <OuiFlex align="center" gap="sm">
                    <CheckIcon class="h-4 w-4 text-accent-success shrink-0" />
                    <OuiText size="sm" color="secondary">Pay for actual usage, not idle time</OuiText>
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
          <OuiCard variant="default" hoverable h="full"
            class="text-center ring-2 ring-accent-primary/50 flex flex-col transition-all hover:shadow-lg hover:scale-[1.02] hover:ring-accent-primary/70">
            <OuiCardBody p="xl" h="full" class="flex flex-col">
              <OuiStack gap="lg" grow>
                <OuiStack gap="md">
                  <OuiBox w="12" h="12" rounded="xl" mx="auto" class="bg-accent-primary/10">
                    <OuiFlex align="center" justify="center" h="full">
                      <RocketLaunchIcon class="h-6 w-6 text-accent-primary" />
                    </OuiFlex>
                  </OuiBox>
                  <OuiFlex align="center" justify="center" gap="sm">
                    <OuiText size="lg" weight="semibold" color="primary">Medium App</OuiText>
                    <OuiBadge color="primary" size="sm">Popular</OuiBadge>
                  </OuiFlex>
                  <OuiText size="sm" color="secondary">For growing teams</OuiText>
                </OuiStack>

                <OuiStack gap="xs" align="center">
                  <OuiText size="4xl" weight="bold" color="primary">~$14</OuiText>
                  <OuiText size="sm" color="secondary">per month</OuiText>
                </OuiStack>

                <OuiStack gap="sm" align="start" class="text-left flex-1">
                  <OuiFlex align="center" gap="sm">
                    <CheckIcon class="h-4 w-4 text-accent-success shrink-0" />
                    <OuiText size="sm" color="secondary">2 GB RAM running 24/7</OuiText>
                  </OuiFlex>
                  <OuiFlex align="center" gap="sm">
                    <CheckIcon class="h-4 w-4 text-accent-success shrink-0" />
                    <OuiText size="sm" color="secondary">1 vCPU core running 24/7</OuiText>
                  </OuiFlex>
                  <OuiFlex align="center" gap="sm">
                    <CheckIcon class="h-4 w-4 text-accent-success shrink-0" />
                    <OuiText size="sm" color="secondary">~50 GB bandwidth/month</OuiText>
                  </OuiFlex>
                  <OuiFlex align="center" gap="sm">
                    <CheckIcon class="h-4 w-4 text-accent-success shrink-0" />
                    <OuiText size="sm" color="secondary">~25 GB storage</OuiText>
                  </OuiFlex>
                  <OuiFlex align="center" gap="sm">
                    <CheckIcon class="h-4 w-4 text-accent-success shrink-0" />
                    <OuiText size="sm" color="secondary">Custom configurations</OuiText>
                  </OuiFlex>
                  <OuiFlex align="center" gap="sm">
                    <CheckIcon class="h-4 w-4 text-accent-success shrink-0" />
                    <OuiText size="sm" color="secondary">Free support</OuiText>
                  </OuiFlex>
                  <OuiFlex align="center" gap="sm">
                    <CheckIcon class="h-4 w-4 text-accent-success shrink-0" />
                    <OuiText size="sm" color="secondary">Built-in redundancy</OuiText>
                  </OuiFlex>
                </OuiStack>
              </OuiStack>
            </OuiCardBody>
          </OuiCard>
        </OuiGrid>

        <OuiStack gap="lg" align="center" maxW="7xl" mx="auto" mt="2xl">
          <OuiFlex gap="md" wrap="wrap" justify="center">
            <OuiButton size="lg" color="primary" class="gap-2 shadow-lg shadow-accent-primary/25"
              @click="navigateTo('/dashboard')">
              <RocketLaunchIcon class="h-5 w-5" />
              Get Started
            </OuiButton>
          </OuiFlex>

          <OuiText size="sm" color="secondary" class="opacity-75 text-center" maxW="2xl">
            * Example costs based on 24/7 usage. You only pay for actual runtime -
            if your app runs part-time, you'll pay less. Use the calculator below
            to estimate your exact costs.
          </OuiText>
        </OuiStack>

        <OuiCard variant="default" maxW="6xl" mx="auto" mt="xl"
          class="border-2 border-accent-primary/20 bg-gradient-to-br from-accent-primary/5 to-accent-secondary/5">
          <OuiCardBody p="xl">
            <OuiStack gap="xl">
              <OuiStack gap="xs" align="center" class="text-center max-w-3xl mx-auto">
                <OuiText size="xl" weight="bold" color="primary">
                  Why Pay-as-You-Go Matters
                </OuiText>
                <OuiText size="sm" color="secondary">
                  Traditional hosting charges you for resources you don't use. We only charge for what you actually
                  consume.
                </OuiText>
              </OuiStack>
              <OuiGrid :cols="{ sm: 1, md: 2, lg: 4 }" gap="lg" w="full" maxW="6xl" mx="auto">
                <OuiCard variant="default" class="bg-surface-muted/30 border-border-muted/50">
                  <OuiCardBody p="lg">
                    <OuiFlex gap="md" align="start">
                      <OuiBox w="10" h="10" rounded="lg" :shrink="false" class="bg-accent-success/10">
                        <OuiFlex align="center" justify="center" h="full">
                          <ServerIcon class="h-5 w-5 text-accent-success" />
                        </OuiFlex>
                      </OuiBox>
                      <OuiStack gap="xs">
                        <OuiText size="sm" weight="semibold" color="primary">
                          Game Servers
                        </OuiText>
                        <OuiText size="sm" color="secondary">
                          Low costs when idle or offline. Traditional hosting charges $5-15/month fixed plans even when
                          your
                          server is empty.
                        </OuiText>
                      </OuiStack>
                    </OuiFlex>
                  </OuiCardBody>
                </OuiCard>
                <OuiCard variant="default" class="bg-surface-muted/30 border-border-muted/50">
                  <OuiCardBody p="lg">
                    <OuiFlex gap="md" align="start">
                      <OuiBox w="10" h="10" rounded="lg" :shrink="false" class="bg-accent-primary/10">
                        <OuiFlex align="center" justify="center" h="full">
                          <ServerIcon class="h-5 w-5 text-accent-primary" />
                        </OuiFlex>
                      </OuiBox>
                      <OuiStack gap="xs">
                        <OuiText size="sm" weight="semibold" color="primary">
                          VPS Instances
                        </OuiText>
                        <OuiText size="sm" color="secondary">
                          Pay for actual CPU, memory, storage, and bandwidth usage, not idle time. Most VPS providers
                          charge
                          full price regardless of utilization.
                        </OuiText>
                      </OuiStack>
                    </OuiFlex>
                  </OuiCardBody>
                </OuiCard>
                <OuiCard variant="default" class="bg-surface-muted/30 border-border-muted/50">
                  <OuiCardBody p="lg">
                    <OuiFlex gap="md" align="start">
                      <OuiBox w="10" h="10" rounded="lg" :shrink="false" class="bg-accent-secondary/10">
                        <OuiFlex align="center" justify="center" h="full">
                          <RocketLaunchIcon class="h-5 w-5 text-accent-secondary" />
                        </OuiFlex>
                      </OuiBox>
                      <OuiStack gap="xs">
                        <OuiText size="sm" weight="semibold" color="primary">
                          Development Environments
                        </OuiText>
                        <OuiText size="sm" color="secondary">
                          Stop paying for resources that sit idle overnight or on weekends. Only pay when you're
                          actively
                          developing.
                        </OuiText>
                      </OuiStack>
                    </OuiFlex>
                  </OuiCardBody>
                </OuiCard>
                <OuiCard variant="default" class="bg-surface-muted/30 border-border-muted/50">
                  <OuiCardBody p="lg">
                    <OuiFlex gap="md" align="start">
                      <OuiBox w="10" h="10" rounded="lg" :shrink="false" class="bg-accent-warning/10">
                        <OuiFlex align="center" justify="center" h="full">
                          <ChartBarIcon class="h-5 w-5 text-accent-warning" />
                        </OuiFlex>
                      </OuiBox>
                      <OuiStack gap="xs">
                        <OuiText size="sm" weight="semibold" color="primary">
                          Variable Workloads
                        </OuiText>
                        <OuiText size="sm" color="secondary">
                          Scale costs automatically with demand - no over-provisioning required. Perfect for
                          applications
                          with
                          varying traffic patterns.
                        </OuiText>
                      </OuiStack>
                    </OuiFlex>
                  </OuiCardBody>
                </OuiCard>
              </OuiGrid>
            </OuiStack>
          </OuiCardBody>
        </OuiCard>
      </OuiContainer>

      <!-- Pricing Calculator Section -->
      <OuiContainer size="full" w="full" maxW="7xl" mx="auto" px="md" py="4xl">
        <PricingCalculator />
      </OuiContainer>

      <!-- CTA Section -->
      <OuiContainer as="section" size="full" w="full" maxW="7xl" mx="auto" py="4xl" px="md">
        <OuiStack align="center">
          <OuiCard variant="default" maxW="5xl" w="full" class="bg-surface-muted/30 border-border-muted/40">
            <OuiCardBody p="2xl">
              <OuiStack align="center" gap="md" class="text-center">
                <OuiText as="h2" size="3xl" weight="bold" color="primary" class="md:text-4xl">
                  Ready to Deploy?
                </OuiText>
                <OuiText size="lg" color="secondary" class="md:text-xl" style="max-width: 48rem;">
                  Deploy from GitHub in minutes. Pay only for the CPU, memory, storage, and bandwidth you actually use.
                </OuiText>
                <OuiFlex gap="md" wrap="wrap" justify="center" mt="lg">
                  <OuiButton size="lg" color="primary" class="gap-2" @click="navigateTo('/dashboard')">
                    <RocketLaunchIcon class="h-5 w-5" />
                    Get Started
                  </OuiButton>

                </OuiFlex>
              </OuiStack>
            </OuiCardBody>
          </OuiCard>
        </OuiStack>
      </OuiContainer>

      <!-- Footer -->
      <AppFooter />
    </OuiStack>
  </OuiContainer>
</template>

<script setup lang="ts">
import { computed, ref, defineAsyncComponent, onMounted, onBeforeUnmount } from "vue";
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
import DeploymentCard from "~/components/deployment/DeploymentCard.vue";
import GameServerCard from "~/components/gameserver/GameServerCard.vue";
import VPSCard from "~/components/vps/VPSCard.vue";
import VPSXTermTerminal from "~/components/vps/VPSXTermTerminal.vue";
import LiveMetrics from "~/components/shared/LiveMetrics.vue";
import OuiFloatingPanel from "~/components/oui/FloatingPanel.vue";
import PreviewProviders from "~/components/preview/PreviewProviders.vue";
import {
  DeploymentStatus,
  DeploymentType,
  BuildStrategy,
  VPSStatus,
  VPSImage,
  Environment,
  type Deployment,
  type GameServer,
  type VPSInstance,
  type GameServerUsageMetrics,
  GameType,
  GameServerStatus
} from "@obiente/proto";
import { useConfig } from "~/composables/useConfig";
import { useConnectClient } from "~/lib/connect-client";
import { DeploymentService } from "@obiente/proto";

const DeploymentPage = defineAsyncComponent(() => import("~/pages/deployments/[id]/index.vue"));
const GameServerPage = defineAsyncComponent(() => import("~/pages/gameservers/[id]/index.vue"));
const VpsPage = defineAsyncComponent(() => import("~/pages/vps/[id].vue"));

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

// Example data for showcase (static preview only, no network calls)
const exampleWebDeployment: Deployment = {
  // Use a mock id to avoid navigation, but satisfy type
  id: "mock-web-deployment",
  name: "Obiente Cloud",
  domain: "obiente.cloud",
  customDomains: [],
  type: DeploymentType.DOCKER,
  buildStrategy: BuildStrategy.RAILPACK,
  status: DeploymentStatus.RUNNING,
  repositoryUrl: "https://github.com/obiente/cloud",
  branch: "main",
  healthStatus: "Healthy",
  bandwidthUsage: BigInt(0),
  storageUsage: BigInt(1024 * 1024 * 50),
  buildTime: 45,
  size: "50 MB",
  environment: Environment.PRODUCTION,
  groups: ["web"],
  containerIds: ["container-1", "container-2"],
  envVars: {},
  $typeName: "obiente.cloud.deployments.v1.Deployment" as const,
};


const exampleGameServer: GameServer = {
  id: "mock-game-server",
  organizationId: "mock-org",
  name: "Minecraft Server",
  status: GameServerStatus.RUNNING,
  port: 25565,
  gameType: GameType.MINECRAFT,
  cpuCores: 2,
  memoryBytes: BigInt(4 * 1024 * 1024 * 1024), // 4 GB
  dockerImage: "itzg/minecraft-server:latest",
  envVars: {},
  storageBytes: BigInt(20 * 1024 * 1024 * 1024), // 20 GB
  createdBy: "mock-user",
  $typeName: "obiente.cloud.gameservers.v1.GameServer" as const,
};

// Transform for GameServerCard which expects string types
const exampleGameServerForCard = computed(() => ({
  id: exampleGameServer.id,
  name: exampleGameServer.name,
  gameType: "minecraft",
  status: "RUNNING",
  port: exampleGameServer.port,
  cpuCores: exampleGameServer.cpuCores,
  memoryBytes: Number(exampleGameServer.memoryBytes),
}));

const exampleVPS: VPSInstance = {
  id: "mock-vps",
  name: "VPS Instance",
  description: "A small VPS for development",
  status: VPSStatus.RUNNING,
  region: "us-east-1",
  image: VPSImage.UBUNTU_22_04,
  size: "small",
  cpuCores: 1,
  memoryBytes: BigInt(2 * 1024 * 1024 * 1024), // 2 GB
  diskBytes: BigInt(20 * 1024 * 1024 * 1024), // 20 GB
  organizationId: "mock-org",
  ipv4Addresses: ["10.15.3.100"],
  ipv6Addresses: [],
  metadata: {},
  createdBy: "mock-user",
  $typeName: "obiente.cloud.vps.v1.VPSInstance" as const,
};

const exampleGameUsageData: Partial<GameServerUsageMetrics> = {
  cpuCoreSeconds: BigInt(7200),
  memoryByteSeconds: BigInt(4 * 1024 * 1024 * 1024 * 7200),
  bandwidthRxBytes: BigInt(1024 * 1024 * 600),
  bandwidthTxBytes: BigInt(1024 * 1024 * 300),
  storageBytes: BigInt(20 * 1024 * 1024 * 1024),
  uptimeSeconds: BigInt(7200),
  estimatedCostCents: BigInt(2200),
  cpuCostCents: BigInt(800),
  memoryCostCents: BigInt(900),
  bandwidthCostCents: BigInt(300),
  storageCostCents: BigInt(200),
};

const exampleGameLiveMetrics = {
  isStreaming: false,
  latestMetric: null,
  currentCpuUsage: 18,
  currentMemoryUsage: 42,
  currentNetworkRx: 210,
  currentNetworkTx: 110,
};

const exampleVpsLiveMetrics = {
  isStreaming: true,
  latestMetric: null,
  currentCpuUsage: 12,
  currentMemoryUsage: 28,
  currentNetworkRx: 80,
  currentNetworkTx: 60,
};

// Live metrics state for homepage preview (PreviewProviders swaps transport)
const deploymentClient = useConnectClient(DeploymentService);
const homepageMetricsStreaming = ref(false);
const homepageLatestMetric = ref<any>(null);
const homepageStreamController = ref<AbortController | null>(null);

// Computed metrics from latest data
const homepageCurrentCpuUsage = computed(() => {
  return homepageLatestMetric.value?.cpuUsagePercent ?? 15;
});

const homepageCurrentMemoryUsage = computed(() => {
  return homepageLatestMetric.value?.memoryUsageBytes ?? 512 * 1024 * 1024;
});

const homepageCurrentNetworkRx = computed(() => {
  return homepageLatestMetric.value?.networkRxBytes ?? 100 * 1024 * 1024;
});

const homepageCurrentNetworkTx = computed(() => {
  return homepageLatestMetric.value?.networkTxBytes ?? 50 * 1024 * 1024;
});

// Start streaming metrics for homepage preview
const startHomepageMetricsStream = async () => {
  if (homepageMetricsStreaming.value || homepageStreamController.value) {
    return;
  }

  if (!import.meta.client) return;

  // Check if mock transport is available
  const previewTransport = (globalThis as any).__OBIENTE_PREVIEW_CONNECT__;
  if (!previewTransport) {
    console.warn("[Homepage] Mock transport not available, metrics will not stream");
    return;
  }

  homepageMetricsStreaming.value = true;
  homepageStreamController.value = new AbortController();

  try {
    const request: any = {
      deploymentId: exampleWebDeployment.id,
      organizationId: "mock-org",
      intervalSeconds: 5,
      aggregate: true,
    };

    console.log("[Homepage] Starting metrics stream with mock transport");
    const stream = await (deploymentClient as any).streamDeploymentMetrics(request, {
      signal: homepageStreamController.value.signal,
    });

    console.log("[Homepage] Metrics stream started, receiving data...");
    for await (const metric of stream) {
      if (homepageStreamController.value?.signal.aborted) {
        break;
      }
      homepageLatestMetric.value = metric;
      console.log("[Homepage] Received metric:", metric);
    }
  } catch (err: any) {
    if (err.name === "AbortError") {
      return;
    }
    // Suppress "missing trailer" errors
    const isMissingTrailerError =
      err.message?.toLowerCase().includes("missing trailer") ||
      err.message?.toLowerCase().includes("trailer") ||
      err.code === "unknown";

    if (!isMissingTrailerError) {
      console.error("[Homepage] Failed to stream metrics:", err);
    }
  } finally {
    homepageMetricsStreaming.value = false;
    homepageStreamController.value = null;
  }
};

// Stop streaming
const stopHomepageMetricsStream = () => {
  if (homepageStreamController.value) {
    homepageStreamController.value.abort();
    homepageStreamController.value = null;
  }
  homepageMetricsStreaming.value = false;
};

// Start streaming when component mounts (client-side only)
// Wait for PreviewProviders to set up the mock transport
onMounted(() => {
  if (!import.meta.client) return;

  // Wait for mock transport to be available
  const checkAndStart = () => {
    const previewTransport = (globalThis as any).__OBIENTE_PREVIEW_CONNECT__;
    if (previewTransport) {
      console.log('[Homepage] Mock transport available, starting metrics stream');
      startHomepageMetricsStream();
    } else {
      // Retry after a short delay if transport not ready yet
      setTimeout(checkAndStart, 100);
    }
  };

  checkAndStart();
});

// Cleanup on unmount
onBeforeUnmount(() => {
  stopHomepageMetricsStream();
});

const activePreview = ref<"web" | "game" | "vps" | null>(null);
const previewKey = ref(0);

const previewPanelOpen = computed({
  get: () => activePreview.value !== null,
  set: value => {
    if (!value) activePreview.value = null;
  },
});

const previewDefaultPosition = computed(() => {
  if (!import.meta.client) return { x: 0, y: 0 };
  return {
    x: window.innerWidth * 0.025,
    y: window.innerHeight * 0.04
  };
});

const previewTitle = computed(() => {
  if (activePreview.value === "web") return "Web App Preview";
  if (activePreview.value === "game") return "Game Server Preview";
  if (activePreview.value === "vps") return "VPS Preview";
  return "Preview";
});

const openPreview = async (type: "web" | "game" | "vps") => {
  // Close existing preview first
  if (activePreview.value !== null) {
    activePreview.value = null;
    await new Promise(resolve => setTimeout(resolve, 50)); // Wait for cleanup
  }
  previewKey.value++; // Force new instance
  activePreview.value = type;
};

const closePreview = () => {
  activePreview.value = null;
};

const handleMockRefresh = () => {
  // No-op refresh handler for mock data
};

// Handle sign-up click
const handleSignUp = () => {
  if (import.meta.client) {
    auth.popupSignup();
  }
  // Server-side: signup is handled via popup, no navigation needed
};

// SEO meta
useHead({
  title: "Obiente Cloud - Docker Container Hosting, Game Server Hosting & VPS Cloud Platform | Pay-As-You-Go",
  meta: [
    {
      name: "description",
      content:
        "Deploy Docker containers from GitHub, host Minecraft and game servers, and launch VPS instances with transparent pay-as-you-go pricing. Automatic SSL, real-time monitoring, full root access. Deploy in under 5 minutes. Perfect for developers, indie studios, and businesses.",
    },
    {
      name: "keywords",
      content:
        "docker hosting, container hosting, docker deployment, github deployment, game server hosting, minecraft hosting, vps hosting, cloud vps, pay as you go cloud, usage based pricing, docker compose hosting, automatic ssl, cloud platform, developer hosting, indie game hosting, startup hosting, vps cloud, linux vps, ubuntu vps, docker cloud, container cloud platform",
    },
    {
      property: "og:title",
      content: "Obiente Cloud - Docker, Game Server & VPS Hosting | Pay-As-You-Go Cloud Platform",
    },
    {
      property: "og:description",
      content:
        "Deploy Docker containers from GitHub, host game servers, and launch VPS instances with transparent pay-as-you-go pricing. Automatic SSL, real-time monitoring, full root access. Deploy in minutes.",
    },
    { property: "og:type", content: "website" },
    { name: "twitter:card", content: "summary_large_image" },
    { name: "twitter:title", content: "Obiente Cloud - Pay-As-You-Go Cloud Hosting Platform" },
    { name: "twitter:description", content: "Deploy Docker containers, host game servers, and launch VPS instances with transparent usage-based pricing. No fixed plans, no hidden fees." },
    { name: "robots", content: "index, follow" },
    { name: "author", content: "Obiente Cloud" },
  ],
});
</script>

<style>
/* Ensure full page background coverage for homepage */
html,
body {
  background-color: var(--oui-background) !important;
}

/* Hero dot patterns */
.hero-dots-primary {
  background-image: radial-gradient(circle, var(--oui-accent-primary) 2px, transparent 2px);
  background-size: 32px 32px;
  background-position: 0 0;
  opacity: 0.35;
}

.hero-dots-secondary {
  background-image: radial-gradient(circle, var(--oui-accent-secondary) 1.8px, transparent 1.8px);
  background-size: 32px 32px;
  background-position: 16px 16px;
  opacity: 0.3;
}
</style>
