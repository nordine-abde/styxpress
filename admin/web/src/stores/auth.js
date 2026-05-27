import { defineStore } from 'pinia'
import { computed, ref } from 'vue'

const injectedToken = window.__STYXPRESS_SESSION__ || ''

export const useAuthStore = defineStore('auth', () => {
    const token = ref(injectedToken)

    const hasToken = computed(() => token.value.trim() !== '')
    const hasInjectedSession = computed(() => injectedToken.trim() !== '')

    function setToken(value) {
        token.value = value.trim()
    }

    function logout() {
        setToken('')
    }

    return {
        token,
        hasToken,
        hasInjectedSession,
        setToken,
        logout
    }
})
