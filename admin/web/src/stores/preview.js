import { defineStore } from 'pinia'
import { ref } from 'vue'
import { apiRequest } from '../api/client'
import { useUiStore } from './ui'

export const usePreviewStore = defineStore('preview', () => {
    const html = ref('')
    const loading = ref(false)
    const error = ref('')

    async function renderPreview(post) {
        const uiStore = useUiStore()
        loading.value = true
        error.value = ''
        try {
            const payload = await apiRequest('/api/render-preview', {
                method: 'POST',
                body: post
            })
            html.value = payload.html || ''
        } catch (err) {
            error.value = err.message
            uiStore.captureError(err)
            throw err
        } finally {
            loading.value = false
        }
    }

    function clearPreview() {
        html.value = ''
        error.value = ''
    }

    return {
        html,
        loading,
        error,
        renderPreview,
        clearPreview
    }
})
