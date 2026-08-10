export const getPriorityLabelKey = (priority: string): string => {
  switch (priority) {
    case 'urgent': return 'pms.urgent'
    case 'high': return 'pms.high'
    case 'medium': return 'pms.medium'
    case 'low': return 'pms.low'
    default: return priority || 'pms.medium'
  }
}

export const getPriorityColor = (priority: string) => {
  switch (priority) {
    case 'urgent': return 'red'
    case 'high': return 'orange'
    case 'medium': return 'yellow'
    case 'low': return 'green'
    default: return 'gray'
  }
}

export const getStatusColor = (status: string) => {
  switch (status) {
    case 'open': return 'gray'
    case 'in_progress': return 'blue'
    case 'completed': return 'green'
    case 'done': return 'green'
    default: return 'gray'
  }
}
