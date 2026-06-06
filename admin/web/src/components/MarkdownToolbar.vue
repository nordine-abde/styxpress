<script setup>
import { ref } from 'vue'
import UiButton from './ui/UiButton.vue'
import UiSelect from './ui/UiSelect.vue'

const emit = defineEmits(['format'])

const blockType = ref('paragraph')
const blockOptions = [
    { value: 'paragraph', label: 'Paragraph' },
    { value: 'h1', label: 'H1' },
    { value: 'h2', label: 'H2' },
    { value: 'h3', label: 'H3' },
    { value: 'quote', label: 'Quote' },
    { value: 'list', label: 'List' }
]

function applyBlock() {
    emit('format', {
        type: 'block',
        value: blockType.value
    })
}
</script>

<template>
    <div class="markdown-toolbar" aria-label="Markdown formatting">
        <div class="block-control">
            <UiSelect v-model="blockType" label="Block" :options="blockOptions" />
            <UiButton tone="ghost" @click="applyBlock">
                Apply
            </UiButton>
        </div>
        <div class="inline-controls">
            <UiButton tone="ghost" title="Bold" aria-label="Bold" @click="$emit('format', { type: 'bold' })">
                B
            </UiButton>
            <UiButton tone="ghost" title="Italic" aria-label="Italic" @click="$emit('format', { type: 'italic' })">
                I
            </UiButton>
            <UiButton tone="ghost" title="Link" aria-label="Link" @click="$emit('format', { type: 'link' })">
                Link
            </UiButton>
        </div>
    </div>
</template>

<style scoped>
.markdown-toolbar {
    display: flex;
    flex-wrap: wrap;
    gap: 0.75rem;
    align-items: end;
    justify-content: space-between;
    border: 1px solid var(--color-border);
    border-radius: 8px;
    padding: 0.7rem;
    background: color-mix(in srgb, var(--color-surface-muted) 62%, white);
}

.block-control,
.inline-controls {
    display: flex;
    flex-wrap: wrap;
    gap: 0.5rem;
    align-items: end;
}

.block-control {
    flex: 1 1 18rem;
}

.block-control :deep(.field) {
    min-width: 11rem;
}

.inline-controls :deep(.ui-button) {
    min-width: 2.5rem;
}
</style>
