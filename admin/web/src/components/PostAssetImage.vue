<script setup>
import { onBeforeUnmount, ref, watch } from 'vue'
import { apiBlobRequest } from '../api/client'

const props = defineProps({
    slug: {
        type: String,
        required: true
    },
    asset: {
        type: String,
        required: true
    },
    alt: {
        type: String,
        default: ''
    }
})

const assetUrl = ref('')
const failed = ref(false)
let requestId = 0

watch(
    () => [props.slug, props.asset],
    () => {
        loadAsset()
    },
    { immediate: true }
)

async function loadAsset() {
    const currentRequest = requestId + 1
    requestId = currentRequest
    failed.value = false
    setAssetUrl('')
    if (!props.slug || !props.asset) {
        return
    }

    try {
        const blob = await apiBlobRequest(`/api/posts/${encodeURIComponent(props.slug)}/assets/${encodeAssetPath(props.asset)}`)
        if (currentRequest !== requestId) {
            return
        }
        setAssetUrl(URL.createObjectURL(blob))
    } catch {
        if (currentRequest === requestId) {
            failed.value = true
        }
    }
}

function encodeAssetPath(path) {
    return path.split('/').map((part) => encodeURIComponent(part)).join('/')
}

function setAssetUrl(nextUrl) {
    if (assetUrl.value) {
        URL.revokeObjectURL(assetUrl.value)
    }
    assetUrl.value = nextUrl
}

onBeforeUnmount(() => {
    requestId += 1
    setAssetUrl('')
})
</script>

<template>
    <div class="asset-preview" :class="{ failed }">
        <img v-if="assetUrl" :src="assetUrl" :alt="alt">
    </div>
</template>

<style scoped>
.asset-preview {
    width: 100%;
    aspect-ratio: 4 / 3;
    overflow: hidden;
    border: 1px solid var(--color-border);
    border-radius: 8px;
    background: var(--color-surface-muted);
}

.asset-preview.failed {
    border-style: dashed;
}

img {
    display: block;
    width: 100%;
    height: 100%;
    object-fit: contain;
}
</style>
