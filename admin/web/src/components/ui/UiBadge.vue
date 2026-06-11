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
        <span class="badge-dot" aria-hidden="true"></span>
        <slot />
    </span>
</template>

<style scoped>
.badge {
    --_badge-color: var(--color-muted);
    display: inline-flex;
    min-height: 1.55rem;
    align-items: center;
    gap: 0.4rem;
    border: 1px solid color-mix(in srgb, var(--color-border) 78%, white);
    border-radius: 999px;
    padding: 0 0.6rem;
    background: color-mix(in srgb, var(--color-surface-muted) 76%, white);
    color: var(--_badge-color);
    font-size: 0.75rem;
    font-weight: 800;
    transition:
        background 0.2s ease,
        border-color 0.2s ease,
        transform 0.18s ease,
        box-shadow 0.2s ease;
}

.badge:hover {
    transform: translateY(-0.5px);
    box-shadow: 0 2px 8px color-mix(in srgb, var(--_badge-color) 12%, transparent);
}

.badge-dot {
    width: 0.38rem;
    height: 0.38rem;
    border-radius: 999px;
    background: var(--_badge-color);
    flex-shrink: 0;
    transition: background 0.2s ease, box-shadow 0.3s ease;
    box-shadow: 0 0 0 0 color-mix(in srgb, var(--_badge-color) 0%, transparent);
}

.badge:hover .badge-dot {
    box-shadow: 0 0 0 2.5px color-mix(in srgb, var(--_badge-color) 18%, transparent);
}

/* --- Success --- */
.success {
    --_badge-color: var(--color-success);
    border-color: color-mix(in srgb, var(--color-success) 28%, var(--color-border));
    background: color-mix(in srgb, var(--color-success) 6%, var(--color-surface-muted));
}

/* --- Warning --- */
.warning {
    --_badge-color: var(--color-warning);
    border-color: color-mix(in srgb, var(--color-warning) 35%, var(--color-border));
    background: color-mix(in srgb, var(--color-warning) 6%, var(--color-surface-muted));
}

/* --- Danger --- */
.danger {
    --_badge-color: var(--color-danger);
    border-color: color-mix(in srgb, var(--color-danger) 35%, var(--color-border));
    background: color-mix(in srgb, var(--color-danger) 6%, var(--color-surface-muted));
}
</style>
