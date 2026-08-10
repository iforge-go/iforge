'use client'

import NextLink from 'next/link'
import { Tabs, TabList, Tab, HStack, Icon, Text } from '@chakra-ui/react'
import { FiGrid, FiCheckSquare, FiColumns, FiInbox, FiBarChart2, FiGitBranch } from 'react-icons/fi'
import { useI18n } from '@/contexts/I18nContext'

export type ProjectTabKey = 'overview' | 'backlog' | 'board' | 'tasks' | 'reports' | 'code' | 'members'

interface ProjectTabsProps {
  projectSlug: string
  current: ProjectTabKey
  mt?: number
}

// 对齐 Jira Scrum 项目 tab 布局：概览 → 需求池 → 看板 → 任务 → 报告 → 代码
// - 需求池含 Sprint 规划 + Epic 管理（Jira Backlog）
// - 看板只显示活跃 Sprint 任务（Jira Board）
// - 任务为全局列表 + 筛选（Jira Issues & filters）
// - 报告聚合速度图 + Sprint 报告（Jira Reports）
// - 代码展示关联仓库（iForge 特色，Jira 需集成）
export default function ProjectTabs({ projectSlug, current, mt }: ProjectTabsProps) {
  const { t } = useI18n()
  const tabs = [
    { key: 'overview' as const, href: `/projects/${projectSlug}`, icon: FiGrid, label: t('pms.overview') },
    { key: 'backlog' as const, href: `/projects/${projectSlug}/backlog`, icon: FiInbox, label: t('pms.backlog') },
    { key: 'board' as const, href: `/projects/${projectSlug}/board`, icon: FiColumns, label: t('pms.board') },
    { key: 'tasks' as const, href: `/projects/${projectSlug}/tasks`, icon: FiCheckSquare, label: t('pms.tasks') },
    { key: 'reports' as const, href: `/projects/${projectSlug}/reports`, icon: FiBarChart2, label: t('pms.reports') },
    { key: 'code' as const, href: `/projects/${projectSlug}/code`, icon: FiGitBranch, label: t('pms.code') },
  ]
  const currentIndex = tabs.findIndex(t => t.key === current)

  return (
    <Tabs index={currentIndex} isManual variant="line" colorScheme="primary" mt={mt}>
      <TabList borderBottomColor="myGray.200">
        {tabs.map(tab => (
          <Tab key={tab.key} as={NextLink} href={tab.href} whiteSpace="nowrap">
            <HStack spacing={2}>
              <Icon as={tab.icon} />
              <Text>{tab.label}</Text>
            </HStack>
          </Tab>
        ))}
      </TabList>
    </Tabs>
  )
}
