import React, { useState } from 'react'
import { Card } from '@/components/Card'
import { Button } from '@/components/Button'

interface NodeCreationWizardProps {
  isOpen: boolean
  onClose: () => void
  onSubmit: (config: NodeConfig) => Promise<void>
}

export interface NodeConfig {
  name: string
  nodeType: string
  isolationBoundary: string
  resourceLimit: number
}

type Step = 'basic' | 'isolation' | 'resources' | 'review'

export const NodeCreationWizard: React.FC<NodeCreationWizardProps> = ({
  isOpen,
  onClose,
  onSubmit,
}) => {
  const [step, setStep] = useState<Step>('basic')
  const [loading, setLoading] = useState(false)
  const [config, setConfig] = useState<NodeConfig>({
    name: '',
    nodeType: 'worker',
    isolationBoundary: 'container',
    resourceLimit: 4,
  })

  const steps: Step[] = ['basic', 'isolation', 'resources', 'review']
  const stepIndex = steps.indexOf(step)

  const handleNext = () => {
    if (stepIndex < steps.length - 1) {
      setStep(steps[stepIndex + 1])
    }
  }

  const handlePrev = () => {
    if (stepIndex > 0) {
      setStep(steps[stepIndex - 1])
    }
  }

  const handleSubmit = async () => {
    if (!config.name.trim()) return
    setLoading(true)
    try {
      await onSubmit(config)
      onClose()
      setConfig({ name: '', nodeType: 'worker', isolationBoundary: 'container', resourceLimit: 4 })
      setStep('basic')
    } finally {
      setLoading(false)
    }
  }

  if (!isOpen) return null

  return (
    <div className="fixed inset-0 bg-black/50 backdrop-blur-sm flex items-center justify-center z-50">
      <Card variant="glass" className="w-full max-w-md">
        {/* Header */}
        <div className="p-6 border-b border-neutral-700">
          <h2 className="text-xl font-bold text-white">Create New Node</h2>
          <div className="flex gap-1 mt-4">
            {steps.map((s, i) => (
              <div
                key={s}
                className={`h-1 flex-1 rounded-full ${
                  i <= stepIndex ? 'bg-primary-500' : 'bg-neutral-700'
                }`}
              />
            ))}
          </div>
        </div>

        {/* Content */}
        <div className="p-6 space-y-4 min-h-[280px]">
          {step === 'basic' && (
            <>
              <div>
                <label className="block text-sm font-medium text-neutral-300 mb-2">
                  Node Name
                </label>
                <input
                  type="text"
                  value={config.name}
                  onChange={(e) =>
                    setConfig({ ...config, name: e.target.value })
                  }
                  placeholder="e.g., worker-1, master-2"
                  className="w-full px-3 py-2 bg-neutral-800 border border-neutral-700 rounded-lg text-white placeholder-neutral-500 focus:border-primary-500 focus:outline-none"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-neutral-300 mb-2">
                  Node Type
                </label>
                <select
                  value={config.nodeType}
                  onChange={(e) =>
                    setConfig({ ...config, nodeType: e.target.value })
                  }
                  className="w-full px-3 py-2 bg-neutral-800 border border-neutral-700 rounded-lg text-white focus:border-primary-500 focus:outline-none"
                >
                  <option value="master">Master</option>
                  <option value="worker">Worker</option>
                  <option value="edge">Edge</option>
                </select>
              </div>
            </>
          )}

          {step === 'isolation' && (
            <>
              <div>
                <label className="block text-sm font-medium text-neutral-300 mb-2">
                  Isolation Boundary
                </label>
                <select
                  value={config.isolationBoundary}
                  onChange={(e) =>
                    setConfig({ ...config, isolationBoundary: e.target.value })
                  }
                  className="w-full px-3 py-2 bg-neutral-800 border border-neutral-700 rounded-lg text-white focus:border-primary-500 focus:outline-none"
                >
                  <option value="container">Container</option>
                  <option value="vm">Virtual Machine</option>
                  <option value="physical">Physical Host</option>
                </select>
              </div>
              <p className="text-xs text-neutral-400 mt-2">
                Isolation boundary determines the failure domain separation level.
              </p>
            </>
          )}

          {step === 'resources' && (
            <>
              <div>
                <label className="block text-sm font-medium text-neutral-300 mb-2">
                  CPU Limit (cores): {config.resourceLimit}
                </label>
                <input
                  type="range"
                  min="1"
                  max="16"
                  value={config.resourceLimit}
                  onChange={(e) =>
                    setConfig({ ...config, resourceLimit: parseInt(e.target.value) })
                  }
                  className="w-full"
                />
              </div>
              <div className="text-sm text-neutral-400">
                <p>Memory will be allocated proportionally to CPU limits.</p>
              </div>
            </>
          )}

          {step === 'review' && (
            <div className="space-y-3">
              <div className="bg-neutral-800/50 rounded-lg p-4 space-y-2">
                <p className="text-sm">
                  <span className="text-neutral-400">Name:</span>{' '}
                  <span className="text-white font-mono">{config.name}</span>
                </p>
                <p className="text-sm">
                  <span className="text-neutral-400">Type:</span>{' '}
                  <span className="text-white font-mono">{config.nodeType}</span>
                </p>
                <p className="text-sm">
                  <span className="text-neutral-400">Isolation:</span>{' '}
                  <span className="text-white font-mono">{config.isolationBoundary}</span>
                </p>
                <p className="text-sm">
                  <span className="text-neutral-400">CPU Limit:</span>{' '}
                  <span className="text-white font-mono">{config.resourceLimit} cores</span>
                </p>
              </div>
            </div>
          )}
        </div>

        {/* Footer */}
        <div className="p-6 border-t border-neutral-700 flex gap-3">
          <Button
            variant="ghost"
            size="md"
            onClick={onClose}
            className="flex-1"
          >
            Cancel
          </Button>
          {stepIndex > 0 && (
            <Button
              variant="secondary"
              size="md"
              onClick={handlePrev}
              className="flex-1"
            >
              Back
            </Button>
          )}
          {stepIndex < steps.length - 1 ? (
            <Button
              variant="primary"
              size="md"
              onClick={handleNext}
              disabled={step === 'basic' && !config.name.trim()}
              className="flex-1"
            >
              Next
            </Button>
          ) : (
            <Button
              variant="primary"
              size="md"
              onClick={handleSubmit}
              loading={loading}
              className="flex-1"
            >
              Create Node
            </Button>
          )}
        </div>
      </Card>
    </div>
  )
}
