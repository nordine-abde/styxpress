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
import styxpressMarkUrl from './assets/styxpress-mark.png'
import { useAuthStore } from './stores/auth'
import { useBuildStore } from './stores/build'
import { useConfigStore } from './stores/config'
import { usePostsStore } from './stores/posts'
import { useSiteConfigStore } from './stores/siteConfig'
import { useSiteWorkspaceStore } from './stores/siteWorkspace'
import { useUiStore } from './stores/ui'

const authStore = useAuthStore()
const buildStore = useBuildStore()
const configStore = useConfigStore()
const postsStore = usePostsStore()
const siteConfigStore = useSiteConfigStore()
const siteWorkspaceStore = useSiteWorkspaceStore()
const uiStore = useUiStore()
const sidebarOpen = ref(true)

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

const hasUnsavedChanges = computed(() => siteConfigStore.isDirty || postsStore.isDirty)

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
                <div>
                    <p class="eyebrow">{{ activeSiteName }}</p>
                    <h2>{{ workspaceTitle }}</h2>
                </div>
                <div class="status-row">
                    <UiBadge :tone="authStore.hasToken ? 'success' : 'warning'">
                        {{ authStore.hasToken ? 'session ready' : 'session missing' }}
                    </UiBadge>
                    <UiBadge>{{ postsStore.posts.length }} posts</UiBadge>
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
    </div>
</template>
