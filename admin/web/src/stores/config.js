import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import { apiRequest } from '../api/client'
import { useUiStore } from './ui'

const defaultConfig = {
    name: '',
    siteBaseUrl: '',
    contentDir: 'content',
    publicDir: 'public',
    contentStorageMode: 'local',
    remoteHost: '',
    remoteUser: '',
    sshKeyPath: '',
    sshUsePassphrase: false,
    remotePublicDir: '',
    remoteContentDir: ''
}

export const useConfigStore = defineStore('config', () => {
    const config = ref({ ...defaultConfig })
    const sites = ref([])
    const activeSiteId = ref('')
    const multiSite = ref(false)
    const loading = ref(false)
    const saving = ref(false)
    const switching = ref(false)
    const error = ref('')

    const activeSite = computed(() => sites.value.find((site) => site.id === activeSiteId.value) || null)

    async function loadConfig() {
        const uiStore = useUiStore()
        loading.value = true
        error.value = ''
        try {
            applySites(await apiRequest('/api/sites'))
            if (!activeSite.value?.config) {
                config.value = mergeConfig(await apiRequest('/api/config'))
            }
        } catch (err) {
            error.value = err.message
            uiStore.captureError(err)
        } finally {
            loading.value = false
        }
    }

    async function saveConfig(nextConfig) {
        const uiStore = useUiStore()
        saving.value = true
        error.value = ''
        try {
            config.value = mergeConfig(await apiRequest('/api/config', {
                method: 'POST',
                body: {
                    ...defaultConfig,
                    ...nextConfig
                }
            }))
            syncActiveSiteConfig(config.value)
            uiStore.setNotice('Configuration saved.')
        } catch (err) {
            error.value = err.message
            uiStore.captureError(err)
            throw err
        } finally {
            saving.value = false
        }
    }

    async function createSite(name) {
        const uiStore = useUiStore()
        switching.value = true
        error.value = ''
        try {
            const site = await apiRequest('/api/sites', {
                method: 'POST',
                body: {
                    name,
                    config: {
                        ...defaultConfig,
                        name
                    }
                }
            })
            await loadConfig()
            uiStore.setNotice(`Site "${site.name}" created.`)
            return site
        } catch (err) {
            error.value = err.message
            uiStore.captureError(err)
            throw err
        } finally {
            switching.value = false
        }
    }

    async function selectSite(id) {
        const uiStore = useUiStore()
        switching.value = true
        error.value = ''
        try {
            const site = await apiRequest(`/api/sites/${encodeURIComponent(id)}/select`, {
                method: 'POST',
                body: {}
            })
            await loadConfig()
            uiStore.setNotice(`Site "${site.name}" selected.`)
            return site
        } catch (err) {
            error.value = err.message
            uiStore.captureError(err)
            throw err
        } finally {
            switching.value = false
        }
    }

    async function deleteSite(id) {
        const uiStore = useUiStore()
        switching.value = true
        error.value = ''
        try {
            applySites(await apiRequest(`/api/sites/${encodeURIComponent(id)}`, {
                method: 'DELETE'
            }))
            uiStore.setNotice('Site deleted.')
        } catch (err) {
            error.value = err.message
            uiStore.captureError(err)
            throw err
        } finally {
            switching.value = false
        }
    }

    function applySites(payload) {
        sites.value = Array.isArray(payload?.sites) ? payload.sites : []
        activeSiteId.value = payload?.activeSiteId || ''
        multiSite.value = payload?.multiSite === true
        const active = sites.value.find((site) => site.id === activeSiteId.value)
        if (active?.config) {
            config.value = mergeConfig(active.config)
        }
    }

    function syncActiveSiteConfig(nextConfig) {
        sites.value = sites.value.map((site) => {
            if (site.id !== activeSiteId.value) {
                return site
            }
            return {
                ...site,
                name: nextConfig.name || site.name,
                config: mergeConfig(nextConfig)
            }
        })
    }

    return {
        config,
        sites,
        activeSiteId,
        multiSite,
        loading,
        saving,
        switching,
        error,
        activeSite,
        loadConfig,
        saveConfig,
        createSite,
        selectSite,
        deleteSite
    }
})

function mergeConfig(value = {}) {
    return {
        ...defaultConfig,
        ...value
    }
}
