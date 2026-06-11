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
    <button
        class="ui-button"
        :class="[tone, { 'is-busy': busy }]"
        :type="type"
        :disabled="busy || disabled"
    >
        <span v-if="busy" class="spinner" aria-hidden="true"></span>
        <span v-if="$slots.icon && !busy" class="btn-icon">
            <slot name="icon" />
        </span>
        <span class="btn-label">
            <slot />
        </span>
    </button>
</template>

<style scoped>
.ui-button {
    position: relative;
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
    transition:
        border-color 0.2s ease,
        background 0.2s ease,
        box-shadow 0.25s ease,
        transform 0.18s ease,
        opacity 0.2s ease;
}

.ui-button:hover:not(:disabled) {
    border-color: color-mix(in srgb, var(--color-accent) 70%, var(--color-accent-strong));
    box-shadow:
        0 6px 14px rgb(89 47 15 / 7%),
        0 0 0 3px color-mix(in srgb, var(--color-accent) 10%, transparent);
    transform: translateY(-1px);
}

.ui-button:active:not(:disabled) {
    transform: translateY(0.5px) scale(0.98);
    box-shadow: 0 2px 6px rgb(89 47 15 / 6%);
    transition-duration: 0.06s;
}

.ui-button:disabled {
    opacity: 0.55;
    cursor: not-allowed;
}

.ui-button.is-busy {
    cursor: wait;
}

/* --- Primary --- */
.primary {
    border-color: color-mix(in srgb, var(--color-accent) 70%, white);
    background: var(--color-accent);
    color: var(--color-heading);
    box-shadow:
        0 6px 14px rgb(89 47 15 / 7%),
        0 0 0 0 color-mix(in srgb, var(--color-accent) 0%, transparent);
}

.primary:hover:not(:disabled) {
    background: color-mix(in srgb, var(--color-accent) 88%, var(--color-accent-strong));
    border-color: var(--color-accent-strong);
    box-shadow:
        0 8px 20px rgb(20 184 166 / 18%),
        0 0 0 3px color-mix(in srgb, var(--color-accent) 18%, transparent);
}

/* --- Danger --- */
.danger {
    border-color: color-mix(in srgb, var(--color-danger) 45%, var(--color-border));
    color: var(--color-danger);
}

.danger:hover:not(:disabled) {
    border-color: color-mix(in srgb, var(--color-danger) 65%, var(--color-border));
    background: color-mix(in srgb, var(--color-danger) 6%, var(--color-surface));
    box-shadow:
        0 6px 14px rgb(89 47 15 / 7%),
        0 0 0 3px color-mix(in srgb, var(--color-danger) 10%, transparent);
}

/* --- Ghost --- */
.ghost {
    border-color: transparent;
    background: color-mix(in srgb, var(--color-surface) 42%, transparent);
    box-shadow: none;
}

.ghost:hover:not(:disabled) {
    background: color-mix(in srgb, var(--color-surface) 80%, transparent);
    border-color: var(--color-border);
    box-shadow: 0 4px 10px rgb(89 47 15 / 5%);
}

/* --- Slots --- */
.btn-icon {
    display: inline-flex;
    align-items: center;
    font-size: 1.05em;
    opacity: 0.85;
    transition: opacity 0.2s ease;
}

.ui-button:hover .btn-icon {
    opacity: 1;
}

.btn-label {
    display: inline-flex;
    align-items: center;
}

/* --- Spinner --- */
.spinner {
    width: 0.85rem;
    height: 0.85rem;
    border: 2px solid currentColor;
    border-right-color: transparent;
    border-radius: 999px;
    animation: spin 0.65s linear infinite;
    flex-shrink: 0;
}

@keyframes spin {
    to {
        transform: rotate(360deg);
    }
}
</style>
