'use client'

import { useState, useEffect, useMemo } from 'react'
import { useParams, usePathname } from 'next/navigation'
import Link from 'next/link'
import { Container, HStack, Text, Box } from '@chakra-ui/react'
import { api } from '@/lib/api'
import { useI18n } from '@/contexts/I18nContext'
import { useCurrentUser } from '@/contexts/UserContext'
import { deriveMyRole, hasRoleAtLeast, ProjectRoleOrEmpty } from '@/lib/projectRole'
import ProjectTabs from './components/ProjectTabs'
import { ProjectContext, getCurrentTab } from './ProjectContext'

// 项目 layout 的 client component（从原 layout.tsx 抽取）
//
// 负责:
//   - 通过 api 获取项目信息和成员列表
//   - 派生当前用户的角色与权限
//   - 用 ProjectContext.Provider 包裹子页面,共享项目数据
//   - 渲染面包屑和 ProjectTabs
//
// server layout([slug]/layout.tsx) 只负责 generateMetadata(动态 title),
// 实际 UI 和交互逻辑都在这里。
export default function ProjectLayoutClient({ children }: { children: React.ReactNode }) {
  const params = useParams()
  const pathname = usePathname()
  const projectSlug = params.slug as string
  const { t } = useI18n()
  const { user } = useCurrentUser()
  const [project, setProject] = useState<any>(null)
  const [members, setMembers] = useState<any[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    if (!user || !projectSlug) return
    Promise.all([
      api.getProject(projectSlug).then(setProject).catch(() => null),
      api.getProjectMembers(projectSlug).then((data) => setMembers(data || [])).catch(() => {}),
    ]).finally(() => setLoading(false))
  }, [user, projectSlug])

  // 派生当前用户的角色与权限（与后端 RequireRole 对齐）
  const myRole = useMemo<ProjectRoleOrEmpty>(
    () => deriveMyRole(user, project, members),
    [user, project, members],
  )
  const canEditScrum = useMemo(() => hasRoleAtLeast(myRole, 'member'), [myRole])
  const canManageProject = useMemo(() => hasRoleAtLeast(myRole, 'admin'), [myRole])
  const isProjectOwner = myRole === 'owner'

  return (
    <ProjectContext.Provider
      value={{
        project,
        loading,
        setProject,
        members,
        setMembers,
        myRole,
        canEditScrum,
        canManageProject,
        isProjectOwner,
      }}
    >
      <Container maxW="container.xl" py={8}>
        <HStack spacing={2} mb={4} fontSize="sm" color="gray.600">
          <Link href="/pms" style={{ textDecoration: 'none' }}>
            <Text _hover={{ color: 'blue.500' }}>{t('nav.pms')}</Text>
          </Link>
          <Text>/</Text>
          <Text color="gray.900" fontWeight="medium">{project?.name || projectSlug}</Text>
        </HStack>
        <ProjectTabs projectSlug={projectSlug} current={getCurrentTab(pathname, projectSlug)} mt={2} />
        <Box mt={6}>
          {children}
        </Box>
      </Container>
    </ProjectContext.Provider>
  )
}
