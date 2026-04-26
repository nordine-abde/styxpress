<script setup>
defineProps({
    tone: {
        type: String,
        default: 'neutral'
    }
})
</script>

<template>
    <span class="badge" :class="tone">
        <slot />
    </span>
</template>

<style scoped>
.badge {
    display: inline-flex;
    min-height: 1.55rem;
    align-items: center;
    border: 1px solid var(--color-border);
    border-radius: 999px;
    padding: 0 0.55rem;
    background: var(--color-surface-muted);
    color: var(--color-muted);
    font-size: 0.75rem;
    font-weight: 800;
}

.success {
    border-color: color-mix(in srgb, var(--color-success) 28%, var(--color-border));
    color: var(--color-success);
}

.warning {
    border-color: color-mix(in srgb, var(--color-warning) 35%, var(--color-border));
    color: var(--color-warning);
}

.danger {
    border-color: color-mix(in srgb, var(--color-danger) 35%, var(--color-border));
    color: var(--color-danger);
}
</style>
