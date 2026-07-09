<template>
  <div class="nav-group" :class="{ 'nav-group-expanded': expanded }">
    <button
      type="button"
      class="nav-group-toggle"
      :aria-expanded="expanded"
      :aria-controls="groupId"
      @click="$emit('toggle')"
    >
      <span class="nav-group-title">{{ title }}</span>
      <span class="nav-group-chevron" aria-hidden="true"></span>
    </button>

    <div
      :id="groupId"
      class="nav-group-panel"
      :class="{ 'nav-group-panel-expanded': expanded }"
      :aria-hidden="!expanded"
    >
      <div class="nav-group-panel-inner">
        <div class="nav-group-items">
          <slot />
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
defineProps({
  title: {
    type: String,
    required: true,
  },
  expanded: {
    type: Boolean,
    default: false,
  },
  groupId: {
    type: String,
    required: true,
  },
});

defineEmits(["toggle"]);
</script>
