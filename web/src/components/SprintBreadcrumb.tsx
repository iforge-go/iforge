'use client'

import { Flex, Heading, Text, Badge } from '@chakra-ui/react'
import { FiRepeat } from 'react-icons/fi'
import NextLink from 'next/link'
import { useI18n } from '@/contexts/I18nContext'

const getStatusColor = (status: string) => {
  switch (status) {
    case 'open': return 'gray.500'
    case 'active': return 'green.500'
    case 'closed': return 'red.500'
    default: return 'gray.500'
  }
}

interface SprintBreadcrumbProps {
  sprint: any
  /** 若提供，则把 Sprint 标题渲染为可点击链接（用于非 Sprint 详情页，如看板页面返回入口）；不传则为纯文本 */
  href?: string
}

export default function SprintBreadcrumb({ sprint, href }: SprintBreadcrumbProps) {
  const { t } = useI18n()

  const getSprintStatusLabel = (status: string) => {
    switch (status) {
      case 'open': return t('pms.statusOpen')
      case 'active': return t('pms.statusActive')
      case 'closed': return t('pms.statusClosed')
      default: return status
    }
  }

  return (
    <Flex align="center" gap={3}>
      <FiRepeat color="#3182ce" />
      {href ? (
        // 看板等非详情页：标题做成链接，悬停下划线提示可点击跳回 Sprint 详情
        <NextLink href={href} style={{ textDecoration: 'none' }}>
          <Heading size="md" _hover={{ color: 'primary.600', textDecoration: 'underline' }}>
            {sprint.title}
          </Heading>
        </NextLink>
      ) : (
        <Heading size="md">{sprint.title}</Heading>
      )}
      <Text fontSize="sm" color="gray.500">
        {sprint.startDate || sprint.endDate
          ? `${sprint.startDate ? new Date(sprint.startDate).toLocaleDateString() : '?'} - ${sprint.endDate ? new Date(sprint.endDate).toLocaleDateString() : '?'}`
          : t('pms.noDatesSet')}
      </Text>
      <Badge borderWidth="1px" borderStyle="solid" bg="transparent" color={getStatusColor(sprint.status)} borderColor={getStatusColor(sprint.status)} fontSize="sm">
        {getSprintStatusLabel(sprint.status)}
      </Badge>
    </Flex>
  )
}
