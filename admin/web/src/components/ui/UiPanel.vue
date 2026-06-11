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
        <header v-if="title || subtitle" class="panel-head">
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
    border: 1px solid color-mix(in srgb, var(--color-border) 72%, white);
    border-radius: 10px;
    padding: 1rem;
    background: color-mix(in srgb, var(--color-surface) 82%, transparent);
    backdrop-filter: blur(12px);
    -webkit-backdrop-filter: blur(12px);
    box-shadow:
        var(--shadow-soft),
        inset 0 1px 0 0 color-mix(in srgb, white 45%, transparent);
    transition:
        box-shadow 0.25s ease,
        border-color 0.25s ease,
        transform 0.2s ease;
}

.panel:hover {
    border-color: color-mix(in srgb, var(--color-border) 90%, var(--color-accent));
    box-shadow:
        0 20px 48px rgb(15 23 42 / 10%),
        inset 0 1px 0 0 color-mix(in srgb, white 45%, transparent);
}

.panel-head {
    display: grid;
    gap: 0.25rem;
    padding-left: 0.7rem;
    border-left: 3px solid color-mix(in srgb, var(--color-accent) 65%, var(--color-border));
    transition: border-color 0.25s ease;
}

.panel:hover .panel-head {
    border-left-color: var(--color-accent);
}

h3 {
    margin: 0;
    color: var(--color-heading);
    font-size: 1rem;
    letter-spacing: -0.01em;
}

p {
    margin: 0;
    color: var(--color-muted);
    font-size: 0.9rem;
    line-height: 1.45;
}
</style>
