<template>
  <OuiContainer size="5xl" py="xl">
    <OuiStack gap="3xl">
      <!-- Page Header -->
      <OuiStack gap="xs">
        <OuiText as="h1" size="4xl" weight="bold" color="primary"
          >Typography Showcase</OuiText
        >
        <OuiText color="secondary"
          >Demonstrating the OUI Text component variants</OuiText
        >
      </OuiStack>

      <!-- Size Variants -->
      <OuiContainer>
        <OuiStack gap="lg">
          <OuiText as="h2" size="2xl" weight="semibold" color="primary"
            >Size Variants</OuiText
          >
          <OuiGrid cols="1" gap="md">
            <OuiFlex align="center" gap="md">
              <OuiText size="xs" color="muted" class="w-12">xs</OuiText>
              <OuiText size="xs"
                >Extra small text for captions and fine print</OuiText
              >
            </OuiFlex>
            <OuiFlex align="center" gap="md">
              <OuiText size="sm" color="muted" class="w-12">sm</OuiText>
              <OuiText size="sm">Small text for secondary information</OuiText>
            </OuiFlex>
            <OuiFlex align="center" gap="md">
              <OuiText size="md" color="muted" class="w-12">md</OuiText>
              <OuiText size="md"
                >Base text for body content and paragraphs</OuiText
              >
            </OuiFlex>
            <OuiFlex align="center" gap="md">
              <OuiText size="lg" color="muted" class="w-12">lg</OuiText>
              <OuiText size="lg">Large text for emphasized content</OuiText>
            </OuiFlex>
            <OuiFlex align="center" gap="md">
              <OuiText size="xl" color="muted" class="w-12">xl</OuiText>
              <OuiText size="xl">Extra large text for subheadings</OuiText>
            </OuiFlex>
            <OuiFlex align="center" gap="md">
              <OuiText size="2xl" color="muted" class="w-12">2xl</OuiText>
              <OuiText size="2xl">2XL text for main headings</OuiText>
            </OuiFlex>
            <OuiFlex align="center" gap="md">
              <OuiText size="3xl" color="muted" class="w-12">3xl</OuiText>
              <OuiText size="3xl">3XL text for page titles</OuiText>
            </OuiFlex>
          </OuiGrid>
        </OuiStack>
      </OuiContainer>

      <!-- Weight Variants -->
      <OuiContainer>
        <OuiStack gap="lg">
          <OuiText as="h2" size="2xl" weight="semibold" color="primary"
            >Weight Variants</OuiText
          >
          <OuiGrid cols="1" gap="md">
            <OuiFlex align="center" gap="md">
              <OuiText color="muted" class="w-20">light</OuiText>
              <OuiText weight="light"
                >Light weight text for subtle emphasis</OuiText
              >
            </OuiFlex>
            <OuiFlex align="center" gap="md">
              <OuiText color="muted" class="w-20">normal</OuiText>
              <OuiText weight="normal"
                >Normal weight text for body content</OuiText
              >
            </OuiFlex>
            <OuiFlex align="center" gap="md">
              <OuiText color="muted" class="w-20">medium</OuiText>
              <OuiText weight="medium">Medium weight text for emphasis</OuiText>
            </OuiFlex>
            <OuiFlex align="center" gap="md">
              <OuiText color="muted" class="w-20">semibold</OuiText>
              <OuiText weight="semibold">Semibold text for headings</OuiText>
            </OuiFlex>
            <OuiFlex align="center" gap="md">
              <OuiText color="muted" class="w-20">bold</OuiText>
              <OuiText weight="bold">Bold text for strong emphasis</OuiText>
            </OuiFlex>
            <OuiFlex align="center" gap="md">
              <OuiText color="muted" class="w-20">extrabold</OuiText>
              <OuiText weight="extrabold"
                >Extra bold text for maximum impact</OuiText
              >
            </OuiFlex>
          </OuiGrid>
        </OuiStack>
      </OuiContainer>

      <!-- Color Variants -->
      <OuiContainer>
        <OuiStack gap="lg">
          <OuiText as="h2" size="2xl" weight="semibold" color="primary"
            >Color Variants</OuiText
          >
          <OuiGrid cols="1" gap="md">
            <OuiFlex align="center" gap="md">
              <OuiText color="muted" class="w-20">primary</OuiText>
              <OuiText color="primary"
                >Primary text color for main content</OuiText
              >
            </OuiFlex>
            <OuiFlex align="center" gap="md">
              <OuiText color="muted" class="w-20">secondary</OuiText>
              <OuiText color="secondary"
                >Secondary text color for supporting content</OuiText
              >
            </OuiFlex>
            <OuiFlex align="center" gap="md">
              <OuiText color="muted" class="w-20">muted</OuiText>
              <OuiText color="muted"
                >Muted text color for subtle information</OuiText
              >
            </OuiFlex>
            <OuiFlex align="center" gap="md">
              <OuiText color="muted" class="w-20">accent</OuiText>
              <OuiText color="accent">Accent text color for highlights</OuiText>
            </OuiFlex>
            <OuiFlex align="center" gap="md">
              <OuiText color="muted" class="w-20">success</OuiText>
              <OuiText color="success"
                >Success text color for positive states</OuiText
              >
            </OuiFlex>
            <OuiFlex align="center" gap="md">
              <OuiText color="muted" class="w-20">warning</OuiText>
              <OuiText color="warning"
                >Warning text color for cautionary states</OuiText
              >
            </OuiFlex>
            <OuiFlex align="center" gap="md">
              <OuiText color="muted" class="w-20">danger</OuiText>
              <OuiText color="danger"
                >Danger text color for error states</OuiText
              >
            </OuiFlex>
          </OuiGrid>
        </OuiStack>
      </OuiContainer>

      <!-- Typography Helpers -->
      <OuiContainer>
        <OuiStack gap="lg">
          <OuiText as="h2" size="2xl" weight="semibold" color="primary"
            >Typography Helpers</OuiText
          >
          <OuiStack gap="lg">
            <OuiStack gap="xs">
              <OuiText color="muted" size="sm"
                >Display Text (Large headlines)</OuiText
              >
              <OuiText as="h1" size="4xl" weight="bold" class="oui-text-display"
                >The Future of Cloud</OuiText
              >
            </OuiStack>
            <OuiStack gap="xs">
              <OuiText color="muted" size="sm">Heading Text</OuiText>
              <OuiText as="h2" size="2xl" class="oui-text-heading"
                >Section Heading</OuiText
              >
            </OuiStack>
            <OuiStack gap="xs">
              <OuiText color="muted" size="sm">Body Text</OuiText>
              <OuiText class="oui-text-body"
                >This is body text that would be used for paragraphs and longer
                content. It has proper line height and spacing for optimal
                readability.</OuiText
              >
            </OuiStack>
            <OuiStack gap="xs">
              <OuiText color="muted" size="sm">Caption Text</OuiText>
              <OuiText class="oui-text-caption"
                >Small caption text for image descriptions and metadata</OuiText
              >
            </OuiStack>
            <OuiStack gap="xs">
              <OuiText color="muted" size="sm">Overline Text</OuiText>
              <OuiText class="oui-text-overline">Category Label</OuiText>
            </OuiStack>
          </OuiStack>
        </OuiStack>
      </OuiContainer>

      <!-- Special Features -->
      <OuiContainer>
        <OuiStack gap="lg">
          <OuiText as="h2" size="2xl" weight="semibold" color="primary"
            >Special Features</OuiText
          >
          <OuiStack gap="lg">
            <OuiStack gap="xs">
              <OuiText color="muted" size="sm">Truncated Text</OuiText>
              <OuiContainer size="xs" class="max-w-64">
                <OuiText truncate
                  >This is a very long line of text that will be truncated with
                  an ellipsis when it exceeds the container width</OuiText
                >
              </OuiContainer>
            </OuiStack>
            <OuiStack gap="xs">
              <OuiText color="muted" size="sm">Text Alignment</OuiText>
              <OuiStack gap="xs">
                <OuiText align="left">Left aligned text</OuiText>
                <OuiText align="center">Center aligned text</OuiText>
                <OuiText align="right">Right aligned text</OuiText>
              </OuiStack>
            </OuiStack>
            <OuiStack gap="xs">
              <OuiText color="muted" size="sm">Text Transform</OuiText>
              <OuiStack gap="xs">
                <OuiText transform="uppercase">Uppercase text</OuiText>
                <OuiText transform="lowercase">LOWERCASE TEXT</OuiText>
                <OuiText transform="capitalize">capitalize each word</OuiText>
              </OuiStack>
            </OuiStack>
            <OuiStack gap="xs">
              <OuiText color="muted" size="sm">Text Decoration</OuiText>
              <OuiStack gap="xs">
                <OuiText decoration="underline">Underlined text</OuiText>
                <OuiText decoration="line-through">Strikethrough text</OuiText>
              </OuiStack>
            </OuiStack>
            <OuiStack gap="xs">
              <OuiText color="muted" size="sm">Line Height Variants</OuiText>
              <OuiStack gap="md" class="max-w-lg">
                <OuiStack gap="xs">
                  <OuiText color="muted" size="xs">Tight leading</OuiText>
                  <OuiText leading="tight"
                    >This text has tight line height which is useful for
                    headings and compact layouts where space is at a
                    premium.</OuiText
                  >
                </OuiStack>
                <OuiStack gap="xs">
                  <OuiText color="muted" size="xs">Normal leading</OuiText>
                  <OuiText leading="normal"
                    >This text has normal line height which provides good
                    readability for most content types and general use
                    cases.</OuiText
                  >
                </OuiStack>
                <OuiStack gap="xs">
                  <OuiText color="muted" size="xs">Relaxed leading</OuiText>
                  <OuiText leading="relaxed"
                    >This text has relaxed line height which improves
                    readability for longer paragraphs and content that needs
                    more breathing room.</OuiText
                  >
                </OuiStack>
              </OuiStack>
            </OuiStack>
          </OuiStack>
        </OuiStack>
      </OuiContainer>

      <!-- Semantic Usage -->
      <OuiContainer>
        <OuiStack gap="lg">
          <OuiText as="h2" size="2xl" weight="semibold" color="primary"
            >Semantic Usage Examples</OuiText
          >
          <OuiCard variant="outline" class="p-6">
            <OuiStack gap="md">
              <OuiText as="h3" size="lg" weight="semibold"
                >Article Title</OuiText
              >
              <OuiText color="muted" size="sm"
                >Published on March 15, 2024 by John Doe</OuiText
              >
              <OuiText
                >This is the article introduction paragraph that provides an
                overview of the content. It uses the default body text styling
                for optimal readability.</OuiText
              >
              <OuiText
                >This is another paragraph that continues the article content.
                Notice how the line height and spacing create a comfortable
                reading experience.</OuiText
              >
              <OuiText
                color="accent"
                decoration="underline"
                as="a"
                href="#"
                class="inline-block"
                >Read more</OuiText
              >
            </OuiStack>
          </OuiCard>
        </OuiStack>
      </OuiContainer>
    </OuiStack>
  </OuiContainer>
</template>

<script setup lang="ts">
  // Import OUI components
  import OuiText from "../components/oui/Text.vue";

  // Page meta
  definePageMeta({
    layout: "default",
  });
</script>
