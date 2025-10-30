<template>
  <OuiStack gap="lg">
    <OuiText size="xl" weight="semibold">Inputs</OuiText>

    <OuiCard>
      <OuiCardBody>
        <OuiText size="lg" weight="medium">Select</OuiText>
        <div class="max-w-sm mt-2">
          <OuiSelect v-model="selectValue" :items="selectItems" label="Framework" />
          <OuiText class="mt-2 text-sm text-secondary">Value: {{ selectValue }}</OuiText>
        </div>
      </OuiCardBody>
    </OuiCard>

    <OuiCard>
      <OuiCardBody>
        <OuiText size="lg" weight="medium">Checkbox</OuiText>
        <div class="mt-2">
          <OuiCheckbox v-model="checkboxA">Option A</OuiCheckbox>
          <OuiCheckbox v-model="checkboxB">Option B</OuiCheckbox>
          <OuiText class="mt-2 text-sm text-secondary">A: {{ checkboxA }} | B: {{ checkboxB }}</OuiText>
        </div>
      </OuiCardBody>
    </OuiCard>

    <OuiCard>
      <OuiCardBody>
        <OuiText size="lg" weight="medium">Radio Group</OuiText>
        <div class="mt-2">
          <OuiRadioGroup v-model="radio" :options="radioOptions" label="Choice" />
          <OuiText class="mt-2 text-sm text-secondary">Value: {{ radio }}</OuiText>
        </div>
      </OuiCardBody>
    </OuiCard>

    <OuiCard>
      <OuiCardBody>
        <OuiText size="lg" weight="medium">Switch</OuiText>
        <div class="mt-2">
          <OuiSwitch v-model="switchOn" label="Enable feature" />
          <OuiText class="mt-2 text-sm text-secondary">On: {{ switchOn }}</OuiText>
        </div>
      </OuiCardBody>
    </OuiCard>

    <OuiCard>
      <OuiCardBody>
        <OuiText size="lg" weight="medium">Input</OuiText>
        <div class="max-w-sm mt-2">
          <OuiInput v-model="textValue" label="Name" placeholder="Jane Doe" />
          <OuiText class="mt-2 text-sm text-secondary">Value: {{ textValue }}</OuiText>
        </div>
      </OuiCardBody>
    </OuiCard>

    <OuiCard>
      <OuiCardBody>
        <OuiText size="lg" weight="medium">Combobox</OuiText>
        <div class="max-w-sm mt-2">
          <Combobox v-model="comboValue" :options="selectItems" label="Pick one" />
          <OuiText class="mt-2 text-sm text-secondary">Value: {{ comboValue }}</OuiText>
        </div>
      </OuiCardBody>
    </OuiCard>
  </OuiStack>
</template>

<script setup lang="ts">
import Combobox from '~/components/oui/Combobox.vue'
import OuiRadioGroup from '~/components/oui/RadioGroup.vue'

definePageMeta({ layout: 'components' })

const selectItems = [
  { label: 'Vue', value: 'vue' },
  { label: 'React', value: 'react' },
  { label: 'Svelte', value: 'svelte' },
]

const radioOptions = [
  { label: 'Alpha', value: 'alpha' },
  { label: 'Beta', value: 'beta' },
  { label: 'Gamma', value: 'gamma' },
]

const selectValue = ref<string | null>('vue')
const checkboxA = ref(false)
const checkboxB = ref(true)
const radio = ref('alpha')
const switchOn = ref(true)
const textValue = ref('')
const comboValue = ref('react')
</script>
