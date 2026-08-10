'use client'

import { IconButton, Menu, MenuButton, MenuItem, MenuList } from '@chakra-ui/react'
import { FiGlobe, FiCheck } from 'react-icons/fi'
import { useI18n } from '@/contexts/I18nContext'

export function LanguageSwitcher() {
  const { t, locale, setLocale } = useI18n()

  return (
    <Menu>
      <MenuButton
        as={IconButton}
        aria-label={t('header.switchLanguage')}
        icon={<FiGlobe />}
        size="sm"
        variant="ghost"
      />
      <MenuList>
        <MenuItem onClick={() => setLocale('zh')} justifyContent="space-between">
          {t('header.chinese')}
          {locale === 'zh' && <FiCheck />}
        </MenuItem>
        <MenuItem onClick={() => setLocale('en')} justifyContent="space-between">
          {t('header.english')}
          {locale === 'en' && <FiCheck />}
        </MenuItem>
      </MenuList>
    </Menu>
  )
}
