<script setup>
defineProps({
    label: {
        type: String,
        required: true
    },
    description: {
        type: String,
        default: ''
    },
    modelValue: {
        type: Boolean,
        default: false
    },
    disabled: {
        type: Boolean,
        default: false
    }
})

defineEmits(['update:modelValue'])
</script>

<template>
    <label class="switch-field">
        <input
            type="checkbox"
            :checked="modelValue"
            :disabled="disabled"
            @change="$emit('update:modelValue', $event.target.checked)"
        >
        <span class="switch-track" aria-hidden="true">
            <span class="switch-thumb"></span>
        </span>
        <span class="switch-copy">
            <span class="switch-label">{{ label }}</span>
            <small v-if="description">{{ description }}</small>
        </span>
    </label>
</template>

<style scoped>
.switch-field {
    display: grid;
    grid-template-columns: auto minmax(0, 1fr);
    gap: 0.7rem;
    align-items: center;
    cursor: pointer;
}

input {
    position: absolute;
    width: 1px;
    height: 1px;
    overflow: hidden;
    clip: rect(0 0 0 0);
    clip-path: inset(50%);
    white-space: nowrap;
}

.switch-track {
    display: inline-flex;
    width: 2.75rem;
    height: 1.55rem;
    align-items: center;
    border: 1px solid var(--color-border);
    border-radius: 999px;
    padding: 0.15rem;
    background: color-mix(in srgb, var(--color-surface-muted) 70%, white);
    pointer-events: none;
    transition: background 160ms ease, border-color 160ms ease;
}

.switch-thumb {
    display: block;
    width: 1.15rem;
    height: 1.15rem;
    border-radius: 999px;
    background: var(--color-surface);
    box-shadow: 0 2px 6px rgb(89 47 15 / 18%);
    transform: translateX(0);
    transition: transform 160ms ease;
}

input:checked + .switch-track {
    border-color: color-mix(in srgb, var(--color-success) 48%, var(--color-border));
    background: color-mix(in srgb, var(--color-success) 28%, var(--color-surface));
}

input:checked + .switch-track .switch-thumb {
    transform: translateX(1.15rem);
}

input:focus-visible + .switch-track {
    outline: 3px solid color-mix(in srgb, var(--color-accent) 30%, transparent);
}

.switch-copy {
    display: grid;
    gap: 0.2rem;
}

.switch-label {
    color: var(--color-heading);
    font-size: 0.88rem;
    font-weight: 800;
}

small {
    color: var(--color-muted);
}
</style>
