import React from 'react'
import { Card } from '../Card'
import { AreaChart } from '../Charts/AreaChart'

interface MetricsChartProps {
  title: string
  data: Array<Record<string, unknown>>
  dataKey: string
  unit?: string
}

export const MetricsChart: React.FC<MetricsChartProps> = ({
  title,
  data,
  dataKey,
  unit = '%',
}) => {
  return (
    <Card variant="glass">
      <div className="p-6 border-b border-neutral-700">
        <h3 className="text-lg font-semibold text-white">{title}</h3>
      </div>
      <div className="p-6">
        <AreaChart data={data} dataKey={dataKey} name={title} height={250} />
      </div>
    </Card>
  )
}
