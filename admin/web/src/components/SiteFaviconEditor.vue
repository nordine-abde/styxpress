<script setup>
import { computed, ref } from 'vue'
import FileField from './ui/FileField.vue'
import UiButton from './ui/UiButton.vue'

const defaultFavicon = 'favicon.ico'
const faviconAccept = '.ico,image/x-icon,image/vnd.microsoft.icon'

const props = defineProps({
    modelValue: {
        type: String,
        default: defaultFavicon
    },
    disabled: {
        type: Boolean,
        default: false
    }
})

const emit = defineEmits(['selected', 'reset'])
const error = ref('')
const canReset = computed(() => props.modelValue && props.modelValue !== defaultFavicon)

function selectFavicon(file) {
    error.value = ''
    if (!file) {
        return
    }
    if (!/\.ico$/i.test(file.name || '')) {
        error.value = 'Choose an .ico file.'
        return
    }
    emit('selected', file)
}
</script>

<template>
    <section class="favicon-editor">
        <div class="section-title">
            <span>Favicon</span>
            <strong>{{ modelValue || defaultFavicon }}</strong>
        </div>

        <div class="favicon-actions">
            <FileField
                label="Choose ICO"
                :accept="faviconAccept"
                :disabled="disabled"
                @selected="selectFavicon"
            />
            <UiButton tone="ghost" :disabled="disabled || !canReset" @click="$emit('reset')">
                Use default
            </UiButton>
        </div>

        <p v-if="error" class="error-text compact-text">
            {{ error }}
        </p>
    </section>
</template>

<style scoped>
.favicon-editor,
.favicon-actions {
    display: grid;
    gap: 0.75rem;
}

.section-title {
    display: grid;
    gap: 0.2rem;
}

.section-title span {
    color: var(--color-muted);
    font-size: 0.78rem;
    font-weight: 800;
    text-transform: uppercase;
}

.section-title strong {
    color: var(--color-heading);
    overflow-wrap: anywhere;
}

.compact-text {
    margin: 0;
}

@media (min-width: 720px) {
    .favicon-actions {
        grid-template-columns: minmax(0, 1fr) auto;
        align-items: end;
    }
}
</style>
