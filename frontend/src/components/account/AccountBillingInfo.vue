<template>
  <div class="rounded-lg border border-slate-200 bg-slate-50 p-3 text-xs text-slate-700 dark:border-slate-700 dark:bg-slate-900/40 dark:text-slate-300">
    <div class="mb-2 flex items-center justify-between gap-2">
      <span class="font-medium text-slate-900 dark:text-slate-100">计费架构</span>
      <span class="rounded bg-slate-200 px-2 py-0.5 font-mono text-[11px] dark:bg-slate-800">
        账号倍率 ×{{ formatRate(architecture.effective_account_rate_multiplier) }}
      </span>
    </div>

    <div class="space-y-1">
      <p>
        <span class="font-medium">余额扣费：</span>
        ActualCost（分组/用户专属倍率后）
      </p>
      <p>
        <span class="font-medium">订阅消耗：</span>
        ActualCost
      </p>
      <p>
        <span class="font-medium">API Key 配额：</span>
        ActualCost
      </p>
      <p>
        <span class="font-medium">账号配额：</span>
        TotalCost × 账号倍率
      </p>
    </div>

    <div v-if="architecture.groups?.length" class="mt-2 border-t border-slate-200 pt-2 dark:border-slate-700">
      <div class="mb-1 font-medium">分组倍率</div>
      <div v-for="group in architecture.groups" :key="group.id" class="flex items-center justify-between gap-2">
        <span class="truncate">{{ group.name }}</span>
        <span class="font-mono">×{{ formatRate(group.rate_multiplier) }}</span>
      </div>
    </div>

    <div v-if="architecture.effective_user_rate_multiplier != null" class="mt-2 border-t border-slate-200 pt-2 dark:border-slate-700">
      <span class="font-medium">用户有效倍率：</span>
      <span class="font-mono">×{{ formatRate(architecture.effective_user_rate_multiplier) }}</span>
      <span class="ml-1 text-slate-500">{{ userRateSourceLabel }}</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { AccountBillingArchitecture } from '@/types'

const props = defineProps<{
  architecture: AccountBillingArchitecture
}>()

const formatRate = (value: number | null | undefined): string => {
  if (value == null || Number.isNaN(Number(value))) return '1'
  return Number(value).toFixed(3).replace(/\.0+$/, '').replace(/(\.\d*?)0+$/, '$1')
}

const userRateSourceLabel = computed(() => {
  switch (props.architecture.user_rate_source) {
    case 'user_group_override':
      return '用户专属覆盖'
    case 'group_default':
      return '分组默认'
    case 'system_default':
      return '系统默认'
    default:
      return ''
  }
})
</script>
