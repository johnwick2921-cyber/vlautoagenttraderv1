import { ConfirmDialogProvider } from './components/common/ConfirmDialog'
import { AuthProvider } from './contexts/AuthContext'
import { LanguageProvider } from './contexts/LanguageContext'
import { AppRoutes } from './router/AppRoutes'
import { SandboxBanner } from './components/common/SandboxBanner'

export default function App() {
  return (
    <LanguageProvider>
      <AuthProvider>
        <ConfirmDialogProvider>
          <SandboxBanner />
          <AppRoutes />
        </ConfirmDialogProvider>
      </AuthProvider>
    </LanguageProvider>
  )
}
