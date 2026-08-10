'use client'

import {
  FormControl,
  FormLabel,
  FormErrorMessage,
  FormHelperText,
  Input,
  InputGroup,
  InputRightElement,
  Spinner,
  Icon,
} from '@chakra-ui/react'
import { useEffect, useRef } from 'react'
import { FiCheckCircle, FiXCircle } from 'react-icons/fi'
import { useI18n } from '@/contexts/I18nContext'
import { useUsernameCheck, UsernameStatus } from '@/hooks/useUsernameCheck'

interface UsernameInputProps {
  value: string
  onChange: (v: string) => void
  /** 校验状态变化回调，父组件据此在提交前判断是否 available */
  onStatusChange?: (status: UsernameStatus) => void
  placeholder?: string
  label?: string
  isRequired?: boolean
  autoComplete?: string
}

/**
 * 用户名输入框 + 实时可用性校验（共享组件）。
 * 复用于注册页 / 初始化向导 / 管理员新建用户。
 * - 可用：右侧绿色对勾 + "用户名可用"
 * - 校验中：右侧转圈
 * - 不可用（格式/保留词/已占用）：右侧红色叉 + 输入框下错误提示
 */
export function UsernameInput({
  value,
  onChange,
  onStatusChange,
  placeholder,
  label,
  isRequired,
  autoComplete,
}: UsernameInputProps) {
  const { t } = useI18n()
  const status = useUsernameCheck(value)

  // 用 ref 回调，避免 onStatusChange 不稳定导致父组件 re-render 循环
  const onStatusChangeRef = useRef(onStatusChange)
  onStatusChangeRef.current = onStatusChange
  useEffect(() => {
    onStatusChangeRef.current?.(status)
  }, [status])

  const showError =
    status === 'invalid' || status === 'reserved' || status === 'taken'
  const errorMessage =
    status === 'invalid'
      ? t('auth.usernameInvalid')
      : status === 'reserved'
        ? t('errors.usernameReserved')
        : status === 'taken'
          ? t('auth.usernameTaken')
          : ''

  return (
    <FormControl isRequired={isRequired} isInvalid={showError}>
      {label && <FormLabel color="myGray.700">{label}</FormLabel>}
      <InputGroup>
        <Input
          type="text"
          value={value}
          onChange={(e) => onChange(e.target.value)}
          placeholder={placeholder}
          autoComplete={autoComplete}
          pr={status !== 'idle' ? '2.5rem' : undefined}
        />
        {status === 'checking' && (
          <InputRightElement>
            <Spinner size="sm" color="myGray.400" />
          </InputRightElement>
        )}
        {status === 'available' && (
          <InputRightElement>
            <Icon as={FiCheckCircle} color="green.500" />
          </InputRightElement>
        )}
        {showError && (
          <InputRightElement>
            <Icon as={FiXCircle} color="red.500" />
          </InputRightElement>
        )}
      </InputGroup>
      {showError ? (
        <FormErrorMessage>{errorMessage}</FormErrorMessage>
      ) : status === 'available' ? (
        <FormHelperText color="green.500">{t('auth.usernameAvailable')}</FormHelperText>
      ) : (
        <FormHelperText color="myGray.500">{t('auth.usernameCheckHint')}</FormHelperText>
      )}
    </FormControl>
  )
}
