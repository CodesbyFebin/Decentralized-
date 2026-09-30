import React from 'react'
import {
  LineChart as RechartLineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
} from 'recharts'

interface LineChartProps {
  data: Array<Record<string, unknown>>
  dataKey: string
  name: string
  stroke?: string
  height?: number
  xAxisKey?: string
}

export const LineChart: React.FC<LineChartProps> = ({
  data,
  dataKey,
  name,
  stroke = '#00d9ff',
  height = 300,
  xAxisKey = 'timestamp',
}) => {
  return (
    <ResponsiveContainer width="100%" height={height}>
      <RechartLineChart data={data} margin={{ top: 5, right: 30, left: 0, bottom: 5 }}>
        <CartesianGrid strokeDasharray="3 3" stroke="#404040" />
        <XAxis dataKey={xAxisKey} stroke="#737373" />
        <YAxis stroke="#737373" />
        <Tooltip
          contentStyle={{
            backgroundColor: '#1a1a1a',
            border: '1px solid #404040',
            borderRadius: '8px',
          }}
          cursor={{ stroke }}
        />
        <Line
          type="monotone"
          dataKey={dataKey}
          stroke={stroke}
          strokeWidth={2}
          dot={false}
          isAnimationActive={false}
          name={name}
        />
      </RechartLineChart>
    </ResponsiveContainer>
  )
}
