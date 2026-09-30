import React from 'react'
import { AppLayout } from '@/layouts/AppLayout'
import { Card } from '@/components/Card'

const Team: React.FC = () => {
  return (
    <AppLayout title="Team" subtitle="Team members and permissions">
      <Card variant="glass">
        <div className="p-12 text-center">
          <h2 className="text-2xl font-bold text-neutral-300 mb-4">Team Management</h2>
          <p className="text-neutral-500">Coming in Phase 2 - Manage team members and roles</p>
        </div>
      </Card>
    </AppLayout>
  )
}

export default Team
