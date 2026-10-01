import { readFileSync } from 'node:fs'
import { describe, expect, it } from 'vitest'

// D2 item 11 (Z28): a custom property declared in BOTH index.css and
// vl-tokens.css is a silent cascade conflict — the later :root rule of equal
// specificity wins and the design-system token changes meaning. Parse the two
// production files and fail on any shared declaration.
function declaredCustomProps(path: string): Set<string> {
  const css = readFileSync(new URL(path, import.meta.url), 'utf8')
  const props = new Set<string>()
  const re = /--[a-z0-9-]+\s*:/g
  let m: RegExpExecArray | null
  while ((m = re.exec(css)) !== null) {
    props.add(m[0].replace(/\s*:$/, '').trim())
  }
  return props
}

describe('theme custom properties', () => {
  it('no custom property is declared in both index.css and vl-tokens.css', () => {
    const index = declaredCustomProps('../index.css')
    const tokens = declaredCustomProps('./vl-tokens.css')
    const shared = [...index].filter((p) => tokens.has(p))
    expect(shared).toEqual([])
  })

  it('the renamed palette uses the vl-neo- prefix, not the bare vl- prefix', () => {
    const index = declaredCustomProps('../index.css')
    for (const name of ['gold', 'bg', 'accent', 'glass', 'border']) {
      expect(index.has(`--vl-neo-${name}`)).toBe(true)
    }
  })
})
