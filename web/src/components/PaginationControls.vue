<template>
  <footer class="pagination">
    <button
      class="button button-secondary"
      type="button"
      :disabled="page <= 1 || loading"
      @click="$emit('previous')"
    >
      Previous
    </button>

    <span>Page {{ page }} of {{ totalPages || 1 }}</span>

    <button
      class="button button-secondary"
      type="button"
      :disabled="page >= totalPages || loading"
      @click="$emit('next')"
    >
      Next
    </button>
  </footer>
</template>

<script setup>
defineProps({
  page: {
    type: Number,
    required: true,
  },
  totalPages: {
    type: Number,
    required: true,
  },
  loading: {
    type: Boolean,
    default: false,
  },
});

defineEmits(["previous", "next"]);
</script>