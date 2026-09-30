import React from 'react'
import { AppLayout } from '@/layouts/AppLayout'
import { Card } from '@/components/Card'

const Compute: React.FC = () => {
  return (
    <AppLayout title="Compute Resources" subtitle="Resource allocation and workloads">
      <Card variant="glass">
        <div className="p-12 text-center">
          <h2 className="text-2xl font-bold text-neutral-300 mb-4">Compute Management</h2>
          <p className="text-neutral-500">Coming in Phase 3 - CPU, memory, and workload distribution</p>
        </div>
      </Card>
    </AppLayout>
  )
}

export default Compute
