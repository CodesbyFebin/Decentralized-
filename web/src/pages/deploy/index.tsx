import React from 'react'
import { AppLayout } from '@/layouts/AppLayout'
import { Card } from '@/components/Card'

const Deploy: React.FC = () => {
  return (
    <AppLayout title="Deployments" subtitle="Application deployment management">
      <Card variant="glass">
        <div className="p-12 text-center">
          <h2 className="text-2xl font-bold text-neutral-300 mb-4">Deployment Management</h2>
          <p className="text-neutral-500">Coming in Phase 2 - Deploy, rollback, and monitor applications</p>
        </div>
      </Card>
    </AppLayout>
  )
}

export default Deploy
