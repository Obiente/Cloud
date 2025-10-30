<template>
  <OuiStack gap="lg">
    <OuiText size="xl" weight="semibold">Navigation</OuiText>

    <OuiCard>
      <OuiCardBody>
        <OuiText size="lg" weight="medium">Tabs</OuiText>
        <div class="mt-2">
          <OuiTabs v-model="tab">
            <template #triggers>
              <Tabs.Trigger value="one" class="px-3 py-2 text-sm" :class="tab === 'one' ? 'text-primary border-b-2 border-primary' : 'text-secondary'">One</Tabs.Trigger>
              <Tabs.Trigger value="two" class="px-3 py-2 text-sm" :class="tab === 'two' ? 'text-primary border-b-2 border-primary' : 'text-secondary'">Two</Tabs.Trigger>
              <Tabs.Trigger value="three" class="px-3 py-2 text-sm" :class="tab === 'three' ? 'text-primary border-b-2 border-primary' : 'text-secondary'">Three</Tabs.Trigger>
            </template>
            <Tabs.Content value="one" class="mt-4">
              <OuiText>Tab One Content</OuiText>
            </Tabs.Content>
            <Tabs.Content value="two" class="mt-4">
              <OuiText>Tab Two Content</OuiText>
            </Tabs.Content>
            <Tabs.Content value="three" class="mt-4">
              <OuiText>Tab Three Content</OuiText>
            </Tabs.Content>
          </OuiTabs>
        </div>
      </OuiCardBody>
    </OuiCard>
  </OuiStack>
</template>

<script setup lang="ts">
import { Tabs } from '@ark-ui/vue/tabs'

definePageMeta({ layout: 'components' })

const tab = ref('one')
</script>
