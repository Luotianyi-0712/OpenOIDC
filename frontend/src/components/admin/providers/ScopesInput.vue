<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { ChevronDown, X } from 'lucide-vue-next'
import { searchScopes, type ScopeOption } from '@/composables/providerScopes'

const props = defineProps<{
  provider: string
  modelValue: string
}>()

const emit = defineEmits<{
  'update:modelValue': [value: string]
}>()

const inputRef = ref<HTMLInputElement>()
const showDropdown = ref(false)
const searchQuery = ref('')
const selectedIndex = ref(0)

// 解析当前已选择的 scopes
const selectedScopes = computed(() => {
  return props.modelValue
    .split(/[\s,]+/)
    .map(s => s.trim())
    .filter(Boolean)
})

// 搜索候选项
const suggestions = computed(() => {
  const results = searchScopes(props.provider, searchQuery.value)
  // 过滤掉已选择的
  return results.filter(s => !selectedScopes.value.includes(s.value))
})

// 监听搜索内容变化，重置选中索引
watch(searchQuery, () => {
  selectedIndex.value = 0
})

function onInput(event: Event) {
  const target = event.target as HTMLInputElement
  searchQuery.value = target.value
  showDropdown.value = true
}

function onFocus() {
  showDropdown.value = true
}

function onBlur() {
  // 延迟关闭，让点击事件能触发
  setTimeout(() => {
    showDropdown.value = false
    searchQuery.value = ''
  }, 200)
}

function selectScope(scope: ScopeOption) {
  const newScopes = [...selectedScopes.value, scope.value]
  emit('update:modelValue', newScopes.join(' '))
  searchQuery.value = ''
  selectedIndex.value = 0
  inputRef.value?.focus()
}

function removeScope(scope: string) {
  const newScopes = selectedScopes.value.filter(s => s !== scope)
  emit('update:modelValue', newScopes.join(' '))
}

function onKeyDown(event: KeyboardEvent) {
  if (!showDropdown.value || suggestions.value.length === 0) {
    return
  }

  switch (event.key) {
    case 'ArrowDown':
      event.preventDefault()
      selectedIndex.value = Math.min(selectedIndex.value + 1, suggestions.value.length - 1)
      break
    case 'ArrowUp':
      event.preventDefault()
      selectedIndex.value = Math.max(selectedIndex.value - 1, 0)
      break
    case 'Enter':
      event.preventDefault()
      if (suggestions.value[selectedIndex.value]) {
        selectScope(suggestions.value[selectedIndex.value])
      }
      break
    case 'Escape':
      event.preventDefault()
      showDropdown.value = false
      searchQuery.value = ''
      break
  }
}
</script>

<template>
  <div class="relative">
    <!-- 已选择的 scopes -->
    <div v-if="selectedScopes.length > 0" class="flex flex-wrap gap-1.5 mb-2">
      <span
        v-for="scope in selectedScopes"
        :key="scope"
        class="inline-flex items-center gap-1 px-2 py-1 bg-blue-50 text-blue-700 rounded text-xs font-medium"
      >
        {{ scope }}
        <button
          type="button"
          class="hover:bg-blue-100 rounded-full p-0.5 transition-colors"
          @click="removeScope(scope)"
        >
          <X class="w-3 h-3" />
        </button>
      </span>
    </div>

    <!-- 输入框 -->
    <div class="relative">
      <input
        ref="inputRef"
        :value="searchQuery"
        type="text"
        class="w-full px-3 py-2 pr-8 border border-border rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-foreground/10"
        :placeholder="selectedScopes.length > 0 ? '添加更多权限...' : '搜索权限范围...'"
        @input="onInput"
        @focus="onFocus"
        @blur="onBlur"
        @keydown="onKeyDown"
      />
      <ChevronDown class="absolute right-2.5 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground pointer-events-none" />
    </div>

    <!-- 下拉候选列表 -->
    <Transition
      enter-active-class="transition duration-100 ease-out"
      enter-from-class="transform scale-95 opacity-0"
      enter-to-class="transform scale-100 opacity-100"
      leave-active-class="transition duration-75 ease-in"
      leave-from-class="transform scale-100 opacity-100"
      leave-to-class="transform scale-95 opacity-0"
    >
      <div
        v-if="showDropdown && suggestions.length > 0"
        class="absolute z-50 w-full mt-1 bg-white border border-border rounded-lg shadow-lg max-h-64 overflow-y-auto"
      >
        <button
          v-for="(suggestion, index) in suggestions"
          :key="suggestion.value"
          type="button"
          class="w-full px-3 py-2 text-left hover:bg-muted transition-colors"
          :class="{ 'bg-muted': index === selectedIndex }"
          @click="selectScope(suggestion)"
        >
          <div class="flex items-start justify-between gap-2">
            <div class="min-w-0 flex-1">
              <div class="text-sm font-medium text-foreground">{{ suggestion.label }}</div>
              <div class="text-xs text-muted-foreground mt-0.5 line-clamp-2">{{ suggestion.description }}</div>
            </div>
          </div>
        </button>
      </div>
    </Transition>

    <!-- 无结果提示 -->
    <div
      v-if="showDropdown && searchQuery && suggestions.length === 0"
      class="absolute z-50 w-full mt-1 bg-white border border-border rounded-lg shadow-lg p-3"
    >
      <p class="text-sm text-muted-foreground text-center">未找到匹配的权限范围</p>
    </div>
  </div>
</template>
