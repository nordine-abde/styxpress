<script setup>
import { computed, onMounted } from 'vue'
import AuthBar from './components/AuthBar.vue'
import ConfigScreen from './components/ConfigScreen.vue'
import FeaturedManager from './components/FeaturedManager.vue'
import PostEditor from './components/PostEditor.vue'
import PostList from './components/PostList.vue'
import PublishPanel from './components/PublishPanel.vue'
import SiteConfigScreen from './components/SiteConfigScreen.vue'
import UiBadge from './components/ui/UiBadge.vue'
import styxpressMarkUrl from './assets/styxpress-mark.png'
import { useAuthStore } from './stores/auth'
import { useConfigStore } from './stores/config'
import { usePostsStore } from './stores/posts'
import { useSiteConfigStore } from './stores/siteConfig'
import { useUiStore } from './stores/ui'

const authStore = useAuthStore()
const configStore = useConfigStore()
const postsStore = usePostsStore()
const siteConfigStore = useSiteConfigStore()
const uiStore = useUiStore()

const activeLabel = computed(() => {
    if (uiStore.activeView === 'config') {
        return 'Configuration'
    }
    if (uiStore.activeView === 'featured') {
        return 'Featured'
    }
    if (uiStore.activeView === 'site') {
        return 'Site'
    }
    return 'Posts'
})

onMounted(async () => {
    if (!authStore.hasToken) {
        uiStore.setActiveView('config')
        return
    }
    await Promise.all([
        configStore.loadConfig(),
        siteConfigStore.loadSiteConfig(),
        postsStore.loadPosts(),
        postsStore.loadFeatured()
    ])
})
</script>

<template>
    <div class="app-shell">
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

            <nav class="nav" aria-label="Admin sections">
                <button
                    type="button"
                    :class="{ active: uiStore.activeView === 'posts' }"
                    @click="uiStore.setActiveView('posts')"
                >
                    <span aria-hidden="true">#</span>
                    Posts
                </button>
                <button
                    type="button"
                    :class="{ active: uiStore.activeView === 'featured' }"
                    @click="uiStore.setActiveView('featured')"
                >
                    <span aria-hidden="true">*</span>
                    Featured
                </button>
                <button
                    type="button"
                    :class="{ active: uiStore.activeView === 'site' }"
                    @click="uiStore.setActiveView('site')"
                >
                    <span aria-hidden="true">S</span>
                    Site
                </button>
                <button
                    type="button"
                    :class="{ active: uiStore.activeView === 'config' }"
                    @click="uiStore.setActiveView('config')"
                >
                    <span aria-hidden="true">~</span>
                    Config
                </button>
            </nav>

            <AuthBar />
        </aside>

        <main class="workspace">
            <header class="topbar">
                <div>
                    <p class="eyebrow">{{ activeLabel }}</p>
                    <h2>{{ activeLabel === 'Posts' ? 'Content workspace' : activeLabel }}</h2>
                </div>
                <div class="status-row">
                    <UiBadge :tone="authStore.hasToken ? 'success' : 'warning'">
                        {{ authStore.hasToken ? 'session ready' : 'session missing' }}
                    </UiBadge>
                    <UiBadge>{{ postsStore.posts.length }} posts</UiBadge>
                </div>
            </header>

            <section v-if="uiStore.activeView === 'posts'" class="posts-layout">
                <PostList />
                <PostEditor />
                <PublishPanel />
            </section>

            <section v-else-if="uiStore.activeView === 'featured'" class="single-layout">
                <FeaturedManager />
            </section>

            <section v-else-if="uiStore.activeView === 'site'" class="single-layout">
                <SiteConfigScreen />
            </section>

            <section v-else class="single-layout">
                <ConfigScreen />
            </section>
        </main>
    </div>
</template>
