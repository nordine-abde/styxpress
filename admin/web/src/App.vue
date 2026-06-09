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
    return configStore.activeSite?.name || configStore.config.name || 'Configured site'
})

const saveAction = computed(() => uiStore.headerSaveAction)
const saveAvailable = computed(() => {
    const action = saveAction.value
    return Boolean(action?.run) && action?.isAvailable?.() !== false
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
        if (view === 'posts') {
            sidebarOpen.value = false
            return
        }
        if (view !== 'sites') {
            sidebarOpen.value = true
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
    sidebarOpen.value = true
}

function openPostsList() {
    if (!confirmDiscardUnsavedChanges()) {
        return
    }
    postsStore.clearSelection()
    uiStore.setActiveView('posts')
    sidebarOpen.value = false
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
    <div v-if="uiStore.activeView === 'sites'" class="site-entry-shell">
        <header class="site-entry-header">
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
                    <h1>Admin</h1>
                </div>
            </div>
            <div class="status-row">
                <UiBadge :tone="authStore.hasToken ? 'success' : 'warning'">
                    {{ authStore.hasToken ? 'session ready' : 'session missing' }}
                </UiBadge>
                <UiBadge>{{ configStore.sites.length }} sites</UiBadge>
            </div>
        </header>

        <main class="site-entry-main">
            <SiteListScreen v-if="authStore.hasToken" />
            <AuthBar />
        </main>
    </div>

    <div v-else class="app-shell" :class="{ 'sidebar-open': sidebarOpen }">
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
                    <h1>Admin</h1>
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
                    <span aria-hidden="true">~</span>
                    Configuration
                </button>
                <button
                    type="button"
                    :class="{ active: uiStore.activeView === 'site' }"
                    @click="setSiteView('site')"
                >
                    <span aria-hidden="true">S</span>
                    Site
                </button>
                <button
                    type="button"
                    :class="{ active: uiStore.activeView === 'posts' }"
                    @click="setSiteView('posts')"
                >
                    <span aria-hidden="true">#</span>
                    Posts
                </button>
            </nav>

            <AuthBar />
        </aside>

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

            <section v-if="uiStore.activeView === 'posts'" class="posts-layout">
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

            <section v-else-if="uiStore.activeView === 'site'" class="wide-layout">
                <SiteConfigScreen />
            </section>

            <section v-else class="single-layout">
                <ConfigScreen />
            </section>
        </main>

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
                        <p class="muted">Only kept in this server session.</p>
                    </div>
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
    </div>
</template>
