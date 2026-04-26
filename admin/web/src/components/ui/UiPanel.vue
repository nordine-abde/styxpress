<script setup>
defineProps({
    title: {
        type: String,
        default: ''
    },
    subtitle: {
        type: String,
        default: ''
    }
})
</script>

<template>
    <section class="panel">
        <header v-if="title || subtitle">
            <h3 v-if="title">{{ title }}</h3>
            <p v-if="subtitle">{{ subtitle }}</p>
        </header>
        <slot />
    </section>
</template>

<style scoped>
.panel {
    display: grid;
    gap: 1rem;
    border: 1px solid var(--color-border);
    border-radius: 8px;
    padding: 1rem;
    background: var(--color-surface);
    box-shadow: var(--shadow-soft);
}

header {
    display: grid;
    gap: 0.2rem;
}

h3 {
    margin: 0;
    color: var(--color-heading);
    font-size: 1rem;
}

p {
    margin: 0;
    color: var(--color-muted);
    font-size: 0.9rem;
}
</style>
