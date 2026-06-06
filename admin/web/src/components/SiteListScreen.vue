<script setup>
import { computed, onMounted, ref } from 'vue'
import ConfirmPrompt from './ui/ConfirmPrompt.vue'
import EmptyState from './ui/EmptyState.vue'
import UiBadge from './ui/UiBadge.vue'
import UiButton from './ui/UiButton.vue'
import UiField from './ui/UiField.vue'
import UiPanel from './ui/UiPanel.vue'
import { useAuthStore } from '../stores/auth'
import { useConfigStore } from '../stores/config'
import { useSiteWorkspaceStore } from '../stores/siteWorkspace'
import { useUiStore } from '../stores/ui'

const authStore = useAuthStore()
const configStore = useConfigStore()
const siteWorkspaceStore = useSiteWorkspaceStore()
const uiStore = useUiStore()
const newSiteName = ref('')

const canCreateSites = computed(() => configStore.multiSite)
const canDelete = computed(() => configStore.multiSite && configStore.sites.length > 0)

onMounted(async () => {
    if (authStore.hasToken && configStore.sites.length === 0) {
        await configStore.loadConfig()
    }
})

async function createSite() {
    const name = newSiteName.value.trim()
    if (!name) {
        return
    }
    await configStore.createSite(name)
    newSiteName.value = ''
    await openActiveSite()
}

async function openSite(site) {
    if (site.id !== configStore.activeSiteId) {
        await configStore.selectSite(site.id)
    }
    await openActiveSite()
}

async function openActiveSite() {
    siteWorkspaceStore.reset()
    uiStore.setActiveView('config')
    await siteWorkspaceStore.loadCurrentSite({ force: true })
}

async function deleteSite(site) {
    await configStore.deleteSite(site.id)
}
</script>

<template>
    <UiPanel title="My sites" subtitle="Choose one site before editing local content and output settings.">
        <form v-if="canCreateSites" class="site-create-row" @submit.prevent="createSite">
            <UiField v-model="newSiteName" label="New site" placeholder="Client blog" />
            <UiButton
                tone="primary"
                type="submit"
                :busy="configStore.switching"
                :disabled="!newSiteName.trim()"
            >
                Create and configure
            </UiButton>
        </form>

        <p v-else class="muted">
            This admin session is using the explicit config file passed with -config.
        </p>

        <p v-if="configStore.loading" class="muted">
            Loading sites...
        </p>

        <EmptyState
            v-if="!configStore.loading && configStore.sites.length === 0"
            title="No sites yet"
            message="Create a site to start."
        />

        <ul v-if="!configStore.loading && configStore.sites.length > 0" class="site-library" aria-label="Configured sites">
            <li v-for="site in configStore.sites" :key="site.id" class="site-card">
                <div class="site-card-main">
                    <div class="site-title-row">
                        <strong>{{ site.name }}</strong>
                        <UiBadge v-if="site.id === configStore.activeSiteId" tone="success">
                            selected
                        </UiBadge>
                    </div>
                    <p>{{ site.config.contentDir }}</p>
                </div>
                <div class="site-card-meta">
                    <span>local content</span>
                    <span>{{ site.config.publicDir || 'public' }}</span>
                </div>
                <div class="site-card-actions">
                    <UiButton
                        tone="primary"
                        :busy="(configStore.switching && site.id !== configStore.activeSiteId) || siteWorkspaceStore.loading"
                        @click="openSite(site)"
                    >
                        Open site
                    </UiButton>
                    <ConfirmPrompt
                        v-if="canDelete"
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
.site-create-row,
.site-library,
.site-card,
.site-card-main,
.site-card-meta {
    display: grid;
    gap: 0.75rem;
}

.site-library {
    margin: 0;
    padding: 0;
    list-style: none;
}

.site-card {
    align-items: center;
    border: 1px solid var(--color-border);
    border-radius: 8px;
    padding: 0.9rem;
    background: var(--color-surface);
}

.site-title-row,
.site-card-actions {
    display: flex;
    flex-wrap: wrap;
    gap: 0.55rem;
    align-items: center;
}

.site-title-row strong {
    color: var(--color-heading);
    font-size: 1rem;
}

.site-card-main p,
.muted {
    margin: 0;
    color: var(--color-muted);
    overflow-wrap: anywhere;
}

.site-card-main p {
    font-size: 0.9rem;
}

.site-card-meta {
    gap: 0.35rem;
    color: var(--color-muted);
    font-size: 0.78rem;
    font-weight: 800;
    text-transform: uppercase;
}

@media (min-width: 720px) {
    .site-create-row {
        grid-template-columns: minmax(0, 1fr) auto;
        align-items: end;
    }

    .site-card {
        grid-template-columns: minmax(0, 1fr) minmax(8rem, auto) auto;
    }
}
</style>
