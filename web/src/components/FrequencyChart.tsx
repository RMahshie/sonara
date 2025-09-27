import { useRef, useEffect, useState } from 'react'
import { line, curveMonotoneX } from 'd3-shape'

interface FrequencyData {
  frequency: number
  magnitude: number
}

interface FrequencyChartProps {
  data: FrequencyData[]
}

const FrequencyChart = ({ data }: FrequencyChartProps) => {
  const containerRef = useRef<HTMLDivElement>(null)
  const [width, setWidth] = useState(800) // fallback width

  // Chart dimensions
  const height = 400
  const padding = 60

  // Dynamic width calculation
  useEffect(() => {
    const updateWidth = () => {
      if (containerRef.current) {
        const containerWidth = containerRef.current.clientWidth
        // Always fit container to prevent overflow, with reasonable minimum for readability
        const usableWidth = Math.max(250, containerWidth) // 250px minimum for basic readability
        setWidth(usableWidth)
      }
    }

    // Initial calculation
    updateWidth()

    // Use ResizeObserver to watch container size changes immediately
    const resizeObserver = new ResizeObserver(updateWidth)
    if (containerRef.current) {
      resizeObserver.observe(containerRef.current)
    }

    // Keep window resize as fallback for edge cases
    window.addEventListener('resize', updateWidth)

    return () => {
      resizeObserver.disconnect()
      window.removeEventListener('resize', updateWidth)
    }
  }, [padding])

  // Fixed ranges for professional audio visualization
  const FREQ_MIN = 20
  const FREQ_MAX = 20000
  const DB_MIN = -25
  const DB_MAX = 25

  // Generate tick marks
  const generateFrequencyTicks = (): Array<{ x: number; label: string; freq: number }> => {
    const ticks: Array<{ x: number; label: string; freq: number }> = []
    const frequencies = [20, 50, 100, 200, 500, 1000, 2000, 5000, 10000, 20000]

    frequencies.forEach(freq => {
      if (freq >= FREQ_MIN && freq <= FREQ_MAX) {
        const x = xScale(freq)
        const label = freq >= 1000 ? `${(freq / 1000).toFixed(0)}k` : freq.toString()
        ticks.push({ x, label, freq })
      }
    })

    return ticks
  }

  const generateDbTicks = (): Array<{ y: number; label: string; db: number }> => {
    const ticks: Array<{ y: number; label: string; db: number }> = []
    const dbValues = [-25, -20, -15, -10, -5, 0, 5, 10, 15, 20, 25]

    dbValues.forEach(db => {
      const y = yScale(db)
      ticks.push({ y, label: db.toString(), db })
    })

    return ticks
  }

  // Scaling functions
  const xScale = (freq: number) => {
    const logFreq = Math.log10(Math.max(FREQ_MIN, Math.min(FREQ_MAX, freq)))
    const logMin = Math.log10(FREQ_MIN)
    const logMax = Math.log10(FREQ_MAX)
    return padding + ((logFreq - logMin) / (logMax - logMin)) * (width - 2 * padding)
  }

  const yScale = (db: number) => {
    const clampedDb = Math.max(DB_MIN, Math.min(DB_MAX, db))
    return height - padding - ((clampedDb - DB_MIN) / (DB_MAX - DB_MIN)) * (height - 2 * padding)
  }

  // Prepare data for rendering
  const filteredData = data
    .filter(d => d.frequency >= FREQ_MIN && d.frequency <= FREQ_MAX && !isNaN(d.magnitude))
    .sort((a, b) => a.frequency - b.frequency)

  // Generate smooth curve path using monotonic interpolation
  const generatePath = () => {
    if (filteredData.length === 0) return ''

    const lineGenerator = line<FrequencyData>()
      .x((d: FrequencyData) => xScale(d.frequency))
      .y((d: FrequencyData) => yScale(d.magnitude))
      .curve(curveMonotoneX)

    return lineGenerator(filteredData) || ''
  }

  const freqTicks = generateFrequencyTicks()
  const dbTicks = generateDbTicks()

  return (
    <div ref={containerRef} className="w-full">
      <svg width={width} height={height} className="border border-racing-green/20 rounded-lg bg-white">
        {/* Grid lines */}
        <defs>
          <pattern id="grid" width="20" height="20" patternUnits="userSpaceOnUse">
            <path d="M 20 0 L 0 0 0 20" fill="none" stroke="#004225" strokeWidth="0.3" opacity="0.05"/>
          </pattern>
        </defs>
        <rect width="100%" height="100%" fill="url(#grid)" />

        {/* Grid lines */}
        {freqTicks.map(tick => (
          <line
            key={`v-grid-${tick.freq}`}
            x1={tick.x}
            y1={padding}
            x2={tick.x}
            y2={height - padding}
            stroke="#004225"
            strokeWidth="0.5"
            opacity="0.1"
          />
        ))}
        {dbTicks.map(tick => (
          <line
            key={`h-grid-${tick.db}`}
            x1={padding}
            y1={tick.y}
            x2={width - padding}
            y2={tick.y}
            stroke="#004225"
            strokeWidth="0.5"
            opacity="0.1"
          />
        ))}

        {/* 0dB reference line */}
        <line
          x1={padding}
          y1={yScale(0)}
          x2={width - padding}
          y2={yScale(0)}
          stroke="#004225"
          strokeWidth="1"
          opacity="0.3"
        />

        {/* Axes */}
        <line x1={padding} y1={height - padding} x2={width - padding} y2={height - padding} stroke="#004225" strokeWidth="1"/>
        <line x1={padding} y1={padding} x2={padding} y2={height - padding} stroke="#004225" strokeWidth="1"/>

        {/* Data curve */}
        {filteredData.length > 0 && (
          <path
            d={generatePath()}
            fill="none"
            stroke="#b8860b"
            strokeWidth="2"
            strokeLinecap="round"
            strokeLinejoin="round"
          />
        )}

        {/* Data points (subtle) */}
        {filteredData.map((d, i) => (
          <circle
            key={`point-${i}`}
            cx={xScale(d.frequency)}
            cy={yScale(d.magnitude)}
            r="1.5"
            fill="#004225"
            opacity="0.6"
          />
        ))}

        {/* Frequency tick marks and labels */}
        {freqTicks.map(tick => (
          <g key={`freq-tick-${tick.freq}`}>
            <line
              x1={tick.x}
              y1={height - padding}
              x2={tick.x}
              y2={height - padding + 5}
              stroke="#004225"
              strokeWidth="1"
            />
            <text
              x={tick.x}
              y={height - padding + 18}
              textAnchor="middle"
              className="text-xs fill-current text-racing-green font-medium"
            >
              {tick.label}
            </text>
          </g>
        ))}

        {/* dB tick marks and labels */}
        {dbTicks.map(tick => (
          <g key={`db-tick-${tick.db}`}>
            <line
              x1={padding - 5}
              y1={tick.y}
              x2={padding}
              y2={tick.y}
              stroke="#004225"
              strokeWidth="1"
            />
            <text
              x={padding - 8}
              y={tick.y + 4}
              textAnchor="end"
              className="text-xs fill-current text-racing-green font-medium"
            >
              {tick.label}
            </text>
          </g>
        ))}

        {/* Axis labels */}
        <text
          x={width / 2}
          y={height - 10}
          textAnchor="middle"
          className="text-sm fill-current text-racing-green font-semibold"
        >
          Frequency (Hz)
        </text>
        <text
          x={15}
          y={height / 2}
          textAnchor="middle"
          transform={`rotate(-90 15 ${height / 2})`}
          className="text-sm fill-current text-racing-green font-semibold"
        >
          Magnitude (dB)
        </text>
      </svg>
    </div>
  )
}

export default FrequencyChart