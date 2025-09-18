interface FrequencyData {
  frequency: number
  amplitude: number
}

interface FrequencyChartProps {
  data: FrequencyData[]
}

const FrequencyChart = ({ data }: FrequencyChartProps) => {
  const width = 600
  const height = 300
  const padding = 40

  const maxAmplitude = Math.max(...data.map(d => d.amplitude))
  const minFreq = Math.min(...data.map(d => d.frequency))
  const maxFreq = Math.max(...data.map(d => d.frequency))

  const xScale = (freq: number) => ((Math.log10(freq) - Math.log10(minFreq)) / (Math.log10(maxFreq) - Math.log10(minFreq))) * (width - 2 * padding) + padding
  const yScale = (amp: number) => height - padding - (amp / maxAmplitude) * (height - 2 * padding)

  const points = data.map(d => `${xScale(d.frequency)},${yScale(d.amplitude)}`).join(' ')

  return (
    <div className="w-full overflow-x-auto">
      <svg width={width} height={height} className="border border-racing-green/20 rounded-lg">
        {/* Grid lines */}
        <defs>
          <pattern id="grid" width="40" height="20" patternUnits="userSpaceOnUse">
            <path d="M 40 0 L 0 0 0 20" fill="none" stroke="#004225" strokeWidth="0.5" opacity="0.1"/>
          </pattern>
        </defs>
        <rect width="100%" height="100%" fill="url(#grid)" />

        {/* Axes */}
        <line x1={padding} y1={height - padding} x2={width - padding} y2={height - padding} stroke="#004225" strokeWidth="2"/>
        <line x1={padding} y1={padding} x2={padding} y2={height - padding} stroke="#004225" strokeWidth="2"/>

        {/* Data line */}
        <polyline
          fill="none"
          stroke="#b8860b"
          strokeWidth="3"
          points={points}
        />

        {/* Data points */}
        {data.map((d, i) => (
          <circle
            key={i}
            cx={xScale(d.frequency)}
            cy={yScale(d.amplitude)}
            r="4"
            fill="#004225"
          />
        ))}

        {/* Labels */}
        <text x={width / 2} y={height - 10} textAnchor="middle" className="text-sm fill-current text-racing-green">
          Frequency (Hz)
        </text>
        <text x={15} y={height / 2} textAnchor="middle" transform={`rotate(-90 15 ${height / 2})`} className="text-sm fill-current text-racing-green">
          Amplitude
        </text>
      </svg>
    </div>
  )
}

export default FrequencyChart