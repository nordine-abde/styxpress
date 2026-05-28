<script setup>
import { computed, ref } from 'vue'
import UiButton from './ui/UiButton.vue'
import UiField from './ui/UiField.vue'
import { useAuthStore } from '../stores/auth'
import { useConfigStore } from '../stores/config'
import { usePostsStore } from '../stores/posts'
import { useUiStore } from '../stores/ui'

const authStore = useAuthStore()
const configStore = useConfigStore()
const postsStore = usePostsStore()
const uiStore = useUiStore()
const tokenInput = ref(authStore.token)
const showSessionBox = computed(() => {
    return !authStore.hasInjectedSession || uiStore.unauthorized || uiStore.notice || uiStore.error
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
    authStore.logout()
    tokenInput.value = ''
    postsStore.newPost()
    uiStore.setActiveView('sites')
}
</script>

<template>
    <section v-if="showSessionBox" class="session-box">
        <template v-if="authStore.hasInjectedSession">
            <p class="label">Local session</p>
            <p class="session-state">
                connected
            </p>
        </template>
        <template v-else>
            <UiField
                v-model="tokenInput"
                label="Session token"
                type="password"
                placeholder="Paste token"
                help="Printed by the local admin server."
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
    gap: 0.45rem;
    border-top: 1px solid var(--color-border);
    padding-top: 1rem;
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
    color: var(--color-text);
    font-weight: 700;
}
</style>
