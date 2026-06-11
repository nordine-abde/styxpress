<script setup>
import { computed, ref } from 'vue'
import UiButton from './ui/UiButton.vue'
import UiField from './ui/UiField.vue'
import { useAuthStore } from '../stores/auth'
import { useConfigStore } from '../stores/config'
import { usePostsStore } from '../stores/posts'
import { useSiteConfigStore } from '../stores/siteConfig'
import { useUiStore } from '../stores/ui'

const authStore = useAuthStore()
const configStore = useConfigStore()
const postsStore = usePostsStore()
const siteConfigStore = useSiteConfigStore()
const uiStore = useUiStore()
const tokenInput = ref(authStore.token)
const showTokenForm = computed(() => !authStore.hasToken || uiStore.unauthorized)
const hasUnsavedChanges = computed(() => siteConfigStore.isDirty || postsStore.isDirty)

const userInitial = computed(() => {
    const name = configStore.config?.name || ''
    return name.trim().charAt(0).toUpperCase() || 'S'
})

async function applyToken() {
    authStore.setToken(tokenInput.value)
    uiStore.clearMessages()
    if (authStore.hasToken) {
        await configStore.loadConfig()
        uiStore.setActiveView('sites')
    }
}

function clearToken() {
    if (!confirmDiscardUnsavedChanges()) {
        return
    }
    authStore.logout()
    tokenInput.value = ''
    postsStore.clearSelection()
    uiStore.setActiveView('sites')
}

function confirmDiscardUnsavedChanges() {
    if (!hasUnsavedChanges.value) {
        return true
    }
    const confirmed = window.confirm('You have unsaved changes. Leave without saving?')
    if (confirmed) {
        siteConfigStore.setDirty(false)
        postsStore.discardDraftChanges()
    }
    return confirmed
}
</script>

<template>
    <section class="session-box">
        <template v-if="!showTokenForm">
            <div class="session-profile">
                <div class="avatar">
                    <span>{{ userInitial }}</span>
                </div>
                <div class="session-info">
                    <p class="label">Local session</p>
                    <p class="session-state">
                        <span class="status-dot"></span>
                        connected
                    </p>
                </div>
            </div>
            <div class="button-row">
                <UiButton tone="ghost" @click="clearToken">
                    Clear
                </UiButton>
            </div>
        </template>
        <template v-else>
            <UiField
                v-model="tokenInput"
                label="Session token"
                type="password"
                placeholder="Paste token"
                help="Copy the API session token from the terminal where styxpress-admin is running."
                @keydown.enter.prevent="applyToken"
            />
            <div class="button-row">
                <UiButton tone="primary" @click="applyToken">
                    Set
                </UiButton>
                <UiButton v-if="authStore.hasToken" tone="ghost" @click="clearToken">
                    Clear
                </UiButton>
            </div>
        </template>
        <p v-if="uiStore.unauthorized" class="error-text">
            The local session was rejected. Restart the admin server and reload.
        </p>
        <p v-if="uiStore.notice" class="success-text">
            {{ uiStore.notice }}
        </p>
        <p v-if="uiStore.error" class="error-text">
            {{ uiStore.error }}
        </p>
    </section>
</template>

<style scoped>
.session-box {
    display: grid;
    gap: 0.6rem;
    border-top: 1px solid var(--color-border);
    padding-top: 1rem;
}

.session-profile {
    display: flex;
    align-items: center;
    gap: 0.7rem;
}

.avatar {
    display: flex;
    align-items: center;
    justify-content: center;
    flex: 0 0 auto;
    width: 2.2rem;
    height: 2.2rem;
    border-radius: 50%;
    background: linear-gradient(135deg, var(--color-accent) 0%, var(--color-accent-strong) 100%);
    color: #ffffff;
    font-size: 0.88rem;
    font-weight: 800;
    letter-spacing: -0.02em;
    user-select: none;
    box-shadow: 0 2px 8px color-mix(in srgb, var(--color-accent) 28%, transparent);
}

.session-info {
    display: grid;
    gap: 0.1rem;
    min-width: 0;
}

p {
    margin: 0;
    font-size: 0.86rem;
}

.label {
    color: var(--color-muted);
    font-size: 0.72rem;
    font-weight: 800;
    letter-spacing: 0.08em;
    text-transform: uppercase;
}

.session-state {
    display: flex;
    align-items: center;
    gap: 0.4rem;
    color: var(--color-text);
    font-weight: 700;
}

@keyframes pulse-status {
    0%, 100% { opacity: 0.6; }
    50% { opacity: 1; }
}

.status-dot {
    display: inline-block;
    width: 7px;
    height: 7px;
    border-radius: 50%;
    background: var(--color-success);
    animation: pulse-status 2s ease infinite;
    box-shadow: 0 0 6px color-mix(in srgb, var(--color-success) 36%, transparent);
}
</style>
