<template>
  <section class="stats-grid">
    <article class="stat-card">
      <span>Total</span>
      <strong>{{ pagination.total }}</strong>
    </article>

    <article class="stat-card">
      <span>Current page</span>
      <strong>{{ pagination.page }}</strong>
    </article>

    <article class="stat-card">
      <span>Total pages</span>
      <strong>{{ pagination.total_pages }}</strong>
    </article>

    <article class="stat-card">
      <span>Page size</span>
      <strong>{{ pagination.limit }}</strong>
    </article>
  </section>
</template>

<script setup>
defineProps({
  pagination: {
    type: Object,
    required: true,
  },
});
</script>