<template>
    <section v-if="message" class="alert">
        <span>{{ message }}</span>
        <button type="button" @click="$emit('dismiss')">Dismiss</button>
</section>
</template>

<script setup>
import "../stylesheets/alert.css";

    defineProps({
    message: {
    type: String,
    default: "",
},
});

    defineEmits(["dismiss"]);
</script>