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
        <OuiButton variant="ghost" size="sm">Web Hosting</OuiButton>
        <OuiButton variant="ghost" size="sm">Game Hosting</OuiButton>
        <OuiButton variant="ghost" size="sm">VPS</OuiButton>
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
              class="bg-linear-to-r from-accent-primary to-accent-primary bg-clip-text text-transparent"
            >
              Obiente Cloud
            </span>
          </OuiText>

          <OuiText size="xl" color="secondary" class="max-w-2xl md:text-2xl">
            The all-in-one platform for web apps, game servers, VPS instances,
            and databases. Deploy in seconds, scale with confidence.
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
            Get Started Free
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
            From web apps to game servers
          </OuiText>
          <OuiText size="lg" color="secondary" class="md:text-xl">
            Deploy anything, anywhere. One platform for all your hosting needs.
          </OuiText>
        </OuiStack>

        <OuiGrid cols="1" cols-md="2" cols-lg="3" gap="xl" class="w-full">
          <!-- Web Deployments -->
          <OuiCard
            variant="raised"
            hoverable
            class="group cursor-pointer transition-all duration-300 hover:-translate-y-2 hover:shadow-xl hover:shadow-accent-primary/10"
          >
            <OuiCardBody class="text-center">
              <OuiStack gap="lg">
                <OuiBox
                  class="mx-auto w-16 h-16 bg-linear-to-r from-accent-primary/20 to-accent-primary/10 rounded-xl flex items-center justify-center group-hover:from-accent-primary/30 group-hover:to-accent-primary/20 transition-colors"
                >
                  <RocketLaunchIcon class="h-8 w-8 text-accent-primary" />
                </OuiBox>
                <OuiStack gap="md">
                  <OuiText as="h3" size="xl" weight="semibold" color="primary"
                    >Web Deployments</OuiText
                  >
                  <OuiText color="secondary">
                    Deploy React, Next.js, Vue, and more. Git-based deployments
                    with automatic builds and global CDN.
                  </OuiText>
                </OuiStack>
              </OuiStack>
            </OuiCardBody>
          </OuiCard>

          <!-- Game Hosting -->
          <OuiCard
            variant="raised"
            hoverable
            class="group cursor-pointer transition-all duration-300 hover:-translate-y-2 hover:shadow-xl hover:shadow-accent-warning/10"
          >
            <OuiCardBody class="text-center">
              <OuiStack gap="lg">
                <OuiBox
                  class="mx-auto w-16 h-16 bg-linear-to-r from-accent-warning/20 to-accent-warning/10 rounded-xl flex items-center justify-center group-hover:from-accent-warning/30 group-hover:to-accent-warning/20 transition-colors"
                >
                  <svg
                    class="h-8 w-8 text-accent-warning"
                    fill="currentColor"
                    viewBox="0 0 24 24"
                  >
                    <path
                      d="M7.5 6.5C7.5 8.981 9.519 11 12 11s4.5-2.019 4.5-4.5S14.481 2 12 2 7.5 4.019 7.5 6.5zM20 21h1v-1c0-3.859-3.141-7-7-7h-4c-3.859 0-7 3.141-7 7v1h1 1 14z"
                    />
                    <path d="M15.5 7.5a2 2 0 11-4 0 2 2 0 014 0z" />
                  </svg>
                </OuiBox>
                <OuiStack gap="md">
                  <OuiText as="h3" size="xl" weight="semibold" color="primary"
                    >Game Hosting</OuiText
                  >
                  <OuiText color="secondary">
                    Host Minecraft, CS2, Rust, and more. Reliable performance
                    with high-quality hardware.
                  </OuiText>
                </OuiStack>
              </OuiStack>
            </OuiCardBody>
          </OuiCard>

          <!-- VPS Infrastructure -->
          <OuiCard
            variant="raised"
            hoverable
            class="group cursor-pointer transition-all duration-300 hover:-translate-y-2 hover:shadow-xl hover:shadow-accent-secondary/10"
          >
            <OuiCardBody class="text-center">
              <OuiStack gap="lg">
                <OuiBox
                  class="mx-auto w-16 h-16 bg-linear-to-r from-accent-secondary/20 to-accent-secondary/10 rounded-xl flex items-center justify-center group-hover:from-accent-secondary/30 group-hover:to-accent-secondary/20 transition-colors"
                >
                  <ServerIcon class="h-8 w-8 text-accent-secondary" />
                </OuiBox>
                <OuiStack gap="md">
                  <OuiText as="h3" size="xl" weight="semibold" color="primary"
                    >VPS Infrastructure</OuiText
                  >
                  <OuiText color="secondary">
                    Full Linux VPS instances with root access. Perfect for
                    custom applications and microservices.
                  </OuiText>
                </OuiStack>
              </OuiStack>
            </OuiCardBody>
          </OuiCard>

          <!-- Database Services -->
          <OuiCard
            variant="raised"
            hoverable
            class="group cursor-pointer transition-all duration-300 hover:-translate-y-2 hover:shadow-xl hover:shadow-accent-success/10"
          >
            <OuiCardBody class="text-center">
              <OuiStack gap="lg">
                <OuiBox
                  class="mx-auto w-16 h-16 bg-linear-to-r from-accent-success/20 to-accent-success/10 rounded-xl flex items-center justify-center group-hover:from-accent-success/30 group-hover:to-accent-success/20 transition-colors"
                >
                  <CircleStackIcon class="h-8 w-8 text-accent-success" />
                </OuiBox>
                <OuiStack gap="md">
                  <OuiText as="h3" size="xl" weight="semibold" color="primary"
                    >Database Services</OuiText
                  >
                  <OuiText color="secondary">
                    Managed PostgreSQL, MySQL, Redis, and MongoDB. Automatic
                    backups and scaling.
                  </OuiText>
                </OuiStack>
              </OuiStack>
            </OuiCardBody>
          </OuiCard>

          <!-- Monitoring & Alerting -->
          <OuiCard
            variant="raised"
            hoverable
            class="group cursor-pointer transition-all duration-300 hover:-translate-y-2 hover:shadow-xl hover:shadow-accent-info/10"
          >
            <OuiCardBody class="text-center">
              <OuiStack gap="lg">
                <OuiBox
                  class="mx-auto w-16 h-16 bg-linear-to-r from-accent-info/20 to-accent-info/10 rounded-xl flex items-center justify-center group-hover:from-accent-info/30 group-hover:to-accent-info/20 transition-colors"
                >
                  <ChartBarIcon class="h-8 w-8 text-accent-info" />
                </OuiBox>
                <OuiStack gap="md">
                  <OuiText as="h3" size="xl" weight="semibold" color="primary"
                    >Monitoring & Alerting</OuiText
                  >
                  <OuiText color="secondary">
                    Real-time metrics, performance monitoring, and instant
                    alerts to keep your services healthy.
                  </OuiText>
                </OuiStack>
              </OuiStack>
            </OuiCardBody>
          </OuiCard>

          <!-- Enterprise Security -->
          <OuiCard
            variant="raised"
            hoverable
            class="group cursor-pointer transition-all duration-300 hover:-translate-y-2 hover:shadow-xl hover:shadow-accent-danger/10"
          >
            <OuiCardBody class="text-center">
              <OuiStack gap="lg">
                <OuiBox
                  class="mx-auto w-16 h-16 bg-linear-to-r from-accent-danger/20 to-accent-danger/10 rounded-xl flex items-center justify-center group-hover:from-accent-danger/30 group-hover:to-accent-danger/20 transition-colors"
                >
                  <ShieldCheckIcon class="h-8 w-8 text-accent-danger" />
                </OuiBox>
                <OuiStack gap="md">
                  <OuiText as="h3" size="xl" weight="semibold" color="primary"
                    >Enterprise Security</OuiText
                  >
                  <OuiText color="secondary">
                    SOC 2 compliant with SSL certificates and comprehensive
                    audit logs for security.
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
            Choose the plan that fits your needs. Pay only for what you use.
          </OuiText>
        </OuiStack>

        <OuiGrid cols="1" cols-md="3" gap="xl" class="w-full max-w-5xl">
          <!-- Starter Plan -->
          <OuiCard
            variant="raised"
            hoverable
            class="text-center group hover:-translate-y-1 transition-all duration-300"
          >
            <OuiCardBody>
              <OuiStack gap="lg">
                <OuiStack gap="md">
                  <OuiText size="lg" weight="semibold" color="primary"
                    >Starter</OuiText
                  >
                  <OuiText size="sm" color="secondary"
                    >Perfect for side projects</OuiText
                  >
                </OuiStack>

                <OuiStack gap="xs" align="center">
                  <OuiText size="3xl" weight="bold" color="primary">$1</OuiText>
                  <OuiText size="sm" color="secondary">per month</OuiText>
                </OuiStack>

                <OuiStack gap="sm" align="start" class="text-left">
                  <OuiFlex align="center" gap="sm">
                    <CheckIcon class="h-4 w-4 text-accent-success shrink-0" />
                    <OuiText size="sm" color="secondary"
                      >1 web deployment</OuiText
                    >
                  </OuiFlex>
                  <OuiFlex align="center" gap="sm">
                    <CheckIcon class="h-4 w-4 text-accent-success shrink-0" />
                    <OuiText size="sm" color="secondary"
                      >1 game server slot</OuiText
                    >
                  </OuiFlex>
                  <OuiFlex align="center" gap="sm">
                    <CheckIcon class="h-4 w-4 text-accent-success shrink-0" />
                    <OuiText size="sm" color="secondary"
                      >1 shared vCPU (Ryzen 7 7900x)</OuiText
                    >
                  </OuiFlex>
                  <OuiFlex align="center" gap="sm">
                    <CheckIcon class="h-4 w-4 text-accent-success shrink-0" />
                    <OuiText size="sm" color="secondary">512MB RAM</OuiText>
                  </OuiFlex>
                  <OuiFlex align="center" gap="sm">
                    <CheckIcon class="h-4 w-4 text-accent-success shrink-0" />
                    <OuiText size="sm" color="secondary">2GB storage</OuiText>
                  </OuiFlex>
                  <OuiFlex align="center" gap="sm">
                    <CheckIcon class="h-4 w-4 text-accent-success shrink-0" />
                    <OuiText size="sm" color="secondary">Free support</OuiText>
                  </OuiFlex>
                </OuiStack>

                <OuiButton
                  variant="outline"
                  size="md"
                  class="w-full"
                  @click="navigateTo('/dashboard')"
                >
                  Start with Starter
                </OuiButton>
              </OuiStack>
            </OuiCardBody>
          </OuiCard>

          <!-- Pro Plan -->
          <OuiCard
            variant="raised"
            hoverable
            class="text-center group hover:-translate-y-1 transition-all duration-300 ring-2 ring-accent-primary/50"
          >
            <OuiCardBody>
              <OuiStack gap="lg">
                <OuiStack gap="md">
                  <OuiFlex align="center" justify="center" gap="sm">
                    <OuiText size="lg" weight="semibold" color="primary"
                      >Pro</OuiText
                    >
                    <OuiBadge color="primary" size="sm">Popular</OuiBadge>
                  </OuiFlex>
                  <OuiText size="sm" color="secondary"
                    >For growing teams</OuiText
                  >
                </OuiStack>

                <OuiStack gap="xs" align="center">
                  <OuiText size="3xl" weight="bold" color="primary"
                    >$29</OuiText
                  >
                  <OuiText size="sm" color="secondary">per month</OuiText>
                </OuiStack>

                <OuiStack gap="sm" align="start" class="text-left">
                  <OuiFlex align="center" gap="sm">
                    <CheckIcon class="h-4 w-4 text-accent-success shrink-0" />
                    <OuiText size="sm" color="secondary"
                      >10 web deployments</OuiText
                    >
                  </OuiFlex>
                  <OuiFlex align="center" gap="sm">
                    <CheckIcon class="h-4 w-4 text-accent-success shrink-0" />
                    <OuiText size="sm" color="secondary"
                      >5 game servers</OuiText
                    >
                  </OuiFlex>
                  <OuiFlex align="center" gap="sm">
                    <CheckIcon class="h-4 w-4 text-accent-success shrink-0" />
                    <OuiText size="sm" color="secondary"
                      >2 shared vCPUs (Ryzen 7 7900x)</OuiText
                    >
                  </OuiFlex>
                  <OuiFlex align="center" gap="sm">
                    <CheckIcon class="h-4 w-4 text-accent-success shrink-0" />
                    <OuiText size="sm" color="secondary"
                      >2GB RAM per instance</OuiText
                    >
                  </OuiFlex>
                  <OuiFlex align="center" gap="sm">
                    <CheckIcon class="h-4 w-4 text-accent-success shrink-0" />
                    <OuiText size="sm" color="secondary">50GB storage</OuiText>
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
                </OuiStack>

                <OuiButton
                  color="primary"
                  size="md"
                  class="w-full shadow-lg shadow-accent-primary/25"
                  @click="navigateTo('/dashboard')"
                >
                  Start Pro Trial
                </OuiButton>
              </OuiStack>
            </OuiCardBody>
          </OuiCard>

          <!-- Enterprise Plan -->
          <OuiCard
            variant="raised"
            hoverable
            class="text-center group hover:-translate-y-1 transition-all duration-300"
          >
            <OuiCardBody>
              <OuiStack gap="lg">
                <OuiStack gap="md">
                  <OuiText size="lg" weight="semibold" color="primary"
                    >Enterprise</OuiText
                  >
                  <OuiText size="sm" color="secondary"
                    >For large organizations</OuiText
                  >
                </OuiStack>

                <OuiStack gap="xs" align="center">
                  <OuiText size="3xl" weight="bold" color="primary"
                    >Pay as you grow</OuiText
                  >
                  <OuiText size="sm" color="secondary"
                    >resource-based pricing</OuiText
                  >
                </OuiStack>

                <OuiStack gap="sm" align="start" class="text-left">
                  <OuiFlex align="center" gap="sm">
                    <CheckIcon class="h-4 w-4 text-accent-success shrink-0" />
                    <OuiText size="sm" color="secondary"
                      >Unlimited deployments</OuiText
                    >
                  </OuiFlex>
                  <OuiFlex align="center" gap="sm">
                    <CheckIcon class="h-4 w-4 text-accent-success shrink-0" />
                    <OuiText size="sm" color="secondary"
                      >Unlimited game servers</OuiText
                    >
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
                </OuiStack>

                <OuiButton variant="outline" size="md" class="w-full">
                  Contact Sales
                </OuiButton>
              </OuiStack>
            </OuiCardBody>
          </OuiCard>
        </OuiGrid>
      </OuiStack>
    </OuiContainer>

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
            Join our growing community of developers and gamers building amazing
            things.
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
            Start for $1/month
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
} from "@heroicons/vue/24/outline";

// Page meta - no auth required for homepage
definePageMeta({
  layout: false, // Use custom layout for homepage
});

// Scroll state for header effect using VueUse
const { y: scrollY } = useWindowScroll();
const isScrolled = computed(() => scrollY.value > 10);

// SEO meta
useHead({
  title: "Obiente Cloud - Web Apps, Game Servers & VPS Hosting",
  meta: [
    {
      name: "description",
      content:
        "Deploy web apps, host game servers, and manage VPS infrastructure with Obiente Cloud. Like Vercel, but for everything.",
    },
    {
      name: "keywords",
      content:
        "web hosting, game hosting, minecraft hosting, vps, cloud platform, deployment, vercel alternative",
    },
    {
      property: "og:title",
      content: "Obiente Cloud - Web Apps, Game Servers & VPS Hosting",
    },
    {
      property: "og:description",
      content:
        "Deploy web apps, host game servers, and manage VPS infrastructure with Obiente Cloud. Like Vercel, but for everything.",
    },
    { property: "og:type", content: "website" },
    { name: "twitter:card", content: "summary_large_image" },
  ],
});
</script>
