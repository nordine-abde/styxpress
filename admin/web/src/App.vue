<script setup>
import { computed, onMounted } from 'vue'
import AuthBar from './components/AuthBar.vue'
import ConfigScreen from './components/ConfigScreen.vue'
import FeaturedManager from './components/FeaturedManager.vue'
import PostEditor from './components/PostEditor.vue'
import PostList from './components/PostList.vue'
import PublishPanel from './components/PublishPanel.vue'
import SSHConnectionGate from './components/SSHConnectionGate.vue'
import SiteConfigScreen from './components/SiteConfigScreen.vue'
import SiteListScreen from './components/SiteListScreen.vue'
import UiButton from './components/ui/UiButton.vue'
import UiBadge from './components/ui/UiBadge.vue'
import styxpressMarkUrl from './assets/styxpress-mark.png'
import { useAuthStore } from './stores/auth'
import { useConfigStore } from './stores/config'
import { usePostsStore } from './stores/posts'
import { usePublishingStore } from './stores/publishing'
import { useSiteWorkspaceStore } from './stores/siteWorkspace'
import { useUiStore } from './stores/ui'

const authStore = useAuthStore()
const configStore = useConfigStore()
const postsStore = usePostsStore()
const publishingStore = usePublishingStore()
const siteWorkspaceStore = useSiteWorkspaceStore()
const uiStore = useUiStore()

const activeLabel = computed(() => {
    if (uiStore.activeView === 'sites') {
        return 'Sites'
    }
    if (uiStore.activeView === 'config') {
        return 'Configuration'
    }
    if (uiStore.activeView === 'access') {
        return 'Site access'
    }
    if (uiStore.activeView === 'featured') {
        return 'Featured'
    }
    if (uiStore.activeView === 'site') {
        return 'Styles'
    }
    return 'Posts'
})

const activeSiteName = computed(() => {
    return configStore.activeSite?.name || configStore.config.name || 'Configured site'
})

onMounted(async () => {
    if (!authStore.hasToken) {
        uiStore.setActiveView('sites')
        return
    }
    await configStore.loadConfig()
})

function leaveSite() {
    publishingStore.resetSSH()
    siteWorkspaceStore.reset()
    uiStore.setActiveView('sites')
}

function setSiteView(view) {
    if (publishingStore.sshBlocksEditing && view !== 'config') {
        uiStore.setActiveView('access')
        return
    }
    uiStore.setActiveView(view)
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

    <div v-else class="app-shell">
        <aside class="sidebar">
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
                    :disabled="publishingStore.sshBlocksEditing"
                    @click="setSiteView('site')"
                >
                    <span aria-hidden="true">S</span>
                    Styles
                </button>
                <button
                    type="button"
                    :class="{ active: uiStore.activeView === 'posts' }"
                    :disabled="publishingStore.sshBlocksEditing"
                    @click="setSiteView('posts')"
                >
                    <span aria-hidden="true">#</span>
                    Posts
                </button>
                <button
                    type="button"
                    :class="{ active: uiStore.activeView === 'featured' }"
                    :disabled="publishingStore.sshBlocksEditing"
                    @click="setSiteView('featured')"
                >
                    <span aria-hidden="true">*</span>
                    Featured
                </button>
            </nav>

            <AuthBar />
        </aside>

        <main class="workspace">
            <header class="topbar">
                <div>
                    <p class="eyebrow">{{ activeSiteName }}</p>
                    <h2>{{ activeLabel === 'Posts' ? 'Content workspace' : activeLabel }}</h2>
                </div>
                <div class="status-row">
                    <UiBadge v-if="publishingStore.sshEnabled" :tone="publishingStore.sshStatusTone">
                        {{ publishingStore.sshStatusLabel }}
                    </UiBadge>
                    <UiBadge :tone="authStore.hasToken ? 'success' : 'warning'">
                        {{ authStore.hasToken ? 'session ready' : 'session missing' }}
                    </UiBadge>
                    <UiBadge>{{ postsStore.posts.length }} posts</UiBadge>
                </div>
            </header>

            <SSHConnectionGate v-if="uiStore.activeView === 'access'" />

            <section v-else-if="uiStore.activeView === 'posts'" class="posts-layout">
                <div class="posts-side-column">
                    <div class="posts-list-column">
                        <PostList />
                    </div>
                    <div class="posts-action-column">
                        <PublishPanel />
                    </div>
                </div>
                <div class="posts-editor-column">
                    <PostEditor />
                </div>
            </section>

            <section v-else-if="uiStore.activeView === 'featured'" class="single-layout">
                <FeaturedManager />
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
