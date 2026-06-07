import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import { apiRequest } from '../api/client'
import { useDeployStore } from './deploy'
import { useUiStore } from './ui'

const defaultConfig = {
    name: '',
    contentDir: 'content',
    publicDir: 'public',
    deploy: {
        enabled: false,
        sftp: {
            host: '',
            port: 22,
            user: '',
            remotePath: '',
            keyPath: '',
            knownHostsPath: '',
            deleteExtra: false
        }
    }
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
            if (!activeSite.value?.config && !multiSite.value) {
                config.value = mergeConfig(await apiRequest('/api/config'))
            }
            syncDeployStore()
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
                body: mergeConfig(nextConfig)
            }))
            syncActiveSiteConfig(config.value)
            syncDeployStore()
            uiStore.setNotice('Configuration saved.')
        } catch (err) {
            error.value = err.message
            uiStore.captureError(err)
            throw err
        } finally {
            saving.value = false
        }
    }

    async function suggestSite(name) {
        return apiRequest(`/api/sites/suggestion?name=${encodeURIComponent(name)}`)
    }

    async function createSite(input) {
        const uiStore = useUiStore()
        const name = typeof input === 'string' ? input : input?.name
        const configInput = typeof input === 'string' ? {} : input?.config || {}
        switching.value = true
        error.value = ''
        try {
            const site = await apiRequest('/api/sites', {
                method: 'POST',
                body: {
                    name,
                    config: configInput
                }
            })
            await loadConfig()
            syncDeployStore()
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
            syncDeployStore()
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
            syncDeployStore()
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
            syncDeployStore()
            return
        }
        config.value = mergeConfig()
        syncDeployStore()
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

    function syncDeployStore() {
        const deployStore = useDeployStore()
        deployStore.syncFromConfig(config.value)
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
        suggestSite,
        createSite,
        selectSite,
        deleteSite
    }
})

function mergeConfig(value = {}) {
    const deploy = value.deploy || {}
    const deployConfig = { ...deploy }
    delete deployConfig.mode
    const sftp = deploy.sftp || {}
    return {
        ...defaultConfig,
        ...value,
        deploy: {
            ...defaultConfig.deploy,
            ...deployConfig,
            enabled: deploy.enabled === true,
            sftp: {
                ...defaultConfig.deploy.sftp,
                ...sftp,
                port: normalizePort(sftp.port),
                deleteExtra: sftp.deleteExtra === true
            }
        }
    }
}

function normalizePort(value) {
    const port = Number(value || defaultConfig.deploy.sftp.port)
    if (!Number.isFinite(port)) {
        return defaultConfig.deploy.sftp.port
    }
    return Math.min(65535, Math.max(1, Math.trunc(port)))
}
