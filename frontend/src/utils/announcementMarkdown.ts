import { Marked } from 'marked'
import DOMPurify from 'dompurify'

const markdown = new Marked({ gfm: true, breaks: true, renderer: { html: () => '' } })

export function renderAnnouncementMarkdown(content: string): string {
  return DOMPurify.sanitize(markdown.parse(content, { async: false }), {
    ALLOWED_TAGS: ['p', 'br', 'strong', 'em', 'del', 's', 'a', 'h1', 'h2', 'h3', 'h4', 'h5', 'h6', 'ul', 'ol', 'li', 'blockquote', 'pre', 'code', 'hr', 'table', 'thead', 'tbody', 'tr', 'th', 'td', 'img', 'input'],
    ALLOWED_ATTR: ['href', 'src', 'alt', 'title', 'type', 'checked', 'disabled', 'start'],
    ALLOW_DATA_ATTR: false,
    ALLOW_ARIA_ATTR: false,
  })
}
