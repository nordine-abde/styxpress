<script setup>
import { ref } from 'vue'
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

async function applyToken() {
    authStore.setToken(tokenInput.value)
    uiStore.clearMessages()
    if (authStore.hasToken) {
        await Promise.all([
            configStore.loadConfig(),
            postsStore.loadPosts(),
            postsStore.loadFeatured()
        ])
    }
}

function logout() {
    authStore.logout()
    tokenInput.value = ''
    postsStore.newPost()
    uiStore.setActiveView('config')
}
</script>

<template>
    <section class="auth-box">
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
            <UiButton v-if="authStore.hasToken" tone="ghost" @click="logout">
                Clear
            </UiButton>
        </div>
        <p v-if="uiStore.unauthorized" class="error-text">
            The saved token was rejected.
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
.auth-box {
    display: grid;
    gap: 0.8rem;
    border-top: 1px solid var(--color-border);
    padding-top: 1rem;
}

p {
    margin: 0;
    font-size: 0.86rem;
}
</style>
