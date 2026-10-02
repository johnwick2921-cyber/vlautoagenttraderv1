import {
  VL_BEGINNER_ONBOARDING_COMPLETED_KEY,
  VL_BEGINNER_WALLET_ADDRESS_KEY,
  VL_USER_MODE_KEY,
} from './storageMigration'

export type UserMode = 'beginner' | 'advanced'

const USER_MODE_KEY = VL_USER_MODE_KEY
const BEGINNER_WALLET_ADDRESS_KEY = VL_BEGINNER_WALLET_ADDRESS_KEY
const BEGINNER_ONBOARDING_COMPLETED_KEY = VL_BEGINNER_ONBOARDING_COMPLETED_KEY

export function getUserMode(): UserMode | null {
  const value = localStorage.getItem(USER_MODE_KEY)
  if (value === 'beginner' || value === 'advanced') {
    return value
  }
  return null
}

export function setUserMode(mode: UserMode) {
  localStorage.setItem(USER_MODE_KEY, mode)
}

export function getPostAuthPath(mode: UserMode | null | undefined): string {
  return mode === 'beginner' ? '/welcome' : '/traders'
}

export function setBeginnerWalletAddress(address: string) {
  localStorage.setItem(BEGINNER_WALLET_ADDRESS_KEY, address)
}

export function getBeginnerWalletAddress(): string | null {
  return localStorage.getItem(BEGINNER_WALLET_ADDRESS_KEY)
}

export function hasCompletedBeginnerOnboarding(): boolean {
  return localStorage.getItem(BEGINNER_ONBOARDING_COMPLETED_KEY) === 'true'
}

export function markBeginnerOnboardingCompleted() {
  localStorage.setItem(BEGINNER_ONBOARDING_COMPLETED_KEY, 'true')
}
