import React from 'react'
import { AppLayout } from '@/layouts/AppLayout'
import { Card } from '@/components/Card'

const Storage: React.FC = () => {
  return (
    <AppLayout title="Storage" subtitle="Distributed storage management">
      <Card variant="glass">
        <div className="p-12 text-center">
          <h2 className="text-2xl font-bold text-neutral-300 mb-4">Storage Management</h2>
          <p className="text-neutral-500">Coming in Phase 2 - Storage buckets, backups, and replication</p>
        </div>
      </Card>
    </AppLayout>
  )
}

export default Storage
