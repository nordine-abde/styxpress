import { defineStore } from 'pinia'
import { computed, ref } from 'vue'

const tokenKey = 'styxpress.sessionToken'

export const useAuthStore = defineStore('auth', () => {
    const token = ref(window.localStorage.getItem(tokenKey) || '')

    const hasToken = computed(() => token.value.trim() !== '')

    function setToken(value) {
        token.value = value.trim()
        if (token.value) {
            window.localStorage.setItem(tokenKey, token.value)
        } else {
            window.localStorage.removeItem(tokenKey)
        }
    }

    function logout() {
        setToken('')
    }

    return {
        token,
        hasToken,
        setToken,
        logout
    }
})
