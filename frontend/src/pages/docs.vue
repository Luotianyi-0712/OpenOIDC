<script setup lang="ts">
import { ExternalLink, Copy, Check, BookOpen, ArrowRight } from 'lucide-vue-next'
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { usePublicConfig } from '@/composables/usePublicConfig'

const { t } = useI18n()
const { settings: publicSettings } = usePublicConfig()

const copiedBlock = ref<string | null>(null)
const activeSection = ref('intro')

function copyCode(id: string, text: string) {
  navigator.clipboard.writeText(text)
  copiedBlock.value = id
  setTimeout(() => { copiedBlock.value = null }, 2000)
}

const navItems = computed(() => [
  { id: 'intro', label: t('docs.nav.intro') },
  { id: 'register-client', label: t('docs.nav.registerClient') },
  { id: 'auth-flow', label: t('docs.nav.authFlow') },
  { id: 'get-tokens', label: t('docs.nav.getTokens') },
  { id: 'userinfo', label: t('docs.nav.userinfo') },
  { id: 'scopes', label: t('docs.nav.scopes') },
  { id: 'security-levels', label: t('docs.nav.securityLevels') },
  { id: 'refresh-revoke', label: t('docs.nav.refreshRevoke') },
  { id: 'discovery', label: t('docs.nav.discovery') },
  { id: 'examples', label: t('docs.nav.examples') },
])

const claimRows = computed(() => [
  { field: 'sub', desc: t('docs.claimSub') },
  { field: 'uid', desc: t('docs.claimUid') },
  { field: 'name', desc: t('docs.claimName') },
  { field: 'email', desc: t('docs.claimEmail') },
  { field: 'email_verified', desc: t('docs.claimEmailVerified') },
  { field: 'picture', desc: t('docs.claimPicture') },
  { field: 'security_level', desc: t('docs.claimSecurityLevel') },
])

const flowSteps = computed(() => [
  t('docs.quickFlowStep1'),
  t('docs.quickFlowStep2'),
  t('docs.quickFlowStep3'),
  t('docs.quickFlowStep4'),
])

// --- Code snippets ---

const baseUrl = computed(() => (publicSettings.value.site_url || window.location.origin).replace(/\/+$/, ''))
const discoveryUrl = computed(() => `${baseUrl.value}/.well-known/openid-configuration`)

const snippetEndpoints = computed(() => `Authorize URL: ${baseUrl.value}/oauth2/authorize
Token URL:     ${baseUrl.value}/oauth2/token
UserInfo URL:  ${baseUrl.value}/oauth2/userinfo
Discovery:     ${discoveryUrl.value}
JWKS:          ${baseUrl.value}/jwks.json
Revoke URL:    ${baseUrl.value}/oauth2/revoke`)

const snippetMinimalJS = computed(() => `const CLIENT_ID = 'YOUR_CLIENT_ID'
const CLIENT_SECRET = process.env.OIDC_CLIENT_SECRET
const REDIRECT_URI = 'https://yourapp.com/callback'
const AUTH_URL = '${baseUrl.value}/oauth2/authorize'
const TOKEN_URL = '${baseUrl.value}/oauth2/token'
const USERINFO_URL = '${baseUrl.value}/oauth2/userinfo'

export function buildAuthorizeUrl() {
  const params = new URLSearchParams({
    response_type: 'code',
    client_id: CLIENT_ID,
    redirect_uri: REDIRECT_URI,
    scope: 'openid profile email security_level',
    state: crypto.randomUUID(),
  })
  return AUTH_URL + '?' + params.toString()
}

export async function exchangeCode(code) {
  const body = new URLSearchParams({
    grant_type: 'authorization_code',
    code,
    redirect_uri: REDIRECT_URI,
    client_id: CLIENT_ID,
    client_secret: CLIENT_SECRET,
  })
  const res = await fetch(TOKEN_URL, {
    method: 'POST',
    headers: { 'Content-Type': 'application/x-www-form-urlencoded', Accept: 'application/json' },
    body,
  })
  if (!res.ok) throw new Error(await res.text())
  return res.json()
}

export async function getUserinfo(accessToken) {
  const res = await fetch(USERINFO_URL, {
    headers: { Authorization: 'Bearer ' + accessToken },
  })
  if (!res.ok) throw new Error(await res.text())
  return res.json()
}`)

const snippetAuthorize = `GET /oauth2/authorize
  ?response_type=code
  &client_id=YOUR_CLIENT_ID
  &redirect_uri=https://yourapp.com/callback
  &scope=openid profile email
  &state=RANDOM_STATE`

const snippetCallback = `https://yourapp.com/callback
  ?code=AUTHORIZATION_CODE
  &state=RANDOM_STATE`

const snippetToken = computed(() => `curl -X POST ${baseUrl.value}/oauth2/token \\
  -H "Content-Type: application/x-www-form-urlencoded" \\
  -d "grant_type=authorization_code" \\
  -d "code=AUTHORIZATION_CODE" \\
  -d "redirect_uri=https://yourapp.com/callback" \\
  -d "client_id=YOUR_CLIENT_ID" \\
  -d "client_secret=YOUR_CLIENT_SECRET"`)

const snippetTokenResponse = `{
  "access_token": "eyJhbGciOiJSUzI1NiIs...",
  "token_type": "Bearer",
  "expires_in": 3600,
  "refresh_token": "dGhpcyBpcyBhIHJlZnJl...",
  "id_token": "eyJhbGciOiJSUzI1NiIs..."
}`

const snippetUserinfo = computed(() => `curl -H "Authorization: Bearer ACCESS_TOKEN" \\
  ${baseUrl.value}/oauth2/userinfo`)

const snippetUserinfoResponse = `{
  "sub": "550e8400-e29b-41d4-a716-446655440000",
  "uid": 100001,
  "name": "Alice",
  "email": "alice@example.com",
  "email_verified": true,
  "picture": "https://...",
  "security_level": 3
}`

const snippetRefresh = computed(() => `curl -X POST ${baseUrl.value}/oauth2/token \\
  -d "grant_type=refresh_token" \\
  -d "refresh_token=YOUR_REFRESH_TOKEN" \\
  -d "client_id=YOUR_CLIENT_ID" \\
  -d "client_secret=YOUR_CLIENT_SECRET"`)

const snippetRevoke = computed(() => `curl -X POST ${baseUrl.value}/oauth2/revoke \\
  -d "token=ACCESS_OR_REFRESH_TOKEN" \\
  -d "client_id=YOUR_CLIENT_ID" \\
  -d "client_secret=YOUR_CLIENT_SECRET"`)

const snippetWellKnown = computed(() => `{
  "issuer": "OIDC_ISSUER_URL",
  "authorization_endpoint": "${baseUrl.value}/oauth2/authorize",
  "token_endpoint": "${baseUrl.value}/oauth2/token",
  "userinfo_endpoint": "${baseUrl.value}/oauth2/userinfo",
  "jwks_uri": "${baseUrl.value}/jwks.json",
  "revocation_endpoint": "${baseUrl.value}/oauth2/revoke",
  "scopes_supported": ["openid", "profile", "email", "security_level"],
  "response_types_supported": ["code"],
  "grant_types_supported": [
    "authorization_code", "refresh_token", "client_credentials"
  ]
}`)

const snippetNodejs = computed(() => `// Node.js  (openid-client)
import { Issuer } from 'openid-client';

const issuer = await Issuer.discover('OIDC_ISSUER_URL');
const client = new issuer.Client({
  client_id:     'YOUR_CLIENT_ID',
  client_secret: 'YOUR_CLIENT_SECRET',
  redirect_uris: ['https://yourapp.com/callback'],
});

// Redirect user
const url = client.authorizationUrl({ scope: 'openid profile email' });

// In your callback handler
const tokenSet = await client.callback(
  'https://yourapp.com/callback',
  req.query
);
const userinfo = await client.userinfo(tokenSet.access_token);`)

const snippetPython = computed(() => `# Python  (authlib)
from authlib.integrations.requests_client import OAuth2Session

session = OAuth2Session(
    client_id='YOUR_CLIENT_ID',
    client_secret='YOUR_CLIENT_SECRET',
    redirect_uri='https://yourapp.com/callback',
)

# Redirect user
url, state = session.create_authorization_url(
    '${baseUrl.value}/oauth2/authorize',
    scope='openid profile email',
)

# In your callback handler
token = session.fetch_token(
    '${baseUrl.value}/oauth2/token',
    authorization_response=request.url,
)
userinfo = session.get('${baseUrl.value}/oauth2/userinfo').json()`)

const snippetGo = computed(() => `// Go  (golang.org/x/oauth2 + coreos/go-oidc)
provider, _ := oidc.NewProvider(ctx, "OIDC_ISSUER_URL")

oauth2Config := &oauth2.Config{
    ClientID:     "YOUR_CLIENT_ID",
    ClientSecret: "YOUR_CLIENT_SECRET",
    RedirectURL:  "https://yourapp.com/callback",
    Endpoint:     provider.Endpoint(),
    Scopes:       []string{oidc.ScopeOpenID, "profile", "email"},
}

// Redirect user
url := oauth2Config.AuthCodeURL(state)

// In your callback handler
token, _ := oauth2Config.Exchange(ctx, code)
userinfo, _ := provider.UserInfo(ctx, oauth2.StaticTokenSource(token))`)

// Intersection Observer
let observer: IntersectionObserver | null = null

onMounted(() => {
  observer = new IntersectionObserver(
    (entries) => {
      for (const entry of entries) {
        if (entry.isIntersecting) activeSection.value = entry.target.id
      }
    },
    { rootMargin: '-120px 0px -60% 0px', threshold: 0 },
  )
  for (const item of navItems.value) {
    const el = document.getElementById(item.id)
    if (el) observer.observe(el)
  }
})

onUnmounted(() => { observer?.disconnect() })

function scrollTo(id: string) {
  document.getElementById(id)?.scrollIntoView({ behavior: 'smooth' })
}
</script>

<template>
  <div class="max-w-[1200px] mx-auto px-6 md:px-10 pt-28 pb-24">
    <!-- Mobile tab bar -->
    <nav class="lg:hidden overflow-x-auto flex gap-1 pb-4 mb-8 border-b border-border -mx-6 px-6 sticky top-16 bg-background z-10">
      <a
        v-for="item in navItems"
        :key="item.id"
        :href="'#' + item.id"
        class="shrink-0 px-3 py-1.5 text-sm rounded-md whitespace-nowrap transition-colors"
        :class="activeSection === item.id ? 'bg-foreground text-background font-medium' : 'text-muted-foreground hover:text-foreground'"
        @click.prevent="scrollTo(item.id)"
      >
        {{ item.label }}
      </a>
    </nav>

    <div class="flex gap-12">
      <!-- Sidebar -->
      <aside class="hidden lg:block w-52 shrink-0">
        <nav class="sticky top-28 space-y-0.5">
          <a
            v-for="item in navItems"
            :key="item.id"
            :href="'#' + item.id"
            class="block text-sm pl-4 py-2 border-l-2 transition-colors"
            :class="activeSection === item.id ? 'border-foreground text-foreground font-medium' : 'border-transparent text-muted-foreground hover:text-foreground hover:border-border'"
            @click.prevent="scrollTo(item.id)"
          >
            {{ item.label }}
          </a>
        </nav>
      </aside>

      <!-- Content -->
      <main class="min-w-0 flex-1 space-y-0">

        <!-- 1. Introduction -->
        <section id="intro" class="border-b border-border pb-12 mb-12">
          <div class="inline-flex items-center gap-2 px-3 py-1 rounded-full border border-border bg-muted text-xs font-medium text-muted-foreground mb-6">
            <BookOpen class="w-3.5 h-3.5" />
            {{ $t('docs.badge') }}
          </div>
          <h1 class="text-3xl font-bold tracking-tight text-foreground mb-4">
            {{ $t('docs.introTitle') }}
          </h1>
          <p class="text-muted-foreground leading-relaxed max-w-2xl">
            {{ $t('docs.introDesc') }}
          </p>
          <div class="mt-6 grid grid-cols-1 sm:grid-cols-3 gap-4">
            <div class="border border-border rounded-xl p-4">
              <div class="text-xs font-semibold text-muted-foreground uppercase tracking-wider mb-1">{{ $t('docs.introProto') }}</div>
              <div class="text-sm font-medium">OAuth 2.0 + OpenID Connect</div>
            </div>
            <div class="border border-border rounded-xl p-4">
              <div class="text-xs font-semibold text-muted-foreground uppercase tracking-wider mb-1">{{ $t('docs.introGrant') }}</div>
              <div class="text-sm font-medium">Authorization Code (+ PKCE)</div>
            </div>
            <div class="border border-border rounded-xl p-4">
              <div class="text-xs font-semibold text-muted-foreground uppercase tracking-wider mb-1">{{ $t('docs.introBase') }}</div>
              <div class="text-sm font-mono font-medium">{{ baseUrl }}</div>
            </div>
          </div>

          <div class="mt-8 grid grid-cols-1 lg:grid-cols-2 gap-6">
            <div class="rounded-xl border border-border overflow-hidden">
              <div class="px-5 py-3 border-b border-border bg-muted/40">
                <h2 class="text-sm font-semibold">{{ $t('docs.endpointsTitle') }}</h2>
              </div>
              <div class="bg-foreground">
                <div class="flex items-center justify-between px-5 py-3 border-b border-white/10">
                  <span class="text-xs text-white/50 font-mono">OAuth2 / OIDC</span>
                  <button class="text-white/50 hover:text-white/80" @click="copyCode('endpoints', snippetEndpoints)">
                    <Check v-if="copiedBlock === 'endpoints'" class="w-3.5 h-3.5" /><Copy v-else class="w-3.5 h-3.5" />
                  </button>
                </div>
                <pre class="px-5 py-4 text-[0.8125rem] font-mono leading-relaxed overflow-x-auto text-white/85"><code>{{ snippetEndpoints }}</code></pre>
              </div>
            </div>

            <div class="rounded-xl border border-border overflow-hidden">
              <div class="px-5 py-3 border-b border-border bg-muted/40">
                <h2 class="text-sm font-semibold">{{ $t('docs.quickFlowTitle') }}</h2>
              </div>
              <ol class="divide-y divide-border">
                <li v-for="(step, index) in flowSteps" :key="step" class="flex gap-3 px-5 py-4 text-sm">
                  <span class="shrink-0 w-6 h-6 rounded-full bg-foreground text-background flex items-center justify-center text-xs font-semibold">
                    {{ index + 1 }}
                  </span>
                  <span class="text-muted-foreground">{{ step }}</span>
                </li>
              </ol>
            </div>
          </div>

          <div class="mt-6 rounded-xl border border-border overflow-hidden">
            <div class="px-5 py-3 border-b border-border bg-muted/40">
              <h2 class="text-sm font-semibold">{{ $t('docs.userFieldsTitle') }}</h2>
            </div>
            <table class="w-full text-sm">
              <tbody class="divide-y divide-border">
                <tr v-for="row in claimRows" :key="row.field">
                  <td class="px-5 py-3 font-mono text-xs font-semibold whitespace-nowrap">{{ row.field }}</td>
                  <td class="px-5 py-3 text-xs text-muted-foreground">{{ row.desc }}</td>
                </tr>
              </tbody>
            </table>
          </div>

          <div class="mt-6">
            <div class="mb-3">
              <h2 class="text-sm font-semibold">{{ $t('docs.minimalExampleTitle') }}</h2>
              <p class="text-xs text-muted-foreground mt-1">{{ $t('docs.minimalExampleDesc') }}</p>
            </div>
            <div class="bg-foreground rounded-xl overflow-hidden">
              <div class="flex items-center justify-between px-5 py-3 border-b border-white/10">
                <span class="text-xs text-white/50 font-mono">fetch</span>
                <button class="text-white/50 hover:text-white/80" @click="copyCode('minimal-js', snippetMinimalJS)">
                  <Check v-if="copiedBlock === 'minimal-js'" class="w-3.5 h-3.5" /><Copy v-else class="w-3.5 h-3.5" />
                </button>
              </div>
              <pre class="px-5 py-4 text-[0.8125rem] font-mono leading-relaxed overflow-x-auto text-white/85"><code>{{ snippetMinimalJS }}</code></pre>
            </div>
          </div>
        </section>

        <!-- 2. Register a Client -->
        <section id="register-client" class="border-b border-border pb-12 mb-12">
          <h2 class="text-2xl font-bold tracking-tight text-foreground mb-4">
            {{ $t('docs.registerTitle') }}
          </h2>
          <p class="text-muted-foreground leading-relaxed mb-6">
            {{ $t('docs.registerDesc') }}
          </p>
          <ol class="list-decimal list-inside space-y-3 text-sm text-foreground">
            <li>{{ $t('docs.registerStep1') }}</li>
            <li>{{ $t('docs.registerStep2') }}</li>
            <li>{{ $t('docs.registerStep3') }}</li>
            <li>{{ $t('docs.registerStep4') }}</li>
          </ol>
          <div class="mt-6 p-4 rounded-xl border border-amber-200 bg-amber-50 text-sm text-amber-800">
            {{ $t('docs.registerWarning') }}
          </div>
        </section>

        <!-- 3. Authorization Flow -->
        <section id="auth-flow" class="border-b border-border pb-12 mb-12">
          <h2 class="text-2xl font-bold tracking-tight text-foreground mb-4">
            {{ $t('docs.authFlowTitle') }}
          </h2>
          <p class="text-muted-foreground leading-relaxed mb-6">
            {{ $t('docs.authFlowDesc') }}
          </p>

          <!-- Flow diagram -->
          <div class="rounded-xl border border-border bg-muted/30 p-6 mb-8">
            <div class="flex flex-col md:flex-row items-stretch gap-3 text-center text-sm">
              <div class="flex-1 border border-border rounded-lg bg-background p-4">
                <div class="font-semibold mb-1">{{ $t('docs.flowApp') }}</div>
                <div class="text-xs text-muted-foreground">{{ $t('docs.flowAppDesc') }}</div>
              </div>
              <div class="flex items-center justify-center text-muted-foreground">
                <ArrowRight class="w-4 h-4 rotate-90 md:rotate-0" />
              </div>
              <div class="flex-1 border border-border rounded-lg bg-background p-4">
                <div class="font-semibold mb-1">{{ $t('docs.flowOIDC') }}</div>
                <div class="text-xs text-muted-foreground">{{ $t('docs.flowOIDCDesc') }}</div>
              </div>
              <div class="flex items-center justify-center text-muted-foreground">
                <ArrowRight class="w-4 h-4 rotate-90 md:rotate-0" />
              </div>
              <div class="flex-1 border border-border rounded-lg bg-background p-4">
                <div class="font-semibold mb-1">{{ $t('docs.flowCallback') }}</div>
                <div class="text-xs text-muted-foreground">{{ $t('docs.flowCallbackDesc') }}</div>
              </div>
            </div>
          </div>

          <h3 class="text-base font-semibold mb-3">{{ $t('docs.authStep1') }}</h3>
          <p class="text-sm text-muted-foreground mb-4">{{ $t('docs.authStep1Desc') }}</p>
          <!-- code -->
          <div class="bg-foreground rounded-xl overflow-hidden mb-8">
            <div class="flex items-center justify-between px-5 py-3 border-b border-white/10">
              <span class="text-xs text-white/50 font-mono">{{ $t('docs.authStep1Label') }}</span>
              <button class="text-white/50 hover:text-white/80" @click="copyCode('authorize', snippetAuthorize)">
                <Check v-if="copiedBlock === 'authorize'" class="w-3.5 h-3.5" /><Copy v-else class="w-3.5 h-3.5" />
              </button>
            </div>
            <pre class="px-5 py-4 text-[0.8125rem] font-mono leading-relaxed overflow-x-auto text-white/85"><code>{{ snippetAuthorize }}</code></pre>
          </div>

          <h3 class="text-base font-semibold mb-3">{{ $t('docs.authStep2') }}</h3>
          <p class="text-sm text-muted-foreground mb-4">{{ $t('docs.authStep2Desc') }}</p>
          <div class="bg-foreground rounded-xl overflow-hidden">
            <div class="flex items-center justify-between px-5 py-3 border-b border-white/10">
              <span class="text-xs text-white/50 font-mono">{{ $t('docs.authStep2Label') }}</span>
              <button class="text-white/50 hover:text-white/80" @click="copyCode('callback', snippetCallback)">
                <Check v-if="copiedBlock === 'callback'" class="w-3.5 h-3.5" /><Copy v-else class="w-3.5 h-3.5" />
              </button>
            </div>
            <pre class="px-5 py-4 text-[0.8125rem] font-mono leading-relaxed overflow-x-auto text-white/85"><code>{{ snippetCallback }}</code></pre>
          </div>
        </section>

        <!-- 4. Get Tokens -->
        <section id="get-tokens" class="border-b border-border pb-12 mb-12">
          <h2 class="text-2xl font-bold tracking-tight text-foreground mb-4">
            {{ $t('docs.tokensTitle') }}
          </h2>
          <p class="text-muted-foreground leading-relaxed mb-6">{{ $t('docs.tokensDesc') }}</p>

          <h3 class="text-base font-semibold mb-3">{{ $t('docs.tokensRequest') }}</h3>
          <div class="bg-foreground rounded-xl overflow-hidden mb-6">
            <div class="flex items-center justify-between px-5 py-3 border-b border-white/10">
              <span class="text-xs text-white/50 font-mono">POST /oauth2/token</span>
              <button class="text-white/50 hover:text-white/80" @click="copyCode('token', snippetToken)">
                <Check v-if="copiedBlock === 'token'" class="w-3.5 h-3.5" /><Copy v-else class="w-3.5 h-3.5" />
              </button>
            </div>
            <pre class="px-5 py-4 text-[0.8125rem] font-mono leading-relaxed overflow-x-auto text-white/85"><code>{{ snippetToken }}</code></pre>
          </div>

          <h3 class="text-base font-semibold mb-3">{{ $t('docs.tokensResponse') }}</h3>
          <div class="bg-foreground rounded-xl overflow-hidden">
            <div class="flex items-center justify-between px-5 py-3 border-b border-white/10">
              <span class="text-xs text-white/50 font-mono">Response</span>
              <button class="text-white/50 hover:text-white/80" @click="copyCode('tokenres', snippetTokenResponse)">
                <Check v-if="copiedBlock === 'tokenres'" class="w-3.5 h-3.5" /><Copy v-else class="w-3.5 h-3.5" />
              </button>
            </div>
            <pre class="px-5 py-4 text-[0.8125rem] font-mono leading-relaxed overflow-x-auto text-white/85"><code>{{ snippetTokenResponse }}</code></pre>
          </div>
          <ul class="mt-6 space-y-2 text-sm text-muted-foreground list-disc list-inside">
            <li><code class="text-xs bg-muted px-1.5 py-0.5 rounded font-mono">access_token</code> — {{ $t('docs.tokensAccessDesc') }}</li>
            <li><code class="text-xs bg-muted px-1.5 py-0.5 rounded font-mono">id_token</code> — {{ $t('docs.tokensIdDesc') }}</li>
            <li><code class="text-xs bg-muted px-1.5 py-0.5 rounded font-mono">refresh_token</code> — {{ $t('docs.tokensRefreshDesc') }}</li>
          </ul>
        </section>

        <!-- 5. UserInfo -->
        <section id="userinfo" class="border-b border-border pb-12 mb-12">
          <h2 class="text-2xl font-bold tracking-tight text-foreground mb-4">
            {{ $t('docs.userinfoTitle') }}
          </h2>
          <p class="text-muted-foreground leading-relaxed mb-6">{{ $t('docs.userinfoDesc') }}</p>

          <div class="bg-foreground rounded-xl overflow-hidden mb-6">
            <div class="flex items-center justify-between px-5 py-3 border-b border-white/10">
              <span class="text-xs text-white/50 font-mono">GET /oauth2/userinfo</span>
              <button class="text-white/50 hover:text-white/80" @click="copyCode('userinfo', snippetUserinfo)">
                <Check v-if="copiedBlock === 'userinfo'" class="w-3.5 h-3.5" /><Copy v-else class="w-3.5 h-3.5" />
              </button>
            </div>
            <pre class="px-5 py-4 text-[0.8125rem] font-mono leading-relaxed overflow-x-auto text-white/85"><code>{{ snippetUserinfo }}</code></pre>
          </div>

          <h3 class="text-base font-semibold mb-3">{{ $t('docs.userinfoResponse') }}</h3>
          <div class="bg-foreground rounded-xl overflow-hidden">
            <div class="flex items-center justify-between px-5 py-3 border-b border-white/10">
              <span class="text-xs text-white/50 font-mono">Response</span>
              <button class="text-white/50 hover:text-white/80" @click="copyCode('userinfores', snippetUserinfoResponse)">
                <Check v-if="copiedBlock === 'userinfores'" class="w-3.5 h-3.5" /><Copy v-else class="w-3.5 h-3.5" />
              </button>
            </div>
            <pre class="px-5 py-4 text-[0.8125rem] font-mono leading-relaxed overflow-x-auto text-white/85"><code>{{ snippetUserinfoResponse }}</code></pre>
          </div>
        </section>

        <!-- 6. Scopes & Claims -->
        <section id="scopes" class="border-b border-border pb-12 mb-12">
          <h2 class="text-2xl font-bold tracking-tight text-foreground mb-4">
            {{ $t('docs.scopesTitle') }}
          </h2>
          <p class="text-muted-foreground leading-relaxed mb-6">{{ $t('docs.scopesDesc') }}</p>
          <div class="rounded-xl border border-border overflow-hidden">
            <table class="w-full text-sm">
              <thead class="bg-muted/50 text-left text-xs font-medium text-muted-foreground uppercase tracking-wider">
                <tr>
                  <th class="px-5 py-3">Scope</th>
                  <th class="px-5 py-3">Claims</th>
                  <th class="px-5 py-3">{{ $t('docs.scopeDesc') }}</th>
                </tr>
              </thead>
              <tbody class="divide-y divide-border">
                <tr><td class="px-5 py-3 font-mono text-xs font-semibold">openid</td><td class="px-5 py-3 text-xs text-muted-foreground">sub</td><td class="px-5 py-3 text-xs text-muted-foreground">{{ $t('docs.scopeOpenid') }}</td></tr>
                <tr><td class="px-5 py-3 font-mono text-xs font-semibold">profile</td><td class="px-5 py-3 text-xs text-muted-foreground">name, picture</td><td class="px-5 py-3 text-xs text-muted-foreground">{{ $t('docs.scopeProfile') }}</td></tr>
                <tr><td class="px-5 py-3 font-mono text-xs font-semibold">email</td><td class="px-5 py-3 text-xs text-muted-foreground">email, email_verified</td><td class="px-5 py-3 text-xs text-muted-foreground">{{ $t('docs.scopeEmail') }}</td></tr>
                <tr><td class="px-5 py-3 font-mono text-xs font-semibold">security_level</td><td class="px-5 py-3 text-xs text-muted-foreground">security_level</td><td class="px-5 py-3 text-xs text-muted-foreground">{{ $t('docs.scopeSecurity') }}</td></tr>
              </tbody>
            </table>
          </div>
        </section>

        <!-- 7. Security Levels -->
        <section id="security-levels" class="border-b border-border pb-12 mb-12">
          <h2 class="text-2xl font-bold tracking-tight text-foreground mb-4">
            {{ $t('docs.secLevelTitle') }}
          </h2>
          <p class="text-muted-foreground leading-relaxed mb-6">{{ $t('docs.secLevelDesc') }}</p>
          <div class="space-y-3">
            <div class="flex items-start gap-4 border border-border rounded-xl p-4">
              <span class="shrink-0 w-8 h-8 rounded-full bg-muted flex items-center justify-center text-sm font-bold">0</span>
              <div><div class="font-semibold text-sm">{{ $t('docs.secLevel0') }}</div><div class="text-xs text-muted-foreground mt-0.5">{{ $t('docs.secLevel0Desc') }}</div></div>
            </div>
            <div class="flex items-start gap-4 border border-border rounded-xl p-4">
              <span class="shrink-0 w-8 h-8 rounded-full bg-blue-50 text-blue-600 flex items-center justify-center text-sm font-bold">1</span>
              <div><div class="font-semibold text-sm">{{ $t('docs.secLevel1') }}</div><div class="text-xs text-muted-foreground mt-0.5">{{ $t('docs.secLevel1Desc') }}</div></div>
            </div>
            <div class="flex items-start gap-4 border border-border rounded-xl p-4">
              <span class="shrink-0 w-8 h-8 rounded-full bg-green-50 text-green-600 flex items-center justify-center text-sm font-bold">2+</span>
              <div><div class="font-semibold text-sm">{{ $t('docs.secLevel2') }}</div><div class="text-xs text-muted-foreground mt-0.5">{{ $t('docs.secLevel2Desc') }}</div></div>
            </div>
          </div>
          <div class="mt-6 p-4 rounded-xl border border-border bg-muted/30 text-sm text-muted-foreground">
            {{ $t('docs.secLevelHint') }}
          </div>
        </section>

        <!-- 8. Refresh & Revoke -->
        <section id="refresh-revoke" class="border-b border-border pb-12 mb-12">
          <h2 class="text-2xl font-bold tracking-tight text-foreground mb-4">
            {{ $t('docs.refreshTitle') }}
          </h2>

          <h3 class="text-base font-semibold mb-3">{{ $t('docs.refreshSub') }}</h3>
          <p class="text-sm text-muted-foreground mb-4">{{ $t('docs.refreshDesc') }}</p>
          <div class="bg-foreground rounded-xl overflow-hidden mb-8">
            <div class="flex items-center justify-between px-5 py-3 border-b border-white/10">
              <span class="text-xs text-white/50 font-mono">POST /oauth2/token</span>
              <button class="text-white/50 hover:text-white/80" @click="copyCode('refresh', snippetRefresh)">
                <Check v-if="copiedBlock === 'refresh'" class="w-3.5 h-3.5" /><Copy v-else class="w-3.5 h-3.5" />
              </button>
            </div>
            <pre class="px-5 py-4 text-[0.8125rem] font-mono leading-relaxed overflow-x-auto text-white/85"><code>{{ snippetRefresh }}</code></pre>
          </div>

          <h3 class="text-base font-semibold mb-3">{{ $t('docs.revokeSub') }}</h3>
          <p class="text-sm text-muted-foreground mb-4">{{ $t('docs.revokeDesc') }}</p>
          <div class="bg-foreground rounded-xl overflow-hidden">
            <div class="flex items-center justify-between px-5 py-3 border-b border-white/10">
              <span class="text-xs text-white/50 font-mono">POST /oauth2/revoke</span>
              <button class="text-white/50 hover:text-white/80" @click="copyCode('revoke', snippetRevoke)">
                <Check v-if="copiedBlock === 'revoke'" class="w-3.5 h-3.5" /><Copy v-else class="w-3.5 h-3.5" />
              </button>
            </div>
            <pre class="px-5 py-4 text-[0.8125rem] font-mono leading-relaxed overflow-x-auto text-white/85"><code>{{ snippetRevoke }}</code></pre>
          </div>
        </section>

        <!-- 9. Discovery -->
        <section id="discovery" class="border-b border-border pb-12 mb-12">
          <h2 class="text-2xl font-bold tracking-tight text-foreground mb-4">
            {{ $t('docs.discoveryTitle') }}
          </h2>
          <p class="text-sm text-muted-foreground mb-2">{{ $t('docs.discoveryDesc') }}</p>
          <p class="text-xs text-muted-foreground mb-4">{{ $t('docs.discoveryIssuerHint') }}</p>
          <div class="flex items-center gap-3 mb-6">
            <code class="text-xs bg-muted border border-border rounded-lg px-3 py-2 font-mono">GET /.well-known/openid-configuration</code>
            <a :href="discoveryUrl" target="_blank" class="inline-flex items-center gap-1.5 text-sm font-medium text-foreground hover:text-foreground/80">
              {{ $t('docs.discoveryLive') }} <ExternalLink class="w-3.5 h-3.5" />
            </a>
          </div>
          <div class="bg-foreground rounded-xl overflow-hidden">
            <div class="flex items-center justify-between px-5 py-3 border-b border-white/10">
              <span class="text-xs text-white/50 font-mono">Response</span>
              <button class="text-white/50 hover:text-white/80" @click="copyCode('wellknown', snippetWellKnown)">
                <Check v-if="copiedBlock === 'wellknown'" class="w-3.5 h-3.5" /><Copy v-else class="w-3.5 h-3.5" />
              </button>
            </div>
            <pre class="px-5 py-4 text-[0.8125rem] font-mono leading-relaxed overflow-x-auto text-white/85"><code>{{ snippetWellKnown }}</code></pre>
          </div>
          <p class="mt-4 text-sm text-muted-foreground">
            {{ $t('docs.discoveryJwksHint') }}
          </p>
        </section>

        <!-- 10. Code Examples -->
        <section id="examples" class="pb-12">
          <h2 class="text-2xl font-bold tracking-tight text-foreground mb-4">
            {{ $t('docs.examplesTitle') }}
          </h2>
          <p class="text-muted-foreground leading-relaxed mb-8">{{ $t('docs.examplesDesc') }}</p>

          <!-- Node.js -->
          <h3 class="text-base font-semibold mb-3">Node.js</h3>
          <div class="bg-foreground rounded-xl overflow-hidden mb-8">
            <div class="flex items-center justify-between px-5 py-3 border-b border-white/10">
              <span class="text-xs text-white/50 font-mono">openid-client</span>
              <button class="text-white/50 hover:text-white/80" @click="copyCode('nodejs', snippetNodejs)">
                <Check v-if="copiedBlock === 'nodejs'" class="w-3.5 h-3.5" /><Copy v-else class="w-3.5 h-3.5" />
              </button>
            </div>
            <pre class="px-5 py-4 text-[0.8125rem] font-mono leading-relaxed overflow-x-auto text-white/85"><code>{{ snippetNodejs }}</code></pre>
          </div>

          <!-- Python -->
          <h3 class="text-base font-semibold mb-3">Python</h3>
          <div class="bg-foreground rounded-xl overflow-hidden mb-8">
            <div class="flex items-center justify-between px-5 py-3 border-b border-white/10">
              <span class="text-xs text-white/50 font-mono">authlib</span>
              <button class="text-white/50 hover:text-white/80" @click="copyCode('python', snippetPython)">
                <Check v-if="copiedBlock === 'python'" class="w-3.5 h-3.5" /><Copy v-else class="w-3.5 h-3.5" />
              </button>
            </div>
            <pre class="px-5 py-4 text-[0.8125rem] font-mono leading-relaxed overflow-x-auto text-white/85"><code>{{ snippetPython }}</code></pre>
          </div>

          <!-- Go -->
          <h3 class="text-base font-semibold mb-3">Go</h3>
          <div class="bg-foreground rounded-xl overflow-hidden">
            <div class="flex items-center justify-between px-5 py-3 border-b border-white/10">
              <span class="text-xs text-white/50 font-mono">go-oidc + oauth2</span>
              <button class="text-white/50 hover:text-white/80" @click="copyCode('go', snippetGo)">
                <Check v-if="copiedBlock === 'go'" class="w-3.5 h-3.5" /><Copy v-else class="w-3.5 h-3.5" />
              </button>
            </div>
            <pre class="px-5 py-4 text-[0.8125rem] font-mono leading-relaxed overflow-x-auto text-white/85"><code>{{ snippetGo }}</code></pre>
          </div>
        </section>

      </main>
    </div>
  </div>
</template>
