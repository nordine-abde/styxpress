import { defineStore } from 'pinia'
import { ref, shallowRef } from 'vue'

export const useUiStore = defineStore('ui', () => {
    const activeView = ref('sites')
    const notice = ref('')
    const error = ref('')
    const unauthorized = ref(false)
    const headerSaveAction = shallowRef(null)

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

    function registerHeaderSaveAction(key, action) {
        headerSaveAction.value = {
            key,
            ...action
        }
    }

    function unregisterHeaderSaveAction(key) {
        if (headerSaveAction.value?.key === key) {
            headerSaveAction.value = null
        }
    }

    return {
        activeView,
        notice,
        error,
        unauthorized,
        headerSaveAction,
        setActiveView,
        setNotice,
        captureError,
        clearMessages,
        registerHeaderSaveAction,
        unregisterHeaderSaveAction
    }
})
