import { defineStore } from 'pinia'
import { ref } from 'vue'
import { useConfigStore } from './config'
import { usePostsStore } from './posts'
import { useSiteConfigStore } from './siteConfig'
import { useUiStore } from './ui'

export const useSiteWorkspaceStore = defineStore('siteWorkspace', () => {
    const loadedSiteId = ref('')
    const loading = ref(false)
    const error = ref('')

    async function loadCurrentSite(options = {}) {
        const configStore = useConfigStore()
        const postsStore = usePostsStore()
        const siteConfigStore = useSiteConfigStore()
        const uiStore = useUiStore()
        const siteId = configStore.activeSiteId || 'single-site'
        if (configStore.multiSite && !configStore.activeSiteId) {
            reset()
            uiStore.setActiveView('sites')
            return
        }

        if (!options.force && loadedSiteId.value === siteId) {
            return
        }

        loading.value = true
        error.value = ''
        postsStore.newPost()

        try {
            await Promise.all([
                siteConfigStore.loadSiteConfig(),
                postsStore.loadPosts(),
                postsStore.loadFeatured()
            ])
            loadedSiteId.value = siteId
        } catch (err) {
            error.value = err.message
            uiStore.captureError(err)
            throw err
        } finally {
            loading.value = false
        }
    }

    function reset() {
        loadedSiteId.value = ''
        error.value = ''
        usePostsStore().reset()
        useSiteConfigStore().reset()
    }

    return {
        loadedSiteId,
        loading,
        error,
        loadCurrentSite,
        reset
    }
})
