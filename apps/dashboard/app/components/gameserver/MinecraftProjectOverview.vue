<template>
  <OuiStack gap="lg" class="p-4">
    <!-- Header with icon and title -->
    <OuiFlex gap="md" align="start">
      <OuiAvatar
        size="xl"
        :src="displayProject.iconUrl || undefined"
        class="bg-surface-muted shrink-0"
        :alt="displayProject.title"
      >
        <CubeIcon v-if="!displayProject.iconUrl" class="w-8 h-8 text-secondary" />
      </OuiAvatar>
      <OuiStack gap="sm" class="flex-1 min-w-0">
        <OuiFlex align="center" gap="sm" wrap="wrap">
          <OuiText as="h2" size="xl" weight="bold" truncate>{{ displayProject.title }}</OuiText>
          <OuiBadge size="sm" variant="secondary">
            {{ formatProjectType(displayProject.projectType) }}
          </OuiBadge>
        </OuiFlex>
        <OuiText size="sm" color="secondary" v-if="displayProject.authors && displayProject.authors.length > 0">
          by {{ displayProject.authors.join(", ") }}
        </OuiText>
      </OuiStack>
    </OuiFlex>

    <!-- Screenshots/Gallery -->
    <div v-if="displayProject.gallery && displayProject.gallery.length > 0">
      <OuiText size="sm" weight="semibold" class="mb-2">Screenshots</OuiText>
      <OuiGrid cols="1" cols-md="2" cols-lg="3" gap="md">
        <div
          v-for="(imageUrl, index) in displayProject.gallery"
          :key="index"
          class="relative aspect-video overflow-hidden rounded-lg border border-border-default bg-surface-muted cursor-pointer group"
          @click="openImageLightbox(imageUrl, index)"
        >
          <img
            :src="imageUrl"
            :alt="`${displayProject.title} screenshot ${index + 1}`"
            class="w-full h-full object-cover transition-transform duration-200 group-hover:scale-105"
            loading="lazy"
          />
        </div>
      </OuiGrid>
    </div>

    <!-- Stats Grid -->
    <OuiGrid cols="2" cols-md="4" gap="md">
      <OuiCard variant="default">
        <OuiCardBody>
          <OuiStack gap="xs">
            <OuiText size="xs" color="secondary">Downloads</OuiText>
            <OuiText size="lg" weight="bold">{{ formatDownloads(project.downloads) }}</OuiText>
          </OuiStack>
        </OuiCardBody>
      </OuiCard>
      <OuiCard variant="default" v-if="project.rating">
        <OuiCardBody>
          <OuiStack gap="xs">
            <OuiText size="xs" color="secondary">Rating</OuiText>
            <OuiText size="lg" weight="bold">{{ project.rating.toFixed(1) }} / 5</OuiText>
          </OuiStack>
        </OuiCardBody>
      </OuiCard>
      <OuiCard variant="default" v-if="project.gameVersions && project.gameVersions.length > 0">
        <OuiCardBody>
          <OuiStack gap="xs">
            <OuiText size="xs" color="secondary">Versions</OuiText>
            <OuiText size="lg" weight="bold">{{ project.gameVersions.length }}</OuiText>
          </OuiStack>
        </OuiCardBody>
      </OuiCard>
      <OuiCard variant="default" v-if="activeVersionFilter && project.gameVersions && project.gameVersions.length > 0">
        <OuiCardBody>
          <OuiStack gap="xs">
            <OuiText size="xs" color="secondary">Compatibility</OuiText>
            <OuiBadge :variant="getCompatibilityStatus(project).variant" size="sm">
              {{ getCompatibilityStatus(project).label }}
            </OuiBadge>
          </OuiStack>
        </OuiCardBody>
      </OuiCard>
    </OuiGrid>

    <!-- Loaders -->
    <div v-if="project.loaders && project.loaders.length > 0">
      <OuiText size="sm" weight="semibold" class="mb-2">Loaders</OuiText>
      <OuiFlex gap="sm" wrap="wrap">
        <OuiBadge v-for="loader in project.loaders" :key="loader" size="sm" variant="secondary">
          {{ loader }}
        </OuiBadge>
      </OuiFlex>
    </div>

    <!-- Game Versions -->
    <div v-if="project.gameVersions && project.gameVersions.length > 0">
      <OuiText size="sm" weight="semibold" class="mb-2">Supported Minecraft Versions</OuiText>
      <OuiFlex gap="sm" wrap="wrap">
        <OuiBadge v-for="version in sortedGameVersions.slice(0, 20)" :key="version" size="sm" variant="outline">
          {{ version }}
        </OuiBadge>
        <OuiText v-if="sortedGameVersions.length > 20" size="xs" color="secondary" class="self-center">
          +{{ sortedGameVersions.length - 20 }} more
        </OuiText>
      </OuiFlex>
    </div>

    <!-- Categories -->
    <div v-if="project.categories && project.categories.length > 0">
      <OuiText size="sm" weight="semibold" class="mb-2">Categories</OuiText>
      <OuiFlex gap="sm" wrap="wrap">
        <OuiBadge v-for="category in project.categories" :key="category" size="sm" variant="secondary">
          {{ category }}
        </OuiBadge>
      </OuiFlex>
    </div>

    <!-- Links -->
    <OuiFlex gap="sm" wrap="wrap" v-if="project.projectUrl || project.sourceUrl || project.issuesUrl">
      <OuiButton
        v-if="project.projectUrl"
        variant="outline"
        size="sm"
        as="a"
        :href="project.projectUrl"
        target="_blank"
        rel="noopener noreferrer"
        class="gap-2"
      >
        <GlobeAltIcon class="w-4 h-4" />
        View on Modrinth
      </OuiButton>
      <OuiButton
        v-if="project.sourceUrl"
        variant="outline"
        size="sm"
        as="a"
        :href="project.sourceUrl"
        target="_blank"
        rel="noopener noreferrer"
        class="gap-2"
      >
        <CodeBracketIcon class="w-4 h-4" />
        Source Code
      </OuiButton>
      <OuiButton
        v-if="project.issuesUrl"
        variant="outline"
        size="sm"
        as="a"
        :href="project.issuesUrl"
        target="_blank"
        rel="noopener noreferrer"
        class="gap-2"
      >
        <ExclamationTriangleIcon class="w-4 h-4" />
        Report Issue
      </OuiButton>
    </OuiFlex>

    <!-- Full Description/Body at the bottom -->
    <div v-if="displayProject.body || displayProject.description">
      <OuiText size="sm" weight="semibold" class="mb-2">Description</OuiText>
      <div v-if="isLoadingFullDetails" class="flex items-center gap-2 text-text-secondary">
        <OuiSkeleton class="h-4 w-20" />
        <OuiText size="xs">Loading full details...</OuiText>
      </div>
      <div v-else class="prose prose-sm dark:prose-invert max-w-none text-text-secondary [&_a]:text-primary [&_a]:hover:underline [&_a]:no-underline [&_img]:max-w-full [&_img]:rounded-lg [&_img]:my-2">
        <div v-html="renderedBody"></div>
      </div>
    </div>

    <!-- Image Lightbox -->
    <OuiDialog
      v-model="isLightboxOpen"
      :title="`${displayProject.title} - Screenshot`"
      @close="closeImageLightbox"
      class="max-w-7xl"
    >
      <div v-if="lightboxImage" class="relative">
        <button
          @click="closeImageLightbox"
          class="absolute top-4 right-4 z-10 p-2 rounded-lg bg-surface-base/90 backdrop-blur border border-border-default hover:bg-surface-muted transition-colors"
          aria-label="Close lightbox"
        >
          <XMarkIcon class="w-5 h-5" />
        </button>
        <img
          :src="lightboxImage"
          :alt="`${displayProject.title} screenshot`"
          class="w-full h-auto rounded-lg"
        />
        <div
          v-if="displayProject.gallery && displayProject.gallery.length > 1"
          class="flex items-center justify-between mt-4"
        >
          <OuiButton
            variant="outline"
            @click="navigateImage('prev')"
            class="gap-2"
          >
            <ChevronLeftIcon class="w-4 h-4" />
            Previous
          </OuiButton>
          <OuiText size="sm" color="secondary">
            {{ lightboxIndex + 1 }} / {{ displayProject.gallery.length }}
          </OuiText>
          <OuiButton
            variant="outline"
            @click="navigateImage('next')"
            class="gap-2"
          >
            Next
            <ChevronRightIcon class="w-4 h-4" />
          </OuiButton>
        </div>
      </div>
    </OuiDialog>
  </OuiStack>
</template>

<script setup lang="ts">
import { computed, ref, onMounted } from "vue";
import {
  CubeIcon,
  GlobeAltIcon,
  CodeBracketIcon,
  ExclamationTriangleIcon,
  XMarkIcon,
  ChevronLeftIcon,
  ChevronRightIcon,
} from "@heroicons/vue/24/outline";
import type { MinecraftProject, MinecraftProjectType } from "@obiente/proto";
import { MinecraftProjectType as ProjectTypeEnum } from "@obiente/proto";
import { GameServerService } from "@obiente/proto";
import { useConnectClient } from "~/lib/connect-client";
import { marked, type Renderer, type Tokens } from "marked";

interface Props {
  project: MinecraftProject;
  activeVersionFilter?: string;
  isInstalling?: boolean;
  gameServerId?: string;
}

const props = withDefaults(defineProps<Props>(), {
  isInstalling: false,
});

defineEmits<{
  install: [];
}>();

const client = useConnectClient(GameServerService);
const isLoadingFullDetails = ref(false);
const fullProject = ref<MinecraftProject | null>(null);

const lightboxImage = ref<string | null>(null);
const lightboxIndex = ref<number>(0);

// Use full project details if available, otherwise use the basic project
const displayProject = computed(() => fullProject.value || props.project);

// Computed property for dialog open state
const isLightboxOpen = computed({
  get: () => lightboxImage.value !== null,
  set: (value: boolean) => {
    if (!value) {
      lightboxImage.value = null;
    }
  },
});

// Fetch full project details if body/gallery are not available
onMounted(async () => {
  // Only fetch if we don't have body or gallery and we have a gameServerId
  if (props.gameServerId && (!props.project.body || !props.project.gallery || props.project.gallery.length === 0)) {
    isLoadingFullDetails.value = true;
    try {
      const response = await client.getMinecraftProject({
        gameServerId: props.gameServerId,
        projectId: props.project.id,
      });
      if (response.project) {
        fullProject.value = response.project;
      }
    } catch (error) {
      // Silently fail - just use the basic project data
    } finally {
      isLoadingFullDetails.value = false;
    }
  }
});

// Configure marked
marked.setOptions({
  breaks: true,
  gfm: true,
});

// Custom renderer to add target="_blank" to links
const createMarkdownRenderer = (): Renderer => {
  const renderer = new marked.Renderer();
  const originalLinkRenderer = renderer.link.bind(renderer);
  renderer.link = (token: Tokens.Link) => {
    const html = originalLinkRenderer(token);
    return html.replace(/^<a /, '<a target="_blank" rel="noopener noreferrer" ');
  };
  return renderer;
};

const markdownRenderer = createMarkdownRenderer();

const renderedBody = computed(() => {
  const body = displayProject.value.body || displayProject.value.description;
  if (!body) return "";
  
  // Check if content is clearly HTML (has complete HTML tags with attributes and closing tags)
  // This is a more strict check - we want to see actual HTML structure
  const hasCompleteHTMLTags = /<[a-z]+[^>]*\s[^>]*>.*?<\/[a-z]+>/i.test(body) || 
                               /<[a-z]+[^>]*\/\s*>/i.test(body) ||
                               /<[a-z]+[^>]*>[\s\S]*<\/[a-z]+>/i.test(body);
  
  // Check for markdown patterns (more comprehensive)
  const hasMarkdownPatterns = 
    /\[[^\]]+\]\([^)]+\)/.test(body) ||  // [text](url) links
    /^#{1,6}\s/m.test(body) ||            // Headers
    /^[-*+]\s/m.test(body) ||             // Lists
    /\|.*\|/m.test(body) ||               // Tables
    /!\[.*?\]\(.*?\)/.test(body);         // Images
  
  // If it has markdown patterns, always try to render as markdown first
  // Only treat as HTML if it has complete HTML tags AND no markdown patterns
  if (hasMarkdownPatterns || !hasCompleteHTMLTags) {
    // Try markdown rendering
    try {
      const html = marked.parse(body, { renderer: markdownRenderer }) as string;
      // Verify we got actual HTML back (not just the input)
      if (html && html !== body && html.includes('<')) {
        return html;
      }
    } catch {
      // Markdown parsing failed, will fall through to HTML or plain text handling
    }
  }
  
  // If markdown rendering failed or content is clearly HTML, process as HTML
  if (hasCompleteHTMLTags) {
    // It's already HTML, just ensure links open in new tabs
    return body.replace(/<a\s+([^>]*?)>/gi, (match, attrs) => {
      if (!attrs.includes('href=')) {
        return match;
      }
      
      // Update target
      if (attrs.includes('target=')) {
        attrs = attrs.replace(/target=["'][^"']*["']/gi, 'target="_blank"');
      } else {
        attrs = attrs.trim() + ' target="_blank"';
      }
      
      // Update rel
      if (attrs.includes('rel=')) {
        attrs = attrs.replace(/rel=["']([^"']*)["']/gi, (_m: string, rel: string) => {
          const rels = rel.split(' ').filter((r: string) => r && r !== 'noopener' && r !== 'noreferrer');
          return `rel="${[...rels, 'noopener', 'noreferrer'].join(' ')}"`;
        });
      } else {
        attrs = attrs.trim() + ' rel="noopener noreferrer"';
      }
      
      return `<a ${attrs}>`;
    });
  }
  
  // Final fallback: plain text with line breaks
  return body.replace(/\n/g, '<br />');
});

function openImageLightbox(imageUrl: string, index: number) {
  lightboxImage.value = imageUrl;
  lightboxIndex.value = index;
}

function closeImageLightbox() {
  lightboxImage.value = null;
}

function navigateImage(direction: "prev" | "next") {
  const gallery = displayProject.value.gallery;
  if (!gallery || gallery.length === 0) return;
  
  if (direction === "next") {
    lightboxIndex.value = (lightboxIndex.value + 1) % gallery.length;
  } else {
    lightboxIndex.value = lightboxIndex.value === 0 ? gallery.length - 1 : lightboxIndex.value - 1;
  }
  const imageUrl = gallery[lightboxIndex.value];
  if (imageUrl) {
    lightboxImage.value = imageUrl;
  }
}

const sortedGameVersions = computed(() => {
  if (!props.project.gameVersions || props.project.gameVersions.length === 0) {
    return [];
  }

  if (!props.activeVersionFilter) {
    // If no server version, just return versions as-is
    return [...props.project.gameVersions];
  }

  const serverVersion = props.activeVersionFilter;
  const serverParts = parseVersion(serverVersion);

  // Calculate distance from server version for each project version
  const versionsWithDistance = props.project.gameVersions.map((version) => {
    const versionParts = parseVersion(version);
    let distance = 0;

    // Calculate distance: prioritize exact matches, then closest versions
    // Exact match gets distance 0
    const normalized = version.replace(/^v/i, "");
    if (normalized === serverVersion || compareVersions(normalized, serverVersion) === 0) {
      return { version, distance: 0 };
    }

    // Calculate distance based on version parts
    // Major version difference gets high weight
    // Minor version difference gets medium weight
    // Patch version difference gets low weight
    const maxLen = Math.max(serverParts.length, versionParts.length);
    for (let i = 0; i < maxLen; i++) {
      const serverPart = serverParts[i] || 0;
      const versionPart = versionParts[i] || 0;
      const diff = Math.abs(serverPart - versionPart);
      
      if (i === 0) {
        // Major version: multiply by 10000
        distance += diff * 10000;
      } else if (i === 1) {
        // Minor version: multiply by 100
        distance += diff * 100;
      } else {
        // Patch version: add directly
        distance += diff;
      }
    }

    return { version, distance };
  });

  // Sort by distance (closest first), then by version number (newer first for same distance)
  versionsWithDistance.sort((a, b) => {
    if (a.distance !== b.distance) {
      return a.distance - b.distance;
    }
    // If same distance, prefer newer versions
    return compareVersions(b.version, a.version);
  });

  return versionsWithDistance.map((v) => v.version);
});

function formatProjectType(type: MinecraftProjectType | undefined) {
  if (type === ProjectTypeEnum.PLUGIN) {
    return "Plugin";
  }
  return "Mod";
}

function formatDownloads(downloads?: bigint | number | null) {
  const value = typeof downloads === "bigint" ? Number(downloads) : downloads || 0;
  if (value >= 1_000_000) return `${(value / 1_000_000).toFixed(1)}M`;
  if (value >= 1_000) return `${(value / 1_000).toFixed(1)}k`;
  return value.toString();
}

function parseVersion(version: string): number[] {
  return version
    .replace(/^v/i, "")
    .split(".")
    .map((v) => parseInt(v, 10) || 0);
}

function compareVersions(v1: string, v2: string): number {
  const parts1 = parseVersion(v1);
  const parts2 = parseVersion(v2);
  const maxLen = Math.max(parts1.length, parts2.length);
  
  for (let i = 0; i < maxLen; i++) {
    const p1 = parts1[i] || 0;
    const p2 = parts2[i] || 0;
    if (p1 < p2) return -1;
    if (p1 > p2) return 1;
  }
  return 0;
}

function getCompatibilityStatus(project: MinecraftProject): { label: string; variant: "success" | "warning" | "secondary"; range?: string } {
  if (!props.activeVersionFilter || !project.gameVersions || project.gameVersions.length === 0) {
    return { label: "Unknown", variant: "secondary" };
  }

  const serverVersion = props.activeVersionFilter;
  const projectVersions = project.gameVersions;

  const isCompatible = projectVersions.some((v) => {
    const normalized = v.replace(/^v/i, "");
    return normalized === serverVersion || compareVersions(normalized, serverVersion) === 0;
  });

  if (isCompatible) {
    return { label: "Compatible", variant: "success" };
  }

  const sortedVersions = [...projectVersions].sort((a, b) => compareVersions(a, b));
  const minVersion = sortedVersions[0];
  const maxVersion = sortedVersions[sortedVersions.length - 1];

  if (!minVersion || !maxVersion) {
    return { label: "Incompatible", variant: "secondary" };
  }

  const serverVsMin = compareVersions(serverVersion, minVersion);
  const serverVsMax = compareVersions(serverVersion, maxVersion);

  if (serverVsMin < 0) {
    return { 
      label: "Newer version", 
      variant: "warning",
      range: `Requires ${minVersion}+`
    };
  } else if (serverVsMax > 0) {
    return { 
      label: "Older version", 
      variant: "warning",
      range: `Up to ${maxVersion}`
    };
  }

  return { label: "Incompatible", variant: "secondary" };
}
</script>

