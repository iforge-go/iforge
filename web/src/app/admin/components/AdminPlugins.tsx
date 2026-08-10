'use client'

import { useState, useEffect, useRef } from 'react'
import {
  Box,
  VStack,
  HStack,
  Text,
  Icon,
  Button,
  SimpleGrid,
  Badge,
  Modal,
  ModalOverlay,
  ModalContent,
  ModalHeader,
  ModalBody,
  ModalFooter,
  ModalCloseButton,
  useDisclosure,
  FormControl,
  FormLabel,
  Input,
  Textarea,
  Select,
  Switch,
  useToast,
  Tabs,
  TabList,
  TabPanels,
  Tab,
  TabPanel,
  Card,
  CardBody,
  CardHeader,
  Heading,
  IconButton,
  Tooltip,
  AlertDialog,
  AlertDialogBody,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogContent,
  AlertDialogOverlay,
} from '@chakra-ui/react'
import { FiPackage, FiDownload, FiTrash2, FiToggleLeft, FiToggleRight, FiSettings } from 'react-icons/fi'
import { useI18n } from '@/contexts/I18nContext'
import { api } from '@/lib/api'

interface Plugin {
  id: number
  name: string
  version: string
  description: string
  author: string
  enabled: boolean
  type: string
  template_id?: string
  config?: string
  created_at: string
}

interface PluginTemplate {
  id: string
  name: string
  description: string
  version: string
  author: string
  category: string
  icon: string
  tags: string[]
  config_schema: Record<string, any>
}

export default function AdminPlugins() {
  const { t } = useI18n()
  const toast = useToast()
  const { isOpen: isInstallOpen, onOpen: onInstallOpen, onClose: onInstallClose } = useDisclosure()
  const { isOpen: isConfigOpen, onOpen: onConfigOpen, onClose: onConfigClose } = useDisclosure()
  
  const [plugins, setPlugins] = useState<Plugin[]>([])
  const [templates, setTemplates] = useState<PluginTemplate[]>([])
  const [selectedTemplate, setSelectedTemplate] = useState<PluginTemplate | null>(null)
  const [selectedPlugin, setSelectedPlugin] = useState<Plugin | null>(null)
  const [editConfig, setEditConfig] = useState<Record<string, any>>({})
  const [installForm, setInstallForm] = useState({
    name: '',
    config: {} as Record<string, any>,
  })
  const [deletePluginId, setDeletePluginId] = useState<number | null>(null)
  const { isOpen: isDeleteOpen, onOpen: onDeleteOpen, onClose: onDeleteClose } = useDisclosure()
  const cancelRef = useRef<HTMLButtonElement>(null!)

  const loadPlugins = async () => {
    try {
      const data = await api.listPlugins()
      setPlugins(data)
    } catch (error) {
      console.error('Failed to load plugins:', error)
    }
  }

  const loadTemplates = async () => {
    try {
      const data = await api.listPluginTemplates()
      setTemplates(data)
    } catch (error) {
      console.error('Failed to load templates:', error)
    }
  }

  useEffect(() => {
    loadPlugins()
    loadTemplates()
  }, [])

  const handleInstall = async () => {
    if (!selectedTemplate) return

    try {
      await api.installPlugin({
        template_id: selectedTemplate.id,
        name: installForm.name,
        config: installForm.config,
      })

      toast({
        title: t('admin.pluginInstalled'),
        status: 'success',
        duration: 3000,
      })

      onInstallClose()
      loadPlugins()
      setInstallForm({ name: '', config: {} })
      setSelectedTemplate(null)
    } catch (error: any) {
      toast({
        title: t('admin.installFailed'),
        description: error.message,
        status: 'error',
        duration: 3000,
      })
    }
  }

  const handleTogglePlugin = async (plugin: Plugin) => {
    try {
      if (plugin.enabled) {
        await api.disablePlugin(plugin.id)
        toast({
          title: t('admin.pluginDisabledMsg'),
          status: 'info',
          duration: 2000,
        })
      } else {
        await api.enablePlugin(plugin.id)
        toast({
          title: t('admin.pluginEnabledMsg'),
          status: 'success',
          duration: 2000,
        })
      }
      loadPlugins()
    } catch (error: any) {
      toast({
        title: 'Error',
        description: error.message,
        status: 'error',
        duration: 3000,
      })
    }
  }

  const handleUninstall = async (pluginId: number) => {
    setDeletePluginId(pluginId)
    onDeleteOpen()
  }

  const confirmDelete = async () => {
    if (!deletePluginId) return

    try {
      await api.uninstallPlugin(deletePluginId)
      toast({
        title: t('admin.pluginUninstalled'),
        status: 'success',
        duration: 3000,
      })
      loadPlugins()
      onDeleteClose()
    } catch (error: any) {
      toast({
        title: 'Error',
        description: error.message,
        status: 'error',
        duration: 3000,
      })
    }
  }

  const handleSaveConfig = async () => {
    if (!selectedPlugin) return

    try {
      await api.updatePlugin(selectedPlugin.id, { config: editConfig })
      toast({
        title: t('admin.configSaved'),
        status: 'success',
        duration: 3000,
      })
      onConfigClose()
      loadPlugins()
    } catch (error: any) {
      toast({
        title: t('admin.saveConfigFailed'),
        description: error.message,
        status: 'error',
        duration: 3000,
      })
    }
  }

  const openConfigModal = (plugin: Plugin) => {
    setSelectedPlugin(plugin)
    // Parse existing config
    try {
      const config = plugin.config ? JSON.parse(plugin.config) : {}
      setEditConfig(config)
    } catch {
      setEditConfig({})
    }
    onConfigOpen()
  }

  const openInstallModal = (template: PluginTemplate) => {
    setSelectedTemplate(template)
    setInstallForm({
      name: template.name,
      config: Object.fromEntries(
        Object.entries(template.config_schema || {}).map(([key, schema]: [string, any]) => [
          key,
          schema.default || '',
        ])
      ),
    })
    onInstallOpen()
  }

  const renderConfigField = (key: string, schema: any) => {
    const value = installForm.config[key] || ''
    const onChange = (newValue: any) => {
      setInstallForm({
        ...installForm,
        config: { ...installForm.config, [key]: newValue },
      })
    }

    const label = (
      <FormLabel>
        {schema.label || key}
        {schema.required && <Text as="span" color="red.500"> *</Text>}
      </FormLabel>
    )

    switch (schema.type) {
      case 'textarea':
        return (
          <FormControl key={key} isRequired={schema.required}>
            {label}
            <Textarea
              value={value}
              onChange={(e) => onChange(e.target.value)}
              placeholder={schema.placeholder}
            />
            {schema.description && <Text fontSize="sm" color="gray.600" mt={1}>{schema.description}</Text>}
          </FormControl>
        )

      case 'select':
        return (
          <FormControl key={key} isRequired={schema.required}>
            {label}
            <Select
              value={value}
              onChange={(e) => onChange(e.target.value)}
              placeholder={schema.placeholder}
            >
              {schema.options?.map((opt: any) => (
                <option key={opt.value} value={opt.value}>
                  {opt.label}
                </option>
              ))}
            </Select>
            {schema.description && <Text fontSize="sm" color="gray.600" mt={1}>{schema.description}</Text>}
          </FormControl>
        )

      case 'boolean':
        return (
          <FormControl key={key} display="flex" alignItems="center">
            <FormLabel mb="0">{schema.label || key}</FormLabel>
            <Switch
              isChecked={value}
              onChange={(e) => onChange(e.target.checked)}
            />
            {schema.description && <Text fontSize="sm" color="gray.600" ml={2}>{schema.description}</Text>}
          </FormControl>
        )

      case 'number':
        return (
          <FormControl key={key} isRequired={schema.required}>
            {label}
            <Input
              type="number"
              value={value}
              onChange={(e) => onChange(Number(e.target.value))}
              placeholder={schema.placeholder}
            />
            {schema.description && <Text fontSize="sm" color="gray.600" mt={1}>{schema.description}</Text>}
          </FormControl>
        )

      default: // string
        return (
          <FormControl key={key} isRequired={schema.required}>
            {label}
            <Input
              value={value}
              onChange={(e) => onChange(e.target.value)}
              placeholder={schema.placeholder}
            />
            {schema.description && <Text fontSize="sm" color="gray.600" mt={1}>{schema.description}</Text>}
          </FormControl>
        )
    }
  }

  return (
    <Box>
      <Tabs variant="enclosed">
        <TabList>
          <Tab>{t('admin.installedPlugins')}</Tab>
          <Tab>{t('admin.pluginMarket')}</Tab>
        </TabList>

        <TabPanels>
          {/* Installed Plugins Tab */}
          <TabPanel>
            {plugins.length === 0 ? (
              <VStack spacing={4} py={12}>
                <Icon as={FiPackage} w={12} h={12} color="gray.400" />
                <Text color="gray.600">{t('admin.noPlugins')}</Text>
              </VStack>
            ) : (
              <SimpleGrid columns={{ base: 1, md: 2, lg: 3 }} spacing={4}>
                {plugins.map((plugin) => (
                  <Card key={plugin.id}>
                    <CardHeader pb={2}>
                      <HStack justify="space-between">
                        <Heading size="sm">{plugin.name}</Heading>
                        <Badge colorScheme={plugin.enabled ? 'green' : 'gray'}>
                          {plugin.enabled ? t('admin.pluginEnabled') : t('admin.pluginDisabled')}
                        </Badge>
                      </HStack>
                    </CardHeader>
                    <CardBody pt={0}>
                      <VStack align="stretch" spacing={2}>
                        <Text fontSize="sm" color="gray.600" noOfLines={2}>
                          {plugin.description}
                        </Text>
                        <HStack fontSize="xs" color="gray.500">
                          <Text>v{plugin.version}</Text>
                          <Text>•</Text>
                          <Text>{plugin.author}</Text>
                        </HStack>
                        <HStack justify="flex-end" pt={2}>
                          <Tooltip label={plugin.enabled ? t('admin.disablePlugin') : t('admin.enablePlugin')}>
                            <IconButton
                              aria-label="Toggle plugin"
                              icon={plugin.enabled ? <FiToggleRight /> : <FiToggleLeft />}
                              onClick={() => handleTogglePlugin(plugin)}
                              size="sm"
                              colorScheme={plugin.enabled ? 'green' : 'gray'}
                            />
                          </Tooltip>
                          <Tooltip label={t('admin.configurePlugin')}>
                            <IconButton
                              aria-label="Configure plugin"
                              icon={<FiSettings />}
                              onClick={() => openConfigModal(plugin)}
                              size="sm"
                            />
                          </Tooltip>
                          <Tooltip label={t('admin.uninstallPlugin')}>
                            <IconButton
                              aria-label="Uninstall plugin"
                              icon={<FiTrash2 />}
                              onClick={() => handleUninstall(plugin.id)}
                              size="sm"
                              colorScheme="red"
                              variant="ghost"
                            />
                          </Tooltip>
                        </HStack>
                      </VStack>
                    </CardBody>
                  </Card>
                ))}
              </SimpleGrid>
            )}
          </TabPanel>

          {/* Plugin Market Tab */}
          <TabPanel>
            {templates.length === 0 ? (
              <VStack spacing={4} py={12}>
                <Icon as={FiPackage} w={12} h={12} color="gray.400" />
                <Text color="gray.600">{t('admin.noTemplates')}</Text>
              </VStack>
            ) : (
              <SimpleGrid columns={{ base: 1, md: 2, lg: 3 }} spacing={4}>
                {templates.map((template) => (
                  <Card key={template.id}>
                    <CardHeader pb={2}>
                      <HStack>
                        <Icon as={FiPackage} w={6} h={6} color="blue.500" />
                        <Heading size="sm">{template.name}</Heading>
                      </HStack>
                    </CardHeader>
                    <CardBody pt={0}>
                      <VStack align="stretch" spacing={2}>
                        <Text fontSize="sm" color="gray.600" noOfLines={2}>
                          {template.description}
                        </Text>
                        <HStack fontSize="xs" color="gray.500">
                          <Text>v{template.version}</Text>
                          <Text>•</Text>
                          <Text>{template.author}</Text>
                        </HStack>
                        {template.tags && template.tags.length > 0 && (
                          <HStack flexWrap="wrap" spacing={1}>
                            {template.tags.map((tag) => (
                              <Badge key={tag} fontSize="xs" colorScheme="blue">
                                {tag}
                              </Badge>
                            ))}
                          </HStack>
                        )}
                        <Button
                          leftIcon={<FiDownload />}
                          size="sm"
                          colorScheme="blue"
                          onClick={() => openInstallModal(template)}
                          mt={2}
                        >
                          {t('admin.installFromTemplate')}
                        </Button>
                      </VStack>
                    </CardBody>
                  </Card>
                ))}
              </SimpleGrid>
            )}
          </TabPanel>
        </TabPanels>
      </Tabs>

      {/* Install Plugin Modal */}
      <Modal isOpen={isInstallOpen} onClose={onInstallClose} size="lg">
        <ModalOverlay />
        <ModalContent>
          <ModalHeader>{t('admin.installPlugin')}</ModalHeader>
          <ModalCloseButton />
          <ModalBody>
            <VStack spacing={4} align="stretch">
              {selectedTemplate && (
                <>
                  <FormControl isRequired>
                    <FormLabel>{t('admin.pluginName')}</FormLabel>
                    <Input
                      value={installForm.name}
                      onChange={(e) => setInstallForm({ ...installForm, name: e.target.value })}
                      placeholder="Enter plugin name"
                    />
                  </FormControl>

                  {selectedTemplate.config_schema &&
                    Object.entries(selectedTemplate.config_schema).map(([key, schema]) =>
                      renderConfigField(key, schema)
                    )}
                </>
              )}
            </VStack>
          </ModalBody>

          <ModalFooter>
            <Button variant="ghost" mr={3} onClick={onInstallClose}>
              {t('admin.cancel')}
            </Button>
            <Button colorScheme="blue" onClick={handleInstall}>
              {t('admin.confirm')}
            </Button>
          </ModalFooter>
        </ModalContent>
      </Modal>

      {/* Configure Plugin Modal */}
      <Modal isOpen={isConfigOpen} onClose={onConfigClose} size="lg">
        <ModalOverlay />
        <ModalContent>
          <ModalHeader>{t('admin.configurePlugin')}</ModalHeader>
          <ModalCloseButton />
          <ModalBody>
            {selectedPlugin && (
              <VStack spacing={4} align="stretch">
                <Text>
                  <strong>{t('admin.pluginName')}:</strong> {selectedPlugin.name}
                </Text>
                <Text>
                  <strong>{t('admin.pluginVersion')}:</strong> {selectedPlugin.version}
                </Text>
                <Text>
                  <strong>{t('admin.pluginStatus')}:</strong>{' '}
                  {selectedPlugin.enabled ? t('admin.pluginEnabled') : t('admin.pluginDisabled')}
                </Text>

                <Box borderTopWidth={1} pt={4}>
                  <Heading size="sm" mb={3}>{t('admin.pluginConfig')}</Heading>
                  <VStack spacing={3} align="stretch">
                    {Object.entries(editConfig).map(([key, value]) => (
                      <FormControl key={key}>
                        <FormLabel>{key}</FormLabel>
                        {typeof value === 'boolean' ? (
                          <Switch
                            isChecked={value}
                            onChange={(e) => setEditConfig({ ...editConfig, [key]: e.target.checked })}
                          />
                        ) : typeof value === 'number' ? (
                          <Input
                            type="number"
                            value={value}
                            onChange={(e) => setEditConfig({ ...editConfig, [key]: Number(e.target.value) })}
                          />
                        ) : (
                          <Input
                            value={String(value)}
                            onChange={(e) => setEditConfig({ ...editConfig, [key]: e.target.value })}
                          />
                        )}
                      </FormControl>
                    ))}
                    {Object.keys(editConfig).length === 0 && (
                      <Text fontSize="sm" color="gray.600">
                        {t('admin.noConfigFields')}
                      </Text>
                    )}
                  </VStack>
                </Box>
              </VStack>
            )}
          </ModalBody>
          <ModalFooter>
            <Button variant="ghost" mr={3} onClick={onConfigClose}>
              {t('admin.cancel')}
            </Button>
            <Button colorScheme="blue" onClick={handleSaveConfig}>
              {t('admin.save')}
            </Button>
          </ModalFooter>
        </ModalContent>
      </Modal>

      {/* Delete Confirmation Dialog */}
      <AlertDialog isOpen={isDeleteOpen} leastDestructiveRef={cancelRef} onClose={onDeleteClose}>
        <AlertDialogOverlay>
          <AlertDialogContent>
            <AlertDialogHeader fontSize="lg" fontWeight="bold">
              {t('admin.uninstallPlugin')}
            </AlertDialogHeader>

            <AlertDialogBody>
              {t('admin.confirmUninstall')}
            </AlertDialogBody>

            <AlertDialogFooter>
              <Button ref={cancelRef} onClick={onDeleteClose}>
                {t('admin.cancel')}
              </Button>
              <Button colorScheme="red" onClick={confirmDelete} ml={3}>
                {t('admin.delete')}
              </Button>
            </AlertDialogFooter>
          </AlertDialogContent>
        </AlertDialogOverlay>
      </AlertDialog>
    </Box>
  )
}
