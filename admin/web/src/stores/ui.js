import { defineStore } from 'pinia'
import { ref } from 'vue'

export const useUiStore = defineStore('ui', () => {
    const activeView = ref('posts')
    const notice = ref('')
    const error = ref('')
    const unauthorized = ref(false)

    function setActiveView(view) {
        activeView.value = view
    }

    function setNotice(message) {
        notice.value = message
        error.value = ''
        window.setTimeout(() => {
            if (notice.value === message) {
                notice.value = ''
            }
        }, 4200)
    }

    function captureError(err) {
        unauthorized.value = err?.status === 401
        error.value = err?.message || 'Request failed'
        notice.value = ''
    }

    function clearMessages() {
        notice.value = ''
        error.value = ''
        unauthorized.value = false
    }

    return {
        activeView,
        notice,
        error,
        unauthorized,
        setActiveView,
        setNotice,
        captureError,
        clearMessages
    }
})
