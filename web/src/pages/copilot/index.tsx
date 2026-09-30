import React from 'react'
import { AppLayout } from '@/layouts/AppLayout'
import { Card } from '@/components/Card'

const Copilot: React.FC = () => {
  return (
    <AppLayout title="RAG Copilot" subtitle="AI-powered assistant">
      <Card variant="glass">
        <div className="p-12 text-center">
          <h2 className="text-2xl font-bold text-neutral-300 mb-4">RAG Copilot Assistant</h2>
          <p className="text-neutral-500">Coming in Phase 7 - AI-powered system assistant and RAG integration</p>
        </div>
      </Card>
    </AppLayout>
  )
}

export default Copilot
