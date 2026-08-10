'use client'

import { Box, VStack, Flex, Text, Progress, IconButton, HStack } from '@chakra-ui/react'
import NextLink from 'next/link'
import { FiSettings, FiChevronLeft, FiChevronRight, FiPlus, FiExternalLink } from 'react-icons/fi'
import { useI18n } from '@/contexts/I18nContext'
import { getEpicColor } from '@/lib/epicColor'

export interface EpicSidebarProps {
  epics: any[]                                  // 含 totalStories/doneStories/progressPercent
  selectedEpicFilter: string                    // 'all' | epicSlug | '__unassigned__'
  onSelect: (filter: string) => void
  totalStoriesCount: number                     // 当前可见 Story 总数(基于搜索/过滤前)
  unassignedCount: number                       // 未分配 Epic 的 Story 数
  projectSlug: string                           // 用于"管理 Epic"链接
  onCollapse?: () => void                       // 收起 Epic 面板
  collapsed?: boolean                           // 当前是否收起
  onExpand?: () => void                         // 展开 Epic 面板
  onCreateEpic?: () => void                     // 新建 Epic（打开 EpicDrawer）
  canEditScrum?: boolean                        // 是否有写权限
}

// 左侧 Epic 面板(参考 Jira Backlog 左侧 Epic Panel)
// 功能:单选过滤 + 进度展示 + 可折叠 + 管理 Epic 入口 + 新建 Epic + 点击进 Epic 详情
// 收起态:Jira 式窄竖条(竖排 EPIC + 展开箭头,始终可见可点)
export default function EpicSidebar({
  epics,
  selectedEpicFilter,
  onSelect,
  totalStoriesCount,
  unassignedCount,
  projectSlug,
  onCollapse,
  collapsed,
  onExpand,
  onCreateEpic,
  canEditScrum,
}: EpicSidebarProps) {
  const { t } = useI18n()

  // 收起态:Jira 式窄竖条(竖排 EPIC 文字 + 展开箭头,整条可点击展开)
  if (collapsed) {
    return (
      <Box
        w="40px"
        flexShrink={0}
        bg="white"
        borderRadius="lg"
        border="1px solid #DFE2EA"
        overflow="hidden"
        h="fit-content"
        maxH="calc(100vh - 200px)"
      >
        <Flex
          direction="column"
          align="center"
          py={3}
          cursor="pointer"
          onClick={onExpand}
          _hover={{ bg: 'gray.50' }}
          minH="120px"
        >
          <IconButton
            aria-label={t('pms.expandEpicPanel')}
            icon={<FiChevronRight />}
            size="xs"
            variant="ghost"
            color="gray.400"
            _hover={{ color: 'blue.500' }}
            mb={2}
          />
          <Text
            fontSize="sm"
            fontWeight="bold"
            color="gray.700"
            sx={{ writingMode: 'vertical-rl', textOrientation: 'mixed', letterSpacing: '0.05em' }}
          >
            {t('pms.epic')}
          </Text>
        </Flex>
      </Box>
    )
  }

  // 展开态:完整面板
  return (
    <Box
      w="220px"
      flexShrink={0}
      bg="white"
      borderRadius="lg"
      border="1px solid #DFE2EA"
      overflow="hidden"
      h="fit-content"
      maxH="calc(100vh - 200px)"
      overflowY="auto"
    >
      {/* 面板标题 + 收起按钮 + 新建 Epic + 管理 Epic 链接 */}
      <Flex px={3} py={2} bg="gray.50" borderBottom="1px solid #F4F6F8" align="center" justify="space-between">
        <HStack spacing={1}>
          {onCollapse && (
            <IconButton
              aria-label={t('pms.collapseEpicPanel')}
              icon={<FiChevronLeft />}
              size="xs"
              variant="ghost"
              color="gray.400"
              _hover={{ color: 'blue.500' }}
              onClick={onCollapse}
            />
          )}
          <Text fontSize="sm" fontWeight="bold" color="gray.700">{t('pms.epic')}</Text>
        </HStack>
        <HStack spacing={1}>
          {canEditScrum && onCreateEpic && (
            <IconButton
              aria-label={t('pms.newEpic')}
              icon={<FiPlus size={14} />}
              size="xs"
              variant="ghost"
              color="gray.400"
              _hover={{ color: 'blue.500' }}
              onClick={onCreateEpic}
            />
          )}
          <IconButton
            as={NextLink}
            href={`/projects/${projectSlug}/epics`}
            aria-label={t('pms.manageEpics')}
            icon={<FiSettings size={14} />}
            size="xs"
            variant="ghost"
            color="gray.400"
            _hover={{ color: 'blue.500' }}
          />
        </HStack>
      </Flex>

      <VStack align="stretch" spacing={0}>
        {/* "全部 Story" 选项 */}
        <EpicFilterRow
          active={selectedEpicFilter === 'all'}
          onClick={() => onSelect('all')}
          colorElement={<Box w="8px" h="8px" borderRadius="full" bg="gray.400" />}
          label={t('pms.allStories')}
          count={String(totalStoriesCount)}
        />

        {/* 每个 Epic 一行(可点击过滤 + 右侧外链图标进 Epic 详情) */}
        {epics.map(epic => {
          const color = getEpicColor(epic.slug)
          const done = epic.doneStories || 0
          const total = epic.totalStories || 0
          const progress = total > 0 ? (done / total) * 100 : 0
          return (
            <EpicFilterRow
              key={epic.slug}
              active={selectedEpicFilter === epic.slug}
              onClick={() => onSelect(epic.slug)}
              colorElement={<Box w="3px" h="20px" bg={color} borderRadius="full" />}
              label={epic.title}
              count={`${done}/${total}`}
              progress={progress}
              progressColor={color}
              trailingElement={
                <IconButton
                  as={NextLink}
                  href={`/projects/${projectSlug}/epics/${epic.slug}`}
                  aria-label={t('pms.viewEpicDetails')}
                  icon={<FiExternalLink size={12} />}
                  size="xs"
                  variant="ghost"
                  color="gray.300"
                  _hover={{ color: 'blue.500' }}
                  onClick={(e: React.MouseEvent) => e.stopPropagation()}
                  flexShrink={0}
                />
              }
            />
          )
        })}

        {/* "未分配 Epic" 选项 */}
        <EpicFilterRow
          active={selectedEpicFilter === '__unassigned__'}
          onClick={() => onSelect('__unassigned__')}
          colorElement={<Box w="8px" h="8px" borderRadius="full" bg="gray.300" borderWidth="1px" borderColor="gray.400" />}
          label={t('pms.noEpic')}
          count={String(unassignedCount)}
        />
      </VStack>
    </Box>
  )
}

// 单行 Epic 过滤选项
function EpicFilterRow({
  active,
  onClick,
  colorElement,
  label,
  count,
  progress,
  progressColor,
  trailingElement,
}: {
  active: boolean
  onClick: () => void
  colorElement: React.ReactNode
  label: string
  count: string
  progress?: number
  progressColor?: string
  trailingElement?: React.ReactNode
}) {
  return (
    <Flex
      direction="column"
      px={3}
      py={2}
      cursor="pointer"
      bg={active ? 'blue.50' : 'white'}
      borderBottom="1px solid #F4F6F8"
      _hover={{ bg: active ? 'blue.50' : 'gray.50' }}
      onClick={onClick}
      position="relative"
    >
      {active && (
        <Box position="absolute" left={0} top={0} bottom={0} w="3px" bg="blue.500" />
      )}
      <Flex align="center">
        {colorElement}
        <Text flex={1} fontSize="sm" color="gray.700" ml={2} noOfLines={1} fontWeight={active ? 'semibold' : 'normal'}>
          {label}
        </Text>
        {trailingElement}
        <Text fontSize="xs" color="gray.400" ml={2} flexShrink={0}>
          {count}
        </Text>
      </Flex>
      {progress !== undefined && (
        <Progress
          value={progress}
          size="xs"
          colorScheme="purple"
          mt={1}
          borderRadius="full"
          sx={{ '& > div': { backgroundColor: progressColor } }}
        />
      )}
    </Flex>
  )
}
