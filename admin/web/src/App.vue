<script setup>
import { computed, onMounted } from 'vue'
import AuthBar from './components/AuthBar.vue'
import ConfigScreen from './components/ConfigScreen.vue'
import FeaturedManager from './components/FeaturedManager.vue'
import PostEditor from './components/PostEditor.vue'
import PostList from './components/PostList.vue'
import PublishPanel from './components/PublishPanel.vue'
import UiBadge from './components/ui/UiBadge.vue'
import { useAuthStore } from './stores/auth'
import { useConfigStore } from './stores/config'
import { usePostsStore } from './stores/posts'
import { useUiStore } from './stores/ui'

const authStore = useAuthStore()
const configStore = useConfigStore()
const postsStore = usePostsStore()
const uiStore = useUiStore()

const activeLabel = computed(() => {
    if (uiStore.activeView === 'config') {
        return 'Configuration'
    }
    if (uiStore.activeView === 'featured') {
        return 'Featured'
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
        postsStore.loadPosts(),
        postsStore.loadFeatured()
    ])
})
</script>

<template>
    <div class="app-shell">
        <aside class="sidebar">
            <div class="brand">
                <span class="mark" aria-hidden="true">S</span>
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
                        {{ authStore.hasToken ? 'token set' : 'token needed' }}
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

            <section v-else class="single-layout">
                <ConfigScreen />
            </section>
        </main>
    </div>
</template>
