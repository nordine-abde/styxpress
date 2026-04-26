<template>
  <main class="shell">
    <section class="hero">
      <p class="eyebrow">Styxpress Admin</p>
      <h1>Local publishing control for a static Markdown blog.</h1>
      <p class="summary">
        This interface will render posts locally, manage content metadata, and publish generated files over SSH.
      </p>
    </section>

    <section class="panel" aria-labelledby="next-steps">
      <h2 id="next-steps">Initial build targets</h2>
      <ul>
        <li>Configure a local publishing profile.</li>
        <li>Create or import Markdown posts.</li>
        <li>Preview rendered HTML before upload.</li>
        <li>Publish content and public files over SSH.</li>
      </ul>
    </section>
  </main>
</template>

<style scoped>
.shell {
  display: grid;
  gap: 2rem;
  width: min(100%, 1080px);
  margin: 0 auto;
  padding: 5rem 1.5rem;
}

.hero {
  max-width: 760px;
}

.eyebrow {
  margin-bottom: 1rem;
  color: var(--color-accent);
  font-size: 0.78rem;
  font-weight: 700;
  letter-spacing: 0.16em;
  text-transform: uppercase;
}

h1 {
  margin: 0;
  color: var(--color-heading);
  font-size: clamp(2.5rem, 8vw, 6.5rem);
  line-height: 0.9;
  letter-spacing: -0.07em;
}

.summary {
  max-width: 620px;
  margin-top: 1.5rem;
  color: var(--color-muted);
  font-size: 1.15rem;
}

.panel {
  border: 1px solid var(--color-border);
  border-radius: 28px;
  padding: 1.5rem;
  background: color-mix(in srgb, var(--color-surface) 86%, transparent);
  box-shadow: 0 24px 80px rgb(31 24 15 / 12%);
}

h2 {
  margin: 0 0 1rem;
  color: var(--color-heading);
  font-size: 1rem;
  font-weight: 800;
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

ul {
  display: grid;
  gap: 0.75rem;
  margin: 0;
  padding: 0;
  list-style: none;
}

li {
  padding: 0.95rem 1rem;
  border-radius: 18px;
  background: var(--color-background);
  color: var(--color-text);
}

@media (min-width: 900px) {
  .shell {
    grid-template-columns: minmax(0, 1fr) 360px;
    align-items: end;
  }
}
</style>
