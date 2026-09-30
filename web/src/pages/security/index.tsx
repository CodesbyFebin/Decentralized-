import React from 'react'
import { AppLayout } from '@/layouts/AppLayout'
import { Card } from '@/components/Card'

const Security: React.FC = () => {
  return (
    <AppLayout title="Security" subtitle="SSL certificates and security policies">
      <Card variant="glass">
        <div className="p-12 text-center">
          <h2 className="text-2xl font-bold text-neutral-300 mb-4">Security Management</h2>
          <p className="text-neutral-500">Coming in Phase 2 - Certificates, policies, and audit logs</p>
        </div>
      </Card>
    </AppLayout>
  )
}

export default Security
