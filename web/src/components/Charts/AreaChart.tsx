import React from 'react'
import {
  AreaChart as RechartAreaChart,
  Area,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
} from 'recharts'

interface AreaChartProps {
  data: Array<Record<string, unknown>>
  dataKey: string
  name: string
  fill?: string
  stroke?: string
  height?: number
  xAxisKey?: string
}

export const AreaChart: React.FC<AreaChartProps> = ({
  data,
  dataKey,
  name,
  fill = 'url(#colorGradient)',
  stroke = '#00d9ff',
  height = 300,
  xAxisKey = 'timestamp',
}) => {
  return (
    <ResponsiveContainer width="100%" height={height}>
      <RechartAreaChart data={data} margin={{ top: 5, right: 30, left: 0, bottom: 5 }}>
        <defs>
          <linearGradient id="colorGradient" x1="0" y1="0" x2="0" y2="1">
            <stop offset="5%" stopColor="#00d9ff" stopOpacity={0.3} />
            <stop offset="95%" stopColor="#00d9ff" stopOpacity={0} />
          </linearGradient>
        </defs>
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
        <Area
          type="monotone"
          dataKey={dataKey}
          stroke={stroke}
          fill={fill}
          isAnimationActive={false}
          name={name}
        />
      </RechartAreaChart>
    </ResponsiveContainer>
  )
}
