import { defineStore } from 'pinia'
import { computed, ref } from 'vue'

export const useAuthStore = defineStore('auth', () => {
    const token = ref('')

    const hasToken = computed(() => token.value.trim() !== '')

    function setToken(value) {
        token.value = value.trim()
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
