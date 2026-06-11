<script setup>
import { ref } from 'vue'
import UiButton from './UiButton.vue'

defineProps({
    label: {
        type: String,
        required: true
    },
    confirmLabel: {
        type: String,
        default: 'Confirm'
    },
    disabled: {
        type: Boolean,
        default: false
    },
    message: {
        type: String,
        default: ''
    }
})

const emit = defineEmits(['confirm'])
const confirming = ref(false)

function confirm() {
    emit('confirm')
    confirming.value = false
}
</script>

<template>
    <div class="confirm-prompt">
        <UiButton v-if="!confirming" tone="danger" :disabled="disabled" @click="confirming = true">
            {{ label }}
        </UiButton>
        <template v-else>
            <p v-if="message" class="confirm-message">
                {{ message }}
            </p>
            <UiButton tone="danger" :disabled="disabled" @click="confirm">
                {{ confirmLabel }}
            </UiButton>
            <UiButton tone="ghost" @click="confirming = false">
                Cancel
            </UiButton>
        </template>
    </div>
</template>

<style scoped>
.confirm-prompt {
    display: flex;
    flex-wrap: wrap;
    gap: 0.5rem;
    align-items: center;
}

.confirm-message {
    flex: 1 1 100%;
    margin: 0;
    color: var(--color-muted);
    font-size: 0.78rem;
}
</style>
