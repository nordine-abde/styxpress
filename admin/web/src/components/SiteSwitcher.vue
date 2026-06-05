<script setup>
import { computed, ref } from 'vue'
import ConfirmPrompt from './ui/ConfirmPrompt.vue'
import EmptyState from './ui/EmptyState.vue'
import UiButton from './ui/UiButton.vue'
import UiField from './ui/UiField.vue'
import UiPanel from './ui/UiPanel.vue'
import { useConfigStore } from '../stores/config'
import { usePostsStore } from '../stores/posts'
import { useSiteConfigStore } from '../stores/siteConfig'

const configStore = useConfigStore()
const postsStore = usePostsStore()
const siteConfigStore = useSiteConfigStore()
const newSiteName = ref('')

const canDelete = computed(() => configStore.multiSite && configStore.sites.length > 0)

async function createSite() {
    const name = newSiteName.value.trim()
    if (!name) {
        return
    }
    await configStore.createSite(name)
    newSiteName.value = ''
    await reloadSiteWorkspace()
}

async function selectSite(site) {
    if (site.id === configStore.activeSiteId) {
        return
    }
    await configStore.selectSite(site.id)
    await reloadSiteWorkspace()
}

async function deleteSite(site) {
    await configStore.deleteSite(site.id)
    if (!configStore.activeSiteId) {
        postsStore.reset()
        siteConfigStore.reset()
        return
    }
    await reloadSiteWorkspace()
}

async function reloadSiteWorkspace() {
    postsStore.newPost()
    await Promise.all([
        siteConfigStore.loadSiteConfig(),
        postsStore.loadPosts(),
        postsStore.loadFeatured()
    ])
}
</script>

<template>
    <UiPanel title="My sites" subtitle="Saved admin configurations in the user config directory.">
        <form v-if="configStore.multiSite" class="new-site" @submit.prevent="createSite">
            <UiField v-model="newSiteName" label="New site name" placeholder="Client blog" />
            <UiButton
                tone="primary"
                type="submit"
                :busy="configStore.switching"
                :disabled="!newSiteName.trim()"
            >
                Create site
            </UiButton>
        </form>

        <p v-else class="muted">
            This admin session is using the explicit config file passed with -config.
        </p>

        <EmptyState
            v-if="configStore.sites.length === 0"
            title="No sites yet"
            message="Create a site to start."
        />

        <ul v-else class="site-list">
            <li v-for="site in configStore.sites" :key="site.id" class="site-item">
                <div>
                    <div class="site-title-row">
                        <strong>{{ site.name }}</strong>
                        <span v-if="site.id === configStore.activeSiteId">active</span>
                    </div>
                    <p>{{ site.config.contentDir }}</p>
                </div>
                <div class="site-actions">
                    <UiButton
                        tone="ghost"
                        :disabled="site.id === configStore.activeSiteId"
                        :busy="configStore.switching && site.id !== configStore.activeSiteId"
                        @click="selectSite(site)"
                    >
                        Select
                    </UiButton>
                    <ConfirmPrompt
                        v-if="configStore.multiSite && canDelete"
                        label="Delete"
                        confirm-label="Delete site"
                        @confirm="deleteSite(site)"
                    />
                </div>
            </li>
        </ul>
    </UiPanel>
</template>

<style scoped>
.new-site {
    display: grid;
    gap: 0.65rem;
}

.site-list {
    display: grid;
    gap: 0.6rem;
    margin: 0;
    padding: 0;
    list-style: none;
}

.site-item {
    display: grid;
    gap: 0.75rem;
    align-items: center;
    border: 1px solid var(--color-border);
    border-radius: 8px;
    padding: 0.85rem;
    background: var(--color-surface);
}

.site-title-row {
    display: flex;
    flex-wrap: wrap;
    gap: 0.5rem;
    align-items: center;
}

strong {
    color: var(--color-heading);
}

span {
    border-radius: 999px;
    padding: 0.15rem 0.45rem;
    background: color-mix(in srgb, var(--color-success) 12%, var(--color-surface));
    color: var(--color-success);
    font-size: 0.72rem;
    font-weight: 800;
    text-transform: uppercase;
}

p {
    margin: 0.25rem 0 0;
    color: var(--color-muted);
    font-size: 0.84rem;
    overflow-wrap: anywhere;
}

.site-actions {
    display: flex;
    flex-wrap: wrap;
    gap: 0.5rem;
}

@media (min-width: 720px) {
    .new-site {
        grid-template-columns: minmax(0, 1fr) auto;
        align-items: end;
    }

    .site-item {
        grid-template-columns: minmax(0, 1fr) auto;
    }
}
</style>
