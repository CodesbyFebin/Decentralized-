import React from 'react'
import { AppLayout } from '@/layouts/AppLayout'
import { Card } from '@/components/Card'

const Settings: React.FC = () => {
  return (
    <AppLayout title="Settings" subtitle="Configuration and preferences">
      <Card variant="glass">
        <div className="p-12 text-center">
          <h2 className="text-2xl font-bold text-neutral-300 mb-4">Settings</h2>
          <p className="text-neutral-500">Coming in Phase 2 - Configure system and user preferences</p>
        </div>
      </Card>
    </AppLayout>
  )
}

export default Settings
