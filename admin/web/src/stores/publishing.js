import { defineStore } from 'pinia'
import { ref } from 'vue'
import { apiRequest } from '../api/client'
import { useUiStore } from './ui'

export const usePublishingStore = defineStore('publishing', () => {
    const testing = ref(false)
    const rendering = ref(false)
    const publishing = ref(false)
    const lastResult = ref(null)
    const error = ref('')

    async function testSSH(passphrase) {
        const uiStore = useUiStore()
        testing.value = true
        error.value = ''
        try {
            await apiRequest('/api/test-ssh', {
                method: 'POST',
                body: { passphrase }
            })
            uiStore.setNotice('SSH connection succeeded.')
        } catch (err) {
            error.value = err.message
            uiStore.captureError(err)
            throw err
        } finally {
            testing.value = false
        }
    }

    async function renderPost(slug) {
        const uiStore = useUiStore()
        rendering.value = true
        error.value = ''
        try {
            lastResult.value = await apiRequest(`/api/posts/${encodeURIComponent(slug)}/render`, {
                method: 'POST',
                body: {}
            })
            uiStore.setNotice('Post rendered locally.')
        } catch (err) {
            error.value = err.message
            uiStore.captureError(err)
            throw err
        } finally {
            rendering.value = false
        }
    }

    async function publishPost(slug, passphrase) {
        const uiStore = useUiStore()
        publishing.value = true
        error.value = ''
        try {
            lastResult.value = await apiRequest(`/api/posts/${encodeURIComponent(slug)}/publish`, {
                method: 'POST',
                body: { passphrase }
            })
            uiStore.setNotice('Post published.')
        } catch (err) {
            error.value = err.message
            uiStore.captureError(err)
            throw err
        } finally {
            publishing.value = false
        }
    }

    return {
        testing,
        rendering,
        publishing,
        lastResult,
        error,
        testSSH,
        renderPost,
        publishPost
    }
})
