import { execFileSync } from 'node:child_process'
import { resolve } from 'node:path'
import { render, screen, cleanup } from '@testing-library/react'
import { afterEach, beforeAll, expect, it, vi } from 'vitest'
import { ChatMessages } from './components/agent/ChatMessages'
import { ChatInput } from './components/agent/ChatInput'
import { GuidePage } from './guide/GuidePage'

let status: Record<string, string>
beforeAll(() => {
  const output = execFileSync(
    'go',
    [
      'test',
      './agent',
      '-run',
      '^TestBrandStatusRenderFixture$',
      '-count=1',
      '-v',
    ],
    { cwd: resolve('..'), encoding: 'utf8', timeout: 120000 }
  )
  const line = output
    .split('\n')
    .find((line) => line.startsWith('BRAND_STATUS_JSON:'))
  if (!line) throw new Error('Real Go status-handler fixture missing')
  status = JSON.parse(line.slice('BRAND_STATUS_JSON:'.length))
}, 120000)
afterEach(() => {
  cleanup()
  vi.unstubAllGlobals()
})

it.each(['en', 'zh', 'id'])(
  'renders the real status card and sender label in %s',
  (language) => {
    const { container } = render(
      <ChatMessages
        messages={[
          {
            id: 'status-fixture',
            role: 'bot',
            text: status[language],
            time: '12:34',
          },
        ]}
      />
    )
    expect(container.textContent).toContain(
      language === 'zh' ? 'VL 状态' : 'VL Status'
    )
    expect(container.textContent).toContain('VL · 12:34')
    expect(container.textContent).not.toMatch(/NOFXi?|VL Trader/i)
  }
)

it.each(['en', 'zh', 'id'])(
  'renders the input placeholder and disclaimer in %s',
  (language) => {
    const { container } = render(
      <ChatInput
        language={language}
        loading={false}
        value=""
        onChange={() => {}}
        onSend={() => {}}
        onStop={() => {}}
      />
    )
    expect(screen.getByRole('textbox').getAttribute('placeholder')).toContain(
      language === 'zh' ? '跟 VL 聊点什么' : 'Ask VL anything'
    )
    expect(container.textContent).toContain('VL may make mistakes.')
    expect(container.textContent).not.toMatch(/NOFXi?/i)
  }
)

it('renders the product title in the served HTML', async () => {
  const output = execFileSync(
    process.execPath,
    [
      '--input-type=module',
      '-e',
      `
  import { createServer } from 'vite';
  import { readFileSync } from 'node:fs';
  const server = await createServer({ server: { middlewareMode: true }, appType: 'custom' });
  try { console.log('BRAND_HTML:' + JSON.stringify(await server.transformIndexHtml('/', readFileSync('index.html', 'utf8')))); }
  finally { await server.close(); }
 `,
    ],
    { encoding: 'utf8', timeout: 20000 }
  )
  const line = output.split('\n').find((line) => line.startsWith('BRAND_HTML:'))
  if (!line) throw new Error('Vite HTML transformation missing')
  const doc = new DOMParser().parseFromString(
    JSON.parse(line.slice('BRAND_HTML:'.length)),
    'text/html'
  )
  expect(doc.title).toBe('VL Intelligent - AI Trading System')
}, 30000)

it('renders the Guide heading and product card', () => {
  vi.stubGlobal('fetch', vi.fn().mockReturnValue(new Promise(() => {})))
  render(<GuidePage />)
  expect(
    screen.queryByRole('heading', { name: 'VL Intelligent System Guide' })
  ).not.toBeNull()
  expect(screen.getByText('VL Intelligent')).toBeTruthy()
  expect(screen.queryByText('NOFX / VL')).toBeNull()
})
