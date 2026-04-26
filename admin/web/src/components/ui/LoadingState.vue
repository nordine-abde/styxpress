<template>
    <div class="loading-state">
        <span aria-hidden="true"></span>
        <slot>Loading</slot>
    </div>
</template>

<style scoped>
.loading-state {
    display: inline-flex;
    align-items: center;
    gap: 0.5rem;
    color: var(--color-muted);
    font-size: 0.9rem;
}

span {
    width: 0.8rem;
    height: 0.8rem;
    border: 2px solid var(--color-muted);
    border-right-color: transparent;
    border-radius: 999px;
    animation: spin 0.7s linear infinite;
}

@keyframes spin {
    to {
        transform: rotate(360deg);
    }
}
</style>
