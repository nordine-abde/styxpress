<script setup>
import { computed, ref } from 'vue'
import ConfirmPrompt from './ui/ConfirmPrompt.vue'
import FileField from './ui/FileField.vue'
import PostAssetImage from './PostAssetImage.vue'
import PostCoverImage from './PostCoverImage.vue'
import UiButton from './ui/UiButton.vue'
import UiField from './ui/UiField.vue'
import UiPanel from './ui/UiPanel.vue'
import UiSelect from './ui/UiSelect.vue'
import { useBuildStore } from '../stores/build'
import { usePostsStore } from '../stores/posts'

const emit = defineEmits(['insert-image'])

const coverAccept = '.jpg,.jpeg,.png,.webp,.avif,image/jpeg,image/png,image/webp,image/avif'
const imageAccept = `${coverAccept},.gif,image/gif`
const imageSizeOptions = [
    { value: 'medium', label: 'Medium' },
    { value: 'small', label: 'Small' },
    { value: 'large', label: 'Large' },
    { value: 'full', label: 'Full width' }
]

const buildStore = useBuildStore()
const postsStore = usePostsStore()
const assetPath = ref('')
const imageSize = ref('medium')
const uploadError = ref('')

const canUpload = computed(() => postsStore.canUploadMedia)
const busy = computed(() => postsStore.saving || postsStore.uploading || buildStore.publishing || buildStore.rendering)
const uploadDisabled = computed(() => !canUpload.value || busy.value)
const uploadHelp = computed(() => {
    if (!canUpload.value) {
        return 'Enter a slug before uploading images.'
    }
    if (postsStore.mediaUploadNeedsSave) {
        return 'Uploads save the current draft first.'
    }
    return ''
})
const imageAssets = computed(() => {
    return (postsStore.draft.assets || []).filter((asset) => isImagePath(asset))
})

async function uploadAsset(file) {
    uploadError.value = ''
    if (!file) {
        return
    }
    if (!isImagePath(file.name) && !file.type.startsWith('image/')) {
        uploadError.value = 'Only image files can be uploaded.'
        return
    }
    await postsStore.uploadAsset(file, assetPath.value)
    assetPath.value = ''
    await renderPublishedPostOutput()
}

async function uploadCover(file) {
    uploadError.value = ''
    if (!file) {
        return
    }
    if (!isCoverImagePath(file.name)) {
        uploadError.value = 'Cover images must be jpg, png, webp, or avif.'
        return
    }
    await postsStore.uploadCover(file)
    await renderPublishedPostOutput()
}

async function deleteCover() {
    await postsStore.deleteCover()
    await renderPublishedPostOutput()
}

async function deleteAsset(asset) {
    await postsStore.deleteAsset(asset)
    await renderPublishedPostOutput()
}

function insertImage(asset) {
    emit('insert-image', {
        asset,
        size: imageSize.value,
        alt: assetLabel(asset)
    })
}

async function renderPublishedPostOutput() {
    if (!postsStore.selectedSlug || postsStore.draft.publishStatus !== 'published') {
        return
    }
    await buildStore.renderPost(postsStore.selectedSlug)
}

function isImagePath(path) {
    return /\.(jpe?g|png|webp|avif|gif)$/i.test(path || '')
}

function isCoverImagePath(path) {
    return /\.(jpe?g|png|webp|avif)$/i.test(path || '')
}

function assetLabel(asset) {
    const filename = asset.split('/').pop() || asset
    return filename.replace(/\.[^.]+$/, '').replaceAll('-', ' ').replaceAll('_', ' ')
}
</script>

<template>
    <aside class="asset-panel">
        <UiPanel title="Images">
            <div class="field-grid">
                <section class="asset-section">
                    <div class="section-title">
                        <span>Cover</span>
                        <strong>{{ postsStore.draft.cover || 'none' }}</strong>
                    </div>
                    <PostCoverImage
                        v-if="postsStore.selectedSlug && postsStore.draft.cover"
                        :slug="postsStore.selectedSlug"
                        :cover="postsStore.draft.cover"
                        :alt="`${postsStore.draft.title} cover`"
                    />
                    <FileField
                        label="Upload cover"
                        :accept="coverAccept"
                        :disabled="uploadDisabled"
                        @selected="uploadCover"
                    />
                    <ConfirmPrompt
                        v-if="postsStore.draft.cover"
                        label="Remove cover"
                        confirm-label="Remove"
                        :disabled="uploadDisabled"
                        @confirm="deleteCover"
                    />
                </section>

                <section class="asset-section">
                    <UiSelect v-model="imageSize" label="Insert size" :options="imageSizeOptions" />
                    <div class="asset-upload">
                        <UiField v-model="assetPath" label="Image path" placeholder="images/photo.png" />
                        <FileField
                            label="Upload image"
                            :accept="imageAccept"
                            :disabled="uploadDisabled"
                            @selected="uploadAsset"
                        />
                    </div>
                    <p v-if="uploadHelp" class="muted compact-text">
                        {{ uploadHelp }}
                    </p>
                    <p v-if="uploadError" class="error-text compact-text">
                        {{ uploadError }}
                    </p>
                </section>

                <ul v-if="imageAssets.length > 0" class="asset-list" aria-label="Post images">
                    <li v-for="asset in imageAssets" :key="asset" class="asset-card">
                        <PostAssetImage
                            v-if="postsStore.selectedSlug"
                            :slug="postsStore.selectedSlug"
                            :asset="asset"
                            :alt="assetLabel(asset)"
                        />
                        <code>{{ asset }}</code>
                        <div class="asset-actions">
                            <UiButton tone="primary" :busy="busy" @click="insertImage(asset)">
                                Insert
                            </UiButton>
                            <ConfirmPrompt
                                v-if="canUpload"
                                label="Remove"
                                confirm-label="Remove"
                                :disabled="uploadDisabled"
                                @confirm="deleteAsset(asset)"
                            />
                        </div>
                    </li>
                </ul>

                <p v-else class="muted compact-text">
                    No images yet.
                </p>
            </div>
        </UiPanel>
    </aside>
</template>

<style scoped>
.asset-panel {
    min-width: 0;
}

.asset-section,
.asset-upload {
    display: grid;
    gap: 0.75rem;
}

.section-title {
    display: grid;
    gap: 0.2rem;
}

.section-title span {
    color: var(--color-muted);
    font-size: 0.78rem;
    font-weight: 800;
    text-transform: uppercase;
}

.section-title strong {
    color: var(--color-heading);
    overflow-wrap: anywhere;
}

.asset-list {
    display: grid;
    gap: 0.75rem;
    margin: 0;
    padding: 0;
    list-style: none;
}

.asset-card {
    display: grid;
    gap: 0.65rem;
    min-width: 0;
    border: 1px solid var(--color-border);
    border-radius: 8px;
    padding: 0.65rem;
    background: var(--color-surface);
}

.asset-actions {
    display: flex;
    flex-wrap: wrap;
    gap: 0.5rem;
    align-items: center;
}

.compact-text {
    margin: 0;
}

code {
    color: var(--color-heading);
    overflow-wrap: anywhere;
}
</style>
