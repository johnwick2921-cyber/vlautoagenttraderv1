// A permanent, unmissable marker so a sandbox instance can never be confused with
// the live system. Renders ONLY when GET /api/config reports sandbox:true.

import { useSystemConfig } from '../../hooks/useSystemConfig'

export function SandboxBanner() {
  const { config } = useSystemConfig()
  if (!config?.sandbox) return null
  return (
    <div
      data-testid="sandbox-banner"
      style={{
        position: 'sticky',
        top: 0,
        zIndex: 100,
        background:
          'repeating-linear-gradient(45deg,#3a2b00,#3a2b00 12px,#2b2000 12px,#2b2000 24px)',
        color: '#F5C542',
        borderBottom: '1px solid rgba(245,197,66,.45)',
        font: '600 11px/1.6 Inter, system-ui, sans-serif',
        letterSpacing: '.14em',
        textAlign: 'center',
        padding: '5px 10px',
      }}
    >
      🧪 SANDBOX — isolated test copy · not live · no real orders
    </div>
  )
}
