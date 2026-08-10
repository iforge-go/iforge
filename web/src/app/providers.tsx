'use client'

import { ChakraProvider, createLocalStorageManager, useToast, UseToastOptions, ToastId } from '@chakra-ui/react'
import { theme } from '@/theme'
import { createContext, useContext } from 'react'
import { I18nProvider } from '@/contexts/I18nContext'
import { UserProvider } from '@/contexts/UserContext'
import { WebSocketProvider } from '@/contexts/WebSocketContext'
import { SiteSettingsProvider } from '@/contexts/SiteSettingsContext'

const colorModeManager = createLocalStorageManager('iforge-color-mode')

// 定义 toast 类型
type GithubToast = (options?: UseToastOptions) => ToastId

const ToastContext = createContext<GithubToast | null>(null)

export function ToastProvider({ children }: { children: React.ReactNode }) {
  const toast = useToast()
  
  const githubToast: GithubToast = (options) => {
    return toast({
      position: 'top',
      duration: 3000,
      isClosable: true,
      ...options,
    })
  }
  
  return (
    <ToastContext.Provider value={githubToast}>
      {children}
    </ToastContext.Provider>
  )
}

export function useGithubToast() {
  const context = useContext(ToastContext)
  if (!context) {
    throw new Error('useGithubToast must be used within ToastProvider')
  }
  return context
}

export function Providers({ children }: { children: React.ReactNode }) {
  return (
    <ChakraProvider theme={theme} colorModeManager={colorModeManager}>
      <SiteSettingsProvider>
        <I18nProvider>
          <UserProvider>
            <WebSocketProvider>
              <ToastProvider>
                {children}
              </ToastProvider>
            </WebSocketProvider>
          </UserProvider>
        </I18nProvider>
      </SiteSettingsProvider>
    </ChakraProvider>
  )
}
