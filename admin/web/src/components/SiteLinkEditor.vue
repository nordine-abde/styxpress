<script setup>
import { computed } from 'vue'
import UiButton from './ui/UiButton.vue'
import UiField from './ui/UiField.vue'

const props = defineProps({
    title: {
        type: String,
        required: true
    },
    modelValue: {
        type: Array,
        default: () => []
    }
})

const emit = defineEmits(['update:modelValue'])

const links = computed(() => props.modelValue || [])

function updateLink(index, field, value) {
    const next = links.value.map((link, currentIndex) => (
        currentIndex === index ? { ...link, [field]: value } : link
    ))
    emit('update:modelValue', next)
}

function addLink() {
    emit('update:modelValue', [...links.value, { label: '', href: '' }])
}

function removeLink(index) {
    emit('update:modelValue', links.value.filter((_, currentIndex) => currentIndex !== index))
}
</script>

<template>
    <section class="link-editor">
        <header>
            <h4>{{ title }}</h4>
            <UiButton tone="ghost" @click="addLink">
                Add link
            </UiButton>
        </header>

        <div v-if="links.length" class="link-list">
            <div v-for="(link, index) in links" :key="index" class="link-row">
                <UiField
                    label="Label"
                    :model-value="link.label"
                    @update:model-value="updateLink(index, 'label', $event)"
                />
                <UiField
                    label="Href"
                    :model-value="link.href"
                    placeholder="/, /feed.xml, https://example.com"
                    @update:model-value="updateLink(index, 'href', $event)"
                />
                <UiButton tone="danger" @click="removeLink(index)">
                    Remove
                </UiButton>
            </div>
        </div>
    </section>
</template>

<style scoped>
.link-editor {
    display: grid;
    gap: 0.75rem;
}

header {
    display: flex;
    flex-wrap: wrap;
    gap: 0.75rem;
    align-items: center;
    justify-content: space-between;
}

h4 {
    margin: 0;
    color: var(--color-heading);
    font-size: 0.92rem;
}

.link-list {
    display: grid;
    gap: 0.7rem;
}

.link-row {
    display: grid;
    gap: 0.7rem;
    align-items: end;
    border: 1px solid var(--color-border);
    border-radius: 8px;
    padding: 0.75rem;
}

@media (min-width: 780px) {
    .link-row {
        grid-template-columns: minmax(0, 0.8fr) minmax(0, 1.4fr) auto;
    }
}
</style>
