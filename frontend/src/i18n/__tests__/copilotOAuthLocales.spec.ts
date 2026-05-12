import { describe, expect, it } from 'vitest'

import en from '../locales/en'
import zh from '../locales/zh'

describe('copilot OAuth locale keys', () => {
  it('puts zh Copilot OAuth copy under admin.accounts.oauth.copilot', () => {
    expect(zh.admin.accounts.oauth.copilot.title).toBe('GitHub Copilot 账户授权')
    expect(zh.admin.accounts.oauth.copilot.generateAuthUrl).toBe('生成设备授权链接')
    expect(zh.admin.accounts.oauth.copilot.userCode).toBe('User Code')
    expect(zh.admin.accounts.gemini.copilot).toBeUndefined()
  })

  it('puts en Copilot OAuth copy under admin.accounts.oauth.copilot', () => {
    expect(en.admin.accounts.oauth.copilot.title).toBe('GitHub Copilot Account Authorization')
    expect(en.admin.accounts.oauth.copilot.generateAuthUrl).toBe('Generate device authorization link')
    expect(en.admin.accounts.oauth.copilot.userCode).toBe('User Code')
    expect(en.admin.accounts.gemini.copilot).toBeUndefined()
  })
})
