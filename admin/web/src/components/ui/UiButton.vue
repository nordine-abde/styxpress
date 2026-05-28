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
    background: color-mix(in srgb, var(--color-surface) 88%, white);
    color: var(--color-heading);
    font-weight: 700;
    box-shadow: 0 6px 14px rgb(89 47 15 / 7%);
}

.ui-button:hover {
    border-color: color-mix(in srgb, var(--color-accent) 70%, var(--color-accent-strong));
    transform: translateY(-1px);
}

.primary {
    border-color: color-mix(in srgb, var(--color-accent) 70%, white);
    background: var(--color-accent);
    color: var(--color-heading);
}

.danger {
    border-color: color-mix(in srgb, var(--color-danger) 45%, var(--color-border));
    color: var(--color-danger);
}

.ghost {
    background: color-mix(in srgb, var(--color-surface) 42%, transparent);
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
