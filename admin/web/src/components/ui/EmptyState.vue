<script setup>
defineProps({
    title: {
        type: String,
        required: true
    },
    message: {
        type: String,
        default: ''
    }
})
</script>

<template>
    <div class="empty-state">
        <strong>{{ title }}</strong>
        <p v-if="message">{{ message }}</p>
        <slot />
    </div>
</template>

<style scoped>
.empty-state {
    display: grid;
    gap: 0.55rem;
    place-items: start;
    border: 1px dashed var(--color-border);
    border-radius: 8px;
    padding: 1rem;
    background: var(--color-surface-muted);
}

strong {
    color: var(--color-heading);
}

p {
    margin: 0;
    color: var(--color-muted);
}
</style>
