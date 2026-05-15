<script setup lang="ts">
</script>

<template>
  <div class="max-w-[800px] mx-auto px-6 md:px-10 pt-32 pb-24">
    <h1 class="text-3xl font-bold tracking-tight text-foreground">
      {{ $t('privacy.title') }}
    </h1>
    <p class="text-sm text-muted-foreground mt-2">
      {{ $t('privacy.lastUpdated') }}
    </p>
    <p class="text-muted-foreground mt-6 leading-relaxed">
      {{ $t('privacy.intro') }}
    </p>

    <!-- Information We Collect -->
    <section class="mt-10">
      <h2 class="text-xl font-semibold text-foreground">
        {{ $t('privacy.collectTitle') }}
      </h2>
      <p class="text-muted-foreground mt-3 leading-relaxed">
        {{ $t('privacy.collectDesc') }}
      </p>
      <ul class="list-disc list-inside text-muted-foreground mt-3 space-y-1.5 leading-relaxed">
        <li>{{ $t('privacy.collectEmail') }}</li>
        <li>{{ $t('privacy.collectName') }}</li>
        <li>{{ $t('privacy.collectSocial') }}</li>
        <li>{{ $t('privacy.collectIp') }}</li>
        <li>{{ $t('privacy.collectSession') }}</li>
      </ul>
    </section>

    <!-- How We Use Your Information -->
    <section class="mt-10">
      <h2 class="text-xl font-semibold text-foreground">
        {{ $t('privacy.useTitle') }}
      </h2>
      <p class="text-muted-foreground mt-3 leading-relaxed">
        {{ $t('privacy.useDesc') }}
      </p>
      <ul class="list-disc list-inside text-muted-foreground mt-3 space-y-1.5 leading-relaxed">
        <li>{{ $t('privacy.useAuth') }}</li>
        <li>{{ $t('privacy.useSecurity') }}</li>
        <li>{{ $t('privacy.useSession') }}</li>
      </ul>
    </section>

    <!-- Data Sharing -->
    <section class="mt-10">
      <h2 class="text-xl font-semibold text-foreground">
        {{ $t('privacy.sharingTitle') }}
      </h2>
      <p class="text-muted-foreground mt-3 leading-relaxed">
        {{ $t('privacy.sharingDesc') }}
      </p>
    </section>

    <!-- Data Security -->
    <section class="mt-10">
      <h2 class="text-xl font-semibold text-foreground">
        {{ $t('privacy.securityTitle') }}
      </h2>
      <p class="text-muted-foreground mt-3 leading-relaxed">
        {{ $t('privacy.securityDesc') }}
      </p>
      <ul class="list-disc list-inside text-muted-foreground mt-3 space-y-1.5 leading-relaxed">
        <li>{{ $t('privacy.securityHash') }}</li>
        <li>{{ $t('privacy.securityTokens') }}</li>
        <li>{{ $t('privacy.securityAudit') }}</li>
      </ul>
    </section>

    <!-- Your Rights -->
    <section class="mt-10">
      <h2 class="text-xl font-semibold text-foreground">
        {{ $t('privacy.rightsTitle') }}
      </h2>
      <p class="text-muted-foreground mt-3 leading-relaxed">
        {{ $t('privacy.rightsDesc') }}
      </p>
      <ul class="list-disc list-inside text-muted-foreground mt-3 space-y-1.5 leading-relaxed">
        <li>{{ $t('privacy.rightsAccess') }}</li>
        <li>{{ $t('privacy.rightsCorrect') }}</li>
        <li>{{ $t('privacy.rightsDelete') }}</li>
        <li>{{ $t('privacy.rightsRevoke') }}</li>
      </ul>
    </section>

    <!-- Contact -->
    <section class="mt-10 border-t border-border pt-10">
      <h2 class="text-xl font-semibold text-foreground">
        {{ $t('privacy.contactTitle') }}
      </h2>
      <p class="text-muted-foreground mt-3 leading-relaxed">
        {{ $t('privacy.contactDesc') }}
      </p>
    </section>
  </div>
</template>
