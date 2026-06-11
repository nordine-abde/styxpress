<script setup>
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import AuthBar from './components/AuthBar.vue'
import ConfigScreen from './components/ConfigScreen.vue'
import PostCreateSetup from './components/PostCreateSetup.vue'
import PostEditor from './components/PostEditor.vue'
import PostList from './components/PostList.vue'
import SiteConfigScreen from './components/SiteConfigScreen.vue'
import SiteListScreen from './components/SiteListScreen.vue'
import UiButton from './components/ui/UiButton.vue'
import UiBadge from './components/ui/UiBadge.vue'
import UiField from './components/ui/UiField.vue'
import styxpressMarkUrl from './assets/styxpress-mark.png'
import { useAuthStore } from './stores/auth'
import { useBuildStore } from './stores/build'
import { useConfigStore } from './stores/config'
import { useDeployStore } from './stores/deploy'
import { usePostsStore } from './stores/posts'
import { useSiteConfigStore } from './stores/siteConfig'
import { useSiteWorkspaceStore } from './stores/siteWorkspace'
import { useUiStore } from './stores/ui'

const authStore = useAuthStore()
const buildStore = useBuildStore()
const configStore = useConfigStore()
const deployStore = useDeployStore()
const postsStore = usePostsStore()
const siteConfigStore = useSiteConfigStore()
const siteWorkspaceStore = useSiteWorkspaceStore()
const uiStore = useUiStore()
const sidebarOpen = ref(true)
const deploySecret = ref('')
const deploySecretOpen = ref(false)

const activeLabel = computed(() => {
    if (uiStore.activeView === 'sites') {
        return 'Sites'
    }
    if (uiStore.activeView === 'config') {
        return 'Configuration'
    }
    if (uiStore.activeView === 'site') {
        return 'Site'
    }
    return 'Posts'
})

const activeSiteName = computed(() => {
    return configStore.activeSite?.name || configStore.config?.name || 'Configured site'
})

const saveAction = computed(() => uiStore.headerSaveAction)
const saveAvailable = computed(() => {
    const action = saveAction.value
    return Boolean(action?.run) && action?.isAvailable?.() !== false && action?.isDirty?.() !== false
})
const headerSaving = computed(() => Boolean(saveAction.value?.isBusy?.()))
const localChangesToSave = computed(() => {
    return Boolean(saveAction.value?.isDirty?.()) || siteConfigStore.isDirty || postsStore.isDirty
})
const hasUnsavedChanges = computed(() => localChangesToSave.value)
const localSaveLabel = computed(() => (
    localChangesToSave.value ? 'Local changes to save' : 'No local changes to save'
))
const deployStatusLabel = computed(() => {
    if (!deployStore.enabled || !deployStore.status.configured) {
        return 'Deploy not configured'
    }
    return deployStore.status.outOfSync ? 'Saved changes to deploy' : 'No saved changes to deploy'
})
const deployStatusTone = computed(() => {
    if (!deployStore.enabled || !deployStore.status.configured) {
        return 'neutral'
    }
    return deployStore.status.outOfSync ? 'warning' : 'success'
})
const deployConfigReady = computed(() => {
    const deploy = configStore.config?.deploy || {}
    const sftp = deploy.sftp || {}
    return deploy.enabled === true &&
        Boolean(String(sftp.host || '').trim()) &&
        Boolean(String(sftp.user || '').trim()) &&
        Boolean(String(sftp.remotePath || '').trim())
})
const deployReady = computed(() => deployStore.enabled && deployStore.status.configured)
const deployBusy = computed(() => deployStore.deploying || deployStore.savingSecret)
const deployTargetLabel = computed(() => {
    const sftp = configStore.config?.deploy?.sftp || {}
    const host = String(sftp.host || '').trim()
    const remotePath = String(sftp.remotePath || '').trim()
    if (!host && !remotePath) {
        return 'the configured SFTP target'
    }
    if (!remotePath) {
        return host
    }
    if (!host) {
        return remotePath
    }
    return `${host}:${remotePath}`
})

const workspaceTitle = computed(() => {
    if (uiStore.activeView === 'posts') {
        if (postsStore.postSetupOpen) {
            return 'New post'
        }
        return postsStore.editorOpen ? (postsStore.draft.title || 'Post editor') : 'Posts'
    }
    return activeLabel.value
})

onMounted(async () => {
    window.addEventListener('beforeunload', handleBeforeUnload)
    if (!authStore.hasToken) {
        uiStore.setActiveView('sites')
        return
    }
    await configStore.loadConfig()
    refreshDeployStatus()
})

onBeforeUnmount(() => {
    window.removeEventListener('beforeunload', handleBeforeUnload)
})

watch(
    () => uiStore.activeView,
    (view) => {
        if (view !== 'sites') {
            sidebarOpen.value = window.matchMedia('(min-width: 900px)').matches
        }
    }
)

watch(
    () => [uiStore.activeView, deployConfigReady.value, configStore.activeSiteId],
    () => {
        refreshDeployStatus()
    }
)

function leaveSite() {
    if (!confirmDiscardUnsavedChanges()) {
        return
    }
    buildStore.reset()
    siteWorkspaceStore.reset()
    uiStore.setActiveView('sites')
}

function setSiteView(view) {
    if (view === 'posts') {
        openPostsList()
        return
    }
    if (view === uiStore.activeView) {
        return
    }
    if (!confirmDiscardUnsavedChanges()) {
        return
    }
    uiStore.setActiveView(view)
    sidebarOpen.value = window.matchMedia('(min-width: 900px)').matches
}

function openPostsList() {
    if (!confirmDiscardUnsavedChanges()) {
        return
    }
    postsStore.clearSelection()
    uiStore.setActiveView('posts')
    sidebarOpen.value = window.matchMedia('(min-width: 900px)').matches
}

function toggleSidebar() {
    sidebarOpen.value = !sidebarOpen.value
}

async function saveFromHeader() {
    const action = saveAction.value
    if (!saveAvailable.value) {
        uiStore.setNotice('Open an editable screen to save local changes.')
        return
    }
    await action.run()
}

async function deployFromHeader() {
    if (localChangesToSave.value) {
        uiStore.setNotice('Save local changes before deploying.')
        return
    }
    if (!deployReady.value) {
        uiStore.setNotice('Configure SFTP deploy before deploying.')
        uiStore.setActiveView('config')
        sidebarOpen.value = true
        return
    }
    if (!deployStore.status.secretSet) {
        deploySecret.value = ''
        deploySecretOpen.value = true
        uiStore.setNotice('Enter the SFTP password or key passphrase to deploy.')
        return
    }
    deploySecretOpen.value = false
    try {
        await deployStore.deployNow({ quietSecretError: true })
    } catch (err) {
        if (deployNeedsSecret(err)) {
            deploySecret.value = ''
            deploySecretOpen.value = true
            uiStore.setNotice('Enter the SFTP password or key passphrase to deploy.')
        }
    }
}

async function saveSecretAndDeploy() {
    if (!deploySecret.value) {
        return
    }
    try {
        await deployStore.saveSecret(deploySecret.value)
        deploySecret.value = ''
        deploySecretOpen.value = false
        await deployStore.deployNow()
    } catch {
        // Stores expose the verification or deploy error to the UI.
    }
}

function cancelDeploySecret() {
    deploySecret.value = ''
    deploySecretOpen.value = false
}

function deployNeedsSecret(err) {
    if (err?.code !== 'invalid_deploy_config') {
        return false
    }
    const message = String(err?.message || '').toLowerCase()
    return message.includes('password') ||
        message.includes('passphrase') ||
        message.includes('encrypted private key') ||
        message.includes('ssh-agent')
}

function refreshDeployStatus() {
    if (uiStore.activeView === 'sites' || !deployConfigReady.value) {
        return
    }
    deployStore.refreshStatus({ quiet: true }).catch(() => {})
}

function handleBeforeUnload(event) {
    if (!hasUnsavedChanges.value) {
        return
    }
    event.preventDefault()
    event.returnValue = ''
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
    <transition name="fade-view" mode="out-in">
        <div v-if="uiStore.activeView === 'sites'" class="site-entry-shell">
            <header class="site-entry-header hero-header">
                <div class="brand hero-brand">
                    <img
                        class="mark"
                        :src="styxpressMarkUrl"
                        alt=""
                        width="80"
                        height="80"
                        aria-hidden="true"
                    >
                    <div class="hero-brand-text">
                        <p class="eyebrow">Styxpress</p>
                        <h1 class="gradient-text">Admin</h1>
                    </div>
                </div>
            </header>

            <main class="site-entry-main">
                <div class="status-row hero-status">
                    <UiBadge :tone="authStore.hasToken ? 'success' : 'warning'">
                        {{ authStore.hasToken ? 'session ready' : 'session missing' }}
                    </UiBadge>
                    <UiBadge>{{ configStore.sites.length }} sites</UiBadge>
                </div>
                <SiteListScreen v-if="authStore.hasToken" />
                <AuthBar />
            </main>
        </div>

        <div v-else class="app-shell" :class="{ 'sidebar-open': sidebarOpen }">
            <transition name="slide-sidebar">
                <aside v-if="sidebarOpen" id="admin-sidebar" class="sidebar">
                    <div class="brand">
                        <img
                            class="mark"
                            :src="styxpressMarkUrl"
                            alt=""
                            width="40"
                            height="40"
                            aria-hidden="true"
                        >
                        <div>
                            <p class="eyebrow">Styxpress</p>
                            <h1 class="gradient-text">Admin</h1>
                        </div>
                        <UiButton class="sidebar-close" tone="ghost" aria-label="Close menu" @click="toggleSidebar">
                            <span class="hamburger-icon" aria-hidden="true">
                                <span></span>
                                <span></span>
                                <span></span>
                            </span>
                        </UiButton>
                    </div>

                    <section class="site-context">
                        <p class="eyebrow">Current site</p>
                        <strong>{{ activeSiteName }}</strong>
                        <UiButton tone="ghost" @click="leaveSite">
                            All sites
                        </UiButton>
                    </section>

                    <nav class="nav" aria-label="Admin sections">
                        <button
                            type="button"
                            :class="{ active: uiStore.activeView === 'config' }"
                            @click="setSiteView('config')"
                        >
                            <svg aria-hidden="true" viewBox="0 0 24 24" width="20" height="20" stroke="currentColor" stroke-width="2" fill="none" stroke-linecap="round" stroke-linejoin="round">
                                <circle cx="12" cy="12" r="3"></circle>
                                <path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z"></path>
                            </svg>
                            Configuration
                            <span v-if="uiStore.headerSaveAction?.key === 'config' && localChangesToSave" class="nav-dirty-dot" aria-label="Unsaved changes"></span>
                        </button>
                        <button
                            type="button"
                            :class="{ active: uiStore.activeView === 'site' }"
                            @click="setSiteView('site')"
                        >
                            <svg aria-hidden="true" viewBox="0 0 24 24" width="20" height="20" stroke="currentColor" stroke-width="2" fill="none" stroke-linecap="round" stroke-linejoin="round">
                                <circle cx="12" cy="12" r="10"></circle>
                                <line x1="2" y1="12" x2="22" y2="12"></line>
                                <path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z"></path>
                            </svg>
                            Site
                            <span v-if="siteConfigStore.isDirty" class="nav-dirty-dot" aria-label="Unsaved changes"></span>
                        </button>
                        <button
                            type="button"
                            :class="{ active: uiStore.activeView === 'posts' }"
                            @click="setSiteView('posts')"
                        >
                            <svg aria-hidden="true" viewBox="0 0 24 24" width="20" height="20" stroke="currentColor" stroke-width="2" fill="none" stroke-linecap="round" stroke-linejoin="round">
                                <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"></path>
                                <polyline points="14 2 14 8 20 8"></polyline>
                                <line x1="16" y1="13" x2="8" y2="13"></line>
                                <line x1="16" y1="17" x2="8" y2="17"></line>
                                <polyline points="10 9 9 9 8 9"></polyline>
                            </svg>
                            Posts
                            <span v-if="postsStore.isDirty" class="nav-dirty-dot" aria-label="Unsaved changes"></span>
                        </button>
                    </nav>

                    <AuthBar />
                </aside>
            </transition>

            <main class="workspace">
                <header class="topbar">
                    <UiButton
                        class="menu-button"
                        tone="ghost"
                        :aria-expanded="sidebarOpen"
                        aria-controls="admin-sidebar"
                        @click="toggleSidebar"
                    >
                        <span class="hamburger-icon" aria-hidden="true">
                            <span></span>
                            <span></span>
                            <span></span>
                        </span>
                        <span>{{ sidebarOpen ? 'Close' : 'Menu' }}</span>
                    </UiButton>
                    <div class="topbar-title">
                        <p class="eyebrow">{{ activeSiteName }}</p>
                        <h2>{{ workspaceTitle }}</h2>
                    </div>
                    <div class="topbar-actions">
                        <div class="status-row header-status">
                            <UiBadge :tone="authStore.hasToken ? 'success' : 'warning'">
                                {{ authStore.hasToken ? 'session ready' : 'session missing' }}
                            </UiBadge>
                            <UiBadge :tone="localChangesToSave ? 'warning' : 'success'">
                                {{ localSaveLabel }}
                            </UiBadge>
                            <UiBadge :tone="deployStatusTone">
                                {{ deployStatusLabel }}
                            </UiBadge>
                            <UiBadge>{{ postsStore.posts.length }} posts</UiBadge>
                        </div>
                        <div class="header-command-row">
                            <UiButton
                                tone="primary"
                                :busy="headerSaving"
                                :disabled="!saveAvailable"
                                @click="saveFromHeader"
                            >
                                Save
                            </UiButton>
                            <UiButton
                                tone="primary"
                                :busy="deployBusy"
                                @click="deployFromHeader"
                            >
                                Deploy
                            </UiButton>
                        </div>
                    </div>
                </header>

                <transition name="fade-view" mode="out-in">
                    <section v-if="uiStore.activeView === 'posts'" class="posts-layout" key="posts">
                        <div v-if="postsStore.postSetupOpen" class="posts-editor-only">
                            <PostCreateSetup />
                        </div>
                        <div v-else-if="!postsStore.editorOpen" class="posts-list-only">
                            <PostList />
                        </div>
                        <div v-else class="posts-editor-only">
                            <PostEditor />
                        </div>
                    </section>

                    <section v-else-if="uiStore.activeView === 'site'" class="wide-layout" key="site">
                        <SiteConfigScreen />
                    </section>

                    <section v-else class="single-layout" key="config">
                        <ConfigScreen />
                    </section>
                </transition>
            </main>

            <transition name="fade-modal">
                <div v-if="deploySecretOpen" class="modal-backdrop" @click.self="cancelDeploySecret">
                    <section
                        class="modal-panel"
                        role="dialog"
                        aria-modal="true"
                        aria-labelledby="deploy-secret-title"
                    >
                        <form class="modal-content" @submit.prevent="saveSecretAndDeploy">
                            <div class="modal-heading">
                                <h3 id="deploy-secret-title">Password / passphrase</h3>
                                <p class="muted">Deploy will sync the generated public folder to {{ deployTargetLabel }}.</p>
                                <p class="muted">The password or key passphrase is only kept in this server session.</p>
                            </div>
                            <input
                                type="hidden"
                                name="username"
                                autocomplete="username"
                                :value="configStore.config?.deploy?.sftp?.user || ''"
                            >
                            <UiField
                                v-model="deploySecret"
                                type="password"
                                label="Password / passphrase"
                                autocomplete="current-password"
                            />
                            <div class="modal-actions">
                                <UiButton
                                    tone="primary"
                                    type="submit"
                                    :busy="deployStore.savingSecret || deployStore.deploying"
                                    :disabled="!deploySecret"
                                >
                                    Verify and deploy
                                </UiButton>
                                <UiButton tone="ghost" @click="cancelDeploySecret">
                                    Cancel
                                </UiButton>
                            </div>
                        </form>
                    </section>
                </div>
            </transition>
        </div>
    </transition>

    <transition name="toast-slide">
        <div v-if="uiStore.notice" class="toast-notification" :class="{ 'toast-error': uiStore.error, 'toast-success': !uiStore.error }" role="alert">
            {{ uiStore.notice }}
        </div>
    </transition>
</template>

<style scoped>
/* Typography & Gradients */
.gradient-text {
    background: linear-gradient(135deg, var(--color-heading) 0%, var(--color-accent) 100%);
    -webkit-background-clip: text;
    -webkit-text-fill-color: transparent;
    background-clip: text;
}

/* Nav Item Dirty Dot */
.nav-dirty-dot {
    width: 6px;
    height: 6px;
    border-radius: 50%;
    background: var(--color-warning);
    margin-left: auto;
    box-shadow: 0 0 6px var(--color-warning);
}

/* Hero Shell Layout */
.hero-header {
    background: transparent;
    border: none;
    box-shadow: none;
    justify-content: center;
    padding: 3rem 1rem 1rem;
}

.hero-brand {
    flex-direction: column;
    text-align: center;
    gap: 1.25rem;
}

.hero-brand .mark {
    width: 6rem;
    height: 6rem;
    box-shadow: 0 0 32px var(--color-accent-glow);
}

.hero-brand-text h1 {
    font-size: 2.5rem;
}

.hero-brand-text .eyebrow {
    font-size: 0.85rem;
}

.hero-status {
    justify-content: center;
    margin-bottom: 2rem;
}

/* Transitions */
.fade-view-enter-active,
.fade-view-leave-active {
    transition: opacity 0.25s ease, transform 0.25s ease;
}
.fade-view-enter-from {
    opacity: 0;
    transform: translateY(10px);
}
.fade-view-leave-to {
    opacity: 0;
    transform: translateY(-10px);
}

.slide-sidebar-enter-active,
.slide-sidebar-leave-active {
    transition: transform 0.3s cubic-bezier(0.4, 0, 0.2, 1), opacity 0.3s ease;
}
.slide-sidebar-enter-from,
.slide-sidebar-leave-to {
    transform: translateX(-100%);
    opacity: 0;
}

.fade-modal-enter-active,
.fade-modal-leave-active {
    transition: opacity 0.25s ease;
}
.fade-modal-enter-from,
.fade-modal-leave-to {
    opacity: 0;
}

.toast-slide-enter-active,
.toast-slide-leave-active {
    transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
}
.toast-slide-enter-from,
.toast-slide-leave-to {
    opacity: 0;
    transform: translateX(100%) translateY(1rem);
}
</style>
