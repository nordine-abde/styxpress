<script setup>
defineProps({
    tone: {
        type: String,
        default: 'neutral'
    },
    type: {
        type: String,
        default: 'button'
    },
    busy: {
        type: Boolean,
        default: false
    },
    disabled: {
        type: Boolean,
        default: false
    }
})
</script>

<template>
    <button class="ui-button" :class="tone" :type="type" :disabled="busy || disabled">
        <span v-if="busy" class="spinner" aria-hidden="true"></span>
        <slot />
    </button>
</template>

<style scoped>
.ui-button {
    display: inline-flex;
    min-height: 2.35rem;
    align-items: center;
    justify-content: center;
    gap: 0.45rem;
    border: 1px solid var(--color-border);
    border-radius: 8px;
    padding: 0 0.85rem;
    background: var(--color-surface);
    color: var(--color-heading);
    font-weight: 700;
}

.ui-button:hover {
    border-color: var(--color-accent);
}

.primary {
    border-color: var(--color-accent);
    background: var(--color-accent);
    color: white;
}

.danger {
    border-color: color-mix(in srgb, var(--color-danger) 45%, var(--color-border));
    color: var(--color-danger);
}

.ghost {
    background: transparent;
}

.spinner {
    width: 0.8rem;
    height: 0.8rem;
    border: 2px solid currentColor;
    border-right-color: transparent;
    border-radius: 999px;
    animation: spin 0.7s linear infinite;
}

@keyframes spin {
    to {
        transform: rotate(360deg);
    }
}
</style>
