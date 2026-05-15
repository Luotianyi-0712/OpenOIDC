<script setup lang="ts">
</script>

<template>
  <div class="max-w-[800px] mx-auto px-6 md:px-10 pt-32 pb-24">
    <h1 class="text-3xl font-bold tracking-tight text-foreground">
      {{ $t('terms.title') }}
    </h1>
    <p class="text-sm text-muted-foreground mt-2">
      {{ $t('terms.lastUpdated') }}
    </p>
    <p class="text-muted-foreground mt-6 leading-relaxed">
      {{ $t('terms.intro') }}
    </p>

    <!-- Acceptance of Terms -->
    <section class="mt-10">
      <h2 class="text-xl font-semibold text-foreground">
        {{ $t('terms.acceptanceTitle') }}
      </h2>
      <p class="text-muted-foreground mt-3 leading-relaxed">
        {{ $t('terms.acceptanceDesc') }}
      </p>
    </section>

    <!-- Service Description -->
    <section class="mt-10">
      <h2 class="text-xl font-semibold text-foreground">
        {{ $t('terms.serviceTitle') }}
      </h2>
      <p class="text-muted-foreground mt-3 leading-relaxed">
        {{ $t('terms.serviceDesc') }}
      </p>
      <ul class="list-disc list-inside text-muted-foreground mt-3 space-y-1.5 leading-relaxed">
        <li>{{ $t('terms.serviceOidc') }}</li>
        <li>{{ $t('terms.serviceSocial') }}</li>
        <li>{{ $t('terms.serviceSecurity') }}</li>
      </ul>
    </section>

    <!-- User Accounts -->
    <section class="mt-10">
      <h2 class="text-xl font-semibold text-foreground">
        {{ $t('terms.accountsTitle') }}
      </h2>
      <p class="text-muted-foreground mt-3 leading-relaxed">
        {{ $t('terms.accountsDesc') }}
      </p>
      <ul class="list-disc list-inside text-muted-foreground mt-3 space-y-1.5 leading-relaxed">
        <li>{{ $t('terms.accountsAccurate') }}</li>
        <li>{{ $t('terms.accountsPassword') }}</li>
        <li>{{ $t('terms.accountsSocial') }}</li>
      </ul>
    </section>

    <!-- Third-Party Applications -->
    <section class="mt-10">
      <h2 class="text-xl font-semibold text-foreground">
        {{ $t('terms.thirdPartyTitle') }}
      </h2>
      <p class="text-muted-foreground mt-3 leading-relaxed">
        {{ $t('terms.thirdPartyDesc') }}
      </p>
      <ul class="list-disc list-inside text-muted-foreground mt-3 space-y-1.5 leading-relaxed">
        <li>{{ $t('terms.thirdPartyConsent') }}</li>
        <li>{{ $t('terms.thirdPartySecurity') }}</li>
      </ul>
    </section>

    <!-- Prohibited Use -->
    <section class="mt-10">
      <h2 class="text-xl font-semibold text-foreground">
        {{ $t('terms.prohibitedTitle') }}
      </h2>
      <p class="text-muted-foreground mt-3 leading-relaxed">
        {{ $t('terms.prohibitedDesc') }}
      </p>
    </section>

    <!-- Limitation of Liability -->
    <section class="mt-10">
      <h2 class="text-xl font-semibold text-foreground">
        {{ $t('terms.liabilityTitle') }}
      </h2>
      <p class="text-muted-foreground mt-3 leading-relaxed">
        {{ $t('terms.liabilityDesc') }}
      </p>
    </section>

    <!-- Changes to Terms -->
    <section class="mt-10 border-t border-border pt-10">
      <h2 class="text-xl font-semibold text-foreground">
        {{ $t('terms.changesTitle') }}
      </h2>
      <p class="text-muted-foreground mt-3 leading-relaxed">
        {{ $t('terms.changesDesc') }}
      </p>
    </section>
  </div>
</template>
