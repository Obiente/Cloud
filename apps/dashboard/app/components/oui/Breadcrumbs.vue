<template>
  <nav class="oui-breadcrumbs" aria-label="Breadcrumb">
    <ol>
      <slot />
    </ol>
  </nav>
</template>

<style scoped>
.oui-breadcrumbs {
  display: flex;
  align-items: center;
  font-size: 13px;
  color: var(--oui-text-secondary);
}

.oui-breadcrumbs ol {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  list-style: none;
  margin: 0;
  padding: 0;
}
</style>
