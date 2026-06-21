<template>
  <OuiStack gap="lg">
    <OuiCard variant="outline" class="border-border-default/60">
      <OuiCardBody class="p-3 space-y-3">
        <OuiFlex wrap="wrap" gap="sm" align="center" justify="between">
          <OuiFlex gap="xs" class="rounded-lg border border-border-muted bg-surface-muted/30 p-1">
            <OuiButton
              size="sm"
              :variant="activeView === 'browse' ? 'soft' : 'ghost'"
              class="gap-2"
              @click="activeView = 'browse'"
            >
              <MagnifyingGlassIcon class="w-4 h-4" />
              Browse
            </OuiButton>
            <OuiButton
              size="sm"
              :variant="activeView === 'installed' ? 'soft' : 'ghost'"
              class="gap-2"
              @click="activeView = 'installed'"
            >
              <FolderIcon class="w-4 h-4" />
              Installed
              <OuiBadge v-if="installedFiles.length" size="xs" variant="secondary">
                {{ installedFiles.length }}
              </OuiBadge>
            </OuiButton>
          </OuiFlex>
          <OuiButton
            v-if="activeView === 'installed' && updateableInstalledFiles.length > 0"
            size="sm"
            color="primary"
            class="gap-2"
            :loading="isBulkUpdating"
            @click="updateAllAvailable"
          >
            <ArrowPathIcon class="w-4 h-4" />
            Update all
          </OuiButton>
        </OuiFlex>

        <template v-if="activeView === 'browse'">
        <!-- Row 1: Search + Type badge -->
        <OuiFlex
          wrap="wrap"
          gap="md"
          align="center"
          class="w-full bg-surface-muted/20 px-4 py-3 rounded-xl border border-border-muted/60 shadow-inner"
        >
          <OuiFlex class="flex-1 min-w-[260px]">
            <OuiInput
              v-model="searchQuery"
              :placeholder="`Search ${projectTypeLabel.toLowerCase()}…`"
              size="sm"
              clearable
            >
              <template #prefix>
                <MagnifyingGlassIcon class="w-4 h-4 text-secondary" />
              </template>
            </OuiInput>
          </OuiFlex>
          <OuiBadge size="sm" variant="secondary">
            <PuzzlePieceIcon v-if="projectType === MinecraftProjectType.PLUGIN" class="w-3.5 h-3.5" />
            <CubeIcon v-else class="w-3.5 h-3.5" />
            {{ projectTypeLabel }}
          </OuiBadge>
        </OuiFlex>

        <!-- Row 2: Filtering controls -->
        <OuiFlex
          wrap="wrap"
          gap="md"
          align="center"
          class="w-full text-xs bg-surface-muted/5 px-4 py-3 rounded-xl border border-border-muted/40"
        >
          <OuiFlex gap="xs" class="shrink-0">
            <OuiBadge
              :as="'button'"
              size="sm"
              :variant="matchServerVersion ? 'primary' : 'secondary'"
              :tone="matchServerVersion ? 'solid' : 'soft'"
              class="px-3 py-1 cursor-pointer"
              @click="toggleMatchServerVersion"
            >
              Match version
              <template v-if="activeVersionFilter">
                &nbsp;(v{{ activeVersionFilter }})
              </template>
            </OuiBadge>
            <OuiBadge
              :as="'button'"
              size="sm"
              :variant="matchServerType ? 'primary' : 'secondary'"
              :tone="matchServerType ? 'solid' : 'soft'"
              class="px-3 py-1 cursor-pointer"
              @click="toggleMatchServerType"
            >
              Match type
            </OuiBadge>
          </OuiFlex>
          <OuiFlex
            v-if="isModdedServer && activeLoaderFilter"
            align="center"
            gap="xs"
            class="shrink-0 whitespace-nowrap px-2 py-1 rounded-full border border-border-muted bg-surface-muted/40"
          >
            <PuzzlePieceIcon class="w-3 h-3 text-secondary" />
            <OuiText size="xs" color="tertiary">{{ activeLoaderFilter }}</OuiText>
          </OuiFlex>
          <OuiBadge
            v-if="activeVersionFilter"
            size="xs"
            variant="secondary"
            class="shrink-0 cursor-pointer"
            :as="'button'"
            @click="toggleMatchServerVersion"
          >
            v{{ activeVersionFilter }}
          </OuiBadge>
          <OuiBadge
            v-if="activeLoaderFilter && !isModdedServer"
            size="xs"
            variant="secondary"
            class="shrink-0 cursor-pointer"
            :as="'button'"
            @click="toggleMatchServerType"
          >
            {{ activeLoaderFilter }}
          </OuiBadge>
          <OuiFlex v-if="selectedCategories.length" gap="xs" align="center" class="flex-wrap">
            <OuiBadge
              v-for="category in selectedCategories"
              :key="`active-${category}`"
              size="xs"
              variant="secondary"
              tone="soft"
              class="cursor-pointer"
              :as="'button'"
              @click="toggleCategory(category)"
            >
              {{ formatCategory(category) }}
            </OuiBadge>
          </OuiFlex>

          <OuiFlex gap="sm" align="center" class="ml-auto shrink-0 whitespace-nowrap">
            <OuiSelect
              v-model="sortBy"
              :items="sortOptions"
              size="sm"
              class="min-w-[130px]"
            />
            <OuiButton
              variant="ghost"
              size="sm"
              class="gap-1 whitespace-nowrap"
              :loading="isLoading"
              @click="refresh"
            >
              <ArrowPathIcon class="w-3.5 h-3.5" :class="{ 'animate-spin': isLoading }" />
              Refresh
            </OuiButton>
          </OuiFlex>
        </OuiFlex>

        <!-- Row 3: Categories -->
        <OuiFlex
          wrap="nowrap"
          gap="sm"
          align="center"
          class="w-full pt-3 border-t border-border-muted/50"
        >
          <OuiText size="xs" weight="semibold" color="tertiary" class="whitespace-nowrap">
            Categories
          </OuiText>
          <OuiFlex wrap="nowrap" gap="xs" class="flex-1 overflow-x-auto pb-1 min-w-0">
            <OuiBadge
              v-for="category in availableCategories"
              :key="category.value"
              :as="'button'"
              size="xs"
              :variant="selectedCategories.includes(category.value) ? 'primary' : 'secondary'"
              :tone="selectedCategories.includes(category.value) ? 'solid' : 'soft'"
              class="px-2 py-1 whitespace-nowrap cursor-pointer"
              @click="toggleCategory(category.value)"
            >
              {{ category.label }}
            </OuiBadge>
          </OuiFlex>
          <OuiButton
            variant="ghost"
            size="xs"
            class="whitespace-nowrap"
            @click="clearCategoryFilters"
            :disabled="selectedCategories.length === 0"
          >
            Clear
          </OuiButton>
        </OuiFlex>
        </template>

        <template v-else>
          <OuiFlex
            wrap="wrap"
            gap="md"
            align="center"
            class="w-full bg-surface-muted/10 px-4 py-3 rounded-xl border border-border-muted/40"
          >
            <OuiBadge size="sm" variant="secondary">
              <PuzzlePieceIcon v-if="projectType === MinecraftProjectType.PLUGIN" class="w-3.5 h-3.5" />
              <CubeIcon v-else class="w-3.5 h-3.5" />
              {{ projectTypeLabel }}
            </OuiBadge>
            <OuiText size="sm" color="tertiary">
              {{ installedFiles.length }} installed, {{ updateableInstalledFiles.length }} updateable
            </OuiText>
            <OuiButton
              variant="ghost"
              size="sm"
              class="gap-1 whitespace-nowrap ml-auto"
              :loading="isInstalledLoading"
              @click="loadInstalledFiles"
            >
              <ArrowPathIcon class="w-3.5 h-3.5" :class="{ 'animate-spin': isInstalledLoading }" />
              Refresh
            </OuiButton>
          </OuiFlex>
        </template>
      </OuiCardBody>
    </OuiCard>

    <OuiAlert v-if="errorMessage" variant="error" :title="errorMessage">
      <OuiButton variant="ghost" size="sm" @click="refresh">Try again</OuiButton>
    </OuiAlert>
    <OuiAlert v-if="installedErrorMessage" variant="error" :title="installedErrorMessage">
      <OuiButton variant="ghost" size="sm" @click="loadInstalledFiles">Try again</OuiButton>
    </OuiAlert>

    <OuiStack v-if="activeView === 'browse'" gap="lg">
      <OuiGrid
        :cols="{ sm: 1, md: 2, xl: 3 }"
        gap="lg"
        :class="[
          'transition-opacity duration-150',
          { 'opacity-60': isRefreshing && projects.length > 0 },
        ]"
      >
        <OuiCard
          v-for="card in projectCards"
          :key="card.project.id"
          variant="default"
          class="border-border-muted/70 hover:border-border-default transition"
        >
          <OuiCardBody>
            <OuiStack gap="md">
              <OuiFlex gap="md" align="start">
                <OuiAvatar
                  size="lg"
                  :src="card.project.iconUrl || undefined"
                  class="bg-surface-muted shrink-0"
                  :alt="card.project.title"
                >
                  <CubeIcon v-if="!card.project.iconUrl" class="w-5 h-5 text-secondary" />
                </OuiAvatar>
                <OuiStack gap="xs" class="flex-1 min-w-0">
                  <OuiFlex align="center" gap="sm" wrap="wrap">
                    <OuiText as="h3" size="lg" weight="semibold" truncate>{{ card.project.title }}</OuiText>
                    <OuiBadge
                      v-for="typeLabel in card.typeLabels"
                      :key="`project-type-${card.project.id}-${typeLabel}`"
                      size="xs"
                      variant="secondary"
                    >
                      {{ typeLabel }}
                    </OuiBadge>
                    <OuiBadge
                      v-if="card.installedFile"
                      size="xs"
                      :variant="card.installedFile.updateAvailable ? 'warning' : 'success'"
                    >
                      {{ card.installedFile.updateAvailable ? 'Update available' : 'Installed' }}
                    </OuiBadge>
                  </OuiFlex>
                  <OuiText size="sm" color="tertiary" :lineClamp="2">
                    {{ card.project.description || "No description provided." }}
                  </OuiText>
                </OuiStack>
              </OuiFlex>

              <OuiStack gap="xs">
                <OuiFlex
                  v-if="card.visibleLoaders.length > 0"
                  wrap="wrap"
                  gap="xs"
                  align="center"
                >
                  <OuiBadge
                    v-for="loader in card.visibleLoaders"
                    :key="`loader-${card.project.id}-${loader}`"
                    size="xs"
                    variant="primary"
                    tone="soft"
                  >
                    {{ loader }}
                  </OuiBadge>
                  <OuiBadge
                    v-if="card.loaderOverflow > 0"
                    size="xs"
                    variant="primary"
                    tone="soft"
                  >
                    +{{ card.loaderOverflow }}
                  </OuiBadge>
                </OuiFlex>

                <OuiFlex
                  v-if="card.visibleCategories.length > 0"
                  wrap="wrap"
                  gap="xs"
                  align="center"
                >
                  <OuiBadge
                    v-for="category in card.visibleCategories"
                    :key="`category-${card.project.id}-${category}`"
                    size="xs"
                    variant="secondary"
                    tone="soft"
                  >
                    {{ formatCategory(category) }}
                  </OuiBadge>
                  <OuiBadge
                    v-if="card.categoryOverflow > 0"
                    size="xs"
                    variant="secondary"
                    tone="soft"
                  >
                    +{{ card.categoryOverflow }}
                  </OuiBadge>
                </OuiFlex>

                <OuiFlex
                  v-if="card.versionRange"
                  wrap="wrap"
                  gap="xs"
                  align="center"
                >
                  <OuiBadge
                    size="xs"
                    variant="secondary"
                    tone="soft"
                  >
                    {{ card.versionRange }}
                  </OuiBadge>
                </OuiFlex>
              </OuiStack>

              <OuiFlex justify="between" align="center" wrap="wrap" gap="sm">
                <OuiStack gap="xs">
                  <OuiText size="xs" color="tertiary">Downloads</OuiText>
                  <OuiText size="sm" weight="semibold" color="primary">
                    {{ card.downloadsLabel }}
                  </OuiText>
                </OuiStack>
                <OuiStack gap="xs" v-if="activeVersionFilter && card.versionRange">
                  <OuiText size="xs" color="tertiary">Compatibility</OuiText>
                  <OuiFlex gap="xs" align="center">
                    <OuiBadge
                      size="xs"
                      :variant="card.compatibility.variant"
                    >
                      {{ card.compatibility.label }}
                    </OuiBadge>
                    <OuiText v-if="card.compatibility.range" size="xs" color="tertiary">
                      {{ card.compatibility.range }}
                    </OuiText>
                  </OuiFlex>
                </OuiStack>
                <OuiFlex gap="sm" class="ml-auto">
                  <OuiButton
                    size="sm"
                    variant="ghost"
                    class="gap-2"
                    @click="openOverview(card.project)"
                  >
                    <EyeIcon class="w-4 h-4" />
                    Details
                  </OuiButton>
                  <OuiButton
                    size="sm"
                    class="gap-2"
                    variant="soft"
                    :loading="selectedProject?.id === card.project.id && installDialogOpen && isVersionsLoading"
                    @click="openInstallDialog(card.project, card.installedFile || undefined)"
                  >
                    <ArrowPathIcon v-if="card.installedFile?.updateAvailable" class="w-4 h-4" />
                    <CloudArrowDownIcon v-else class="w-4 h-4" />
                    {{ card.installedFile ? 'Update' : 'Install' }}
                  </OuiButton>
                </OuiFlex>
              </OuiFlex>
            </OuiStack>
          </OuiCardBody>
        </OuiCard>

        <template v-if="activeSkeletons.length">
          <OuiCard
            v-for="skeleton in activeSkeletons"
            :key="skeleton.id"
            :class="[
              'pointer-events-none select-none border-border-muted/70 bg-surface-muted/30',
              { 'opacity-70': showInfiniteSkeletons },
            ]"
          >
            <OuiCardBody>
              <OuiStack gap="md">
                <OuiFlex gap="md" align="start">
                  <OuiSkeleton width="3.5rem" height="3.5rem" variant="rectangle" rounded />
                  <OuiStack gap="xs" class="flex-1 min-w-0">
                    <OuiFlex align="center" gap="sm" wrap="wrap">
                      <OuiSkeleton :width="skeleton.titleWidth" height="1.25rem" variant="text" />
                      <OuiSkeleton
                        :width="skeleton.typeBadgeWidth"
                        height="0.95rem"
                        variant="rectangle"
                        class="rounded-full opacity-80"
                      />
                    </OuiFlex>
                    <OuiSkeleton :width="skeleton.descriptionWidth" height="2.25rem" variant="text" />
                  </OuiStack>
                </OuiFlex>

                <OuiFlex wrap="wrap" gap="xs">
                  <OuiSkeleton
                    v-for="(badgeWidth, badgeIndex) in skeleton.badgeWidths"
                    :key="`badge-${skeleton.id}-${badgeIndex}`"
                    :width="badgeWidth"
                    height="0.85rem"
                    variant="rectangle"
                    class="rounded-full opacity-80"
                  />
                </OuiFlex>

                <OuiFlex justify="between" align="center" wrap="wrap" gap="sm">
                  <OuiStack gap="xs" class="min-w-[150px]">
                    <OuiSkeleton width="4rem" height="0.75rem" variant="text" class="opacity-70" />
                    <OuiSkeleton :width="skeleton.statValueWidth" height="1.1rem" variant="text" />
                  </OuiStack>
                  <OuiStack gap="xs" class="min-w-[160px]">
                    <OuiSkeleton width="5rem" height="0.75rem" variant="text" class="opacity-70" />
                    <OuiSkeleton :width="skeleton.compatWidth" height="1.1rem" variant="text" />
                  </OuiStack>
                  <OuiFlex gap="sm" class="ml-auto">
                    <OuiSkeleton
                      :width="skeleton.buttonShortWidth"
                      height="2rem"
                      variant="rectangle"
                      class="rounded-lg"
                    />
                    <OuiSkeleton
                      :width="skeleton.buttonPrimaryWidth"
                      height="2rem"
                      variant="rectangle"
                      class="rounded-lg"
                    />
                  </OuiFlex>
                </OuiFlex>
              </OuiStack>
            </OuiCardBody>
          </OuiCard>
        </template>
      </OuiGrid>

      <OuiFlex
        v-if="!isLoading && projects.length === 0"
        direction="col"
        align="center"
        gap="sm"
        class="text-center py-16"
      >
        <OuiBox class="w-16 h-16 rounded-full bg-surface-muted flex items-center justify-center">
          <FolderIcon class="w-8 h-8 text-secondary" />
        </OuiBox>
        <OuiText size="lg" weight="semibold">No results yet</OuiText>
        <OuiText color="tertiary">
          Try adjusting your search or disabling some filters.
        </OuiText>
      </OuiFlex>

      <OuiBox
        v-if="hasMore"
        ref="loadMoreSentinel"
        class="h-8 w-full"
        aria-hidden="true"
      />
    </OuiStack>

    <OuiStack v-else gap="md">
      <template v-if="isInstalledLoading && installedFiles.length === 0">
        <OuiCard
          v-for="index in 4"
          :key="`installed-skeleton-${index}`"
          class="pointer-events-none select-none border-border-muted/70 bg-surface-muted/30"
        >
          <OuiCardBody>
            <OuiFlex gap="md" align="center">
              <OuiSkeleton width="2.75rem" height="2.75rem" variant="rectangle" rounded />
              <OuiStack gap="xs" class="flex-1">
                <OuiSkeleton width="14rem" height="1rem" variant="text" />
                <OuiSkeleton width="22rem" height="0.8rem" variant="text" />
              </OuiStack>
              <OuiSkeleton width="7rem" height="2rem" variant="rectangle" class="rounded-lg" />
            </OuiFlex>
          </OuiCardBody>
        </OuiCard>
      </template>

      <OuiCard
        v-for="file in installedFiles"
        :key="file.filename"
        variant="default"
        class="border-border-muted/70"
      >
        <OuiCardBody>
          <OuiFlex wrap="wrap" gap="md" align="center">
            <OuiAvatar
              size="md"
              :src="file.iconUrl || undefined"
              class="bg-surface-muted shrink-0"
              :alt="file.title || file.filename"
            >
              <PuzzlePieceIcon class="w-5 h-5 text-secondary" />
            </OuiAvatar>
            <OuiStack gap="xs" class="flex-1 min-w-[220px]">
              <OuiFlex gap="sm" align="center" wrap="wrap">
                <OuiText size="md" weight="semibold">{{ file.title || file.filename }}</OuiText>
                <OuiBadge size="xs" :variant="file.managed ? 'success' : 'secondary'">
                  {{ file.managed ? 'Managed' : 'Manual' }}
                </OuiBadge>
                <OuiBadge v-if="file.updateAvailable" size="xs" variant="warning">
                  Update available
                </OuiBadge>
              </OuiFlex>
              <OuiFlex gap="sm" wrap="wrap" align="center">
                <OuiText size="xs" color="tertiary">{{ file.filename }}</OuiText>
                <OuiText size="xs" color="tertiary">{{ formatBytes(file.sizeBytes) }}</OuiText>
                <OuiText v-if="file.versionNumber" size="xs" color="tertiary">v{{ file.versionNumber }}</OuiText>
                <OuiText v-if="file.latestVersionNumber" size="xs" color="tertiary">
                  Latest v{{ file.latestVersionNumber }}
                </OuiText>
              </OuiFlex>
            </OuiStack>
            <OuiFlex gap="sm" class="ml-auto">
              <OuiButton
                v-if="file.managed && file.projectId"
                size="sm"
                variant="ghost"
                class="gap-2"
                @click="openInstalledDetails(file)"
              >
                <EyeIcon class="w-4 h-4" />
                Details
              </OuiButton>
              <OuiButton
                v-if="file.managed && file.projectId"
                size="sm"
                :variant="file.updateAvailable ? 'soft' : 'ghost'"
                class="gap-2"
                :loading="updatingFilename === file.filename"
                @click="updateInstalledFile(file)"
              >
                <ArrowPathIcon class="w-4 h-4" />
                {{ file.updateAvailable ? 'Update' : 'Reinstall' }}
              </OuiButton>
            </OuiFlex>
          </OuiFlex>
        </OuiCardBody>
      </OuiCard>

      <OuiFlex
        v-if="!isInstalledLoading && installedFiles.length === 0"
        direction="col"
        align="center"
        gap="sm"
        class="text-center py-16"
      >
        <OuiBox class="w-16 h-16 rounded-full bg-surface-muted flex items-center justify-center">
          <FolderIcon class="w-8 h-8 text-secondary" />
        </OuiBox>
        <OuiText size="lg" weight="semibold">Nothing installed yet</OuiText>
        <OuiText color="tertiary">
          Install from the catalog or upload jars into the {{ installLocationDescription }}.
        </OuiText>
      </OuiFlex>
    </OuiStack>

    <OuiDialog v-model:open="installDialogOpen" :title="installDialogTitle">
      <OuiStack gap="md">
        <OuiAlert variant="muted" v-if="selectedProject">
          {{ selectedInstalledFile ? 'Updating' : 'Installing' }} into <strong>{{ installLocationDescription }}</strong>. Restart the server after changes.
        </OuiAlert>

        <OuiStack gap="xs">
          <OuiFlex align="center" justify="between" gap="md" class="flex-wrap">
            <OuiText size="sm" weight="semibold">Select version</OuiText>
            <OuiFlex align="center" gap="sm">
              <OuiText size="xs" color="tertiary">Include prereleases</OuiText>
              <OuiSwitch v-model="includePrereleaseVersions" size="sm" />
            </OuiFlex>
          </OuiFlex>
          <OuiSelect
            v-model="selectedVersionId"
            :disabled="isVersionsLoading || versionOptions.length === 0"
            :items="versionSelectItems"
            placeholder="Select a version"
          />
          <OuiStack
            v-if="selectedVersion"
            gap="xs"
            class="rounded-lg border border-border-muted bg-surface-muted/20 px-3 py-2"
          >
            <OuiFlex wrap="wrap" gap="xs" align="center">
              <OuiBadge size="xs" variant="secondary">{{ selectedVersion.versionNumber }}</OuiBadge>
              <OuiBadge
                v-if="versionReleaseLabel(selectedVersion)"
                size="xs"
                :variant="isStableVersionOption(selectedVersion) ? 'success' : 'warning'"
                tone="soft"
              >
                {{ versionReleaseLabel(selectedVersion) }}
              </OuiBadge>
              <OuiBadge
                v-for="loader in selectedVersion.loaders"
                :key="`selected-version-loader-${loader}`"
                size="xs"
                variant="secondary"
                tone="soft"
              >
                {{ loader }}
              </OuiBadge>
            </OuiFlex>
            <OuiText size="xs" color="tertiary" class="leading-relaxed break-words">
              Supports {{ formatVersionSupport(selectedVersion.gameVersions) }}
            </OuiText>
          </OuiStack>
          <OuiText v-if="isVersionsLoading" size="xs" color="tertiary">
            Loading versions…
          </OuiText>
          <OuiText v-else-if="versionOptions.length === 0" size="xs" color="tertiary">
            No versions available for this project.
          </OuiText>
        </OuiStack>
      </OuiStack>

      <template #footer>
        <OuiButton variant="ghost" @click="closeInstallDialog">Cancel</OuiButton>
        <OuiButton
          color="primary"
          :loading="isInstalling"
          :disabled="!selectedVersionId"
          @click="installSelected"
        >
          {{ selectedInstalledFile ? 'Update & Restart Later' : 'Install & Restart Later' }}
        </OuiButton>
      </template>
    </OuiDialog>

    <!-- Tabbed Window Group -->
    <TabbedWindowGroup
      v-if="openTabs.length > 0"
      v-model="activeTabId"
      :tabs="openTabs"
      :initial-position="windowPosition"
      :initial-size="windowSize"
      @close="closeWindowGroup"
      @tab-close="closeOverview"
      @tabs-reorder="handleTabsReorder"
      @tab-drag-out="handleTabDragOut"
      @tab-drop-external="(tabId, event) => handleTabDropExternal('primary', tabId, event)"
    >
      <template v-for="tab in openTabs" :key="tab.id" #[`tab-${tab.id}`]>
        <MinecraftProjectOverview
          :project="tab.project"
          :active-version-filter="activeVersionFilter"
          :game-server-id="gameServerId"
        />
      </template>
      <template #footer="{ activeTab: activeTabData }">
        <OuiButton
          v-if="activeTabData"
          color="primary"
          size="lg"
          :loading="selectedProject?.id === activeTabData.project.id && isInstalling"
          @click="openInstallDialog(activeTabData.project, getInstalledProject(activeTabData.project) || undefined)"
          class="w-full gap-2"
        >
          <CloudArrowDownIcon class="w-5 h-5" />
          Install {{ formatProjectType(activeTabData.project.projectType) }}
        </OuiButton>
      </template>
    </TabbedWindowGroup>
    <TabbedWindowGroup
      v-for="window in detachedWindows"
      :key="window.id"
      v-model="window.activeTabId"
      :tabs="window.tabs"
      :initial-position="window.position"
      :initial-size="window.size"
      :persist-rect="false"
      @close="closeDetachedWindow(window.id)"
      @tab-close="(tabId) => handleDetachedTabClose(window.id, tabId)"
      @tabs-reorder="(newOrder) => handleDetachedTabsReorder(window.id, newOrder)"
      @tab-drag-out="(tabId, event) => handleDetachedTabDragOut(window.id, tabId, event)"
      @tab-drop-external="(tabId, event) => handleTabDropExternal(window.id, tabId, event)"
    >
      <template v-for="tab in window.tabs" :key="tab.id" #[`tab-${tab.id}`]>
        <MinecraftProjectOverview
          :project="tab.project"
          :active-version-filter="activeVersionFilter"
          :game-server-id="gameServerId"
        />
      </template>
      <template #footer="{ activeTab: activeTabData }">
        <OuiButton
          v-if="activeTabData"
          color="primary"
          size="lg"
          :loading="selectedProject?.id === activeTabData.project.id && isInstalling"
          @click="openInstallDialog(activeTabData.project, getInstalledProject(activeTabData.project) || undefined)"
          class="w-full gap-2"
        >
          <CloudArrowDownIcon class="w-5 h-5" />
          Install {{ formatProjectType(activeTabData.project.projectType) }}
        </OuiButton>
      </template>
    </TabbedWindowGroup>
  </OuiStack>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, shallowRef, watch } from "vue";
import {
  ArrowPathIcon,
  CloudArrowDownIcon,
  CubeIcon,
  EyeIcon,
  FolderIcon,
  MagnifyingGlassIcon,
  PuzzlePieceIcon,
} from "@heroicons/vue/24/outline";
import { create } from "@bufbuild/protobuf";
import { useDebounceFn, useIntersectionObserver } from "@vueuse/core";
import { randomTextWidthByType } from "~/composables/useSkeletonVariations";
import { useConnectClient } from "~/lib/connect-client";
import { useToast } from "~/composables/useToast";
import {
  GameServerService,
  MinecraftProjectSchema,
  MinecraftProjectType,
} from "@obiente/proto";
import type {
  InstalledMinecraftProjectFile,
  MinecraftProject,
  MinecraftProjectVersion,
} from "@obiente/proto";
import TabbedWindowGroup from "./TabbedWindowGroup.vue";
import MinecraftProjectOverview from "./MinecraftProjectOverview.vue";
import OuiSwitch from "~/components/oui/Switch.vue";

interface OpenTab {
  id: string;
  project: MinecraftProject;
  title: string;
}

interface SkeletonCardState {
  id: string;
  titleWidth: string;
  descriptionWidth: string;
  typeBadgeWidth: string;
  badgeWidths: string[];
  statValueWidth: string;
  compatWidth: string;
  buttonShortWidth: string;
  buttonPrimaryWidth: string;
}

interface DetachedWindow {
  id: string;
  tabs: OpenTab[];
  activeTabId: string;
  position: { x: number; y: number };
  size: { width: number; height: number };
}

interface ProjectCardView {
  project: MinecraftProject;
  installedFile: InstalledMinecraftProjectFile | null;
  typeLabels: string[];
  visibleLoaders: string[];
  loaderOverflow: number;
  visibleCategories: string[];
  categoryOverflow: number;
  versionRange: string;
  compatibility: { label: string; variant: "success" | "warning" | "secondary"; range?: string };
  downloadsLabel: string;
}

const PAGE_SIZE = 18;
const INITIAL_SKELETON_COUNT = 6;

const props = defineProps<{
  gameServerId: string;
  serverType?: string | null;
  serverVersion?: string | null;
}>();

const client = useConnectClient(GameServerService);
const { toast } = useToast();

const projects = shallowRef<MinecraftProject[]>([]);
const installedFiles = shallowRef<InstalledMinecraftProjectFile[]>([]);
const cursor = ref<string | undefined>();
const hasMore = ref(false);
const isLoading = ref(false);
const isLoadingMore = ref(false);
const isInstalledLoading = ref(false);
const isBulkUpdating = ref(false);
const errorMessage = ref<string | null>(null);
const installedErrorMessage = ref<string | null>(null);
const hasLoadedOnce = ref(false);
const isRefreshing = ref(false);
const activeView = ref<"browse" | "installed">("installed");
const updatingFilename = ref("");
let skeletonIdCounter = 0;
const initialSkeletons = ref<SkeletonCardState[]>(createSkeletonStates(INITIAL_SKELETON_COUNT));
const infiniteSkeletons = ref<SkeletonCardState[]>([]);

const searchQuery = ref("");
const matchServerVersion = ref(Boolean(props.serverVersion));
const matchServerType = ref(true);
const selectedCategories = ref<string[]>([]);
const sortBy = ref("relevance");

const showSkeletonPlaceholders = computed(
  () => isLoading.value && (!hasLoadedOnce.value || projects.value.length === 0)
);
const showInfiniteSkeletons = computed(
  () => isLoadingMore.value && projects.value.length > 0 && infiniteSkeletons.value.length > 0
);
const activeSkeletons = computed<SkeletonCardState[]>(() => {
  if (showSkeletonPlaceholders.value) {
    return initialSkeletons.value;
  }
  if (showInfiniteSkeletons.value) {
    return infiniteSkeletons.value;
  }
  return [];
});

// Available categories for filtering (common Modrinth categories)
const availableCategories = [
  { value: "adventure", label: "Adventure" },
  { value: "magic", label: "Magic" },
  { value: "technology", label: "Technology" },
  { value: "worldgen", label: "World Generation" },
  { value: "food", label: "Food" },
  { value: "library", label: "Library" },
  { value: "optimization", label: "Optimization" },
  { value: "storage", label: "Storage" },
  { value: "utility", label: "Utility" },
  { value: "decoration", label: "Decoration" },
  { value: "combat", label: "Combat" },
  { value: "economy", label: "Economy" },
  { value: "social", label: "Social" },
  { value: "cursed", label: "Cursed" },
  { value: "fabric", label: "Fabric" },
  { value: "forge", label: "Forge" },
  { value: "multiloader", label: "Multi-Loader" },
];

// Sort options
const sortOptions = [
  { label: "Relevance", value: "relevance" },
  { label: "Most Downloaded", value: "downloads" },
  { label: "Recently Updated", value: "updated" },
];

const inferredLoader = computed(() => {
  const serverType = (props.serverType || "").toUpperCase();
  const mapping: Record<string, string> = {
    FORGE: "forge",
    NEOFORGE: "neoforge",
    FABRIC: "fabric",
    QUILT: "quilt",
    MAGMA: "magma",
    CATSERVER: "catserver",
    PAPER: "paper",
    PURPUR: "purpur",
    SPIGOT: "spigot",
    BUKKIT: "bukkit",
    FOLIA: "folia",
    VELOCITY: "velocity",
    WATERFALL: "waterfall",
  };
  return mapping[serverType] || "";
});

const projectType = computed(() => {
  const serverType = (props.serverType || "").toUpperCase();
  if (["PAPER", "PURPUR", "SPIGOT", "BUKKIT", "FOLIA", "VELOCITY", "WATERFALL"].includes(serverType)) {
    return MinecraftProjectType.PLUGIN;
  }
  return MinecraftProjectType.MOD;
});

const isModdedServer = computed(() => projectType.value === MinecraftProjectType.MOD);

const projectTypeLabel = computed(() => {
  return projectType.value === MinecraftProjectType.PLUGIN ? "Plugins" : "Mods";
});

const activeVersionFilter = computed(() => {
  if (!matchServerVersion.value) return "";
  return props.serverVersion ? props.serverVersion.replace(/^v/i, "") : "";
});

const activeLoaderFilter = computed(() => {
  // For modded servers, always filter by loader
  if (isModdedServer.value) {
    return inferredLoader.value;
  }
  // For plugin servers, only filter if matchServerType is enabled
  if (!matchServerType.value) return "";
  return inferredLoader.value;
});

const installDialogOpen = ref(false);
const selectedProject = ref<MinecraftProject | null>(null);
const selectedInstalledFile = ref<InstalledMinecraftProjectFile | null>(null);
const versionOptions = shallowRef<MinecraftProjectVersion[]>([]);
const selectedVersionId = ref<string>("");
const includePrereleaseVersions = ref(false);
const isVersionsLoading = ref(false);
const isInstalling = ref(false);
const loadMoreSentinel = ref<HTMLElement | null>(null);

const openTabs = ref<OpenTab[]>([]);
const activeTabId = ref<string>("");
const detachedWindows = ref<DetachedWindow[]>([]);
const windowPosition = ref({ x: 100, y: 100 });
const windowSize = ref({ width: 800, height: 600 });
let tabCounter = 0;
let detachedWindowCounter = 0;
let loadProjectsRequestId = 0;
let loadInstalledRequestId = 0;

const installDialogTitle = computed(() =>
  selectedProject.value
    ? `${selectedInstalledFile.value ? "Update" : "Install"} ${selectedProject.value.title}`
    : "Install content"
);
const installLocationDescription = computed(() => {
  return projectType.value === MinecraftProjectType.PLUGIN ? "plugins directory" : "mods directory";
});
const selectedVersion = computed(() => {
  return versionOptions.value.find((version) => version.id === selectedVersionId.value) || null;
});
const versionSelectItems = computed(() =>
  versionOptions.value.map((version) => ({
    label: formatVersionOptionLabel(version),
    value: version.id,
  }))
);

const debouncedSearch = useDebounceFn(() => refresh(), 350);
const { stop: stopAutoLoadObserver } = useIntersectionObserver(
  loadMoreSentinel,
  ([entry]) => {
    if (
      entry?.isIntersecting &&
      hasMore.value &&
      !isLoading.value &&
      !isLoadingMore.value &&
      !isRefreshing.value
    ) {
      loadMore();
    }
  },
  {
    rootMargin: "400px 0px 0px 0px",
  }
);

watch(searchQuery, () => debouncedSearch());
watch(matchServerVersion, () => refresh());
watch(matchServerType, () => refresh());
watch(
  selectedCategories,
  () => refresh(),
  { deep: true }
);
watch(sortBy, () => refresh());
watch(includePrereleaseVersions, () => {
  if (installDialogOpen.value && selectedProject.value) {
    fetchVersions(selectedProject.value.id);
  }
});
watch(
  () => props.serverVersion,
  () => {
    if (props.serverVersion) {
      matchServerVersion.value = true;
    }
    refresh();
    loadInstalledFiles();
  }
);
watch(
  () => props.serverType,
  () => {
    refresh();
    loadInstalledFiles();
  }
);

const installedByProjectId = computed(() => {
  const map = new Map<string, InstalledMinecraftProjectFile>();
  for (const file of installedFiles.value) {
    if (file.projectId) {
      map.set(file.projectId, file);
    }
  }
  return map;
});

const projectCards = computed<ProjectCardView[]>(() => {
  const installed = installedByProjectId.value;
  return projects.value.map((project) => {
    const displayCategories = displayableCategories(project);
    return {
      project,
      installedFile: installed.get(project.id) || null,
      typeLabels: getProjectTypeLabels(project),
      visibleLoaders: (project.loaders || []).slice(0, 3),
      loaderOverflow: Math.max(0, (project.loaders?.length || 0) - 3),
      visibleCategories: displayCategories.slice(0, 4),
      categoryOverflow: Math.max(0, displayCategories.length - 4),
      versionRange: (project.gameVersions?.length || 0) > 0 ? formatVersionRange(project.gameVersions || []) : "",
      compatibility: getCompatibilityStatus(project),
      downloadsLabel: formatDownloads(project.downloads),
    };
  });
});

const updateableInstalledFiles = computed(() =>
  installedFiles.value.filter((file) => file.managed && file.projectId && file.updateAvailable && file.latestVersionId)
);

async function refresh() {
  cursor.value = undefined;
  await loadProjects({ reset: true });
}

async function refreshCatalogState() {
  await Promise.all([loadProjects({ reset: true }), loadInstalledFiles()]);
}

async function loadMore() {
  if (!hasMore.value || isLoadingMore.value) return;
  isLoadingMore.value = true;
  regenerateInfiniteSkeletons(PAGE_SIZE);
  await loadProjects();
  isLoadingMore.value = false;
}

function buildQueryPayload(cursorValue?: string) {
  return {
    gameServerId: props.gameServerId,
    query: searchQuery.value || undefined,
    projectType: projectType.value,
    gameVersions: activeVersionFilter.value ? [activeVersionFilter.value] : [],
    loaders: activeLoaderFilter.value ? [activeLoaderFilter.value] : [],
    categories: selectedCategories.value.length > 0 ? selectedCategories.value : [],
    cursor: cursorValue,
    limit: PAGE_SIZE,
  };
}

function toggleCategory(category: string) {
  const index = selectedCategories.value.indexOf(category);
  if (index > -1) {
    selectedCategories.value.splice(index, 1);
  } else {
    selectedCategories.value.push(category);
  }
}

function clearCategoryFilters() {
  selectedCategories.value = [];
}

function toggleMatchServerVersion() {
  matchServerVersion.value = !matchServerVersion.value;
}

function toggleMatchServerType() {
  matchServerType.value = !matchServerType.value;
}

function displayableCategories(project: MinecraftProject): string[] {
  const categories = project.categories || [];
  if (categories.length === 0) {
    return [];
  }
  const loaderSet = new Set((project.loaders || []).map((loader) => loader.toLowerCase()));
  return categories.filter((category) => !loaderSet.has(category.toLowerCase()));
}

function formatCategory(category: string): string {
  return category
    .split(/[-_]/)
    .map(word => word.charAt(0).toUpperCase() + word.slice(1))
    .join(" ");
}

function formatVersionRange(versions: string[]): string {
  if (!versions || versions.length === 0) {
    return "Any version";
  }
  const normalized = versions.map((v) => v.replace(/^v/i, ""));
  const sorted = [...normalized].sort((a, b) => compareVersions(a, b));
  const first = sorted[0];
  const last = sorted[sorted.length - 1];
  if (!first) {
    return "Various versions";
  }
  if (!last || first === last) {
    return `v${first}`;
  }
  return `v${first} - v${last}`;
}

function regenerateInfiniteSkeletons(count: number) {
  if (count <= 0) {
    infiniteSkeletons.value = [];
    return;
  }
  infiniteSkeletons.value = createSkeletonStates(count);
}

function createSkeletonStates(count: number): SkeletonCardState[] {
  return Array.from({ length: count }, () => createSkeletonState());
}

function createSkeletonState(): SkeletonCardState {
  const badgeCount = 3 + Math.floor(Math.random() * 2);
  return {
    id: `skeleton-card-${++skeletonIdCounter}`,
    titleWidth: randomTextWidthByType("title"),
    descriptionWidth: randomTextWidthByType("subtitle"),
    typeBadgeWidth: randomTextWidthByType("label"),
    badgeWidths: Array.from({ length: badgeCount }, () => randomTextWidthByType("short")),
    statValueWidth: randomTextWidthByType("value"),
    compatWidth: randomTextWidthByType("label"),
    buttonShortWidth: randomTextWidthByType("subtitle"),
    buttonPrimaryWidth: randomTextWidthByType("title"),
  };
}

async function loadProjects(opts: { reset?: boolean } = {}) {
  if (!props.gameServerId) return;
  const requestId = ++loadProjectsRequestId;
  const isReset = Boolean(opts.reset);
  if (isReset) {
    isLoading.value = true;
    isLoadingMore.value = false;
    infiniteSkeletons.value = [];
    isRefreshing.value = hasLoadedOnce.value && projects.value.length > 0;
    if (!hasLoadedOnce.value || projects.value.length === 0) {
      projects.value = [];
      initialSkeletons.value = createSkeletonStates(INITIAL_SKELETON_COUNT);
    }
    errorMessage.value = null;
  }

  try {
    const payload = buildQueryPayload(cursor.value);
    const response = await client.listMinecraftProjects(payload);
    if (requestId !== loadProjectsRequestId) return;
    const fetchedProjects = response.projects ?? [];
    const fetchedCount = fetchedProjects.length;
    if (isReset) {
      projects.value = fetchedProjects;
    } else {
      projects.value = [...projects.value, ...fetchedProjects];
    }
    cursor.value = response.nextCursor ?? undefined;
    hasMore.value = Boolean(response.hasMore);
    hasLoadedOnce.value = true;
    if (!isReset && isLoadingMore.value) {
      regenerateInfiniteSkeletons(fetchedCount);
    }
  } catch (err: unknown) {
    if (requestId !== loadProjectsRequestId) return;
    console.error(err);
    errorMessage.value = (err as Error | undefined)?.message ?? "Failed to load catalog";
    toast.error("Failed to load Minecraft catalog", errorMessage.value || undefined);
  } finally {
    if (requestId !== loadProjectsRequestId) return;
    if (isReset) {
      isLoading.value = false;
      isRefreshing.value = false;
    }
  }
}

async function loadInstalledFiles() {
  if (!props.gameServerId) return;
  const requestId = ++loadInstalledRequestId;
  isInstalledLoading.value = true;
  installedErrorMessage.value = null;
  try {
    const response = await client.listInstalledMinecraftProjects({
      gameServerId: props.gameServerId,
      projectType: projectType.value,
      checkUpdates: true,
    });
    if (requestId !== loadInstalledRequestId) return;
    installedFiles.value = response.files ?? [];
  } catch (err: unknown) {
    if (requestId !== loadInstalledRequestId) return;
    console.error(err);
    installedErrorMessage.value = (err as Error | undefined)?.message ?? "Failed to load installed content";
    toast.error("Failed to load installed content", installedErrorMessage.value || undefined);
  } finally {
    if (requestId !== loadInstalledRequestId) return;
    isInstalledLoading.value = false;
  }
}

function getInstalledProject(project: MinecraftProject) {
  return installedByProjectId.value.get(project.id) || null;
}

function formatProjectType(type: MinecraftProjectType | string | undefined | null) {
  // Handle both enum values and numeric values (protobuf can send numbers)
  // PLUGIN = 2, MOD = 1, UNSPECIFIED = 0
  if (type == null) {
    // Fallback to server's default project type
    return projectType.value === MinecraftProjectType.PLUGIN ? "Plugin" : "Mod";
  }
  
  if (typeof type === "string") {
    const normalized = type.toUpperCase().trim();
    if (normalized.includes("PLUGIN")) {
      return "Plugin";
    }
    if (normalized.includes("MOD")) {
      return "Mod";
    }
    const parsed = Number(type);
    if (!Number.isNaN(parsed)) {
      return formatProjectType(parsed as MinecraftProjectType);
    }
  }

  // Convert to number for reliable comparison
  const typeNum = typeof type === "number" ? type : Number(type);
  
  // Check numeric values (PLUGIN = 2, MOD = 1)
  if (typeNum === 2) {
    return "Plugin";
  }
  if (typeNum === 1) {
    return "Mod";
  }
  
  // Also check enum constants as fallback
  if (type === MinecraftProjectType.PLUGIN) {
    return "Plugin";
  }
  if (type === MinecraftProjectType.MOD) {
    return "Mod";
  }
  
  // Final fallback: use server's default project type
  return projectType.value === MinecraftProjectType.PLUGIN ? "Plugin" : "Mod";
}

function getProjectTypeLabels(project: MinecraftProject): string[] {
  const candidateTypes =
    (project as any).projectTypes ||
    (project as any).types ||
    (project as any).project_types ||
    [];

  const labels = Array.isArray(candidateTypes)
    ? candidateTypes
        .map((t) => formatProjectType(t as any))
        .filter((label) => Boolean(label))
    : [];

  if (labels.length > 0) {
    return Array.from(new Set(labels));
  }

  const fallback = formatProjectType(project.projectType);
  return fallback ? [fallback] : [];
}

function formatDownloads(downloads?: bigint | number | null) {
  const value = typeof downloads === "bigint" ? Number(downloads) : downloads || 0;
  if (value >= 1_000_000) return `${(value / 1_000_000).toFixed(1)}M`;
  if (value >= 1_000) return `${(value / 1_000).toFixed(1)}k`;
  return value.toString();
}

function formatBytes(bytes?: bigint | number | null) {
  const value = typeof bytes === "bigint" ? Number(bytes) : bytes || 0;
  if (value >= 1024 * 1024) return `${(value / (1024 * 1024)).toFixed(1)} MB`;
  if (value >= 1024) return `${(value / 1024).toFixed(1)} KB`;
  return `${value} B`;
}

function formatVersionOptionLabel(version: MinecraftProjectVersion) {
  const parts = [version.versionNumber || version.name || "Unnamed version"];
  const releaseLabel = versionReleaseLabel(version);
  if (releaseLabel && !isStableVersionOption(version)) {
    parts.push(releaseLabel);
  }
  const support = summarizeVersionSupport(version.gameVersions);
  if (support) {
    parts.push(support);
  }
  if (version.loaders?.length) {
    parts.push(version.loaders.slice(0, 3).join(", "));
  }
  return parts.join(" - ");
}

function versionReleaseLabel(version: MinecraftProjectVersion) {
  const type = (version.versionType || "").trim().toLowerCase();
  if (type === "alpha") return "Alpha";
  if (type === "beta") return "Beta";
  const combined = `${version.versionNumber || ""} ${version.name || ""}`.toLowerCase();
  if (combined.includes("snapshot")) return "Snapshot";
  if (combined.includes("alpha")) return "Alpha";
  if (combined.includes("beta")) return "Beta";
  if (combined.includes("rc")) return "RC";
  return "";
}

function isStableVersionOption(version: MinecraftProjectVersion) {
  return !versionReleaseLabel(version);
}

function summarizeVersionSupport(versions?: string[] | null) {
  const normalized = normalizeMinecraftVersions(versions);
  if (normalized.length === 0) return "Any Minecraft version";
  if (normalized.length === 1) return normalized[0];
  if (normalized.length <= 3) return normalized.join(", ");
  return `${normalized[0]}-${normalized[normalized.length - 1]}`;
}

function formatVersionSupport(versions?: string[] | null) {
  const normalized = normalizeMinecraftVersions(versions);
  if (normalized.length === 0) return "any Minecraft version";
  if (normalized.length <= 12) return normalized.join(", ");
  return `${normalized.slice(0, 8).join(", ")} and ${normalized.length - 8} more through ${normalized[normalized.length - 1]}`;
}

function normalizeMinecraftVersions(versions?: string[] | null) {
  const unique = Array.from(
    new Set(
      (versions || [])
        .map((version) => version.replace(/^v/i, "").trim())
        .filter(Boolean)
    )
  );
  return unique.sort((a, b) => compareVersions(a, b));
}

function parseVersion(version: string): number[] {
  // Parse version string like "1.20.1" into [1, 20, 1]
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
  if (!activeVersionFilter.value || !project.gameVersions || project.gameVersions.length === 0) {
    return { label: "Unknown", variant: "secondary" };
  }

  const serverVersion = activeVersionFilter.value;
  const projectVersions = project.gameVersions;

  // Check if server version is in the project's supported versions
  const isCompatible = projectVersions.some((v) => {
    const normalized = v.replace(/^v/i, "");
    return normalized === serverVersion || compareVersions(normalized, serverVersion) === 0;
  });

  if (isCompatible) {
    return { label: "Compatible", variant: "success" };
  }

  // Find min and max versions
  const sortedVersions = [...projectVersions].sort((a, b) => compareVersions(a, b));
  const minVersion = sortedVersions[0];
  const maxVersion = sortedVersions[sortedVersions.length - 1];

  if (!minVersion || !maxVersion) {
    return { label: "Incompatible", variant: "secondary" };
  }

  const serverVsMin = compareVersions(serverVersion, minVersion);
  const serverVsMax = compareVersions(serverVersion, maxVersion);

  if (serverVsMin < 0) {
    // Server version is older than project's minimum
    return { 
      label: "Newer version", 
      variant: "warning",
      range: `Requires ${minVersion}+`
    };
  } else if (serverVsMax > 0) {
    // Server version is newer than project's maximum
    return { 
      label: "Older version", 
      variant: "warning",
      range: `Up to ${maxVersion}`
    };
  }

  return { label: "Incompatible", variant: "secondary" };
}

function closeInstallDialog() {
  installDialogOpen.value = false;
  selectedProject.value = null;
  selectedInstalledFile.value = null;
  versionOptions.value = [];
  selectedVersionId.value = "";
  includePrereleaseVersions.value = false;
}

function projectFromInstalled(file: InstalledMinecraftProjectFile): MinecraftProject {
  return create(MinecraftProjectSchema, {
    id: file.projectId || file.id,
    slug: file.projectSlug || file.projectId || file.filename,
    title: file.title || file.filename,
    description: file.managed ? "Installed from Modrinth." : "Installed manually.",
    projectType: file.projectType || projectType.value,
    iconUrl: file.iconUrl || "",
    categories: [],
    loaders: file.loaders || [],
    gameVersions: file.gameVersions || [],
    authors: [],
    downloads: 0n,
    rating: 0,
    latestVersionId: file.latestVersionId,
    projectUrl: undefined,
    sourceUrl: undefined,
    issuesUrl: undefined,
    body: undefined,
    gallery: [],
  });
}

function openInstalledDetails(file: InstalledMinecraftProjectFile) {
  if (!file.projectId) return;
  openOverview(projectFromInstalled(file));
}

function openOverview(project: MinecraftProject) {
  // Initialize window position on first tab
  if (openTabs.value.length === 0) {
    windowPosition.value = {
      x: (window.innerWidth - windowSize.value.width) / 2,
      y: (window.innerHeight - windowSize.value.height) / 2,
    };
  }

  // Always create a new tab (allow multiple tabs for the same project)
  // Count how many tabs already exist for this project to make title unique
  const sameProjectCount = openTabs.value.filter((t) => t.project.id === project.id).length;
  const tabTitle = sameProjectCount > 0 ? `${project.title} (${sameProjectCount + 1})` : project.title;

  const newTab: OpenTab = {
    id: `tab-${++tabCounter}`,
    project,
    title: tabTitle,
  };

  openTabs.value.push(newTab);
  activeTabId.value = newTab.id;
}

function removeTabFromPrimary(tabId: string): OpenTab | null {
  const index = openTabs.value.findIndex((t) => t.id === tabId);
  if (index === -1) {
    return null;
  }
  const [removed] = openTabs.value.splice(index, 1);
  if (activeTabId.value === tabId) {
    if (openTabs.value.length > 0) {
      const newIndex = Math.max(0, index - 1);
      const newTab = openTabs.value[newIndex] || openTabs.value[0];
      activeTabId.value = newTab?.id || "";
    } else {
      activeTabId.value = "";
    }
  }
  return removed ?? null;
}

function closeOverview(tabId: string) {
  removeTabFromPrimary(tabId);
}

function handleTabsReorder(newOrder: Array<{ id: string; label: string }>) {
  // Reorder openTabs to match the new order from DraggableTabs
  const newTabsOrder: OpenTab[] = [];
  for (const orderedTab of newOrder) {
    const existingTab = openTabs.value.find((t) => t.id === orderedTab.id);
    if (existingTab) {
      newTabsOrder.push(existingTab);
    }
  }
  // Add any tabs that weren't in the new order (shouldn't happen, but safety check)
  for (const tab of openTabs.value) {
    if (!newTabsOrder.find((t) => t.id === tab.id)) {
      newTabsOrder.push(tab);
    }
  }
  openTabs.value = newTabsOrder;
}

function findDetachedWindow(windowId: string) {
  return detachedWindows.value.find((w) => w.id === windowId);
}

function removeTabFromDetached(window: DetachedWindow, tabId: string): OpenTab | null {
  const index = window.tabs.findIndex((t) => t.id === tabId);
  if (index === -1) {
    return null;
  }
  const [removed] = window.tabs.splice(index, 1);
  if (window.activeTabId === tabId) {
    if (window.tabs.length > 0) {
      const newIndex = Math.max(0, index - 1);
      const newTab = window.tabs[newIndex] || window.tabs[0];
      window.activeTabId = newTab?.id || "";
    } else {
      window.activeTabId = "";
    }
  }
  return removed ?? null;
}

function computeDetachedWindowPosition(event?: DragEvent) {
  if (typeof window === "undefined") {
    return { x: 120, y: 120 };
  }
  const padding = 24;
  const defaultX = Math.max(padding, (window.innerWidth - windowSize.value.width) / 2);
  const defaultY = Math.max(padding, (window.innerHeight - windowSize.value.height) / 2);
  if (!event) {
    return { x: defaultX, y: defaultY };
  }
  const clampedX = Math.min(
    Math.max(event.clientX - windowSize.value.width / 2, padding),
    window.innerWidth - windowSize.value.width - padding
  );
  const clampedY = Math.min(
    Math.max(event.clientY - 60, padding),
    window.innerHeight - windowSize.value.height - padding
  );
  return { x: clampedX, y: clampedY };
}

function createDetachedWindowFromTab(tab: OpenTab, event?: DragEvent) {
  const newWindow: DetachedWindow = {
    id: `detached-${++detachedWindowCounter}`,
    tabs: [tab],
    activeTabId: tab.id,
    position: computeDetachedWindowPosition(event),
    size: { ...windowSize.value },
  };
  detachedWindows.value.push(newWindow);
}

function closeDetachedWindow(windowId: string) {
  const index = detachedWindows.value.findIndex((w) => w.id === windowId);
  if (index !== -1) {
    detachedWindows.value.splice(index, 1);
  }
}

function handleDetachedTabClose(windowId: string, tabId: string) {
  const window = findDetachedWindow(windowId);
  if (!window) return;
  removeTabFromDetached(window, tabId);
  if (window.tabs.length === 0) {
    closeDetachedWindow(windowId);
  }
}

function handleDetachedTabsReorder(windowId: string, newOrder: Array<{ id: string; label?: string }>) {
  const window = findDetachedWindow(windowId);
  if (!window) return;
  const newTabsOrder: OpenTab[] = [];
  for (const orderedTab of newOrder) {
    const existingTab = window.tabs.find((t) => t.id === orderedTab.id);
    if (existingTab) {
      newTabsOrder.push(existingTab);
    }
  }
  for (const tab of window.tabs) {
    if (!newTabsOrder.includes(tab)) {
      newTabsOrder.push(tab);
    }
  }
  window.tabs = newTabsOrder;
}

function detachTabFromWindow(windowId: string, tabId: string, event?: DragEvent) {
  if (windowId === "primary") {
    const tab = removeTabFromPrimary(tabId);
    if (tab) {
      createDetachedWindowFromTab(tab, event);
    }
    return;
  }
  const window = findDetachedWindow(windowId);
  if (!window) return;
  const tab = removeTabFromDetached(window, tabId);
  if (tab) {
    createDetachedWindowFromTab(tab, event);
  }
  if (window.tabs.length === 0) {
    closeDetachedWindow(windowId);
  }
}

function handleTabDragOut(tabId: string, event: DragEvent) {
  detachTabFromWindow("primary", tabId, event);
}

function handleDetachedTabDragOut(windowId: string, tabId: string, event: DragEvent) {
  detachTabFromWindow(windowId, tabId, event);
}

function findTabInAllWindows(tabId: string): { windowId: string; tab: OpenTab; index: number } | null {
  // Check primary window
  const primaryTab = openTabs.value.find((t) => t.id === tabId);
  if (primaryTab) {
    const primaryIndex = openTabs.value.findIndex((t) => t.id === tabId);
    return { windowId: "primary", tab: primaryTab, index: primaryIndex };
  }
  
  // Check detached windows
  for (const window of detachedWindows.value) {
    const tab = window.tabs.find((t) => t.id === tabId);
    if (tab) {
      const index = window.tabs.findIndex((t) => t.id === tabId);
      return { windowId: window.id, tab, index };
    }
  }
  
  return null;
}

function moveTabToWindow(sourceWindowId: string, tabId: string, targetWindowId: string) {
  const tabInfo = findTabInAllWindows(tabId);
  if (!tabInfo) return;
  
  const { tab } = tabInfo;
  
  // Remove from source window
  if (sourceWindowId === "primary") {
    const index = openTabs.value.findIndex((t) => t.id === tabId);
    if (index !== -1) {
      openTabs.value.splice(index, 1);
      if (activeTabId.value === tabId) {
        if (openTabs.value.length > 0) {
          activeTabId.value = openTabs.value[Math.max(0, index - 1)]?.id || openTabs.value[0]?.id || "";
        } else {
          activeTabId.value = "";
        }
      }
    }
  } else {
    const sourceWindow = findDetachedWindow(sourceWindowId);
    if (sourceWindow) {
      const index = sourceWindow.tabs.findIndex((t) => t.id === tabId);
      if (index !== -1) {
        sourceWindow.tabs.splice(index, 1);
        if (sourceWindow.activeTabId === tabId) {
          if (sourceWindow.tabs.length > 0) {
            sourceWindow.activeTabId = sourceWindow.tabs[Math.max(0, index - 1)]?.id || sourceWindow.tabs[0]?.id || "";
          } else {
            sourceWindow.activeTabId = "";
          }
        }
        if (sourceWindow.tabs.length === 0) {
          closeDetachedWindow(sourceWindowId);
        }
      }
    }
  }
  
  // Add to target window
  if (targetWindowId === "primary") {
    openTabs.value.push(tab);
    activeTabId.value = tab.id;
  } else {
    const targetWindow = findDetachedWindow(targetWindowId);
    if (targetWindow) {
      targetWindow.tabs.push(tab);
      targetWindow.activeTabId = tab.id;
    }
  }
}

function handleTabDropExternal(targetWindowId: string, tabId: string, event: DragEvent) {
  // Find which window the tab is coming from
  const tabInfo = findTabInAllWindows(tabId);
  if (!tabInfo) return;
  
  const sourceWindowId = tabInfo.windowId;
  
  // Don't move if it's the same window
  if (sourceWindowId === targetWindowId) return;
  
  moveTabToWindow(sourceWindowId, tabId, targetWindowId);
}

function closeWindowGroup() {
  openTabs.value = [];
  activeTabId.value = "";
}

async function openInstallDialog(project: MinecraftProject, installedFile?: InstalledMinecraftProjectFile) {
  selectedProject.value = project;
  selectedInstalledFile.value = installedFile || null;
  installDialogOpen.value = true;
  versionOptions.value = [];
  selectedVersionId.value = "";
  includePrereleaseVersions.value = false;
  await fetchVersions(project.id);
  if (installedFile?.latestVersionId) {
    selectedVersionId.value = installedFile.latestVersionId;
  }
}

async function fetchVersions(projectId: string) {
  isVersionsLoading.value = true;
  try {
    // Don't filter by loader/game version when fetching versions for installation
    // This allows users to see all available versions, not just ones matching search filters
    // Note: Backend will respect empty arrays and not auto-fill filters
    const response = await client.getMinecraftProjectVersions({
      gameServerId: props.gameServerId,
      projectId,
      projectType: projectType.value,
      loaders: activeLoaderFilter.value ? [activeLoaderFilter.value] : [],
      gameVersions: activeVersionFilter.value ? [activeVersionFilter.value] : [],
      limit: 300,
      includePrereleases: includePrereleaseVersions.value,
    });
    versionOptions.value = response.versions ?? [];
    if (versionOptions.value.length && versionOptions.value[0]?.id) {
      selectedVersionId.value = versionOptions.value[0].id;
    } else if (versionOptions.value.length === 0) {
      // Show a helpful message if no versions are available
      toast.warning("No versions available", "This project may not have any compatible versions.");
    }
  } catch (err: unknown) {
    console.error(err);
    toast.error("Failed to load versions", (err as Error | undefined)?.message || "Unknown error");
    versionOptions.value = [];
  } finally {
    isVersionsLoading.value = false;
  }
}

async function installSelected() {
  if (!selectedProject.value || !selectedVersionId.value) return;
  isInstalling.value = true;
  try {
    if (selectedInstalledFile.value) {
      await client.updateMinecraftProjectFile({
        gameServerId: props.gameServerId,
        projectId: selectedProject.value.id,
        versionId: selectedVersionId.value,
        projectType: projectType.value,
        currentFilename: selectedInstalledFile.value.filename,
        projectTitle: selectedProject.value.title,
        projectSlug: selectedProject.value.slug,
        projectIconUrl: selectedProject.value.iconUrl,
      });
      toast.success(`${selectedProject.value.title} updated`, "Restart the server to enable the new version.");
    } else {
      await client.installMinecraftProjectFile({
        gameServerId: props.gameServerId,
        projectId: selectedProject.value.id,
        versionId: selectedVersionId.value,
        projectType: projectType.value,
        projectTitle: selectedProject.value.title,
        projectSlug: selectedProject.value.slug,
        projectIconUrl: selectedProject.value.iconUrl,
      });
      toast.success(`${selectedProject.value.title} installed`, "Restart the server to enable it.");
    }
    closeInstallDialog();
    await refreshCatalogState();
  } catch (err: unknown) {
    console.error(err);
    toast.error(selectedInstalledFile.value ? "Failed to update" : "Failed to install", (err as Error | undefined)?.message || "Unknown error");
  } finally {
    isInstalling.value = false;
  }
}

async function updateInstalledFile(file: InstalledMinecraftProjectFile, opts: { refreshInstalled?: boolean } = {}) {
  if (!file.projectId) return;
  const refreshInstalled = opts.refreshInstalled ?? true;
  const targetVersionId = file.latestVersionId || file.versionId;
  if (!targetVersionId) return;
  updatingFilename.value = file.filename;
  try {
    await client.updateMinecraftProjectFile({
      gameServerId: props.gameServerId,
      projectId: file.projectId,
      versionId: targetVersionId,
      projectType: file.projectType || projectType.value,
      currentFilename: file.filename,
      projectTitle: file.title,
      projectSlug: file.projectSlug,
      projectIconUrl: file.iconUrl,
    });
    toast.success(`${file.title || file.filename} updated`, "Restart the server to apply the change.");
    if (refreshInstalled) {
      await loadInstalledFiles();
    }
  } catch (err: unknown) {
    console.error(err);
    toast.error("Failed to update", (err as Error | undefined)?.message || "Unknown error");
  } finally {
    updatingFilename.value = "";
  }
}

async function updateAllAvailable() {
  if (updateableInstalledFiles.value.length === 0) return;
  isBulkUpdating.value = true;
  try {
    const filesToUpdate = [...updateableInstalledFiles.value];
    for (const file of filesToUpdate) {
      await updateInstalledFile(file, { refreshInstalled: false });
    }
    await loadInstalledFiles();
  } finally {
    isBulkUpdating.value = false;
  }
}

onMounted(() => {
  refreshCatalogState();
});

onBeforeUnmount(() => {
  stopAutoLoadObserver();
});
</script>
