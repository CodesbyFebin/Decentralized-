import React from 'react'
import {
  BarChart as RechartBarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend,
  ResponsiveContainer,
} from 'recharts'

interface BarChartProps {
  data: Array<Record<string, unknown>>
  dataKeys: Array<{ key: string; name: string; fill: string }>
  height?: number
  xAxisKey?: string
}

export const BarChart: React.FC<BarChartProps> = ({
  data,
  dataKeys,
  height = 300,
  xAxisKey = 'name',
}) => {
  return (
    <ResponsiveContainer width="100%" height={height}>
      <RechartBarChart data={data} margin={{ top: 5, right: 30, left: 0, bottom: 5 }}>
        <CartesianGrid strokeDasharray="3 3" stroke="#404040" />
        <XAxis dataKey={xAxisKey} stroke="#737373" />
        <YAxis stroke="#737373" />
        <Tooltip
          contentStyle={{
            backgroundColor: '#1a1a1a',
            border: '1px solid #404040',
            borderRadius: '8px',
          }}
        />
        <Legend />
        {dataKeys.map((dk) => (
          <Bar
            key={dk.key}
            dataKey={dk.key}
            fill={dk.fill}
            name={dk.name}
            isAnimationActive={false}
          />
        ))}
      </RechartBarChart>
    </ResponsiveContainer>
  )
}
