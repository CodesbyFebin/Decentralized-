import React from 'react'
import { AppLayout } from '@/layouts/AppLayout'
import { Card } from '@/components/Card'

const Analytics: React.FC = () => {
  return (
    <AppLayout title="Analytics" subtitle="System metrics and analytics">
      <Card variant="glass">
        <div className="p-12 text-center">
          <h2 className="text-2xl font-bold text-neutral-300 mb-4">Analytics Dashboard</h2>
          <p className="text-neutral-500">Coming in Phase 2 - Traffic, errors, and performance metrics</p>
        </div>
      </Card>
    </AppLayout>
  )
}

export default Analytics
