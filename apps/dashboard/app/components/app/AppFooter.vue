<template>
  <footer class="border-t border-surface-muted/20 mt-24">
    <OuiContainer py="6xl" size="7xl">
      <OuiGrid cols="1" cols-md="4" gap="xl">
        <!-- Brand -->
        <OuiStack gap="lg">
          <OuiFlex align="center" gap="md">
            <OuiText size="xl" class="font-bold text-white">
              Obiente Cloud
            </OuiText>
          </OuiFlex>
          <OuiText size="sm" class="text-muted">
            Deploy, Scale, and Manage Your Cloud Infrastructure
          </OuiText>
        </OuiStack>

        <!-- Product -->
        <OuiStack gap="md">
          <OuiText size="sm" class="font-semibold text-white">Product</OuiText>
          <OuiStack gap="sm">
            <OuiText
              size="sm"
              class="text-muted hover:text-white cursor-pointer"
            >
              VPS Hosting
            </OuiText>
            <OuiText
              size="sm"
              class="text-muted hover:text-white cursor-pointer"
            >
              Game Servers
            </OuiText>
            <OuiText
              size="sm"
              class="text-muted hover:text-white cursor-pointer"
            >
              Web Hosting
            </OuiText>
            <OuiText
              size="sm"
              class="text-muted hover:text-white cursor-pointer"
            >
              Pricing
            </OuiText>
          </OuiStack>
        </OuiStack>

        <!-- Support -->
        <OuiStack gap="md">
          <OuiText size="sm" class="font-semibold text-white">Support</OuiText>
          <OuiStack gap="sm">
            <OuiText
              size="sm"
              class="text-muted hover:text-white cursor-pointer"
            >
              Documentation
            </OuiText>
            <OuiText
              size="sm"
              class="text-muted hover:text-white cursor-pointer"
            >
              Contact Us
            </OuiText>
            <OuiText
              size="sm"
              class="text-muted hover:text-white cursor-pointer"
            >
              Status Page
            </OuiText>
            <OuiText
              size="sm"
              class="text-muted hover:text-white cursor-pointer"
            >
              Community
            </OuiText>
          </OuiStack>
        </OuiStack>

        <!-- Company -->
        <OuiStack gap="md">
          <OuiText size="sm" class="font-semibold text-white">Company</OuiText>
          <OuiStack gap="sm">
            <OuiText
              size="sm"
              class="text-muted hover:text-white cursor-pointer"
            >
              About
            </OuiText>
            <OuiText
              size="sm"
              class="text-muted hover:text-white cursor-pointer"
            >
              Privacy
            </OuiText>
            <OuiText
              size="sm"
              class="text-muted hover:text-white cursor-pointer"
            >
              Terms
            </OuiText>
          </OuiStack>
        </OuiStack>
      </OuiGrid>

      <div class="border-t border-surface-muted/20 mt-12 pt-8">
        <OuiFlex
          justify="between"
          align="center"
          class="flex-col md:flex-row gap-4"
        >
          <OuiText size="sm" class="text-muted">
            © 2024 Obiente Cloud. All rights reserved.
          </OuiText>
          <OuiFlex align="center" gap="lg">
            <OuiText
              size="sm"
              class="text-muted hover:text-white cursor-pointer"
            >
              Privacy Policy
            </OuiText>
            <OuiText
              size="sm"
              class="text-muted hover:text-white cursor-pointer"
            >
              Terms of Service
            </OuiText>
          </OuiFlex>
        </OuiFlex>
      </div>
    </OuiContainer>
  </footer>
</template>
