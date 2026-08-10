'use client'

import {
  Text,
  VStack,
  HStack,
  Box,
  Badge,
  Avatar,
  Button,
  Menu,
  MenuButton,
  MenuList,
  MenuItem,
  Icon,
  Input,
  Popover,
  PopoverTrigger,
  PopoverContent,
  PopoverBody,
} from '@chakra-ui/react'
import Link from 'next/link'
import { FiPlus, FiX, FiFlag, FiLock, FiUnlock } from 'react-icons/fi'
import { useI18n } from '@/contexts/I18nContext'
import { Issue, Label, Milestone, User, UserSearchResult, Participant } from '@/lib/api'

// 判断颜色亮度，决定文字颜色
function getTextColor(bgColor: string): string {
  const hex = bgColor.replace('#', '')
  const r = parseInt(hex.substring(0, 2), 16)
  const g = parseInt(hex.substring(2, 4), 16)
  const b = parseInt(hex.substring(4, 6), 16)
  const brightness = (r * 299 + g * 587 + b * 114) / 1000
  return brightness > 128 ? 'myGray.800' : 'white'
}

interface IssueSidebarProps {
  issue: Issue
  owner: string
  repoName: string
  assignees: User[]
  allLabels: Label[]
  issueLabels: Label[]
  milestones: Milestone[]
  currentMilestone: Milestone | null
  allParticipants: Participant[]
  canManageIssue: boolean
  canLockIssue: boolean
  dateLocale: string
  lockLoading: boolean
  onLockAction: (action: 'lock' | 'unlock') => void
  // Assignee
  userSearchKeyword: string
  onUserSearchKeywordChange: (value: string) => void
  userSearchResults: UserSearchResult[]
  userSearchLoading: boolean
  isAssigneePopoverOpen: boolean
  onAssigneePopoverOpenChange: (open: boolean) => void
  assigneeLoading: boolean
  onAddAssignee: (username: string) => void
  onRemoveAssignee: (username: string) => void
  // Label
  onAddLabel: (labelId: number) => void
  onRemoveLabel: (labelId: number) => void
  // Milestone
  onSetMilestone: (milestoneId: number | null) => void
}

export function IssueSidebar({
  issue,
  owner,
  repoName,
  assignees,
  allLabels,
  issueLabels,
  milestones,
  currentMilestone,
  allParticipants,
  canManageIssue,
  canLockIssue,
  dateLocale,
  lockLoading,
  onLockAction,
  userSearchKeyword,
  onUserSearchKeywordChange,
  userSearchResults,
  userSearchLoading,
  isAssigneePopoverOpen,
  onAssigneePopoverOpenChange,
  assigneeLoading,
  onAddAssignee,
  onRemoveAssignee,
  onAddLabel,
  onRemoveLabel,
  onSetMilestone,
}: IssueSidebarProps) {
  const { t } = useI18n()

  return (
    <VStack w="250px" align="stretch" spacing={4}>
      {/* 指派人员 */}
      <Box borderWidth="1px" borderRadius="lg" overflow="hidden" bg="white">
        <Box p={4}>
          <VStack spacing={3} align="stretch">
            <HStack justify="space-between">
              <Text fontWeight="semibold" fontSize="sm">{t('repo.assignees')}</Text>
              {canManageIssue && (
                <Popover placement="bottom-end" isOpen={isAssigneePopoverOpen} onOpen={() => onAssigneePopoverOpenChange(true)} onClose={() => onAssigneePopoverOpenChange(false)}>
                  <PopoverTrigger>
                    <Button
                      size="xs"
                      variant="ghost"
                      leftIcon={<Icon as={FiPlus} />}
                      isLoading={assigneeLoading}
                    >
                      {t('repo.add')}
                    </Button>
                  </PopoverTrigger>
                  <PopoverContent maxW="180px" w="180px">
                    <PopoverBody p={2}>
                      <VStack spacing={2} align="stretch">
                        <Input
                          size="sm"
                          placeholder={t('repo.searchUsers')}
                          value={userSearchKeyword}
                          onChange={(e) => onUserSearchKeywordChange(e.target.value)}
                        />
                        {userSearchLoading && (
                          <Text fontSize="xs" color="myGray.500">{t('common.loading')}</Text>
                        )}
                        {!userSearchLoading && userSearchKeyword.trim() && (
                          <VStack spacing={0} align="stretch" maxH="240px" overflowY="auto">
                            {userSearchResults
                              .filter(u => !assignees.find(a => a.userName === u.userName))
                              .map(user => (
                                <Button
                                  key={user.userName}
                                  variant="ghost"
                                  justifyContent="flex-start"
                                  size="sm"
                                  onClick={() => onAddAssignee(user.userName)}
                                >
                                  <HStack spacing={2}>
                                    <Avatar size="xs" name={user.fullName || user.userName} src={user.image || undefined} />
                                    <VStack spacing={0} align="start">
                                      <Text fontSize="sm" fontWeight="medium">{user.userName}</Text>
                                      {user.fullName && (
                                        <Text fontSize="xs" color="myGray.500">{user.fullName}</Text>
                                      )}
                                    </VStack>
                                  </HStack>
                                </Button>
                              ))}
                            {userSearchResults
                              .filter(u => !assignees.find(a => a.userName === u.userName))
                              .length === 0 && (
                              <Text fontSize="xs" color="myGray.500" p={2}>
                                {t('repo.noAssignees')}
                              </Text>
                            )}
                          </VStack>
                        )}
                      </VStack>
                    </PopoverBody>
                  </PopoverContent>
                </Popover>
              )}
            </HStack>
            {assignees.length === 0 ? (
              <Text fontSize="sm" color="myGray.500">{t('repo.noAssignees')}</Text>
            ) : (
              <VStack spacing={2} align="stretch">
                {assignees.map(assignee => (
                  <HStack key={assignee.userName} justify="space-between">
                    <HStack spacing={2} flex={1} minW={0}>
                      <Avatar size="xs" name={assignee.fullName || assignee.userName} src={assignee.image || undefined} />
                      <Text fontSize="sm" color="myGray.900" noOfLines={1}>{assignee.userName}</Text>
                    </HStack>
                    {canManageIssue && (
                    <Button
                      size="xs"
                      variant="ghost"
                      onClick={() => onRemoveAssignee(assignee.userName)}
                    >
                      <Icon as={FiX} />
                    </Button>
                    )}
                  </HStack>
                ))}
              </VStack>
            )}
          </VStack>
        </Box>
      </Box>

      {/* 标签 */}
      <Box borderWidth="1px" borderRadius="lg" overflow="hidden" bg="white">
        <Box p={4}>
          <VStack spacing={3} align="stretch">
            <HStack justify="space-between">
              <Text fontWeight="semibold" fontSize="sm">{t('repo.labelsSection')}</Text>
              {canManageIssue && (
              <Menu>
                <MenuButton
                  as={Button}
                  size="xs"
                  variant="ghost"
                  leftIcon={<Icon as={FiPlus} />}
                >
                  {t('repo.add')}
                </MenuButton>
                <MenuList>
                  {allLabels.length === 0 ? (
                    <MenuItem isDisabled>{t('repo.noLabels')}</MenuItem>
                  ) : (
                    allLabels
                      .filter(label => !issueLabels.find(l => l.labelId === label.labelId))
                      .map(label => (
                        <MenuItem
                          key={label.labelId}
                          onClick={() => onAddLabel(label.labelId)}
                        >
                          <HStack spacing={2}>
                            <Badge
                              bg={`#${label.color}`}
                              color={getTextColor(label.color)}
                              fontSize="xs"
                              px={2}
                              py={0.5}
                              borderRadius="md"
                            >
                              {label.labelName}
                            </Badge>
                          </HStack>
                        </MenuItem>
                      ))
                  )}
                </MenuList>
              </Menu>
              )}
            </HStack>
            {issueLabels.length === 0 ? (
              <Text fontSize="sm" color="myGray.500">{t('repo.noLabels')}</Text>
            ) : (
              <VStack spacing={2} align="stretch">
                {issueLabels.map(label => (
                  <HStack key={label.labelId} justify="space-between">
                    <Badge
                      bg={`#${label.color}`}
                      color={getTextColor(label.color)}
                      fontSize="xs"
                      px={2}
                      py={1}
                      borderRadius="md"
                      flex={1}
                    >
                      {label.labelName}
                    </Badge>
                    {canManageIssue && (
                    <Button
                      size="xs"
                      variant="ghost"
                      onClick={() => onRemoveLabel(label.labelId)}
                    >
                      <Icon as={FiX} />
                    </Button>
                    )}
                  </HStack>
                ))}
              </VStack>
            )}
          </VStack>
        </Box>
      </Box>

      {/* 里程碑 */}
      <Box borderWidth="1px" borderRadius="lg" overflow="hidden" bg="white">
        <Box p={4}>
          <VStack spacing={3} align="stretch">
            <HStack justify="space-between">
              <Text fontWeight="semibold" fontSize="sm">{t('repo.milestone')}</Text>
              {canManageIssue && (
              <Menu>
                <MenuButton
                  as={Button}
                  size="xs"
                  variant="ghost"
                  leftIcon={<Icon as={FiFlag} />}
                >
                  {t('repo.set')}
                </MenuButton>
                <MenuList>
                  {milestones.length === 0 ? (
                    <MenuItem isDisabled>{t('repo.noMilestones')}</MenuItem>
                  ) : (
                    <>
                      {milestones
                        .filter(m => !m.closedDate)
                        .map(milestone => (
                          <MenuItem
                            key={milestone.milestoneId}
                            onClick={() => onSetMilestone(milestone.milestoneId)}
                          >
                            <Text fontSize="sm">{milestone.title}</Text>
                          </MenuItem>
                        ))}
                      {currentMilestone && (
                        <MenuItem
                          onClick={() => onSetMilestone(null)}
                        >
                          <Text fontSize="sm" color="myGray.500">{t('repo.clearMilestone')}</Text>
                        </MenuItem>
                      )}
                    </>
                  )}
                </MenuList>
              </Menu>
              )}
            </HStack>
            {currentMilestone ? (
              <VStack spacing={2} align="stretch">
                <Link href={`/${owner}/${repoName}/milestones/${currentMilestone.milestoneId}`}>
                  <Text fontSize="sm" color="primary.600" fontWeight="medium" _hover={{ textDecoration: 'underline' }}>
                    {currentMilestone.title}
                  </Text>
                </Link>
                {currentMilestone.dueDate && (
                  <Text fontSize="xs" color="myGray.500">
                    {t('repo.dueDate', { date: new Date(currentMilestone.dueDate).toLocaleDateString(dateLocale) })}
                  </Text>
                )}
              </VStack>
            ) : (
              <Text fontSize="sm" color="myGray.500">{t('repo.noMilestones')}</Text>
            )}
          </VStack>
        </Box>
      </Box>

      {/* 参与者 */}
      <Box borderWidth="1px" borderRadius="lg" overflow="hidden" bg="white" borderColor="myGray.200">
        <Box p={4}>
          <Text fontWeight="semibold" fontSize="sm" mb={3}>{t('repo.participants')}</Text>
          <VStack spacing={2} align="stretch">
            {allParticipants.map(p => (
              <HStack key={p.userName} spacing={2}>
                <Avatar size="xs" name={p.fullName || p.userName} src={p.image || undefined} />
                <Text fontSize="sm" color="myGray.900">{p.userName}</Text>
              </HStack>
            ))}
          </VStack>
        </Box>
      </Box>

      {/* 锁定议题 */}
      {canLockIssue && (
        <Box borderWidth="1px" borderRadius="lg" overflow="hidden" bg="white">
          <Box p={4}>
            <Button
              leftIcon={<Icon as={issue.locked ? FiUnlock : FiLock} />}
              variant={issue.locked ? 'outline' : 'ghost'}
              size="sm"
              colorScheme={issue.locked ? 'green' : 'red'}
              onClick={() => onLockAction(issue.locked ? 'unlock' : 'lock')}
              isLoading={lockLoading}
              w="full"
            >
              {issue.locked ? t('repo.unlockIssue') : t('repo.lockIssue')}
            </Button>
          </Box>
        </Box>
      )}
    </VStack>
  )
}
