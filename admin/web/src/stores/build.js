import { defineStore } from 'pinia'
import { ref } from 'vue'
import { apiRequest } from '../api/client'
import { useDeployStore } from './deploy'
import { usePostsStore } from './posts'
import { useUiStore } from './ui'

export const useBuildStore = defineStore('build', () => {
    const rendering = ref(false)
    const publishing = ref(false)
    const lastResult = ref(null)
    const error = ref('')

    async function publishPost(slug) {
        const uiStore = useUiStore()
        const deployStore = useDeployStore()
        const postsStore = usePostsStore()
        publishing.value = true
        error.value = ''
        try {
            lastResult.value = await apiRequest(`/api/posts/${encodeURIComponent(slug)}/publish`, {
                method: 'POST',
                body: {}
            })
            await postsStore.loadPosts()
            await postsStore.selectPost(slug)
            deployStore.applyBuildResult(lastResult.value)
            uiStore.setNotice(lastResult.value?.deploy ? 'Post saved, rendered, and deployed.' : 'Post saved and rendered.')
        } catch (err) {
            error.value = err.message
            uiStore.captureError(err)
            throw err
        } finally {
            publishing.value = false
        }
    }

    async function renderSite() {
        const uiStore = useUiStore()
        const deployStore = useDeployStore()
        rendering.value = true
        error.value = ''
        try {
            lastResult.value = await apiRequest('/api/site/render', {
                method: 'POST',
                body: {}
            })
            deployStore.applyBuildResult(lastResult.value)
            uiStore.setNotice(lastResult.value?.deploy ? 'Site rendered and deployed.' : 'Site rendered locally.')
        } catch (err) {
            error.value = err.message
            uiStore.captureError(err)
            throw err
        } finally {
            rendering.value = false
        }
    }

    function reset() {
        lastResult.value = null
        error.value = ''
        const deployStore = useDeployStore()
        deployStore.reset()
    }

    return {
        rendering,
        publishing,
        lastResult,
        error,
        publishPost,
        renderSite,
        reset
    }
})
