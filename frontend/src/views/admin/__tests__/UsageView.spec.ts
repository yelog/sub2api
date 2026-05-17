import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

import UsageView from '../UsageView.vue'

const { list, getStats, getSnapshotV2, getModelStats, getById, exportList, saveAs } = vi.hoisted(() => {
  vi.stubGlobal('localStorage', {
    getItem: vi.fn(() => null),
    setItem: vi.fn(),
    removeItem: vi.fn(),
  })

  return {
    list: vi.fn(),
    getStats: vi.fn(),
    getSnapshotV2: vi.fn(),
    getModelStats: vi.fn(),
    getById: vi.fn(),
    exportList: vi.fn(),
    saveAs: vi.fn(),
  }
})

const messages: Record<string, string> = {
  'admin.dashboard.timeRange': 'Time Range',
  'admin.dashboard.day': 'Day',
  'admin.dashboard.hour': 'Hour',
  'admin.usage.failedToLoadUser': 'Failed to load user',
}

const formatLocalDate = (date: Date): string => {
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}

vi.mock('@/api/admin', () => ({
  adminAPI: {
    usage: {
      list,
      getStats,
    },
    dashboard: {
      getSnapshotV2,
      getModelStats,
    },
    users: {
      getById,
    },
  },
}))

vi.mock('@/api/admin/usage', () => ({
  adminUsageAPI: {
    list: exportList,
  },
}))

vi.mock('file-saver', () => ({
  saveAs,
}))

vi.mock('xlsx', () => ({
  utils: {
    aoa_to_sheet: vi.fn((rows) => ({ rows })),
    sheet_add_aoa: vi.fn((sheet, rows) => {
      sheet.rows.push(...rows)
    }),
    book_new: vi.fn(() => ({ sheets: [] })),
    book_append_sheet: vi.fn((book, sheet, name) => {
      book.sheets.push({ sheet, name })
    }),
  },
  write: vi.fn(() => new ArrayBuffer(0)),
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError: vi.fn(),
    showWarning: vi.fn(),
    showSuccess: vi.fn(),
    showInfo: vi.fn(),
  }),
}))

vi.mock('@/utils/format', () => ({
  formatReasoningEffort: (value: string | null | undefined) => value ?? '-',
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => messages[key] ?? key,
    }),
  }
})

vi.mock('vue-router', () => ({
  useRoute: () => ({
    query: {}
  })
}))

const AppLayoutStub = { template: '<div><slot /></div>' }
const UsageFiltersStub = { template: '<div><slot name="after-reset" /></div>' }
const UsageFiltersExportStub = {
  props: ['modelValue'],
  emits: ['update:modelValue', 'change', 'export'],
  template: `
    <div>
      <button data-test="fast-filter" @click="setFast">fast</button>
      <button data-test="export" @click="$emit('export')">export</button>
      <slot name="after-reset" />
    </div>
  `,
  methods: {
    setFast() {
      this.$emit('update:modelValue', { ...this.modelValue, openai_fast: 'fast' })
      this.$emit('change')
    },
  },
}
const ModelDistributionChartStub = {
  props: ['metric'],
  emits: ['update:metric'],
  template: `
    <div data-test="model-chart">
      <span class="metric">{{ metric }}</span>
      <button class="switch-metric" @click="$emit('update:metric', 'actual_cost')">switch</button>
    </div>
  `,
}
const GroupDistributionChartStub = {
  props: ['metric'],
  emits: ['update:metric'],
  template: `
    <div data-test="group-chart">
      <span class="metric">{{ metric }}</span>
      <button class="switch-metric" @click="$emit('update:metric', 'actual_cost')">switch</button>
    </div>
  `,
}

describe('admin UsageView distribution metric toggles', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    list.mockReset()
    getStats.mockReset()
    getSnapshotV2.mockReset()
    getModelStats.mockReset()
    getById.mockReset()
    exportList.mockReset()
    saveAs.mockReset()

    list.mockResolvedValue({
      items: [],
      total: 0,
      pages: 0,
    })
    getStats.mockResolvedValue({
      total_requests: 0,
      total_input_tokens: 0,
      total_output_tokens: 0,
      total_cache_tokens: 0,
      total_tokens: 0,
      total_cost: 0,
      total_actual_cost: 0,
      average_duration_ms: 0,
    })
    getSnapshotV2.mockResolvedValue({
      trend: [],
      models: [],
      groups: [],
    })
    getModelStats.mockResolvedValue({ models: [] })
    exportList.mockResolvedValue({ items: [], total: 0, pages: 0 })
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('keeps model and group metric toggles independent without refetching chart data', async () => {
    const wrapper = mount(UsageView, {
      global: {
        stubs: {
          AppLayout: AppLayoutStub,
          UsageStatsCards: true,
          UsageFilters: UsageFiltersStub,
          UsageTable: true,
          UsageExportProgress: true,
          UsageCleanupDialog: true,
          UserBalanceHistoryModal: true,
          Pagination: true,
          Select: true,
          DateRangePicker: true,
          Icon: true,
          TokenUsageTrend: true,
          ModelDistributionChart: ModelDistributionChartStub,
          GroupDistributionChart: GroupDistributionChartStub,
        },
      },
    })

    vi.advanceTimersByTime(120)
    await flushPromises()

    expect(getSnapshotV2).toHaveBeenCalledTimes(1)
    const now = new Date()
    const yesterday = new Date(now.getTime() - 24 * 60 * 60 * 1000)
    expect(getSnapshotV2).toHaveBeenCalledWith(expect.objectContaining({
      start_date: formatLocalDate(yesterday),
      end_date: formatLocalDate(now),
      granularity: 'hour'
    }))

    const modelChart = wrapper.find('[data-test="model-chart"]')
    const groupChart = wrapper.find('[data-test="group-chart"]')

    expect(modelChart.find('.metric').text()).toBe('tokens')
    expect(groupChart.find('.metric').text()).toBe('tokens')

    await modelChart.find('.switch-metric').trigger('click')
    await flushPromises()

    expect(modelChart.find('.metric').text()).toBe('actual_cost')
    expect(groupChart.find('.metric').text()).toBe('tokens')
    expect(getSnapshotV2).toHaveBeenCalledTimes(1)

    await groupChart.find('.switch-metric').trigger('click')
    await flushPromises()

    expect(modelChart.find('.metric').text()).toBe('actual_cost')
    expect(groupChart.find('.metric').text()).toBe('actual_cost')
    expect(getSnapshotV2).toHaveBeenCalledTimes(1)
  })
})

describe('admin UsageView OpenAI Fast usage support', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    list.mockReset()
    getStats.mockReset()
    getSnapshotV2.mockReset()
    getModelStats.mockReset()
    getById.mockReset()
    exportList.mockReset()
    saveAs.mockReset()

    list.mockResolvedValue({ items: [], total: 0, pages: 0 })
    getStats.mockResolvedValue({
      total_requests: 0,
      total_input_tokens: 0,
      total_output_tokens: 0,
      total_cache_tokens: 0,
      total_tokens: 0,
      total_cost: 0,
      total_actual_cost: 0,
      average_duration_ms: 0,
    })
    getSnapshotV2.mockResolvedValue({ trend: [], models: [], groups: [] })
    getModelStats.mockResolvedValue({ models: [] })
    exportList.mockResolvedValue({ items: [], total: 0, pages: 0 })
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  const mountUsageView = () => mount(UsageView, {
    global: {
      stubs: {
        AppLayout: AppLayoutStub,
        UsageStatsCards: true,
        UsageFilters: UsageFiltersExportStub,
        UsageTable: true,
        UsageExportProgress: true,
        UsageCleanupDialog: true,
        UserBalanceHistoryModal: true,
        Pagination: true,
        Select: true,
        DateRangePicker: true,
        Icon: true,
        TokenUsageTrend: true,
        ModelDistributionChart: true,
        GroupDistributionChart: true,
        EndpointDistributionChart: true,
      },
    },
  })

  it('passes openai_fast to list, stats, and export requests', async () => {
    const wrapper = mountUsageView()
    await flushPromises()

    await wrapper.find('[data-test="fast-filter"]').trigger('click')
    await flushPromises()

    expect(list).toHaveBeenLastCalledWith(expect.objectContaining({ openai_fast: 'fast' }), expect.anything())
    expect(getStats).toHaveBeenLastCalledWith(expect.objectContaining({ openai_fast: 'fast' }))

    await wrapper.find('[data-test="export"]').trigger('click')
    await flushPromises()

    expect(exportList).toHaveBeenCalledWith(expect.objectContaining({ openai_fast: 'fast' }), expect.anything())
  })

  it('exports a fast_bucket column derived from service_tier priority', async () => {
    exportList.mockResolvedValueOnce({
      total: 2,
      pages: 1,
      items: [
        { request_id: 'fast-req', created_at: '2026-05-17T00:00:00Z', service_tier: 'priority', input_tokens: 1, output_tokens: 2, cache_read_tokens: 0, cache_creation_tokens: 0, duration_ms: 10, total_cost: 0.1, actual_cost: 0.1, account_rate_multiplier: 1 },
        { request_id: 'normal-req', created_at: '2026-05-17T00:01:00Z', service_tier: null, input_tokens: 3, output_tokens: 4, cache_read_tokens: 0, cache_creation_tokens: 0, duration_ms: 20, total_cost: 0.2, actual_cost: 0.2, account_rate_multiplier: 1 },
      ],
    })

    const wrapper = mountUsageView()
    await flushPromises()
    await wrapper.find('[data-test="export"]').trigger('click')
    await flushPromises()

    const xlsx = await import('xlsx')
    const headerRows = vi.mocked(xlsx.utils.aoa_to_sheet).mock.calls[0][0] as unknown[][]
    const exportedRows = vi.mocked(xlsx.utils.sheet_add_aoa).mock.calls[0][1] as unknown[][]
    const fastBucketIndex = headerRows[0].indexOf('fast_bucket')
    expect(fastBucketIndex).toBeGreaterThanOrEqual(0)
    expect(exportedRows[0][fastBucketIndex]).toBe('fast')
    expect(exportedRows[1][fastBucketIndex]).toBe('non_fast')
  })
})
