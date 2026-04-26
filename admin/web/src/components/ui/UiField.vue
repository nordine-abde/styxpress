<script setup>
defineProps({
    label: {
        type: String,
        required: true
    },
    help: {
        type: String,
        default: ''
    },
    modelValue: {
        type: [String, Number],
        default: ''
    },
    type: {
        type: String,
        default: 'text'
    },
    multiline: {
        type: Boolean,
        default: false
    },
    rows: {
        type: Number,
        default: 8
    },
    placeholder: {
        type: String,
        default: ''
    }
})

defineEmits(['update:modelValue'])
</script>

<template>
    <label class="field">
        <span>{{ label }}</span>
        <textarea
            v-if="multiline"
            :value="modelValue"
            :rows="rows"
            :placeholder="placeholder"
            @input="$emit('update:modelValue', $event.target.value)"
        ></textarea>
        <input
            v-else
            :type="type"
            :value="modelValue"
            :placeholder="placeholder"
            @input="$emit('update:modelValue', $event.target.value)"
        />
        <small v-if="help">{{ help }}</small>
    </label>
</template>

<style scoped>
.field {
    display: grid;
    gap: 0.35rem;
}

span {
    color: var(--color-heading);
    font-size: 0.82rem;
    font-weight: 800;
}

input,
textarea {
    width: 100%;
    border: 1px solid var(--color-border);
    border-radius: 8px;
    padding: 0.72rem 0.8rem;
    background: var(--color-surface);
    color: var(--color-text);
}

textarea {
    min-height: 9rem;
    resize: vertical;
    font-family: 'SFMono-Regular', Consolas, 'Liberation Mono', monospace;
    line-height: 1.55;
}

input:focus,
textarea:focus {
    border-color: var(--color-accent);
    outline: 3px solid color-mix(in srgb, var(--color-accent) 18%, transparent);
}

small {
    color: var(--color-muted);
}
</style>
