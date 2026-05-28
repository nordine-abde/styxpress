<script setup>
defineProps({
    label: {
        type: String,
        required: true
    },
    modelValue: {
        type: String,
        required: true
    },
    options: {
        type: Array,
        required: true
    }
})

defineEmits(['update:modelValue'])
</script>

<template>
    <label class="field">
        <span>{{ label }}</span>
        <select :value="modelValue" @change="$emit('update:modelValue', $event.target.value)">
            <option v-for="option in options" :key="option.value" :value="option.value">
                {{ option.label }}
            </option>
        </select>
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

select {
    width: 100%;
    border: 1px solid color-mix(in srgb, var(--color-border) 86%, white);
    border-radius: 8px;
    padding: 0.72rem 0.8rem;
    background: color-mix(in srgb, var(--color-surface) 92%, white);
    color: var(--color-text);
}
</style>
