<template>
  <div class="max-w-4xl mx-auto p-8 space-y-12">
    <!-- Page Header -->
    <div class="mb-8">
      <OuiText as="h1" size="4xl" weight="bold" color="primary"
        >Typography Showcase</OuiText
      >
      <OuiText color="secondary" class="mt-2"
        >Demonstrating the OUI Text component variants</OuiText
      >
    </div>

    <!-- Size Variants -->
    <section class="space-y-6">
      <OuiText as="h2" size="2xl" weight="semibold" color="primary"
        >Size Variants</OuiText
      >
      <div class="space-y-4">
        <div class="grid gap-4">
          <div class="flex items-center gap-4">
            <OuiText size="xs" color="muted" class="w-12">xs</OuiText>
            <OuiText size="xs"
              >Extra small text for captions and fine print</OuiText
            >
          </div>
          <div class="flex items-center gap-4">
            <OuiText size="sm" color="muted" class="w-12">sm</OuiText>
            <OuiText size="sm">Small text for secondary information</OuiText>
          </div>
          <div class="flex items-center gap-4">
            <OuiText size="md" color="muted" class="w-12">md</OuiText>
            <OuiText size="md"
              >Base text for body content and paragraphs</OuiText
            >
          </div>
          <div class="flex items-center gap-4">
            <OuiText size="lg" color="muted" class="w-12">lg</OuiText>
            <OuiText size="lg">Large text for emphasized content</OuiText>
          </div>
          <div class="flex items-center gap-4">
            <OuiText size="xl" color="muted" class="w-12">xl</OuiText>
            <OuiText size="xl">Extra large text for subheadings</OuiText>
          </div>
          <div class="flex items-center gap-4">
            <OuiText size="2xl" color="muted" class="w-12">2xl</OuiText>
            <OuiText size="2xl">2XL text for main headings</OuiText>
          </div>
          <div class="flex items-center gap-4">
            <OuiText size="3xl" color="muted" class="w-12">3xl</OuiText>
            <OuiText size="3xl">3XL text for page titles</OuiText>
          </div>
        </div>
      </div>
    </section>

    <!-- Weight Variants -->
    <section class="space-y-6">
      <OuiText as="h2" size="2xl" weight="semibold" color="primary"
        >Weight Variants</OuiText
      >
      <div class="space-y-4">
        <div class="grid gap-4">
          <div class="flex items-center gap-4">
            <OuiText color="muted" class="w-20">light</OuiText>
            <OuiText weight="light"
              >Light weight text for subtle emphasis</OuiText
            >
          </div>
          <div class="flex items-center gap-4">
            <OuiText color="muted" class="w-20">normal</OuiText>
            <OuiText weight="normal"
              >Normal weight text for body content</OuiText
            >
          </div>
          <div class="flex items-center gap-4">
            <OuiText color="muted" class="w-20">medium</OuiText>
            <OuiText weight="medium">Medium weight text for emphasis</OuiText>
          </div>
          <div class="flex items-center gap-4">
            <OuiText color="muted" class="w-20">semibold</OuiText>
            <OuiText weight="semibold">Semibold text for headings</OuiText>
          </div>
          <div class="flex items-center gap-4">
            <OuiText color="muted" class="w-20">bold</OuiText>
            <OuiText weight="bold">Bold text for strong emphasis</OuiText>
          </div>
          <div class="flex items-center gap-4">
            <OuiText color="muted" class="w-20">extrabold</OuiText>
            <OuiText weight="extrabold"
              >Extra bold text for maximum impact</OuiText
            >
          </div>
        </div>
      </div>
    </section>

    <!-- Color Variants -->
    <section class="space-y-6">
      <OuiText as="h2" size="2xl" weight="semibold" color="primary"
        >Color Variants</OuiText
      >
      <div class="space-y-4">
        <div class="grid gap-4">
          <div class="flex items-center gap-4">
            <OuiText color="muted" class="w-20">primary</OuiText>
            <OuiText color="primary"
              >Primary text color for main content</OuiText
            >
          </div>
          <div class="flex items-center gap-4">
            <OuiText color="muted" class="w-20">secondary</OuiText>
            <OuiText color="secondary"
              >Secondary text color for supporting content</OuiText
            >
          </div>
          <div class="flex items-center gap-4">
            <OuiText color="muted" class="w-20">muted</OuiText>
            <OuiText color="muted"
              >Muted text color for subtle information</OuiText
            >
          </div>
          <div class="flex items-center gap-4">
            <OuiText color="muted" class="w-20">accent</OuiText>
            <OuiText color="accent">Accent text color for highlights</OuiText>
          </div>
          <div class="flex items-center gap-4">
            <OuiText color="muted" class="w-20">success</OuiText>
            <OuiText color="success"
              >Success text color for positive states</OuiText
            >
          </div>
          <div class="flex items-center gap-4">
            <OuiText color="muted" class="w-20">warning</OuiText>
            <OuiText color="warning"
              >Warning text color for cautionary states</OuiText
            >
          </div>
          <div class="flex items-center gap-4">
            <OuiText color="muted" class="w-20">danger</OuiText>
            <OuiText color="danger">Danger text color for error states</OuiText>
          </div>
        </div>
      </div>
    </section>

    <!-- Typography Helpers -->
    <section class="space-y-6">
      <OuiText as="h2" size="2xl" weight="semibold" color="primary"
        >Typography Helpers</OuiText
      >
      <div class="space-y-6">
        <div>
          <OuiText color="muted" size="sm" class="mb-2"
            >Display Text (Large headlines)</OuiText
          >
          <OuiText as="h1" size="4xl" weight="bold" class="oui-text-display"
            >The Future of Cloud</OuiText
          >
        </div>
        <div>
          <OuiText color="muted" size="sm" class="mb-2">Heading Text</OuiText>
          <OuiText as="h2" size="2xl" class="oui-text-heading"
            >Section Heading</OuiText
          >
        </div>
        <div>
          <OuiText color="muted" size="sm" class="mb-2">Body Text</OuiText>
          <OuiText class="oui-text-body"
            >This is body text that would be used for paragraphs and longer
            content. It has proper line height and spacing for optimal
            readability.</OuiText
          >
        </div>
        <div>
          <OuiText color="muted" size="sm" class="mb-2">Caption Text</OuiText>
          <OuiText class="oui-text-caption"
            >Small caption text for image descriptions and metadata</OuiText
          >
        </div>
        <div>
          <OuiText color="muted" size="sm" class="mb-2">Overline Text</OuiText>
          <OuiText class="oui-text-overline">Category Label</OuiText>
        </div>
      </div>
    </section>

    <!-- Special Features -->
    <section class="space-y-6">
      <OuiText as="h2" size="2xl" weight="semibold" color="primary"
        >Special Features</OuiText
      >
      <div class="space-y-6">
        <div>
          <OuiText color="muted" size="sm" class="mb-2">Truncated Text</OuiText>
          <div class="max-w-64">
            <OuiText truncate
              >This is a very long line of text that will be truncated with an
              ellipsis when it exceeds the container width</OuiText
            >
          </div>
        </div>
        <div>
          <OuiText color="muted" size="sm" class="mb-2">Text Alignment</OuiText>
          <div class="space-y-2">
            <OuiText align="left">Left aligned text</OuiText>
            <OuiText align="center">Center aligned text</OuiText>
            <OuiText align="right">Right aligned text</OuiText>
          </div>
        </div>
        <div>
          <OuiText color="muted" size="sm" class="mb-2">Text Transform</OuiText>
          <div class="space-y-2">
            <OuiText transform="uppercase">Uppercase text</OuiText>
            <OuiText transform="lowercase">LOWERCASE TEXT</OuiText>
            <OuiText transform="capitalize">capitalize each word</OuiText>
          </div>
        </div>
        <div>
          <OuiText color="muted" size="sm" class="mb-2"
            >Text Decoration</OuiText
          >
          <div class="space-y-2">
            <OuiText decoration="underline">Underlined text</OuiText>
            <OuiText decoration="line-through">Strikethrough text</OuiText>
          </div>
        </div>
        <div>
          <OuiText color="muted" size="sm" class="mb-2"
            >Line Height Variants</OuiText
          >
          <div class="space-y-4 max-w-lg">
            <div>
              <OuiText color="muted" size="xs" class="mb-1"
                >Tight leading</OuiText
              >
              <OuiText leading="tight"
                >This text has tight line height which is useful for headings
                and compact layouts where space is at a premium.</OuiText
              >
            </div>
            <div>
              <OuiText color="muted" size="xs" class="mb-1"
                >Normal leading</OuiText
              >
              <OuiText leading="normal"
                >This text has normal line height which provides good
                readability for most content types and general use
                cases.</OuiText
              >
            </div>
            <div>
              <OuiText color="muted" size="xs" class="mb-1"
                >Relaxed leading</OuiText
              >
              <OuiText leading="relaxed"
                >This text has relaxed line height which improves readability
                for longer paragraphs and content that needs more breathing
                room.</OuiText
              >
            </div>
          </div>
        </div>
      </div>
    </section>

    <!-- Semantic Usage -->
    <section class="space-y-6">
      <OuiText as="h2" size="2xl" weight="semibold" color="primary"
        >Semantic Usage Examples</OuiText
      >
      <div class="space-y-6">
        <div class="p-6 border border-border-muted rounded-lg">
          <OuiText as="h3" size="lg" weight="semibold" class="mb-4"
            >Article Title</OuiText
          >
          <OuiText color="muted" size="sm" class="mb-4"
            >Published on March 15, 2024 by John Doe</OuiText
          >
          <OuiText class="mb-4"
            >This is the article introduction paragraph that provides an
            overview of the content. It uses the default body text styling for
            optimal readability.</OuiText
          >
          <OuiText class="mb-2"
            >This is another paragraph that continues the article content.
            Notice how the line height and spacing create a comfortable reading
            experience.</OuiText
          >
          <OuiText
            color="accent"
            decoration="underline"
            as="a"
            href="#"
            class="inline-block"
            >Read more</OuiText
          >
        </div>
      </div>
    </section>
  </div>
</template>

<script setup lang="ts">
// Import OUI components
import OuiText from "../components/oui/Text.vue";

// Page meta
definePageMeta({
  layout: "default",
});
</script>
