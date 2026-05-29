const navToggle = document.querySelector('[data-nav-toggle]')
const navLinks = document.querySelector('[data-nav-links]')

if (navToggle && navLinks) {
    navToggle.addEventListener('click', () => {
        const open = navLinks.classList.toggle('open')
        navToggle.setAttribute('aria-expanded', String(open))
    })

    navLinks.addEventListener('click', (event) => {
        if (event.target instanceof HTMLAnchorElement) {
            navLinks.classList.remove('open')
            navToggle.setAttribute('aria-expanded', 'false')
        }
    })
}

document.querySelectorAll('pre').forEach((pre) => {
    const wrapper = document.createElement('div')
    wrapper.className = 'code-block'
    pre.parentNode.insertBefore(wrapper, pre)
    wrapper.appendChild(pre)

    const button = document.createElement('button')
    button.type = 'button'
    button.className = 'copy-button'
    button.textContent = 'Copy'
    wrapper.appendChild(button)

    button.addEventListener('click', async () => {
        const text = pre.innerText
        try {
            await navigator.clipboard.writeText(text)
            button.textContent = 'Copied'
            window.setTimeout(() => {
                button.textContent = 'Copy'
            }, 1600)
        } catch {
            button.textContent = 'Select'
        }
    })
})

const tocLinks = [...document.querySelectorAll('.toc a[href^="#"]')]
const sections = tocLinks
    .map((link) => document.querySelector(link.getAttribute('href')))
    .filter(Boolean)

if (tocLinks.length && sections.length && 'IntersectionObserver' in window) {
    const observer = new IntersectionObserver((entries) => {
        const visible = entries
            .filter((entry) => entry.isIntersecting)
            .sort((a, b) => b.intersectionRatio - a.intersectionRatio)[0]

        if (!visible) {
            return
        }

        tocLinks.forEach((link) => {
            link.classList.toggle('active', link.getAttribute('href') === `#${visible.target.id}`)
        })
    }, {
        rootMargin: '-20% 0px -65% 0px',
        threshold: [0.08, 0.2, 0.4]
    })

    sections.forEach((section) => observer.observe(section))
}
