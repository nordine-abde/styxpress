<script setup>
defineProps({
    label: {
        type: String,
        required: true
    },
    accept: {
        type: String,
        default: ''
    }
})

defineEmits(['selected'])
</script>

<template>
    <label class="file-field">
        <span>{{ label }}</span>
        <input
            type="file"
            :accept="accept"
            @change="$emit('selected', $event.target.files[0]); $event.target.value = ''"
        />
    </label>
</template>

<style scoped>
.file-field {
    display: grid;
    gap: 0.35rem;
}

span {
    color: var(--color-heading);
    font-size: 0.82rem;
    font-weight: 800;
}

input {
    width: 100%;
    border: 1px solid var(--color-border);
    border-radius: 8px;
    padding: 0.6rem;
    background: var(--color-surface);
}
</style>
