<template>
  <div v-if="open" class="dialog-backdrop" @click.self="$emit('close')">
    <article class="panel dialog-panel" :class="{ 'dialog-panel--wide': wide }">
      <div class="panel-header">
        <div>
          <p class="eyebrow">{{ eyebrow }}</p>
          <h3>{{ title }}</h3>
        </div>

        <button class="icon-button" type="button" @click="$emit('close')">×</button>
      </div>

      <slot />
    </article>
  </div>
</template>

<script setup>
import "../stylesheets/dialog.css";

defineProps({
  open: {
    type: Boolean,
    required: true,
  },
  eyebrow: {
    type: String,
    required: true,
  },
  title: {
    type: String,
    required: true,
  },
  wide: {
    type: Boolean,
    default: false,
  },
});

defineEmits(["close"]);
</script>
