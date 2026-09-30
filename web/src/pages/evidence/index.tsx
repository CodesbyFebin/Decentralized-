import React from 'react'
import { AppLayout } from '@/layouts/AppLayout'
import { Card } from '@/components/Card'

const Evidence: React.FC = () => {
  return (
    <AppLayout title="Evidence" subtitle="Qualification and verification records">
      <Card variant="glass">
        <div className="p-12 text-center">
          <h2 className="text-2xl font-bold text-neutral-300 mb-4">Evidence Viewer</h2>
          <p className="text-neutral-500">Coming in Phase 7 - Qualification evidence, chaos results, and audit trails</p>
        </div>
      </Card>
    </AppLayout>
  )
}

export default Evidence
