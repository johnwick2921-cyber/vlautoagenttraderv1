import productName from '../../../branding/product.txt?raw'
import personaName from '../../../branding/persona.txt?raw'

export const PRODUCT_NAME = productName
export const PERSONA_NAME = personaName

// Project branding constants.
//
// This is a personal fork; original NoFx-era external links (twitter / telegram
// / community github) have been removed because they're not relevant to this
// project. Add your own URLs here if you want the header social links to
// reappear; otherwise the HeaderBar renders without them.

export const OFFICIAL_LINKS = {
  twitter: '',
  telegram: '',
  github: '',
} as const

export const BRAND_INFO = {
  name: PERSONA_NAME,
  tagline: 'AI Trading Dashboard',
  version: '1.0.0',
} as const
