import { defineStore } from 'pinia'
import { ref } from 'vue'
import { apiRequest } from '../api/client'
import { useUiStore } from './ui'

const defaultConfig = {
    siteBaseUrl: '',
    contentDir: 'content',
    publicDir: 'public',
    contentStorageMode: 'local',
    remoteHost: '',
    remoteUser: '',
    sshKeyPath: '',
    remotePublicDir: '',
    remoteContentDir: ''
}

export const useConfigStore = defineStore('config', () => {
    const config = ref({ ...defaultConfig })
    const loading = ref(false)
    const saving = ref(false)
    const error = ref('')

    async function loadConfig() {
        const uiStore = useUiStore()
        loading.value = true
        error.value = ''
        try {
            config.value = {
                ...defaultConfig,
                ...(await apiRequest('/api/config'))
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
            config.value = await apiRequest('/api/config', {
                method: 'POST',
                body: {
                    ...defaultConfig,
                    ...nextConfig
                }
            })
            uiStore.setNotice('Configuration saved.')
        } catch (err) {
            error.value = err.message
            uiStore.captureError(err)
            throw err
        } finally {
            saving.value = false
        }
    }

    return {
        config,
        loading,
        saving,
        error,
        loadConfig,
        saveConfig
    }
})
