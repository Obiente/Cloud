<template>
  <Menu.Separator class="oui-context-menu-separator" />
</template>

<script setup lang="ts">
import { Menu } from "@ark-ui/vue/menu";
</script>

<style scoped>
.oui-context-menu-separator {
  height: 1px;
  margin: 6px 4px;
  background: var(--oui-border-muted);
}
</style>
