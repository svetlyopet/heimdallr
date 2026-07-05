<template>
  <nav v-if="items.length > 0" class="breadcrumb" aria-label="Breadcrumb">
    <template v-for="(item, index) in items" :key="`${item.label}-${index}`">
      <span v-if="index > 0" class="breadcrumb-separator">›</span>
      <RouterLink v-if="item.to" :to="item.to">{{ item.label }}</RouterLink>
      <span v-else>{{ item.label }}</span>
    </template>
  </nav>
</template>

<script setup>
import { RouterLink } from "vue-router";

defineProps({
  items: {
    type: Array,
    default: () => [],
  },
});
</script>
